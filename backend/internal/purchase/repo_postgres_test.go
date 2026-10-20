package purchase

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

// Uses only a named local disposable database or this repository's CI PostgreSQL service.
func openPurchaseTestDB(t *testing.T) (context.Context, *pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("PURCHASE_TEST_DATABASE_URL")
	ci := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn = os.Getenv("DATABASE_URL")
		ci = true
	}
	if dsn == "" {
		t.Skip("set PURCHASE_TEST_DATABASE_URL for a disposable candella_purchase_test database")
	}
	u, err := url.Parse(dsn)
	expected := "/candella_purchase_test"
	if ci {
		expected = "/candella"
	}
	if err != nil || u.Path != expected || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || (ci && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("purchase integration test requires a local disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
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
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	app, err := database.Connect(ctx, dsn, database.AppRole)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	return ctx, admin, app
}
func TestPurchaseReceivingPostgres(t *testing.T) {
	ctx, admin, app := openPurchaseTestDB(t)
	userID, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	storeA, storeB := id.New(id.Store), id.New(id.Store)
	supplierA, supplierB := id.New(id.Supplier), id.New(id.Supplier)
	variantA, variantB := id.New(id.ProductVariant), id.New(id.ProductVariant)
	must := func(q string, args ...any) {
		t.Helper()
		if _, err := admin.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	must(`INSERT INTO users (id,email,full_name) VALUES ($1,$2,'Buyer')`, userID, userID+"@example.test")
	for _, tenantID := range []string{tenantA, tenantB} {
		must(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Shop',$2,'pos_general','PKR')`, tenantID, tenantID)
	}
	must(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$4,'Main',true)`, storeA, tenantA, storeB, tenantB)
	must(`INSERT INTO suppliers (id,tenant_id,name) VALUES ($1,$2,'Vendor A'),($3,$4,'Vendor B')`, supplierA, tenantA, supplierB, tenantB)
	productA, productB := id.New(id.Product), id.New(id.Product)
	must(`INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,'Soap'),($3,$4,'Other')`, productA, tenantA, productB, tenantB)
	must(`INSERT INTO product_variants (id,tenant_id,product_id,sku,price_minor) VALUES ($1,$2,$3,'SOAP',15000),($4,$5,$6,'OTHER',15000)`, variantA, tenantA, productA, variantB, tenantB, productB)
	must(`INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,1)`, id.New(id.StockLevel), tenantA, storeA, variantA)
	svc := NewService(NewPostgresRepository(app))
	input := CreateInput{StoreID: storeA, SupplierID: supplierA, IdempotencyKey: "order-create-001", Items: []LineInput{{VariantID: variantA, Quantity: 6, UnitCostMinor: 13500}}}
	if _, _, err := svc.Create(ctx, tenantA, userID, CreateInput{StoreID: storeB, SupplierID: supplierA, IdempotencyKey: "wrong-store-001", Items: input.Items}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-tenant store: %v", err)
	}
	if _, _, err := svc.Create(ctx, tenantA, userID, CreateInput{StoreID: storeA, SupplierID: supplierB, IdempotencyKey: "wrong-vendor-001", Items: input.Items}); !errors.Is(err, ErrSupplierUnavailable) {
		t.Fatalf("cross-tenant vendor: %v", err)
	}
	if _, _, err := svc.Create(ctx, tenantA, userID, CreateInput{StoreID: storeA, SupplierID: supplierA, IdempotencyKey: "wrong-product-01", Items: []LineInput{{VariantID: variantB, Quantity: 1, UnitCostMinor: 100}}}); !errors.Is(err, ErrVariantUnavailable) {
		t.Fatalf("cross-tenant product: %v", err)
	}
	order, created, err := svc.Create(ctx, tenantA, userID, input)
	if err != nil || !created || order.Status != "draft" || order.TotalCostMinor != 81000 || order.Items[0].ReceivedQuantity != 0 {
		t.Fatalf("draft: %+v %v", order, err)
	}
	var stock int64
	if err := admin.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantA, storeA, variantA).Scan(&stock); err != nil || stock != 1 {
		t.Fatalf("draft changed stock: %d %v", stock, err)
	}
	replay, created, err := svc.Create(ctx, tenantA, userID, input)
	if err != nil || created || replay.ID != order.ID {
		t.Fatalf("draft replay: %+v %v", replay, err)
	}
	changed := input
	changed.Items = []LineInput{{VariantID: variantA, Quantity: 5, UnitCostMinor: 13500}}
	if _, _, err := svc.Create(ctx, tenantA, userID, changed); !errors.Is(err, ErrKeyReuse) {
		t.Fatalf("changed draft replay: %v", err)
	}
	if _, _, _, err := svc.Receive(ctx, tenantA, order.ID, userID, ReceiveInput{IdempotencyKey: "before-approve-01", Items: []ReceiveLineInput{{VariantID: variantA, Quantity: 1}}}); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("draft receive: %v", err)
	}
	approved, changedApproval, err := svc.Approve(ctx, tenantA, order.ID, userID)
	if err != nil || !changedApproval || approved.Status != "approved" {
		t.Fatalf("approval: %+v %v", approved, err)
	}
	if _, changedApproval, err := svc.Approve(ctx, tenantA, order.ID, userID); err != nil || changedApproval {
		t.Fatalf("approval replay: %v %v", changedApproval, err)
	}
	first := ReceiveInput{IdempotencyKey: "receipt-one-001", Items: []ReceiveLineInput{{VariantID: variantA, Quantity: 2}}}
	partial, receipt, created, err := svc.Receive(ctx, tenantA, order.ID, userID, first)
	if err != nil || !created || partial.Status != "partially_received" || partial.Items[0].ReceivedQuantity != 2 || receipt.TotalCostMinor != 27000 {
		t.Fatalf("partial: %+v %+v %v", partial, receipt, err)
	}
	if err := admin.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantA, storeA, variantA).Scan(&stock); err != nil || stock != 3 {
		t.Fatalf("stock after partial: %d %v", stock, err)
	}
	var movements int
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE tenant_id=$1 AND goods_receipt_id=$2 AND kind='purchase'`, tenantA, receipt.ID).Scan(&movements); err != nil || movements != 1 {
		t.Fatalf("ledger: %d %v", movements, err)
	}
	retry, replayedReceipt, created, err := svc.Receive(ctx, tenantA, order.ID, userID, first)
	if err != nil || created || retry.ID != order.ID || replayedReceipt.ID != receipt.ID {
		t.Fatalf("receipt retry: %+v %+v %v", retry, replayedReceipt, err)
	}
	changedReceipt := first
	changedReceipt.Items = []ReceiveLineInput{{VariantID: variantA, Quantity: 1}}
	if _, _, _, err := svc.Receive(ctx, tenantA, order.ID, userID, changedReceipt); !errors.Is(err, ErrKeyReuse) {
		t.Fatalf("changed receipt retry: %v", err)
	}
	if _, _, _, err := svc.Receive(ctx, tenantA, order.ID, userID, ReceiveInput{IdempotencyKey: "too-many-001", Items: []ReceiveLineInput{{VariantID: variantA, Quantity: 5}}}); !errors.Is(err, ErrOverReceive) {
		t.Fatalf("over receive: %v", err)
	}
	must(`UPDATE products SET name='Renamed product' WHERE id=$1`, productA)
	full, _, created, err := svc.Receive(ctx, tenantA, order.ID, userID, ReceiveInput{IdempotencyKey: "receipt-two-001", Items: []ReceiveLineInput{{VariantID: variantA, Quantity: 4}}})
	if err != nil || !created || full.Status != "received" || full.Items[0].ProductName != "Soap" || full.Items[0].ReceivedQuantity != 6 {
		t.Fatalf("full receipt: %+v %v", full, err)
	}
	if _, _, _, err := svc.Receive(ctx, tenantA, order.ID, userID, ReceiveInput{IdempotencyKey: "receipt-three-01", Items: []ReceiveLineInput{{VariantID: variantA, Quantity: 1}}}); !errors.Is(err, ErrFullyReceived) {
		t.Fatalf("fully received: %v", err)
	}
	if _, err := svc.Find(ctx, tenantB, order.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant order: %v", err)
	}
	list, err := svc.List(store.WithSelection(ctx, storeA, userID, "admin"), tenantA, 50, 0)
	if err != nil || len(list) != 1 || list[0].ID != order.ID {
		t.Fatalf("store list: %+v %v", list, err)
	}
	if _, err := svc.List(store.WithSelection(ctx, storeB, userID, "admin"), tenantA, 50, 0); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-tenant store list: %v", err)
	}
	// Two deliveries for four units race; only one may consume the remaining quantity.
	must(`UPDATE products SET name='Soap' WHERE id=$1`, productA)
	secondInput := CreateInput{StoreID: storeA, SupplierID: supplierA, IdempotencyKey: "order-create-002", Items: []LineInput{{VariantID: variantA, Quantity: 4, UnitCostMinor: 12500}}}
	second, _, err := svc.Create(ctx, tenantA, userID, secondInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Approve(ctx, tenantA, second.ID, userID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, key := range []string{"race-receipt-01", "race-receipt-02"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, _, _, err := svc.Receive(context.Background(), tenantA, second.ID, userID, ReceiveInput{IdempotencyKey: key, Items: []ReceiveLineInput{{VariantID: variantA, Quantity: 4}}})
			results <- err
		}(key)
	}
	wg.Wait()
	close(results)
	success, over := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrFullyReceived) || errors.Is(err, ErrOverReceive) {
			over++
		} else {
			t.Fatalf("race receive: %v", err)
		}
	}
	if success != 1 || over != 1 {
		t.Fatalf("race success=%d rejected=%d", success, over)
	}
	if err := admin.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantA, storeA, variantA).Scan(&stock); err != nil || stock != 11 {
		t.Fatalf("race stock=%d %v", stock, err)
	}
}
