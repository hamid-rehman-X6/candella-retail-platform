package store

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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/id"
)

func TestStoreManagementPostgres(t *testing.T) {
	dsn := os.Getenv("STORE_TEST_DATABASE_URL")
	ci := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn, ci = os.Getenv("DATABASE_URL"), true
	}
	if dsn == "" {
		t.Skip("set STORE_TEST_DATABASE_URL for a disposable candella_store_test database")
	}
	u, err := url.Parse(dsn)
	wantPath := "/candella_store_test"
	if ci {
		wantPath = "/candella"
	}
	if err != nil || u.Path != wantPath || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || (ci && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("store integration test requires a local disposable PostgreSQL database")
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
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	ownerID, memberID := id.New(id.User), id.New(id.User)
	tenantA, tenantB := id.New(id.Tenant), id.New(id.Tenant)
	defaultA, defaultB := id.New(id.Store), id.New(id.Store)
	for _, userID := range []string{ownerID, memberID} {
		mustExec(`INSERT INTO users (id,email,full_name) VALUES ($1,$2,'Store tester')`, userID, userID+"@example.test")
	}
	for _, tenantID := range []string{tenantA, tenantB} {
		mustExec(`INSERT INTO tenants (id,name,slug,industry_type,currency_code) VALUES ($1,'Test shop',$2,'pos_general','USD')`, tenantID, tenantID)
	}
	mustExec(`INSERT INTO stores (id,tenant_id,name,is_default) VALUES ($1,$2,'Main',true),($3,$4,'Main',true)`, defaultA, tenantA, defaultB, tenantB)
	mustExec(`INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status) VALUES ($1,$2,$3,'owner','active'),($4,$5,$6,'member','active')`, id.New(id.TenantMembership), tenantA, ownerID, id.New(id.TenantMembership), tenantA, memberID)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(pool, audit.NewRecorder(pool, log), log)
	call := func(method, body, actingTenant, actingUser, role, storeID string, fn func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/stores", strings.NewReader(body))
		requestCtx := tenant.WithContext(req.Context(), &tenant.Context{TenantID: actingTenant, Role: role})
		requestCtx = auth.WithUser(requestCtx, &auth.User{ID: actingUser})
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("storeID", storeID)
		req = req.WithContext(context.WithValue(requestCtx, chi.RouteCtxKey, routeCtx))
		w := httptest.NewRecorder()
		fn(w, req)
		return w
	}
	if got := call("POST", `{"name":"Branch"}`, tenantA, memberID, "member", "", h.Create).Code; got != http.StatusForbidden {
		t.Fatalf("member created store: %d", got)
	}
	if got := call("POST", `{"name":"Branch"}`, tenantA, ownerID, "owner", "", h.Create).Code; got != http.StatusCreated {
		t.Fatalf("owner create status=%d", got)
	}
	var branchID string
	if err := pool.QueryRow(ctx, `SELECT id FROM stores WHERE tenant_id=$1 AND name='Branch'`, tenantA).Scan(&branchID); err != nil {
		t.Fatal(err)
	}
	if got := call("POST", `{"name":"branch"}`, tenantA, ownerID, "owner", "", h.Create).Code; got != http.StatusConflict {
		t.Fatalf("duplicate store status=%d", got)
	}
	if got := call("PATCH", `{"name":"Mall branch"}`, tenantA, ownerID, "owner", branchID, h.Rename).Code; got != http.StatusOK {
		t.Fatalf("rename status=%d", got)
	}
	listCount := func(userID, role string) int {
		w := call("GET", "", tenantA, userID, role, "", h.List)
		if w.Code != http.StatusOK {
			t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
		}
		var payload struct {
			Data struct {
				Stores []Location `json:"stores"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return len(payload.Data.Stores)
	}
	if count := listCount(memberID, "member"); count != 1 {
		t.Fatalf("cashier initially saw %d stores", count)
	}
	if count := listCount(ownerID, "owner"); count != 2 {
		t.Fatalf("owner saw %d stores", count)
	}
	grant := `{"userId":"` + memberID + `","allowed":true}`
	if got := call("POST", grant, tenantA, ownerID, "owner", branchID, h.Access).Code; got != http.StatusOK {
		t.Fatalf("grant status=%d", got)
	}
	if count := listCount(memberID, "member"); count != 2 {
		t.Fatalf("granted cashier saw %d stores", count)
	}
	revoke := `{"userId":"` + memberID + `","allowed":false}`
	if got := call("POST", revoke, tenantA, ownerID, "owner", branchID, h.Access).Code; got != http.StatusOK {
		t.Fatalf("revoke status=%d", got)
	}
	if count := listCount(memberID, "member"); count != 1 {
		t.Fatalf("revoked cashier saw %d stores", count)
	}
	if got := call("POST", "", tenantA, ownerID, "owner", defaultB, h.SetDefault).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant default status=%d", got)
	}
	for _, storeID := range []string{branchID, defaultA} {
		if got := call("POST", "", tenantA, ownerID, "owner", storeID, h.SetDefault).Code; got != http.StatusOK {
			t.Fatalf("default switch status=%d", got)
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM stores WHERE tenant_id=$1 AND is_default`, tenantA).Scan(&count); err != nil || count != 1 {
			t.Fatalf("default count=%d err=%v", count, err)
		}
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, storeID := range []string{branchID, defaultA} {
		wg.Add(1)
		go func(storeID string) {
			defer wg.Done()
			statuses <- call("POST", "", tenantA, ownerID, "owner", storeID, h.SetDefault).Code
		}(storeID)
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("concurrent default status=%d", status)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM stores WHERE tenant_id=$1 AND is_default`, tenantA).Scan(&count); err != nil || count != 1 {
		t.Fatalf("concurrent default count=%d err=%v", count, err)
	}
}
