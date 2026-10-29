package tax

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

func withTaxTx(ctx context.Context, db *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	return database.WithTenantTx(ctx, db, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx)
	})
}

const rateSelect = `SELECT t.id, t.name, t.rate_bps, t.inclusive, t.status, t.created_at,
	(SELECT count(*) FROM products p WHERE p.tenant_id=t.tenant_id AND p.tax_rate_id=t.id AND p.status='active')
	FROM tax_rates t`

func scanRate(row pgx.Row) (*Rate, error) {
	var r Rate
	err := row.Scan(&r.ID, &r.Name, &r.RateBps, &r.Inclusive, &r.Status, &r.CreatedAt, &r.ProductCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}

func (r *postgresRepository) List(ctx context.Context, tenantID string, includeArchived bool) ([]Rate, error) {
	rates := []Rate{}
	err := withTaxTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, rateSelect+` WHERE t.tenant_id=$1 AND ($2 OR t.status='active')
			ORDER BY t.status, lower(t.name), t.id`, tenantID, includeArchived)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			rate, err := scanRate(rows)
			if err != nil {
				return err
			}
			rates = append(rates, *rate)
		}
		return rows.Err()
	})
	return rates, err
}

func conflict(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrConflict
	}
	return err
}

func (r *postgresRepository) Create(ctx context.Context, tenantID string, in RateInput) (*Rate, error) {
	var rate *Rate
	err := withTaxTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rateID := id.New(id.TaxRate)
		if _, err := tx.Exec(ctx, `INSERT INTO tax_rates (id, tenant_id, name, rate_bps, inclusive) VALUES ($1,$2,$3,$4,$5)`,
			rateID, tenantID, in.Name, in.RateBps, in.Inclusive); err != nil {
			return err
		}
		var err error
		rate, err = scanRate(tx.QueryRow(ctx, rateSelect+` WHERE t.tenant_id=$1 AND t.id=$2`, tenantID, rateID))
		return err
	})
	return rate, conflict(err)
}

func (r *postgresRepository) Update(ctx context.Context, tenantID, rateID string, in RateUpdate) (*Rate, error) {
	var rate *Rate
	err := withTaxTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// A product points at its rate; archiving one still in use would leave
		// those products charging a rate nobody can see or pick any more.
		if in.Status != nil && *in.Status == "archived" {
			var inUse int64
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id=$1 AND tax_rate_id=$2`,
				tenantID, rateID).Scan(&inUse); err != nil {
				return err
			}
			if inUse > 0 {
				return &Error{http.StatusConflict, "tax_rate_in_use",
					fmt.Sprintf("%d product(s) still use this rate. Move them to another rate, or to no tax, before archiving it.", inUse)}
			}
		}
		tag, err := tx.Exec(ctx, `UPDATE tax_rates SET name=COALESCE($3, name), rate_bps=COALESCE($4, rate_bps),
			inclusive=COALESCE($5, inclusive), status=COALESCE($6, status) WHERE tenant_id=$1 AND id=$2`,
			tenantID, rateID, in.Name, in.RateBps, in.Inclusive, in.Status)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		rate, err = scanRate(tx.QueryRow(ctx, rateSelect+` WHERE t.tenant_id=$1 AND t.id=$2`, tenantID, rateID))
		return err
	})
	return rate, conflict(err)
}
