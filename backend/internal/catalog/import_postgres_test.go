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
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

func TestCatalogImportPostgres(t *testing.T) {
	dsn := os.Getenv("CATALOG_TEST_DATABASE_URL")
	ci := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn, ci = os.Getenv("DATABASE_URL"), true
	}
	if dsn == "" {
		t.Skip("set CATALOG_TEST_DATABASE_URL for a disposable candella_catalog_test database")
	}
	u, err := url.Parse(dsn)
	wantPath := "/candella_catalog_test"
	if ci {
		wantPath = "/candella"
	}
	if err != nil || u.Path != wantPath || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || (ci && (u.User == nil || u.User.Username() != "candella")) {
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
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("disposable DB migrations: %v", err)
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
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	userID, tenantA, tenantB := id.New(id.User), id.New(id.Tenant), id.New(id.Tenant)
	mustExec(`INSERT INTO users (id,email,full_name) VALUES ($1,$2,'CSV tester')`, userID, userID+"@example.test")
	for _, tenantID := range []string{tenantA, tenantB} {
		mustExec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'CSV shop',$2,'pos_general','PKR')`, tenantID, tenantID)
		mustExec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true)`, id.New(id.Store), tenantID)
	}
	svc := NewImportService(NewImportPostgresRepository(appPool))
	rows := []ImportRow{{Line: 2, ProductName: "School shirt", VariantName: "Small", SKU: "SHIRT-S", SellingPrice: "150.50", OpeningQuantity: "7"},
		{Line: 3, ProductName: "School shirt", VariantName: "Large", SKU: "SHIRT-L", SellingPrice: "200", OpeningQuantity: "0"}}
	preview, err := svc.Preview(ctx, tenantA, "create", "", rows)
	if err != nil || preview.CreateCount != 2 || preview.InvalidCount != 0 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	key := id.New("import")
	result, err := svc.Confirm(ctx, tenantA, userID, "create", "", key, preview.Hash, rows)
	if err != nil || result.Created != 2 || result.Replayed {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(result.Rows) != 2 || result.Rows[0].SKU != "SHIRT-S" || result.Rows[0].Action != "create" {
		t.Fatalf("missing row-level result: %+v", result.Rows)
	}
	replayed, err := svc.Confirm(ctx, tenantA, userID, "create", "", key, preview.Hash, rows)
	if err != nil || !replayed.Replayed || replayed.Created != 2 {
		t.Fatalf("retry=%+v err=%v", replayed, err)
	}
	if len(replayed.Rows) != 2 || replayed.Rows[0].SKU != "SHIRT-S" {
		t.Fatalf("retry lost row-level result: %+v", replayed.Rows)
	}
	var quantity, ledger, variants int64
	if err := pool.QueryRow(ctx, `SELECT s.quantity,COUNT(m.id) FROM stock_levels s JOIN stock_movements m ON m.tenant_id=s.tenant_id AND m.store_id=s.store_id AND m.variant_id=s.variant_id WHERE s.tenant_id=$1 GROUP BY s.quantity`, tenantA).Scan(&quantity, &ledger); err != nil || quantity != 7 || ledger != 1 {
		t.Fatalf("stock=%d movements=%d err=%v", quantity, ledger, err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM product_variants WHERE tenant_id=$1`, tenantA).Scan(&variants); err != nil || variants != 2 {
		t.Fatalf("variants=%d err=%v", variants, err)
	}
	duplicate, err := svc.Preview(ctx, tenantA, "create", "", rows)
	if err != nil || duplicate.InvalidCount != 2 {
		t.Fatalf("duplicate preview=%+v err=%v", duplicate, err)
	}
	other, err := svc.Preview(ctx, tenantB, "create", "", rows[:1])
	if err != nil || other.InvalidCount != 0 {
		t.Fatalf("cross-tenant preview=%+v err=%v", other, err)
	}
	otherExport, err := svc.Export(ctx, tenantB)
	if err != nil || len(otherExport) != 1 {
		t.Fatalf("cross-tenant export=%+v err=%v", otherExport, err)
	}
	update := []ImportRow{{Line: 2, ProductName: "School shirt", VariantName: "Small", SKU: "SHIRT-S", SellingPrice: "140", OpeningQuantity: "0"}}
	updatePreview, err := svc.Preview(ctx, tenantA, "update", "", update)
	if err != nil || updatePreview.UpdateCount != 1 {
		t.Fatalf("update preview=%+v err=%v", updatePreview, err)
	}
	updated, err := svc.Confirm(ctx, tenantA, userID, "update", "", id.New("import"), updatePreview.Hash, update)
	if err != nil || updated.Updated != 1 {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	var price int64
	if err := pool.QueryRow(ctx, `SELECT price_minor FROM product_variants WHERE tenant_id=$1 AND sku='SHIRT-S'`, tenantA).Scan(&price); err != nil || price != 14000 {
		t.Fatalf("price=%d err=%v", price, err)
	}
	if err := pool.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1`, tenantA).Scan(&quantity); err != nil || quantity != 7 {
		t.Fatalf("stock changed on update: %d err=%v", quantity, err)
	}
	update[0].OpeningQuantity = "3"
	bad, err := svc.Preview(ctx, tenantA, "update", "", update)
	if err != nil || bad.InvalidCount != 1 {
		t.Fatalf("stock update accepted: %+v err=%v", bad, err)
	}
}
