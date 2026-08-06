package workspace

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/id"
)

type repoPostgres struct {
	db *pgxpool.Pool
}

// NewPostgresRepository wires the Repository to a pgx pool.
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &repoPostgres{db: db}
}

func (r *repoPostgres) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM tenants WHERE slug = $1 AND deleted_at IS NULL)`,
		slug).Scan(&exists)
	return exists, err
}

func (r *repoPostgres) CreateWithOwner(ctx context.Context, w *Workspace, ownerUserID string) error {
	w.ID = id.New(id.Tenant)
	w.Role = "owner"

	// Tenant + owner membership must be created together, or not at all.
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, industry_type, currency_code, timezone, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			w.ID, w.Name, w.Slug, w.IndustryType, w.CurrencyCode, w.Timezone, ownerUserID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO tenant_memberships (id, tenant_id, user_id, role, status)
			VALUES ($1, $2, $3, 'owner', 'active')`,
			id.New(id.TenantMembership), w.ID, ownerUserID)
		return err
	})
}

func (r *repoPostgres) ListForUser(ctx context.Context, userID string) ([]Workspace, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.name, t.slug, t.industry_type, t.status, t.currency_code,
		       t.timezone, m.role, t.created_at
		FROM tenant_memberships m
		JOIN tenants t ON t.id = m.tenant_id AND t.deleted_at IS NULL
		WHERE m.user_id = $1 AND m.status = 'active'
		ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.IndustryType, &w.Status,
			&w.CurrencyCode, &w.Timezone, &w.Role, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
