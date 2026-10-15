package workspace_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/internal/workspace"
	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/config"
	"candella-ecosystem/backend/pkg/crypto"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
	"candella-ecosystem/backend/pkg/mailer"
)

// TestTenantAccessBoundariesPostgres runs the real RequireAuth → RequireTenant
// chain against a disposable database. Every request that is not an active
// member, of a live and usable workspace, holding a live session of an active
// account must be refused before reaching a tenant-scoped handler.
func TestTenantAccessBoundariesPostgres(t *testing.T) {
	dsn := os.Getenv("TENANT_TEST_DATABASE_URL")
	ci := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn, ci = os.Getenv("DATABASE_URL"), true
	}
	if dsn == "" {
		t.Skip("set TENANT_TEST_DATABASE_URL for a disposable candella_tenant_test database")
	}
	u, err := url.Parse(dsn)
	wantPath := "/candella_tenant_test"
	if ci {
		wantPath = "/candella"
	}
	if err != nil || u.Path != wantPath || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || (ci && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("tenant integration test requires a local disposable PostgreSQL database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	migrateDisposable(ctx, t, dsn)
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	// --- fixtures -------------------------------------------------------------
	type person struct{ id, token string }
	newPerson := func(status string) person {
		p := person{id: id.New(id.User), token: "tok-" + id.New(id.Session)}
		exec(`INSERT INTO users (id,email,full_name,status,email_verified_at) VALUES ($1,$2,'Tenant tester',$3,now())`,
			p.id, p.id+"@example.test", status)
		exec(`INSERT INTO sessions (id,user_id,token_hash,expires_at) VALUES ($1,$2,$3,now() + interval '1 hour')`,
			id.New(id.Session), p.id, crypto.HashToken(p.token))
		return p
	}
	newTenant := func(status string) string {
		tid := id.New(id.Tenant)
		exec(`INSERT INTO tenants (id,name,slug,industry_type,status) VALUES ($1,'Access shop',$2,'pos_general',$3)`,
			tid, tid, status)
		return tid
	}
	join := func(tenantID string, p person, role, status string) {
		exec(`INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status) VALUES ($1,$2,$3,$4,$5)`,
			id.New(id.TenantMembership), tenantID, p.id, role, status)
	}

	shopA, shopB := newTenant("active"), newTenant("active")
	suspendedShop, cancelledShop, deletedShop := newTenant("suspended"), newTenant("cancelled"), newTenant("active")
	exec(`UPDATE tenants SET deleted_at = now() WHERE id = $1`, deletedShop)

	owner := newPerson("active")
	join(shopA, owner, "owner", "active")
	join(suspendedShop, owner, "owner", "active")
	join(cancelledShop, owner, "owner", "active")
	join(deletedShop, owner, "owner", "active")

	cashier := newPerson("active")
	join(shopA, cashier, "member", "active")

	outsider := newPerson("active")
	join(shopB, outsider, "owner", "active")

	suspendedMember, removedMember, invitedMember := newPerson("active"), newPerson("active"), newPerson("active")
	join(shopA, suspendedMember, "member", "suspended")
	join(shopA, removedMember, "member", "removed")
	join(shopA, invitedMember, "member", "invited")

	disabled := newPerson("disabled")
	join(shopA, disabled, "owner", "active")

	expired := newPerson("active")
	join(shopA, expired, "member", "active")
	exec(`UPDATE sessions SET expires_at = now() - interval '1 minute' WHERE user_id = $1`, expired.id)

	revoked := newPerson("active")
	join(shopA, revoked, "member", "active")
	exec(`UPDATE sessions SET revoked_at = now() WHERE user_id = $1`, revoked.id)

	// --- server: the production middleware chain in front of a probe -----------
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{SessionCookieName: "candella_session", SessionTTL: time.Hour}
	authSvc := auth.NewService(auth.NewPostgresRepository(appPool), mailer.NewConsoleMailer(log),
		audit.NewRecorder(appPool, log), cfg, log, nil, []byte("tenant-access-test-challenge-key!"))
	authHandler := auth.NewHandler(authSvc, cfg)
	wsHandler := workspace.NewHandler(workspace.NewService(workspace.NewPostgresRepository(appPool)))

	router := chi.NewRouter()
	router.With(authHandler.RequireAuth, wsHandler.RequireTenant).Get("/probe", func(w http.ResponseWriter, r *http.Request) {
		tc, _ := tenant.FromContext(r.Context())
		_ = json.NewEncoder(w).Encode(map[string]string{"tenantId": tc.TenantID, "role": tc.Role})
	})
	server := httptest.NewServer(router)
	defer server.Close()

	probe := func(p *person, tenantID string) (int, string, map[string]string) {
		t.Helper()
		req, _ := http.NewRequest("GET", server.URL+"/probe", nil)
		if p != nil {
			req.AddCookie(&http.Cookie{Name: "candella_session", Value: p.token})
		}
		if tenantID != "" {
			req.Header.Set(workspace.TenantHeader, tenantID)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var body struct {
			Error *struct{ Code string } `json:"error"`
		}
		raw, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(raw, &body)
		code := ""
		if body.Error != nil {
			code = body.Error.Code
		}
		var ok map[string]string
		if resp.StatusCode == http.StatusOK {
			_ = json.Unmarshal(raw, &ok)
		}
		return resp.StatusCode, code, ok
	}

	// --- allowed --------------------------------------------------------------
	for _, tc := range []struct {
		name string
		who  person
		role string
	}{
		{"owner of the workspace", owner, "owner"},
		{"active member of the workspace", cashier, "member"},
	} {
		t.Run("allows "+tc.name, func(t *testing.T) {
			status, _, got := probe(&tc.who, shopA)
			if status != http.StatusOK || got["tenantId"] != shopA || got["role"] != tc.role {
				t.Fatalf("status=%d context=%v, want 200 for %s as %s", status, got, shopA, tc.role)
			}
		})
	}

	// --- refused --------------------------------------------------------------
	for _, tc := range []struct {
		name       string
		who        *person
		tenantID   string
		wantStatus int
		wantCode   string
	}{
		{"no session", nil, shopA, http.StatusUnauthorized, "unauthenticated"},
		{"expired session", &expired, shopA, http.StatusUnauthorized, "unauthenticated"},
		{"revoked session", &revoked, shopA, http.StatusUnauthorized, "unauthenticated"},
		{"disabled account", &disabled, shopA, http.StatusUnauthorized, "unauthenticated"},
		{"missing workspace header", &owner, "", http.StatusBadRequest, "tenant_required"},
		{"another tenant's workspace", &outsider, shopA, http.StatusNotFound, "workspace_not_found"},
		{"member reaching into a workspace they don't belong to", &cashier, shopB, http.StatusNotFound, "workspace_not_found"},
		{"workspace that doesn't exist", &owner, id.New(id.Tenant), http.StatusNotFound, "workspace_not_found"},
		{"suspended membership", &suspendedMember, shopA, http.StatusNotFound, "workspace_not_found"},
		{"removed membership", &removedMember, shopA, http.StatusNotFound, "workspace_not_found"},
		{"invited but not accepted membership", &invitedMember, shopA, http.StatusNotFound, "workspace_not_found"},
		{"soft-deleted workspace", &owner, deletedShop, http.StatusNotFound, "workspace_not_found"},
		{"suspended workspace", &owner, suspendedShop, http.StatusForbidden, "workspace_suspended"},
		{"cancelled workspace", &owner, cancelledShop, http.StatusForbidden, "workspace_suspended"},
	} {
		t.Run("refuses "+tc.name, func(t *testing.T) {
			status, code, _ := probe(tc.who, tc.tenantID)
			if status != tc.wantStatus || code != tc.wantCode {
				t.Fatalf("got %d %q, want %d %q", status, code, tc.wantStatus, tc.wantCode)
			}
		})
	}
}

func migrateDisposable(ctx context.Context, t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Packages run in parallel against one CI database; serialise migrations.
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtext('candella_integration_migrations'))`)
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("disposable DB migrations: %v", err)
	}
}
