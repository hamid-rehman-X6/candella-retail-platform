package sales

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

// Only a named local disposable database or this repository's CI PostgreSQL
// service may be migrated by this test.
func TestCashCheckoutPostgres(t *testing.T) {
	dsn := os.Getenv("SALES_TEST_DATABASE_URL")
	ciDatabase := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn = os.Getenv("DATABASE_URL")
		ciDatabase = true
	}
	if dsn == "" {
		t.Skip("set SALES_TEST_DATABASE_URL for a disposable candella_sales_test database")
	}
	u, err := url.Parse(dsn)
	expectedDatabase := "/candella_sales_test"
	if ciDatabase {
		expectedDatabase = "/candella"
	}
	if err != nil || u.Path != expectedDatabase || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		(ciDatabase && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("sales integration test requires a local disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("migrations on disposable test DB: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Seed as the login; run the code under test as the API role, so RLS and
	// grants apply exactly as they do in the server.
	appPool, err := database.Connect(ctx, dsn, database.AppRole)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	userID, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	storeA, storeB := id.New(id.Store), id.New(id.Store)
	variantA, variantB, concurrentVariant := id.New(id.ProductVariant), id.New(id.ProductVariant), id.New(id.ProductVariant)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO users (id,email,full_name) VALUES ($1,$2,'Cashier')`, userID, userID+"@example.test")
	for _, tenantID := range []string{tenantA, tenantB} {
		mustExec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Test shop',$2,'pos_general','USD')`, tenantID, tenantID)
	}
	mustExec(`UPDATE tenants SET timezone='Asia/Karachi' WHERE id=$1`, tenantA)
	mustExec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$4,'Main',true)`, storeA, tenantA, storeB, tenantB)
	for _, seed := range []struct{ tenantID, variantID, storeID, sku string }{
		{tenantA, variantA, storeA, "A-SKU"}, {tenantB, variantB, storeB, "B-SKU"},
		{tenantA, concurrentVariant, storeA, "CONCURRENT-SKU"},
	} {
		productID := id.New(id.Product)
		mustExec(`INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,'Original name')`, productID, seed.tenantID)
		mustExec(`INSERT INTO product_variants (id,tenant_id,product_id,sku,price_minor) VALUES ($1,$2,$3,$4,150)`,
			seed.variantID, seed.tenantID, productID, seed.sku)
		mustExec(`INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,5)`,
			id.New(id.StockLevel), seed.tenantID, seed.storeID, seed.variantID)
	}
	svc := NewService(NewPostgresRepository(appPool))
	input := CheckoutInput{IdempotencyKey: "cash-sale-001", CashReceivedMinor: 500,
		Items: []CartItem{{VariantID: variantA, Quantity: 2}}}
	sale, created, err := svc.Checkout(ctx, tenantA, userID, "Cashier", input)
	if err != nil || !created || sale.GrandTotalMinor != 300 || sale.ChangeMinor != 200 || len(sale.Items) != 1 {
		t.Fatalf("sale=%+v created=%v err=%v", sale, created, err)
	}
	var stock, movementCount, paymentCount int
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND variant_id=$2`, tenantA, variantA).Scan(&stock); err != nil || stock != 3 {
		t.Fatalf("stock=%d err=%v", stock, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE tenant_id=$1 AND order_id=$2 AND kind='sale'`, tenantA, sale.ID).Scan(&movementCount); err != nil || movementCount != 1 {
		t.Fatalf("sale movements=%d err=%v", movementCount, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE tenant_id=$1 AND order_id=$2 AND amount_minor=300 AND cash_received_minor=500`, tenantA, sale.ID).Scan(&paymentCount); err != nil || paymentCount != 1 {
		t.Fatalf("payments=%d err=%v", paymentCount, err)
	}
	// A retry after a lost response returns the same receipt and does not move stock.
	replay, replayCreated, err := svc.Checkout(ctx, tenantA, userID, "Cashier", input)
	if err != nil || replayCreated || replay.ID != sale.ID {
		t.Fatalf("replay=%+v created=%v err=%v", replay, replayCreated, err)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND variant_id=$2`, tenantA, variantA).Scan(&stock); err != nil || stock != 3 {
		t.Fatalf("stock after replay=%d err=%v", stock, err)
	}
	changed := input
	changed.CashReceivedMinor = 600
	if _, _, err := svc.Checkout(ctx, tenantA, userID, "Cashier", changed); !errors.Is(err, ErrKeyReuse) {
		t.Fatalf("changed retry error=%v", err)
	}
	// Receipt snapshots survive catalog edits.
	mustExec(`UPDATE product_variants SET price_minor=999 WHERE id=$1`, variantA)
	receipt, err := svc.Receipt(ctx, tenantA, sale.ID)
	if err != nil || receipt.Items[0].UnitPriceMinor != 150 || receipt.GrandTotalMinor != 300 || receipt.CashierName != "Cashier" {
		t.Fatalf("receipt snapshot=%+v err=%v", receipt, err)
	}
	mustExec(`UPDATE users SET full_name='Renamed cashier' WHERE id=$1`, userID)
	mustExec(`UPDATE products SET name='Renamed product' WHERE tenant_id=$1`, tenantA)
	receipt, err = svc.Receipt(ctx, tenantA, sale.ID)
	if err != nil || receipt.Items[0].ProductName != "Original name" || receipt.CashierName != "Cashier" {
		t.Fatalf("receipt changed after edits: %+v err=%v", receipt, err)
	}
	day := sale.CreatedAt.In(time.FixedZone("PKT", 5*60*60)).Format("2006-01-02")
	list, more, err := svc.Search(ctx, tenantA, SalesFilter{FromDate: day, ToDate: day, OrderNumber: sale.OrderNumber, Cashier: "Cashier", MinAmountMinor: amount(300), MaxAmountMinor: amount(300), Limit: 1})
	if err != nil || more || len(list) != 1 || list[0].ID != sale.ID || list[0].CashierName != "Cashier" {
		t.Fatalf("filtered sales=%+v more=%v err=%v", list, more, err)
	}
	list, _, err = svc.Search(ctx, tenantB, SalesFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("cross-tenant sales=%+v err=%v", list, err)
	}
	// 23:30 UTC is already the next calendar day in this workspace.
	mustExec(`UPDATE orders SET created_at='2026-05-01 23:30:00+00' WHERE id=$1`, sale.ID)
	list, _, err = svc.Search(ctx, tenantA, SalesFilter{FromDate: "2026-05-02", ToDate: "2026-05-02", Limit: 1})
	if err != nil || len(list) != 1 || list[0].ID != sale.ID {
		t.Fatalf("workspace-date search=%+v err=%v", list, err)
	}
	list, _, err = svc.Search(ctx, tenantA, SalesFilter{FromDate: "2026-05-01", ToDate: "2026-05-01", Limit: 1})
	if err != nil || len(list) != 0 {
		t.Fatalf("prior workspace day included sale: %+v err=%v", list, err)
	}
	// Insufficient stock never persists an order, payment, or movement.
	_, _, err = svc.Checkout(ctx, tenantA, userID, "Cashier", CheckoutInput{IdempotencyKey: "cash-sale-002", CashReceivedMinor: 10000,
		Items: []CartItem{{VariantID: variantA, Quantity: 4}}})
	if e, ok := asSaleError(err); !ok || e.Code != "insufficient_stock" {
		t.Fatalf("insufficient stock error=%v", err)
	}
	var orderCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE tenant_id=$1`, tenantA).Scan(&orderCount); err != nil || orderCount != 1 {
		t.Fatalf("orders after failure=%d err=%v", orderCount, err)
	}
	// Same variant from another workspace is invisible for both sales and receipts.
	if _, err := svc.Receipt(ctx, tenantB, sale.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant receipt error=%v", err)
	}
	_, _, err = svc.Checkout(ctx, tenantB, userID, "Cashier", CheckoutInput{IdempotencyKey: "other-tenant-001", CashReceivedMinor: 500,
		Items: []CartItem{{VariantID: variantA, Quantity: 1}}})
	if e, ok := asSaleError(err); !ok || e.Code != "variant_not_found" {
		t.Fatalf("cross-tenant checkout error=%v", err)
	}
	// Two cashiers racing for five units can commit only one four-unit sale.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, key := range []string{"race-sale-001", "race-sale-002"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, _, err := svc.Checkout(context.Background(), tenantA, userID, "Cashier", CheckoutInput{IdempotencyKey: key, CashReceivedMinor: 1000,
				Items: []CartItem{{VariantID: concurrentVariant, Quantity: 4}}})
			results <- err
		}(key)
	}
	wg.Wait()
	close(results)
	successes, shortages := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if e, ok := asSaleError(err); ok && e.Code == "insufficient_stock" {
			shortages++
		} else {
			t.Fatalf("race checkout error=%v", err)
		}
	}
	if successes != 1 || shortages != 1 {
		t.Fatalf("race successes=%d shortages=%d", successes, shortages)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND variant_id=$2`, tenantA, concurrentVariant).Scan(&stock); err != nil || stock != 1 {
		t.Fatalf("race stock=%d err=%v", stock, err)
	}
	// Concurrent retries of one request share a single committed order.
	type checkoutResult struct {
		sale    *Sale
		created bool
		err     error
	}
	retries := make(chan checkoutResult, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sale, created, err := svc.Checkout(context.Background(), tenantA, userID, "Cashier",
				CheckoutInput{IdempotencyKey: "same-request-001", CashReceivedMinor: 150,
					Items: []CartItem{{VariantID: concurrentVariant, Quantity: 1}}})
			retries <- checkoutResult{sale, created, err}
		}()
	}
	wg.Wait()
	close(retries)
	createdCount := 0
	var commonID string
	for result := range retries {
		if result.err != nil || result.sale == nil {
			t.Fatalf("concurrent retry error=%v", result.err)
		}
		if commonID != "" && result.sale.ID != commonID {
			t.Fatalf("concurrent retries produced different orders: %s and %s", commonID, result.sale.ID)
		}
		commonID = result.sale.ID
		if result.created {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("concurrent retries created %d orders", createdCount)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND variant_id=$2`, tenantA, concurrentVariant).Scan(&stock); err != nil || stock != 0 {
		t.Fatalf("stock after concurrent retry=%d err=%v", stock, err)
	}
	list, more, err = svc.Search(ctx, tenantA, SalesFilter{Limit: 1})
	if err != nil || !more || len(list) != 1 {
		t.Fatalf("first page=%+v more=%v err=%v", list, more, err)
	}
	second, _, err := svc.Search(ctx, tenantA, SalesFilter{Limit: 1, Offset: 1})
	if err != nil || len(second) != 1 || second[0].ID == list[0].ID {
		t.Fatalf("second page=%+v err=%v", second, err)
	}
	branchID := id.New(id.Store)
	mustExec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Branch',false)`, branchID, tenantA)
	mustExec(`INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,2)`, id.New(id.StockLevel), tenantA, branchID, variantA)
	mustExec(`INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status) VALUES ($1,$2,$3,'member','active')`, id.New(id.TenantMembership), tenantA, userID)
	branchCtx := store.WithSelection(ctx, branchID, userID, "member")
	branchInput := CheckoutInput{StoreID: branchID, IdempotencyKey: "branch-sale-001", CashReceivedMinor: 1000, Items: []CartItem{{VariantID: variantA, Quantity: 1}}}
	if _, _, err := svc.Checkout(branchCtx, tenantA, userID, "Cashier", branchInput); err != store.ErrAccess {
		t.Fatalf("unassigned cashier sold branch stock: %v", err)
	}
	mustExec(`INSERT INTO store_access (tenant_id,store_id,user_id) VALUES ($1,$2,$3)`, tenantA, branchID, userID)
	branchSale, created, err := svc.Checkout(branchCtx, tenantA, userID, "Cashier", branchInput)
	if err != nil || !created || branchSale.StoreID != branchID || branchSale.StoreName != "Branch" {
		t.Fatalf("branch sale=%+v created=%t err=%v", branchSale, created, err)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantA, branchID, variantA).Scan(&stock); err != nil || stock != 1 {
		t.Fatalf("branch stock=%d err=%v", stock, err)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantA, storeA, variantA).Scan(&stock); err != nil || stock != 3 {
		t.Fatalf("main stock changed=%d err=%v", stock, err)
	}
	mustExec(`UPDATE stores SET name='Renamed branch' WHERE id=$1`, branchID)
	branchReceipt, err := svc.Receipt(ctx, tenantA, branchSale.ID)
	if err != nil || branchReceipt.StoreName != "Branch" {
		t.Fatalf("store receipt snapshot=%+v err=%v", branchReceipt, err)
	}
	if _, _, err := svc.Checkout(store.WithSelection(ctx, storeB, userID, "member"), tenantA, userID, "Cashier", CheckoutInput{StoreID: storeB, IdempotencyKey: "wrong-store-001", CashReceivedMinor: 1000, Items: []CartItem{{VariantID: variantA, Quantity: 1}}}); err != store.ErrNotFound {
		t.Fatalf("cross-tenant store sale: %v", err)
	}
	mustExec(`DELETE FROM store_access WHERE tenant_id=$1 AND store_id=$2 AND user_id=$3`, tenantA, branchID, userID)
	memberCtx := store.WithActor(ctx, userID, "member")
	if _, err := svc.Receipt(memberCtx, tenantA, branchSale.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked cashier read branch receipt: %v", err)
	}
	memberSales, _, err := svc.Search(memberCtx, tenantA, SalesFilter{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range memberSales {
		if item.ID == branchSale.ID {
			t.Fatal("revoked cashier saw branch sale in history")
		}
	}
}
