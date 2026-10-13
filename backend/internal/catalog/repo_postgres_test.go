package catalog

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/id"
)

// Only an explicitly named local disposable database may be migrated by this test.
func TestCreateProductWithOpeningStockPostgres(t *testing.T) {
	dsn := os.Getenv("CATALOG_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set CATALOG_TEST_DATABASE_URL for a disposable candella_catalog_test database")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Path != "/candella_catalog_test" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
		t.Fatal("CATALOG_TEST_DATABASE_URL must name a local disposable candella_catalog_test database")
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
}
