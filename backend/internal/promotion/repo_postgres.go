package promotion

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

func withPromotionTx(ctx context.Context, db *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	return database.WithTenantTx(ctx, db, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx)
	})
}

const selectPromotion = `SELECT id, name, kind, percent_bps, amount_minor, applies_to, min_subtotal_minor,
	to_char(starts_on, 'YYYY-MM-DD'), to_char(ends_on, 'YYYY-MM-DD'), status, created_at FROM promotions`

func scan(row pgx.Row) (*Promotion, error) {
	var p Promotion
	err := row.Scan(&p.ID, &p.Name, &p.Kind, &p.PercentBps, &p.AmountMinor, &p.AppliesTo, &p.MinSubtotalMinor,
		&p.StartsOn, &p.EndsOn, &p.Status, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

func conflict(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrConflict
	}
	return err
}

func (r *postgresRepository) List(ctx context.Context, tenantID string, includeArchived bool) ([]Promotion, error) {
	out := []Promotion{}
	err := withPromotionTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, selectPromotion+` WHERE tenant_id=$1 AND ($2 OR status='active')
			ORDER BY status, lower(name), id`, tenantID, includeArchived)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scan(rows)
			if err != nil {
				return err
			}
			out = append(out, *p)
		}
		return rows.Err()
	})
	return out, err
}

func (r *postgresRepository) Create(ctx context.Context, tenantID string, in Input) (*Promotion, error) {
	var p *Promotion
	err := withPromotionTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		promotionID := id.New(id.Promotion)
		if _, err := tx.Exec(ctx, `INSERT INTO promotions (id, tenant_id, name, kind, percent_bps, amount_minor,
			applies_to, min_subtotal_minor, starts_on, ends_on) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			promotionID, tenantID, in.Name, in.Kind, in.PercentBps, in.AmountMinor, in.AppliesTo,
			in.MinSubtotalMinor, in.StartsOn, in.EndsOn); err != nil {
			return err
		}
		var err error
		p, err = scan(tx.QueryRow(ctx, selectPromotion+` WHERE tenant_id=$1 AND id=$2`, tenantID, promotionID))
		return err
	})
	return p, conflict(err)
}

// Update replaces a promotion's terms. Sales already made keep the discount
// they were given; only later sales see the change.
func (r *postgresRepository) Update(ctx context.Context, tenantID, promotionID string, in Input, status *string) (*Promotion, error) {
	var p *Promotion
	err := withPromotionTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE promotions SET name=$3, kind=$4, percent_bps=$5, amount_minor=$6, applies_to=$7,
			min_subtotal_minor=$8, starts_on=$9, ends_on=$10, status=COALESCE($11, status)
			WHERE tenant_id=$1 AND id=$2`, tenantID, promotionID, in.Name, in.Kind, in.PercentBps, in.AmountMinor,
			in.AppliesTo, in.MinSubtotalMinor, in.StartsOn, in.EndsOn, status)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		p, err = scan(tx.QueryRow(ctx, selectPromotion+` WHERE tenant_id=$1 AND id=$2`, tenantID, promotionID))
		return err
	})
	return p, conflict(err)
}

func (r *postgresRepository) Settings(ctx context.Context, tenantID string) (Settings, error) {
	var s Settings
	err := r.db.QueryRow(ctx, `SELECT cashier_discount_limit_bps FROM tenants WHERE id=$1`, tenantID).Scan(&s.CashierLimitBps)
	return s, err
}

func (r *postgresRepository) SaveSettings(ctx context.Context, tenantID string, s Settings) (Settings, error) {
	err := r.db.QueryRow(ctx, `UPDATE tenants SET cashier_discount_limit_bps=$2 WHERE id=$1 RETURNING cashier_discount_limit_bps`,
		tenantID, s.CashierLimitBps).Scan(&s.CashierLimitBps)
	return s, err
}

// ForSale returns a promotion a sale may use today, in a transaction that is
// already scoped to the tenant. today is the business day (YYYY-MM-DD) in the
// workspace's timezone.
func ForSale(ctx context.Context, tx pgx.Tx, tenantID, promotionID, appliesTo, today string) (*Promotion, error) {
	p, err := scan(tx.QueryRow(ctx, selectPromotion+` WHERE tenant_id=$1 AND id=$2 FOR SHARE`, tenantID, promotionID))
	if errors.Is(err, ErrNotFound) {
		return nil, &Error{http.StatusBadRequest, "promotion_unavailable", "This promotion was not found. Refresh and choose again."}
	}
	if err != nil {
		return nil, err
	}
	switch {
	case p.Status != "active":
		return nil, &Error{http.StatusConflict, "promotion_unavailable", fmt.Sprintf("%s has ended and can no longer be used.", p.Name)}
	case p.StartsOn != nil && today < *p.StartsOn:
		return nil, &Error{http.StatusConflict, "promotion_not_started", fmt.Sprintf("%s starts on %s.", p.Name, *p.StartsOn)}
	case p.EndsOn != nil && today > *p.EndsOn:
		return nil, &Error{http.StatusConflict, "promotion_expired", fmt.Sprintf("%s ended on %s.", p.Name, *p.EndsOn)}
	case p.AppliesTo != appliesTo:
		return nil, &Error{http.StatusBadRequest, "promotion_wrong_scope",
			fmt.Sprintf("%s applies to %s, not %s.", p.Name, scopeWords(p.AppliesTo), scopeWords(appliesTo))}
	}
	return p, nil
}

func scopeWords(appliesTo string) string {
	if appliesTo == "item" {
		return "single items"
	}
	return "the whole order"
}
