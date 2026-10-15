// Package database constructs and owns the application's PostgreSQL connection
// pool (pgxpool). It is the single place that knows how to dial the database.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Restricted roles (migration 0025) that a pool switches into on connect, so
// the privileges a query runs with never depend on which login the DSN uses.
const (
	// AppRole serves tenant requests. It cannot bypass Row-Level Security.
	AppRole = "candella_app"
	// PlatformRole serves the platform console: cross-tenant reads only.
	PlatformRole = "candella_platform"
)

// Connect opens a pgx connection pool for the given DSN and verifies it with a
// ping. The caller owns the returned pool and must Close it on shutdown.
//
// With a role, every connection switches into it as soon as it opens, and
// Connect fails unless queries really do run as that role without the power
// to bypass RLS. An empty role keeps the login's own privileges; that is for
// operator tools such as cmd/admin, never for the API.
func Connect(ctx context.Context, dsn, role string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("database: parse dsn: %w", err)
	}

	// Conservative pool defaults; tune via load testing later.
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	if role != "" {
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, "SET ROLE "+pgx.Identifier{role}.Sanitize())
			return err
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database: create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		if role != "" {
			return nil, fmt.Errorf("database: ping as %s: %w (run migrations, and grant the role to the database login)", role, err)
		}
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	if role != "" {
		if err := verifyRole(pingCtx, pool, role); err != nil {
			pool.Close()
			return nil, err
		}
	}

	return pool, nil
}

// verifyRole refuses a pool whose queries would not be bound by RLS.
func verifyRole(ctx context.Context, pool *pgxpool.Pool, role string) error {
	var current string
	var super, bypass bool
	err := pool.QueryRow(ctx, `
		SELECT current_user, r.rolsuper, r.rolbypassrls
		FROM pg_roles r WHERE r.rolname = current_user`).Scan(&current, &super, &bypass)
	if err != nil {
		return fmt.Errorf("database: checking role %s: %w", role, err)
	}
	if current != role {
		return fmt.Errorf("database: queries run as %q, want %q", current, role)
	}
	if super || bypass {
		return fmt.Errorf("database: role %s can bypass row-level security", role)
	}
	return nil
}
