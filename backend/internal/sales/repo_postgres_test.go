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

	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/id"
)

// This integration test only runs against an explicitly named disposable database.
// It applies migrations there; it must never point at a developer or production DB.
func TestCashCheckoutPostgres(t *testing.T) {
	dsn := os.Getenv("SALES_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set SALES_TEST_DATABASE_URL for a disposable candella_sales_test database")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/candella_sales_test" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
		t.Fatal("SALES_TEST_DATABASE_URL must name a local disposable candella_sales_test database")
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
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("migrations on disposable test DB: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
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
	svc := NewService(NewPostgresRepository(pool))
	input := CheckoutInput{IdempotencyKey: "cash-sale-001", CashReceivedMinor: 500,
		Items: []CartItem{{VariantID: variantA, Quantity: 2}}}
	sale, created, err := svc.Checkout(ctx, tenantA, userID, input)
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
	replay, replayCreated, err := svc.Checkout(ctx, tenantA, userID, input)
	if err != nil || replayCreated || replay.ID != sale.ID {
		t.Fatalf("replay=%+v created=%v err=%v", replay, replayCreated, err)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND variant_id=$2`, tenantA, variantA).Scan(&stock); err != nil || stock != 3 {
		t.Fatalf("stock after replay=%d err=%v", stock, err)
	}
	changed := input
	changed.CashReceivedMinor = 600
	if _, _, err := svc.Checkout(ctx, tenantA, userID, changed); !errors.Is(err, ErrKeyReuse) {
		t.Fatalf("changed retry error=%v", err)
	}
	// Receipt snapshots survive catalog edits.
	mustExec(`UPDATE product_variants SET price_minor=999 WHERE id=$1`, variantA)
	receipt, err := svc.Receipt(ctx, tenantA, sale.ID)
	if err != nil || receipt.Items[0].UnitPriceMinor != 150 || receipt.GrandTotalMinor != 300 {
		t.Fatalf("receipt snapshot=%+v err=%v", receipt, err)
	}
	// Insufficient stock never persists an order, payment, or movement.
	_, _, err = svc.Checkout(ctx, tenantA, userID, CheckoutInput{IdempotencyKey: "cash-sale-002", CashReceivedMinor: 10000,
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
	_, _, err = svc.Checkout(ctx, tenantB, userID, CheckoutInput{IdempotencyKey: "other-tenant-001", CashReceivedMinor: 500,
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
			_, _, err := svc.Checkout(context.Background(), tenantA, userID, CheckoutInput{IdempotencyKey: key, CashReceivedMinor: 1000,
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
			sale, created, err := svc.Checkout(context.Background(), tenantA, userID,
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
}
