package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantTx runs fn inside a transaction that has been stamped with the
// current tenant, committing on success and rolling back on error.
//
// The stamp is what PostgreSQL Row-Level Security policies read:
//
//	CREATE POLICY tenant_isolation ON orders
//	  USING (tenant_id = current_setting('app.current_tenant_id', true));
//
// Application-level `WHERE tenant_id = $1` filtering remains the primary control;
// this is the seatbelt for the day somebody forgets one. Every tenant-scoped
// repository write should go through here so enabling those policies later is a
// migration, not a refactor.
//
// set_config(..., true) is used rather than `SET LOCAL` because only the former
// accepts a bind parameter — building the statement by string concatenation
// would put a client-supplied value directly into SQL.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	if tenantID == "" {
		return fmt.Errorf("database: WithTenantTx called without a tenant")
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`SELECT set_config('app.current_tenant_id', $1, true)`, tenantID); err != nil {
			return fmt.Errorf("database: setting tenant scope: %w", err)
		}
		return fn(tx)
	})
}
