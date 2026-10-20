package customer

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db} }
func (r *postgresRepository) tx(ctx context.Context, tenantID string, fn func(pgx.Tx) error) error {
	return database.WithTenantTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx)
	})
}
func scanCustomer(row pgx.Row) (*Customer, error) {
	var c Customer
	err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}
func (r *postgresRepository) List(ctx context.Context, tenantID, search string, full bool, limit, offset int) ([]Customer, error) {
	out := []Customer{}
	err := r.tx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id,name,CASE WHEN $3 THEN phone ELSE '' END,
			CASE WHEN $3 THEN email ELSE '' END,status,created_at,updated_at FROM customers
			WHERE tenant_id=$1 AND ($3 OR status='active') AND ($2='' OR name ILIKE '%'||$2||'%'
			OR phone ILIKE '%'||$2||'%' OR phone ILIKE '%'||$6||'%' OR email ILIKE '%'||$2||'%')
			ORDER BY (status='archived'),lower(name),id LIMIT $4 OFFSET $5`, tenantID, search, full, limit, offset, normalizePhone(search))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c Customer
			if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}
func (r *postgresRepository) Create(ctx context.Context, tenantID string, in Input) (*Customer, error) {
	var out *Customer
	err := r.tx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		out, err = scanCustomer(tx.QueryRow(ctx, `INSERT INTO customers (id,tenant_id,name,phone,email)
			VALUES ($1,$2,$3,$4,$5) RETURNING id,name,phone,email,status,created_at,updated_at`,
			id.New(id.Customer), tenantID, in.Name, in.Phone, in.Email))
		return err
	})
	return out, err
}
func (r *postgresRepository) Update(ctx context.Context, tenantID, customerID string, in UpdateInput) (*Customer, error) {
	var out *Customer
	err := r.tx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		out, err = scanCustomer(tx.QueryRow(ctx, `UPDATE customers SET
			name=COALESCE($3,name),phone=COALESCE($4,phone),email=COALESCE($5,email),status=COALESCE($6,status)
			WHERE tenant_id=$1 AND id=$2 AND (COALESCE($4,phone)<>'' OR COALESCE($5,email)<>'')
			RETURNING id,name,phone,email,status,created_at,updated_at`, tenantID, customerID, in.Name, in.Phone, in.Email, in.Status))
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE tenant_id=$1 AND id=$2)`, tenantID, customerID).Scan(&exists); e != nil {
				return e
			}
			if exists {
				return ErrInvalid
			}
			return ErrNotFound
		}
		return err
	})
	return out, err
}
func (r *postgresRepository) History(ctx context.Context, tenantID, customerID string, limit, offset int) ([]SaleSummary, error) {
	out := []SaleSummary{}
	err := r.tx(ctx, tenantID, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE tenant_id=$1 AND id=$2)`, tenantID, customerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		rows, err := tx.Query(ctx, `SELECT id,order_number,store_name_snapshot,currency_code,grand_total_minor,created_at
			FROM orders WHERE tenant_id=$1 AND customer_id=$2 ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, tenantID, customerID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var sale SaleSummary
			if err := rows.Scan(&sale.ID, &sale.OrderNumber, &sale.StoreName, &sale.CurrencyCode, &sale.GrandTotalMinor, &sale.CreatedAt); err != nil {
				return err
			}
			out = append(out, sale)
		}
		return rows.Err()
	})
	return out, err
}
