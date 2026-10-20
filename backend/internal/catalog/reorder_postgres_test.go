package catalog

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
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

func TestLowStockPostgres(t *testing.T) {
	dsn := os.Getenv("CATALOG_TEST_DATABASE_URL")
	ci := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn, ci = os.Getenv("DATABASE_URL"), true
	}
	if dsn == "" {
		t.Skip("set CATALOG_TEST_DATABASE_URL for a disposable candella_catalog_test database")
	}
	u, err := url.Parse(dsn)
	expected := "/candella_catalog_test"
	if ci {
		expected = "/candella"
	}
	if err != nil || u.Path != expected || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || (ci && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("reorder test requires a local disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	app, err := database.Connect(ctx, dsn, database.AppRole)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	must := func(q string, args ...any) {
		t.Helper()
		if _, err := admin.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	userID, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	storeA, storeOther, storeB := id.New(id.Store), id.New(id.Store), id.New(id.Store)
	productA, productB := id.New(id.Product), id.New(id.Product)
	variantA, variantB := id.New(id.ProductVariant), id.New(id.ProductVariant)
	supplierA, supplierB := id.New(id.Supplier), id.New(id.Supplier)
	must(`INSERT INTO users (id,email,full_name) VALUES ($1,$2,'Buyer')`, userID, userID+"@example.test")
	for _, tenantID := range []string{tenantA, tenantB} {
		must(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Shop',$2,'pos_general','PKR')`, tenantID, tenantID)
	}
	must(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$4,'Other',false),($5,$6,'Main',true)`, storeA, tenantA, storeOther, tenantA, storeB, tenantB)
	must(`INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,'Soap'),($3,$4,'Other')`, productA, tenantA, productB, tenantB)
	must(`INSERT INTO product_variants (id,tenant_id,product_id,sku,price_minor) VALUES ($1,$2,$3,'SOAP',100),($4,$5,$6,'OTHER',100)`, variantA, tenantA, productA, variantB, tenantB, productB)
	must(`INSERT INTO suppliers (id,tenant_id,name) VALUES ($1,$2,'Vendor A'),($3,$4,'Vendor B')`, supplierA, tenantA, supplierB, tenantB)
	must(`INSERT INTO supplier_variants (tenant_id,supplier_id,variant_id) VALUES ($1,$2,$3)`, tenantA, supplierA, variantA)
	svc := NewService(NewPostgresRepository(app))
	selected := func(storeID string) context.Context { return store.WithSelection(ctx, storeID, userID, "admin") }
	if err := svc.SaveReorder(selected(storeA), tenantA, variantA, ReorderInput{StoreID: storeA, Threshold: ptr(0)}); err != nil {
		t.Fatal(err)
	}
	low, err := svc.ListLowStock(selected(storeA), tenantA, 100, 0)
	if err != nil || len(low) != 1 || low[0].Quantity != 0 || low[0].SuggestedQuantity != 1 || low[0].PreferredSupplierID == nil || *low[0].PreferredSupplierID != supplierA {
		t.Fatalf("zero stock alert: %+v %v", low, err)
	}
	stock, _, err := svc.ListStock(selected(storeA), tenantA, "", 100, 0)
	if err != nil || len(stock) != 1 || stock[0].ReorderThreshold == nil || *stock[0].ReorderThreshold != 0 {
		t.Fatalf("stock setting: %+v %v", stock, err)
	}
	if _, _, err := svc.RecordMovement(selected(storeA), tenantA, userID, MovementInput{
		StoreID: storeA, IdempotencyKey: "reorder-receipt-001", VariantID: variantA,
		Kind: "receive", Quantity: ptr(1), Note: "Delivery",
	}); err != nil {
		t.Fatalf("receiving stock: %v", err)
	}
	low, err = svc.ListLowStock(selected(storeA), tenantA, 100, 0)
	if err != nil || len(low) != 0 {
		t.Fatalf("above zero threshold: %+v %v", low, err)
	}
	if err := svc.SaveReorder(selected(storeA), tenantA, variantA, ReorderInput{StoreID: storeA, Threshold: ptr(1), PreferredSupplierID: &supplierA}); err != nil {
		t.Fatal(err)
	}
	low, err = svc.ListLowStock(selected(storeA), tenantA, 100, 0)
	if err != nil || len(low) != 1 || low[0].Quantity != 1 || low[0].SuggestedQuantity != 1 {
		t.Fatalf("at threshold: %+v %v", low, err)
	}
	if err := svc.SaveReorder(selected(storeA), tenantA, variantA, ReorderInput{StoreID: storeA, Threshold: ptr(1), PreferredSupplierID: &supplierB}); !errors.Is(err, ErrSupplierNotActive) {
		t.Fatalf("cross-tenant supplier: %v", err)
	}
	if err := svc.SaveReorder(selected(storeB), tenantA, variantA, ReorderInput{StoreID: storeB, Threshold: ptr(1)}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-tenant store: %v", err)
	}
	if err := svc.SaveReorder(selected(storeA), tenantA, variantB, ReorderInput{StoreID: storeA, Threshold: ptr(1)}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant variant: %v", err)
	}
	if err := svc.SaveReorder(selected(storeOther), tenantA, variantA, ReorderInput{StoreID: storeOther, Threshold: ptr(2)}); err != nil {
		t.Fatal(err)
	}
	other, err := svc.ListLowStock(selected(storeOther), tenantA, 100, 0)
	if err != nil || len(other) != 1 || other[0].Quantity != 0 || *other[0].ReorderThreshold != 2 {
		t.Fatalf("other store: %+v %v", other, err)
	}
	if got, err := svc.ListLowStock(selected(storeB), tenantB, 100, 0); err != nil || len(got) != 0 {
		t.Fatalf("other tenant: %+v %v", got, err)
	}
	must(`UPDATE products SET status='archived' WHERE id=$1`, productA)
	if got, err := svc.ListLowStock(selected(storeA), tenantA, 100, 0); err != nil || len(got) != 0 {
		t.Fatalf("archived product: %+v %v", got, err)
	}
	if err := svc.SaveReorder(selected(storeA), tenantA, variantA, ReorderInput{StoreID: storeA, Threshold: ptr(2)}); !errors.Is(err, ErrArchived) {
		t.Fatalf("archived setting: %v", err)
	}
	if err := svc.SaveReorder(selected(storeA), tenantA, variantA, ReorderInput{StoreID: storeA}); err != nil {
		t.Fatal(err)
	}
	must(`UPDATE products SET status='active' WHERE id=$1`, productA)
	if got, err := svc.ListLowStock(selected(storeA), tenantA, 100, 0); err != nil || len(got) != 0 {
		t.Fatalf("disabled alert: %+v %v", got, err)
	}
}
