package platform

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/id"
)

type repoPostgres struct {
	db *pgxpool.Pool
}

// NewPostgresRepository wires the platform repository to a pgx pool.
//
// The server passes a separate pool running as the `candella_platform` database
// role, so cross-tenant reads are granted by an explicit DB role rather than by
// an escape hatch inside the RLS policies. cmd/admin passes the operator's own
// login, which grant and revoke need.
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &repoPostgres{db: db}
}

// ----- Admin identity -------------------------------------------------------

const adminColumns = `
	a.id, a.user_id, u.email, u.full_name, a.role, a.status, a.last_access_at, a.created_at`

func (r *repoPostgres) GetAdminByUserID(ctx context.Context, userID string) (*Admin, error) {
	var a Admin
	err := r.db.QueryRow(ctx, `
		SELECT `+adminColumns+`
		FROM platform_admins a
		JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL
		WHERE a.user_id = $1 AND a.status = 'active'`, userID).
		Scan(&a.ID, &a.UserID, &a.Email, &a.FullName, &a.Role, &a.Status, &a.LastAccessAt, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *repoPostgres) TouchAdminAccess(ctx context.Context, adminID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE platform_admins SET last_access_at = now() WHERE id = $1`, adminID)
	return err
}

func (r *repoPostgres) GrantAdmin(ctx context.Context, email, role string) (*Admin, error) {
	var userID string
	err := r.db.QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`, email).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Re-granting an existing admin updates their role and reactivates them.
	_, err = r.db.Exec(ctx, `
		INSERT INTO platform_admins (id, user_id, role, status)
		VALUES ($1, $2, $3, 'active')
		ON CONFLICT (user_id) DO UPDATE
		SET role = EXCLUDED.role, status = 'active'`,
		id.New(id.PlatformAdmin), userID, role)
	if err != nil {
		return nil, err
	}
	return r.GetAdminByUserID(ctx, userID)
}

func (r *repoPostgres) RevokeAdmin(ctx context.Context, email string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM platform_admins
		WHERE user_id = (SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL)`, email)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repoPostgres) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+adminColumns+`
		FROM platform_admins a
		JOIN users u ON u.id = a.user_id
		ORDER BY a.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := make([]Admin, 0)
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.ID, &a.UserID, &a.Email, &a.FullName, &a.Role,
			&a.Status, &a.LastAccessAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		admins = append(admins, a)
	}
	return admins, rows.Err()
}

// ----- Overview -------------------------------------------------------------

func (r *repoPostgres) GetOverview(ctx context.Context) (*Overview, error) {
	var o Overview

	// One pass over tenants rather than five COUNT queries.
	err := r.db.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE status = 'active'),
			count(*) FILTER (WHERE status = 'trialing'),
			count(*) FILTER (WHERE status = 'suspended'),
			count(*) FILTER (WHERE status = 'cancelled'),
			count(*) FILTER (WHERE created_at > now() - interval '7 days')
		FROM tenants
		WHERE deleted_at IS NULL`).
		Scan(&o.Tenants.Total, &o.Tenants.Active, &o.Tenants.Trialing,
			&o.Tenants.Suspended, &o.Tenants.Cancelled, &o.NewTenants7d)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE created_at > now() - interval '7 days')
		FROM users
		WHERE deleted_at IS NULL`).Scan(&o.TotalUsers, &o.NewUsers7d)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT count(*) FROM sessions
		WHERE revoked_at IS NULL AND expires_at > now()`).Scan(&o.ActiveSessions)
	if err != nil {
		return nil, err
	}

	o.RecentActivity, err = r.RecentActivity(ctx, 10)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ----- Tenants --------------------------------------------------------------

// tenantSummarySelect carries the member count and owner email as correlated
// subqueries — cheap at this scale and far clearer than two GROUP BY joins.
const tenantSummarySelect = `
	SELECT t.id, t.name, t.slug, t.industry_type, t.status, t.currency_code,
	       (SELECT count(*) FROM tenant_memberships m
	         WHERE m.tenant_id = t.id AND m.status = 'active'),
	       COALESCE((SELECT u.email FROM tenant_memberships m
	                  JOIN users u ON u.id = m.user_id
	                 WHERE m.tenant_id = t.id AND m.role = 'owner'
	                 ORDER BY m.created_at LIMIT 1), ''),
	       t.created_at
	FROM tenants t
	WHERE t.deleted_at IS NULL`

func (r *repoPostgres) ListTenants(ctx context.Context, q TenantQuery) ([]TenantSummary, int, error) {
	// Build the shared filter once so the page and its total can never diverge.
	var where strings.Builder
	args := make([]any, 0, 3)

	if s := strings.TrimSpace(q.Search); s != "" {
		args = append(args, "%"+strings.ToLower(s)+"%")
		where.WriteString(" AND (lower(t.name) LIKE $1 OR lower(t.slug) LIKE $1)")
	}
	if q.Status != "" {
		args = append(args, q.Status)
		where.WriteString(" AND t.status = $" + strconv.Itoa(len(args)))
	}
	if q.Industry != "" {
		args = append(args, q.Industry)
		where.WriteString(" AND t.industry_type = $" + strconv.Itoa(len(args)))
	}
	filter := where.String()

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM tenants t WHERE t.deleted_at IS NULL`+filter, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, q.Limit, q.Offset)
	rows, err := r.db.Query(ctx,
		tenantSummarySelect+filter+
			" ORDER BY t.created_at DESC LIMIT $"+strconv.Itoa(len(args)-1)+" OFFSET $"+strconv.Itoa(len(args)),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]TenantSummary, 0, q.Limit)
	for rows.Next() {
		t, err := scanTenantSummary(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

func (r *repoPostgres) GetTenant(ctx context.Context, tenantID string) (*TenantDetail, error) {
	var d TenantDetail
	row := r.db.QueryRow(ctx, tenantSummarySelect+" AND t.id = $1", tenantID)

	summary, err := scanTenantSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d.TenantSummary = summary

	if err := r.db.QueryRow(ctx,
		`SELECT timezone FROM tenants WHERE id = $1`, tenantID).Scan(&d.Timezone); err != nil {
		return nil, err
	}

	d.Members, err = r.tenantMembers(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	d.RecentActivity, err = r.tenantActivity(ctx, tenantID, 25)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *repoPostgres) tenantMembers(ctx context.Context, tenantID string) ([]TenantMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.email, u.full_name, m.role, m.status, u.mfa_enabled,
		       u.last_login_at, m.created_at
		FROM tenant_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.tenant_id = $1
		ORDER BY m.created_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]TenantMember, 0)
	for rows.Next() {
		var m TenantMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.FullName, &m.Role, &m.Status,
			&m.MFAEnabled, &m.LastLoginAt, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// ----- Activity -------------------------------------------------------------

const activitySelect = `
	SELECT l.id, l.tenant_id, t.name, l.actor_type, u.email, l.action,
	       l.entity_type, l.entity_id, host(l.ip_address), l.created_at
	FROM audit_logs l
	LEFT JOIN tenants t ON t.id = l.tenant_id
	LEFT JOIN users u ON u.id = l.user_id`

func (r *repoPostgres) RecentActivity(ctx context.Context, limit int) ([]ActivityEntry, error) {
	rows, err := r.db.Query(ctx, activitySelect+` ORDER BY l.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return scanActivity(rows)
}

func (r *repoPostgres) tenantActivity(ctx context.Context, tenantID string, limit int) ([]ActivityEntry, error) {
	rows, err := r.db.Query(ctx,
		activitySelect+` WHERE l.tenant_id = $1 ORDER BY l.created_at DESC LIMIT $2`,
		tenantID, limit)
	if err != nil {
		return nil, err
	}
	return scanActivity(rows)
}

// ----- scan helpers ---------------------------------------------------------

// scanner is the shared surface of pgx.Row and pgx.Rows, so one function can
// read a tenant summary from either a single-row query or an iteration.
type scanner interface{ Scan(dest ...any) error }

func scanTenantSummary(s scanner) (TenantSummary, error) {
	var t TenantSummary
	err := s.Scan(&t.ID, &t.Name, &t.Slug, &t.IndustryType, &t.Status,
		&t.CurrencyCode, &t.MemberCount, &t.OwnerEmail, &t.CreatedAt)
	return t, err
}

func scanActivity(rows pgx.Rows) ([]ActivityEntry, error) {
	defer rows.Close()

	entries := make([]ActivityEntry, 0)
	for rows.Next() {
		var e ActivityEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.TenantName, &e.ActorType,
			&e.ActorEmail, &e.Action, &e.EntityType, &e.EntityID,
			&e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
