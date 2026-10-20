package workspace

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
	"candella-ecosystem/backend/pkg/mailer"
)

type staffTestMailer struct {
	last mailer.Email
	fail bool
}

func (m *staffTestMailer) Send(_ context.Context, message mailer.Email) error {
	m.last = message
	if m.fail {
		return errors.New("mail offline")
	}
	return nil
}

func staffToken(t *testing.T, message mailer.Email) string {
	t.Helper()
	if message.HTML == "" || !strings.Contains(message.HTML, "Accept invitation") {
		t.Fatal("invitation email has no HTML action")
	}
	for _, part := range strings.Fields(message.Text) {
		if !strings.HasPrefix(part, "https://") {
			continue
		}
		link, err := url.Parse(part)
		if err == nil && link.Query().Get("token") != "" {
			return link.Query().Get("token")
		}
	}
	t.Fatal("invitation token missing from plain-text email")
	return ""
}

// Migrations run only on a named local disposable DB or this repo's CI service.
func TestStaffInvitationLifecyclePostgres(t *testing.T) {
	dsn := os.Getenv("STAFF_TEST_DATABASE_URL")
	ciDatabase := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn, ciDatabase = os.Getenv("DATABASE_URL"), true
	}
	if dsn == "" {
		t.Skip("set STAFF_TEST_DATABASE_URL for a local disposable candella_staff_test database")
	}
	u, err := url.Parse(dsn)
	expected := "/candella_staff_test"
	if ciDatabase {
		expected = "/candella"
	}
	if err != nil || u.Path != expected || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		(ciDatabase && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("staff integration test requires a local disposable PostgreSQL database")
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	ownerA, ownerB, admin, invited, wrong, extraOwner := id.New(id.User), id.New(id.User), id.New(id.User), id.New(id.User), id.New(id.User), id.New(id.User)
	for _, seed := range []struct {
		id, email string
		verified  bool
	}{
		{ownerA, "owner-a@example.test", true}, {ownerB, "owner-b@example.test", true}, {admin, "admin@example.test", true},
		{invited, "staff@example.test", true}, {wrong, "wrong@example.test", true}, {extraOwner, "co-owner@example.test", true},
	} {
		var verifiedAt *time.Time
		if seed.verified {
			now := time.Now()
			verifiedAt = &now
		}
		exec(`INSERT INTO users (id,email,full_name,email_verified_at) VALUES ($1,$2,$3,$4)`, seed.id, seed.email, seed.email, verifiedAt)
	}
	tenantA, tenantB := id.New(id.Tenant), id.New(id.Tenant)
	for _, tenantID := range []string{tenantA, tenantB} {
		exec(`INSERT INTO tenants (id,name,slug,industry_type) VALUES ($1,'Staff test',$2,'pos_general')`, tenantID, tenantID)
	}
	for _, seed := range []struct{ tenantID, userID, role string }{{tenantA, ownerA, "owner"}, {tenantA, admin, "admin"}, {tenantB, ownerB, "owner"}} {
		exec(`INSERT INTO tenant_memberships (id,tenant_id,user_id,role,status) VALUES ($1,$2,$3,$4,'active')`, id.New(id.TenantMembership), seed.tenantID, seed.userID, seed.role)
	}
	repo := NewStaffPostgresRepository(appPool)
	mail := &staffTestMailer{}
	svc := NewStaffService(repo, mail, "http://localhost:3000")
	invitation, err := svc.Invite(ctx, tenantA, ownerA, " STAFF@example.test ", "member")
	if err != nil || invitation.Email != "staff@example.test" || mail.last.To != "staff@example.test" {
		t.Fatalf("invite=%+v mail=%+v err=%v", invitation, mail.last, err)
	}
	firstToken := staffToken(t, mail.last)
	if _, err := svc.Invite(ctx, tenantA, ownerA, "staff@example.test", "member"); err != ErrStaffConflict {
		t.Fatalf("duplicate invite err=%v", err)
	}
	if _, err := svc.Invite(ctx, tenantA, admin, "co-owner@example.test", "owner"); err != ErrStaffForbidden {
		t.Fatalf("admin owner invite err=%v", err)
	}
	if _, err := svc.Accept(ctx, firstToken, wrong, "wrong@example.test", true); err != ErrInvitationEmail {
		t.Fatalf("wrong email accepted: %v", err)
	}
	if _, err := svc.Accept(ctx, firstToken, invited, "staff@example.test", false); err != ErrInvitationInvalid {
		t.Fatalf("unverified accepted: %v", err)
	}
	if _, _, err := svc.List(ctx, tenantB, ownerA); err == nil {
		t.Fatal("cross-tenant staff list succeeded")
	}
	if _, err := svc.Resend(ctx, tenantB, ownerB, invitation.ID); err != ErrStaffNotFound {
		t.Fatalf("cross-tenant resend err=%v", err)
	}
	if _, err := svc.Resend(ctx, tenantA, ownerA, invitation.ID); err != nil {
		t.Fatal(err)
	}
	newToken := staffToken(t, mail.last)
	if firstToken == newToken {
		t.Fatal("resend kept original token")
	}
	if _, err := svc.Accept(ctx, firstToken, invited, "staff@example.test", true); err != ErrInvitationInvalid {
		t.Fatalf("old token accepted: %v", err)
	}
	joinedTenant, err := svc.Accept(ctx, newToken, invited, "staff@example.test", true)
	if err != nil || joinedTenant != tenantA {
		t.Fatalf("accept tenant=%s err=%v", joinedTenant, err)
	}
	if _, err := svc.Accept(ctx, newToken, invited, "staff@example.test", true); err != ErrInvitationInvalid {
		t.Fatalf("token replay err=%v", err)
	}
	if m, err := NewService(NewPostgresRepository(appPool)).ResolveMembership(ctx, invited, tenantA); err != nil || m.Role != "member" {
		t.Fatalf("accepted membership=%+v err=%v", m, err)
	}
	if err := svc.ChangeRole(ctx, tenantA, admin, admin, "owner"); err != ErrStaffForbidden {
		t.Fatalf("self elevation err=%v", err)
	}
	if err := svc.ChangeRole(ctx, tenantA, admin, ownerA, "member"); err != ErrStaffForbidden {
		t.Fatalf("admin changed owner err=%v", err)
	}
	if err := svc.Remove(ctx, tenantA, ownerA, ownerA); err != ErrLastOwner {
		t.Fatalf("last owner removal err=%v", err)
	}
	if err := svc.ChangeRole(ctx, tenantA, ownerA, ownerA, "admin"); err != ErrLastOwner {
		t.Fatalf("last owner demotion err=%v", err)
	}
	if err := svc.ChangeRole(ctx, tenantA, ownerA, invited, "admin"); err != nil {
		t.Fatalf("role change: %v", err)
	}
	if err := svc.Remove(ctx, tenantA, ownerA, invited); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if _, err := NewService(NewPostgresRepository(appPool)).ResolveMembership(ctx, invited, tenantA); err != ErrNoMembership {
		t.Fatalf("removed member has access: %v", err)
	}
	ownerInvite, err := svc.Invite(ctx, tenantA, ownerA, "co-owner@example.test", "owner")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, tenantA, ownerA, ownerInvite.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Accept(ctx, staffToken(t, mail.last), extraOwner, "co-owner@example.test", true); err != ErrInvitationInvalid {
		t.Fatalf("revoked invite accepted: %v", err)
	}
	if err := svc.Revoke(ctx, tenantA, ownerA, ownerInvite.ID); err != ErrStaffNotFound {
		t.Fatalf("revoke replay err=%v", err)
	}
}
