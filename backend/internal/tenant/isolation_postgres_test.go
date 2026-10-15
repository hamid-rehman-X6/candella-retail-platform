package tenant_test

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

// tenantTables holds every table that stores one tenant's retail data. Each must
// enforce RLS; the schema guard below fails when a new tenant table is added
// without either joining this list or being declared control plane.
var tenantTables = []string{
	"stores", "store_access", "products", "product_variants", "stock_levels",
	"stock_movements", "document_sequences", "orders", "order_items", "payments",
	"suppliers", "supplier_variants", "catalog_import_batches", "registers", "register_sessions",
}

// controlPlaneTables carry a tenant_id but are read before a tenant is in scope:
// membership resolution, invitation acceptance by token, and the audit trail.
// Access to them is filtered in the application, and the API cannot read the
// audit trail at all.
var controlPlaneTables = []string{"tenant_memberships", "staff_invitations", "audit_logs"}

const (
	permissionDenied = "42501" // also raised by an RLS WITH CHECK violation
	foreignKey       = "23503"
)

type isolationEnv struct {
	ctx      context.Context
	dsn      string
	admin    *pgxpool.Pool // the login: seeds and checks, bypassing RLS
	app      *pgxpool.Pool
	platform *pgxpool.Pool
}

// Only a named local disposable database or this repository's CI PostgreSQL
// service may be used; the test creates tenants and migrates the schema.
func setupIsolation(t *testing.T) *isolationEnv {
	t.Helper()
	dsn := os.Getenv("ISOLATION_TEST_DATABASE_URL")
	ciDatabase := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn = os.Getenv("DATABASE_URL")
		ciDatabase = true
	}
	if dsn == "" {
		t.Skip("set ISOLATION_TEST_DATABASE_URL for a disposable candella_isolation_test database")
	}
	u, err := url.Parse(dsn)
	expectedDatabase := "/candella_isolation_test"
	if ciDatabase {
		expectedDatabase = "/candella"
	}
	if err != nil || u.Path != expectedDatabase || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		(ciDatabase && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("isolation test requires a local disposable PostgreSQL database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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

	env := &isolationEnv{ctx: ctx, dsn: dsn}
	if env.admin, err = pgxpool.New(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(env.admin.Close)
	if env.app, err = database.Connect(ctx, dsn, database.AppRole); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(env.app.Close)
	if env.platform, err = database.Connect(ctx, dsn, database.PlatformRole); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(env.platform.Close)
	return env
}

type seededTenant struct {
	tenantID, userID, storeID, productID, variantID, orderID, supplierID, registerID, shiftID string
}

// seedTenant writes one row of every tenant table for a new tenant, as the
// login, so the tests can probe each table from the restricted roles.
func seedTenant(t *testing.T, env *isolationEnv) seededTenant {
	t.Helper()
	s := seededTenant{
		tenantID: id.New(id.Tenant), userID: id.New(id.User), storeID: id.New(id.Store),
		productID: id.New(id.Product), variantID: id.New(id.ProductVariant),
		orderID: id.New(id.Order), supplierID: id.New(id.Supplier),
		registerID: id.New(id.Register), shiftID: id.New(id.RegisterSession),
	}
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, email, full_name) VALUES ($1, $1 || '@example.test', 'Owner')`, []any{s.userID}},
		{`INSERT INTO tenants (id, name, slug, industry_type, currency_code, created_by)
		  VALUES ($1, 'Shop', $1, 'pos_general', 'PKR', $2)`, []any{s.tenantID, s.userID}},
		{`INSERT INTO tenant_memberships (id, tenant_id, user_id, role, status)
		  VALUES ($1, $2, $3, 'owner', 'active')`, []any{id.New(id.TenantMembership), s.tenantID, s.userID}},
		{`INSERT INTO stores (id, tenant_id, name, is_default) VALUES ($1, $2, 'Main', true)`, []any{s.storeID, s.tenantID}},
		{`INSERT INTO store_access (tenant_id, store_id, user_id) VALUES ($1, $2, $3)`, []any{s.tenantID, s.storeID, s.userID}},
		{`INSERT INTO products (id, tenant_id, name) VALUES ($1, $2, 'Original')`, []any{s.productID, s.tenantID}},
		{`INSERT INTO product_variants (id, tenant_id, product_id, sku, price_minor)
		  VALUES ($1, $2, $3, $1, 15000)`, []any{s.variantID, s.tenantID, s.productID}},
		{`INSERT INTO stock_levels (id, tenant_id, store_id, variant_id, quantity)
		  VALUES ($1, $2, $3, $4, 9)`, []any{id.New(id.StockLevel), s.tenantID, s.storeID, s.variantID}},
		{`INSERT INTO document_sequences (tenant_id) VALUES ($1)`, []any{s.tenantID}},
		{`INSERT INTO registers (id, tenant_id, store_id, name) VALUES ($1, $2, $3, 'Counter 1')`,
			[]any{s.registerID, s.tenantID, s.storeID}},
		{`INSERT INTO register_sessions (id, tenant_id, store_id, register_id, register_name_snapshot,
		    opened_by, opener_name_snapshot, opening_cash_minor)
		  VALUES ($1, $2, $3, $4, 'Counter 1', $5, 'Owner', 5000)`,
			[]any{s.shiftID, s.tenantID, s.storeID, s.registerID, s.userID}},
		{`INSERT INTO orders (id, tenant_id, store_id, order_number, idempotency_key, request_hash,
		    currency_code, subtotal_minor, grand_total_minor, created_by, store_name_snapshot, register_session_id)
		  VALUES ($1, $2, $3, 'POS-000001', $1, repeat('a', 64), 'PKR', 15000, 15000, $4, 'Main', $5)`,
			[]any{s.orderID, s.tenantID, s.storeID, s.userID, s.shiftID}},
		{`INSERT INTO order_items (id, tenant_id, order_id, variant_id, product_name_snapshot,
		    variant_name_snapshot, sku_snapshot, quantity, unit_price_minor, line_total_minor)
		  VALUES ($1, $2, $3, $4, 'Original', 'Default', 'SKU', 1, 15000, 15000)`,
			[]any{id.New(id.OrderItem), s.tenantID, s.orderID, s.variantID}},
		{`INSERT INTO payments (id, tenant_id, order_id, method, status, amount_minor,
		    cash_received_minor, change_minor, currency_code)
		  VALUES ($1, $2, $3, 'cash', 'completed', 15000, 20000, 5000, 'PKR')`,
			[]any{id.New(id.Payment), s.tenantID, s.orderID}},
		{`INSERT INTO stock_movements (id, tenant_id, store_id, variant_id, kind, delta,
		    quantity_before, quantity_after, order_id)
		  VALUES ($1, $2, $3, $4, 'sale', -1, 10, 9, $5)`,
			[]any{id.New(id.StockMovement), s.tenantID, s.storeID, s.variantID, s.orderID}},
		{`INSERT INTO suppliers (id, tenant_id, name) VALUES ($1, $2, 'Supplier')`, []any{s.supplierID, s.tenantID}},
		{`INSERT INTO supplier_variants (tenant_id, supplier_id, variant_id) VALUES ($1, $2, $3)`,
			[]any{s.tenantID, s.supplierID, s.variantID}},
		{`INSERT INTO catalog_import_batches (id, tenant_id, idempotency_key, request_hash, mode, created_by)
		  VALUES ($1, $2, $1, repeat('b', 64), 'create', $3)`,
			[]any{id.New(id.Prefix("cimp")), s.tenantID, s.userID}},
	}
	for _, st := range statements {
		if _, err := env.admin.Exec(env.ctx, st.query, st.args...); err != nil {
			t.Fatalf("seeding %q: %v", st.query, err)
		}
	}
	return s
}

// inTenant runs one statement as the API would for tenantID.
func inTenant(env *isolationEnv, tenantID, query string, args ...any) (int64, error) {
	var affected int64
	err := database.WithTenantTx(env.ctx, env.app, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(env.ctx, query, args...)
		affected = tag.RowsAffected()
		return err
	})
	return affected, err
}

func countInTenant(t *testing.T, env *isolationEnv, tenantID, query string, args ...any) int {
	t.Helper()
	var n int
	err := database.WithTenantTx(env.ctx, env.app, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(env.ctx, query, args...).Scan(&n)
	})
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func expectCode(t *testing.T, err error, code, what string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("%s: err = %v, want SQLSTATE %s", what, err, code)
	}
}

func TestTenantIsolation(t *testing.T) {
	env := setupIsolation(t)
	a, b := seedTenant(t, env), seedTenant(t, env)
	both := []string{a.tenantID, b.tenantID}

	t.Run("every tenant table enforces RLS", func(t *testing.T) {
		rows, err := env.admin.Query(env.ctx, `
			SELECT c.relname, c.relrowsecurity, c.relforcerowsecurity,
			       EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid = c.oid
			               AND pg_get_expr(p.polqual, p.polrelid) LIKE '%app.current_tenant_id%'
			               AND pg_get_expr(p.polwithcheck, p.polrelid) LIKE '%app.current_tenant_id%')
			FROM pg_class c
			JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped
			WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r', 'p')`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		seen := 0
		for rows.Next() {
			var name string
			var rls, force, policy bool
			if err := rows.Scan(&name, &rls, &force, &policy); err != nil {
				t.Fatal(err)
			}
			switch {
			case slices.Contains(controlPlaneTables, name):
			case !slices.Contains(tenantTables, name):
				t.Errorf("%s has a tenant_id but is neither a covered tenant table nor declared control plane", name)
			case !rls || !force || !policy:
				t.Errorf("%s: rls=%v force=%v tenant policy=%v, want all true", name, rls, force, policy)
			default:
				seen++
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if seen != len(tenantTables) {
			t.Errorf("found %d of %d tenant tables", seen, len(tenantTables))
		}
	})

	t.Run("foreign keys between tenant tables include tenant_id", func(t *testing.T) {
		rows, err := env.admin.Query(env.ctx, `
			SELECT c.conrelid::regclass::text, pg_get_constraintdef(c.oid)
			FROM pg_constraint c
			WHERE c.contype = 'f' AND c.connamespace = 'public'::regnamespace
			  AND EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid = c.conrelid AND attname = 'tenant_id')
			  AND EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid = c.confrelid AND attname = 'tenant_id')
			  AND NOT EXISTS (
			    SELECT 1 FROM unnest(c.conkey, c.confkey) AS k(src, dst)
			    JOIN pg_attribute s ON s.attrelid = c.conrelid AND s.attnum = k.src AND s.attname = 'tenant_id'
			    JOIN pg_attribute d ON d.attrelid = c.confrelid AND d.attnum = k.dst AND d.attname = 'tenant_id')`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var table, def string
			if err := rows.Scan(&table, &def); err != nil {
				t.Fatal(err)
			}
			t.Errorf("%s: %s can point at another tenant's row; key it on tenant_id too", table, def)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no tenant in scope sees no tenant rows", func(t *testing.T) {
		// A repository that forgets to scope its query fails closed.
		for _, table := range tenantTables {
			var n int
			if err := env.app.QueryRow(env.ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil {
				t.Fatalf("%s: %v", table, err)
			}
			if n != 0 {
				t.Errorf("%s: %d rows visible without a tenant", table, n)
			}
		}
	})

	t.Run("a tenant sees only its own rows", func(t *testing.T) {
		for _, table := range tenantTables {
			own := countInTenant(t, env, a.tenantID, `SELECT count(*) FROM `+table+` WHERE tenant_id = $1`, a.tenantID)
			other := countInTenant(t, env, a.tenantID, `SELECT count(*) FROM `+table+` WHERE tenant_id <> $1`, a.tenantID)
			if own == 0 || other != 0 {
				t.Errorf("%s as tenant A: own=%d other=%d, want own>0 other=0", table, own, other)
			}
		}
	})

	t.Run("a tenant cannot change another tenant's catalog, stock or sales", func(t *testing.T) {
		changes := []string{
			`UPDATE products SET name = 'Hijacked' WHERE tenant_id = $1`,
			`UPDATE product_variants SET price_minor = 1 WHERE tenant_id = $1`,
			`UPDATE stock_levels SET quantity = 999 WHERE tenant_id = $1`,
			`UPDATE stores SET name = 'Hijacked' WHERE tenant_id = $1`,
			`UPDATE suppliers SET name = 'Hijacked' WHERE tenant_id = $1`,
			`UPDATE document_sequences SET current_number = 999 WHERE tenant_id = $1`,
			`DELETE FROM store_access WHERE tenant_id = $1`,
			`UPDATE register_sessions SET status = 'closed' WHERE tenant_id = $1`,
			`UPDATE registers SET name = 'Hijacked' WHERE tenant_id = $1`,
		}
		for _, change := range changes {
			affected, err := inTenant(env, a.tenantID, change, b.tenantID)
			if err != nil || affected != 0 {
				t.Errorf("%s as tenant A: affected=%d err=%v, want 0 rows", change, affected, err)
			}
		}
		var name string
		var quantity, price int64
		if err := env.admin.QueryRow(env.ctx, `
			SELECT p.name, v.price_minor, s.quantity FROM products p
			JOIN product_variants v ON v.tenant_id = p.tenant_id AND v.product_id = p.id
			JOIN stock_levels s ON s.tenant_id = v.tenant_id AND s.variant_id = v.id
			WHERE p.id = $1`, b.productID).Scan(&name, &price, &quantity); err != nil {
			t.Fatal(err)
		}
		if name != "Original" || price != 15000 || quantity != 9 {
			t.Fatalf("tenant B changed: name=%q price=%d quantity=%d", name, price, quantity)
		}
	})

	t.Run("a tenant cannot write rows into another tenant", func(t *testing.T) {
		_, err := inTenant(env, a.tenantID, `INSERT INTO products (id, tenant_id, name) VALUES ($1, $2, 'Planted')`,
			id.New(id.Product), b.tenantID)
		expectCode(t, err, permissionDenied, "product for tenant B")
		_, err = inTenant(env, a.tenantID, `
			INSERT INTO orders (id, tenant_id, store_id, order_number, idempotency_key, request_hash,
			  currency_code, subtotal_minor, grand_total_minor, created_by, store_name_snapshot)
			VALUES ($1, $2, $3, 'POS-999999', $1, repeat('c', 64), 'PKR', 0, 0, $4, 'Main')`,
			id.New(id.Order), b.tenantID, b.storeID, a.userID)
		expectCode(t, err, permissionDenied, "sale for tenant B")
		_, err = inTenant(env, a.tenantID, `
			INSERT INTO stock_movements (id, tenant_id, store_id, variant_id, kind, delta, quantity_before, quantity_after)
			VALUES ($1, $2, $3, $4, 'receive', 5, 9, 14)`,
			id.New(id.StockMovement), b.tenantID, b.storeID, b.variantID)
		expectCode(t, err, permissionDenied, "stock movement for tenant B")
	})

	t.Run("composite keys stop references to another tenant's rows", func(t *testing.T) {
		_, err := inTenant(env, a.tenantID, `
			INSERT INTO stock_levels (id, tenant_id, store_id, variant_id, quantity) VALUES ($1, $2, $3, $4, 1)`,
			id.New(id.StockLevel), a.tenantID, a.storeID, b.variantID)
		expectCode(t, err, foreignKey, "stock for tenant B's variant")
		_, err = inTenant(env, a.tenantID, `
			INSERT INTO order_items (id, tenant_id, order_id, variant_id, product_name_snapshot,
			  variant_name_snapshot, sku_snapshot, quantity, unit_price_minor, line_total_minor)
			VALUES ($1, $2, $3, $4, 'X', 'X', 'X', 1, 1, 1)`,
			id.New(id.OrderItem), a.tenantID, b.orderID, a.variantID)
		expectCode(t, err, foreignKey, "line on tenant B's order")
		_, err = inTenant(env, a.tenantID, `
			INSERT INTO register_sessions (id, tenant_id, store_id, register_id, register_name_snapshot,
			  opened_by, opener_name_snapshot, opening_cash_minor)
			VALUES ($1, $2, $3, $4, 'X', $5, 'X', 0)`,
			id.New(id.RegisterSession), a.tenantID, a.storeID, b.registerID, a.userID)
		expectCode(t, err, foreignKey, "shift on tenant B's counter")
	})

	t.Run("ledgers and the audit trail are append-only for the API", func(t *testing.T) {
		for _, query := range []string{
			`UPDATE stock_movements SET delta = delta WHERE tenant_id = $1`,
			`DELETE FROM stock_movements WHERE tenant_id = $1`,
			`UPDATE orders SET grand_total_minor = 0 WHERE tenant_id = $1`,
			`DELETE FROM orders WHERE tenant_id = $1`,
			`UPDATE payments SET amount_minor = 0 WHERE tenant_id = $1`,
			`DELETE FROM order_items WHERE tenant_id = $1`,
			`DELETE FROM audit_logs WHERE tenant_id = $1`,
			`UPDATE audit_logs SET action = 'x' WHERE tenant_id = $1`,
			`SELECT count(*) FROM audit_logs WHERE tenant_id = $1`,
		} {
			_, err := inTenant(env, a.tenantID, query, a.tenantID)
			expectCode(t, err, permissionDenied, query)
		}
		_, err := env.app.Exec(env.ctx, `SELECT count(*) FROM platform_admins`)
		expectCode(t, err, permissionDenied, "API reading platform_admins")
	})

	t.Run("the platform role reads every tenant", func(t *testing.T) {
		for _, table := range tenantTables {
			var tenants int
			if err := env.platform.QueryRow(env.ctx,
				`SELECT count(DISTINCT tenant_id) FROM `+table+` WHERE tenant_id = ANY($1)`, both).Scan(&tenants); err != nil {
				t.Fatalf("%s: %v", table, err)
			}
			if tenants != 2 {
				t.Errorf("platform sees %d of 2 tenants in %s", tenants, table)
			}
		}
		var members int
		if err := env.platform.QueryRow(env.ctx, `
			SELECT count(*) FROM tenant_memberships m JOIN users u ON u.id = m.user_id
			WHERE m.tenant_id = ANY($1)`, both).Scan(&members); err != nil || members != 2 {
			t.Fatalf("members=%d err=%v", members, err)
		}
	})

	t.Run("the platform role cannot write or read secrets", func(t *testing.T) {
		for _, query := range []string{
			`UPDATE products SET name = 'Hijacked' WHERE tenant_id = $1`,
			`DELETE FROM orders WHERE tenant_id = $1`,
			`INSERT INTO stores (id, tenant_id, name) VALUES ('str_platform0001', $1, 'Planted')`,
			`UPDATE tenants SET status = 'suspended' WHERE id = $1`,
			`UPDATE tenant_memberships SET role = 'owner' WHERE tenant_id = $1`,
			`INSERT INTO audit_logs (id, tenant_id, actor_type, action) VALUES ('alog_platform001', $1, 'system', 'x')`,
		} {
			_, err := env.platform.Exec(env.ctx, query, a.tenantID)
			expectCode(t, err, permissionDenied, "platform: "+query)
		}
		for _, query := range []string{
			`SELECT password_hash FROM users LIMIT 1`,
			`SELECT token_hash FROM sessions LIMIT 1`,
			`SELECT totp_secret_encrypted FROM user_mfa LIMIT 1`,
			`SELECT count(*) FROM mfa_backup_codes`,
			`SELECT count(*) FROM password_reset_tokens`,
			`SELECT count(*) FROM verification_codes`,
			`SELECT count(*) FROM staff_invitations`,
			`UPDATE platform_admins SET role = 'owner'`,
		} {
			_, err := env.platform.Exec(env.ctx, query)
			expectCode(t, err, permissionDenied, "platform: "+query)
		}
		if _, err := env.platform.Exec(env.ctx, `UPDATE platform_admins SET last_access_at = now() WHERE false`); err != nil {
			t.Fatalf("platform must be able to stamp its last access: %v", err)
		}
	})

	t.Run("a pool refuses a role that bypasses RLS", func(t *testing.T) {
		var login string
		if err := env.admin.QueryRow(env.ctx, `SELECT session_user`).Scan(&login); err != nil {
			t.Fatal(err)
		}
		var bypass bool
		if err := env.admin.QueryRow(env.ctx,
			`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = $1`, login).Scan(&bypass); err != nil {
			t.Fatal(err)
		}
		if !bypass {
			t.Skip("the test login cannot bypass RLS, so there is nothing to refuse")
		}
		pool, err := database.Connect(env.ctx, env.dsn, login)
		if err == nil {
			pool.Close()
			t.Fatalf("Connect accepted %s, which bypasses RLS", login)
		}
	})
}
