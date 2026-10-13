package catalog

import (
	"context"
	"database/sql"
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

// Only an explicitly named local disposable database or this repository's
// GitHub Actions PostgreSQL service may be migrated by this test.
func TestCreateProductWithOpeningStockPostgres(t *testing.T) {
	dsn := os.Getenv("CATALOG_TEST_DATABASE_URL")
	ciDatabase := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn = os.Getenv("DATABASE_URL")
		ciDatabase = true
	}
	if dsn == "" {
		t.Skip("set CATALOG_TEST_DATABASE_URL for a disposable candella_catalog_test database")
	}
	u, err := url.Parse(dsn)
	expectedDatabase := "/candella_catalog_test"
	if ciDatabase {
		expectedDatabase = "/candella"
	}
	if err != nil || u.Path != expectedDatabase || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		(ciDatabase && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("integration test requires a local disposable PostgreSQL database")
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
	userID, tenantA, tenantB, storeID := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant), id.New(id.Store)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO users (id,email,full_name) VALUES ($1,$2,'Catalog tester')`, userID, userID+"@example.test")
	for _, tenantID := range []string{tenantA, tenantB} {
		mustExec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Test shop',$2,'pos_general','USD')`, tenantID, tenantID)
	}
	mustExec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true)`, storeID, tenantA)
	svc := NewService(NewPostgresRepository(pool))
	opening := int64(7)
	product, err := svc.CreateProduct(ctx, tenantA, userID, CreateProductInput{
		Name: "Soap", Variant: CreateVariantInput{Name: "Blue", PriceMinor: 150}, InitialQuantity: &opening,
	})
	if err != nil || len(product.Variants) != 1 || product.Variants[0].Quantity != 7 || product.Variants[0].SKU == "" {
		t.Fatalf("product=%+v err=%v", product, err)
	}
	var balance, before, after, movementCount int64
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND variant_id=$2`, tenantA, product.Variants[0].ID).Scan(&balance); err != nil || balance != 7 {
		t.Fatalf("opening balance=%d err=%v", balance, err)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity_before,quantity_after FROM stock_movements WHERE tenant_id=$1 AND variant_id=$2 AND kind='receive'`, tenantA, product.Variants[0].ID).Scan(&before, &after); err != nil || before != 0 || after != 7 {
		t.Fatalf("opening movement=%d -> %d err=%v", before, after, err)
	}
	variant, err := svc.AddVariant(ctx, tenantA, product.ID, CreateVariantInput{Name: "Green", PriceMinor: 150})
	if err != nil || variant.SKU == "" || variant.SKU == product.Variants[0].SKU {
		t.Fatalf("new variant=%+v err=%v", variant, err)
	}
	otherProducts, err := svc.ListProducts(ctx, tenantB, "", 50, 0)
	if err != nil || len(otherProducts) != 0 {
		t.Fatalf("cross-tenant products=%+v err=%v", otherProducts, err)
	}
	// A tenant without a default store cannot receive opening stock. The product
	// and variant inserts must roll back with the failed stock operation.
	if _, err := svc.CreateProduct(ctx, tenantB, userID, CreateProductInput{
		Name: "Should roll back", Variant: CreateVariantInput{PriceMinor: 100}, InitialQuantity: &opening,
	}); err == nil {
		t.Fatal("expected missing-store error")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id=$1`, tenantB).Scan(&movementCount); err != nil || movementCount != 0 {
		t.Fatalf("failed creation left %d products, err=%v", movementCount, err)
	}

	variantID := product.Variants[0].ID
	receiveKey := id.New(id.StockMovement)
	receive := MovementInput{IdempotencyKey: receiveKey, VariantID: variantID, Kind: "receive", Quantity: ptr(3), Note: "Delivery 12"}
	first, created, err := svc.RecordMovement(ctx, tenantA, userID, receive)
	if err != nil || !created || first.QuantityBefore != 7 || first.QuantityAfter != 10 {
		t.Fatalf("first receipt=%+v created=%t err=%v", first, created, err)
	}
	replay, created, err := svc.RecordMovement(ctx, tenantA, userID, receive)
	if err != nil || created || replay.ID != first.ID || replay.QuantityAfter != first.QuantityAfter {
		t.Fatalf("receipt replay=%+v created=%t err=%v", replay, created, err)
	}
	changed := receive
	changed.Quantity = ptr(4)
	if _, _, err := svc.RecordMovement(ctx, tenantA, userID, changed); err != ErrMovementKeyReuse {
		t.Fatalf("changed request with same key: %v", err)
	}
	if _, _, err := svc.RecordMovement(ctx, tenantB, userID, receive); err != ErrNotFound {
		t.Fatalf("cross-tenant variant was accessible: %v", err)
	}

	count := MovementInput{IdempotencyKey: id.New(id.StockMovement), VariantID: variantID, Kind: "count", CountedQuantity: ptr(4), Note: "Physical count"}
	counted, created, err := svc.RecordMovement(ctx, tenantA, userID, count)
	if err != nil || !created || counted.Delta != -6 || counted.QuantityAfter != 4 {
		t.Fatalf("count correction=%+v created=%t err=%v", counted, created, err)
	}
	replay, created, err = svc.RecordMovement(ctx, tenantA, userID, count)
	if err != nil || created || replay.ID != counted.ID {
		t.Fatalf("count replay=%+v created=%t err=%v", replay, created, err)
	}

	concurrent := MovementInput{IdempotencyKey: id.New(id.StockMovement), VariantID: variantID, Kind: "receive", Quantity: ptr(2)}
	type result struct {
		movement *Movement
		created  bool
		err      error
	}
	results := make(chan result, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, isNew, e := svc.RecordMovement(ctx, tenantA, userID, concurrent)
			results <- result{m, isNew, e}
		}()
	}
	wg.Wait()
	close(results)
	newCount := 0
	var concurrentID string
	for res := range results {
		if res.err != nil || res.movement == nil {
			t.Fatalf("concurrent receipt error=%v", res.err)
		}
		if concurrentID != "" && res.movement.ID != concurrentID {
			t.Fatalf("concurrent receipts had different movements: %s and %s", concurrentID, res.movement.ID)
		}
		concurrentID = res.movement.ID
		if res.created {
			newCount++
		}
	}
	if newCount != 1 {
		t.Fatalf("concurrent receipts created %d movements, want 1", newCount)
	}

	var ledgerSum int64
	if err := pool.QueryRow(ctx, `SELECT s.quantity, COALESCE(SUM(m.delta),0), COUNT(m.id)
		FROM stock_levels s JOIN stock_movements m ON m.tenant_id=s.tenant_id AND m.store_id=s.store_id AND m.variant_id=s.variant_id
		WHERE s.tenant_id=$1 AND s.variant_id=$2 GROUP BY s.quantity`, tenantA, variantID).
		Scan(&balance, &ledgerSum, &movementCount); err != nil || balance != 6 || ledgerSum != balance || movementCount != 4 {
		t.Fatalf("balance=%d ledger=%d movements=%d err=%v", balance, ledgerSum, movementCount, err)
	}
	rows, err := pool.Query(ctx, `SELECT quantity_before, quantity_after, delta FROM stock_movements
		WHERE tenant_id=$1 AND variant_id=$2 ORDER BY created_at, id`, tenantA, variantID)
	if err != nil {
		t.Fatal(err)
	}
	previous := int64(0)
	for rows.Next() {
		var from, to, delta int64
		if err := rows.Scan(&from, &to, &delta); err != nil {
			t.Fatal(err)
		}
		if from != previous || to-from != delta {
			t.Fatalf("broken ledger link: %d -> %d, delta=%d, previous=%d", from, to, delta, previous)
		}
		previous = to
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if previous != balance {
		t.Fatalf("ledger ends at %d, balance %d", previous, balance)
	}

	archived := "archived"
	if _, err := svc.UpdateProduct(ctx, tenantA, product.ID, UpdateProductInput{Status: &archived}); err != nil {
		t.Fatal(err)
	}
	replay, created, err = svc.RecordMovement(ctx, tenantA, userID, receive)
	if err != nil || created || replay.ID != first.ID {
		t.Fatalf("archived retry=%+v created=%t err=%v", replay, created, err)
	}
	newReceive := receive
	newReceive.IdempotencyKey = id.New(id.StockMovement)
	if _, _, err := svc.RecordMovement(ctx, tenantA, userID, newReceive); err != ErrArchived {
		t.Fatalf("new receipt on archived product: %v", err)
	}

	mustExec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true)`, id.New(id.Store), tenantB)
	other, err := svc.CreateProduct(ctx, tenantB, userID, CreateProductInput{Name: "Other soap", Variant: CreateVariantInput{PriceMinor: 150}})
	if err != nil {
		t.Fatal(err)
	}
	otherReceive := MovementInput{IdempotencyKey: receiveKey, VariantID: other.Variants[0].ID, Kind: "receive", Quantity: ptr(1)}
	otherMovement, created, err := svc.RecordMovement(ctx, tenantB, userID, otherReceive)
	if err != nil || !created || otherMovement.QuantityAfter != 1 {
		t.Fatalf("tenant B receipt=%+v created=%t err=%v", otherMovement, created, err)
	}
}
