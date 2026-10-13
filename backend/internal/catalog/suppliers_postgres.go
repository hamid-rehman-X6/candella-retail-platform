package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"candella-ecosystem/backend/pkg/id"
)

func (r *postgresRepository) ListSuppliers(ctx context.Context, tenantID, search string, limit, offset int) ([]Supplier, error) {
	out := make([]Supplier, 0)
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id, name, phone, email, address, status, created_at, updated_at
			FROM suppliers WHERE tenant_id=$1 AND ($2='' OR name ILIKE '%' || $2 || '%'
			OR phone ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%')
			ORDER BY (status='archived'), lower(name), id LIMIT $3 OFFSET $4`, tenantID, search, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var supplier Supplier
			if err := rows.Scan(&supplier.ID, &supplier.Name, &supplier.Phone, &supplier.Email, &supplier.Address,
				&supplier.Status, &supplier.CreatedAt, &supplier.UpdatedAt); err != nil {
				return err
			}
			out = append(out, supplier)
		}
		return rows.Err()
	})
	return out, err
}

func (r *postgresRepository) CreateSupplier(ctx context.Context, tenantID string, in SupplierInput) (*Supplier, error) {
	supplier := &Supplier{ID: id.New(id.Supplier), Name: in.Name, Phone: in.Phone, Email: in.Email, Address: in.Address, Status: "active"}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `INSERT INTO suppliers (id, tenant_id, name, phone, email, address)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at, updated_at`,
			supplier.ID, tenantID, in.Name, in.Phone, in.Email, in.Address).
			Scan(&supplier.CreatedAt, &supplier.UpdatedAt)
	})
	return supplier, err
}

func (r *postgresRepository) UpdateSupplier(ctx context.Context, tenantID, supplierID string, in SupplierUpdateInput) (*Supplier, error) {
	supplier := &Supplier{}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE suppliers SET name=COALESCE($3,name), phone=COALESCE($4,phone),
			email=COALESCE($5,email), address=COALESCE($6,address), status=COALESCE($7,status)
			WHERE tenant_id=$1 AND id=$2
			RETURNING id, name, phone, email, address, status, created_at, updated_at`,
			tenantID, supplierID, in.Name, in.Phone, in.Email, in.Address, in.Status).
			Scan(&supplier.ID, &supplier.Name, &supplier.Phone, &supplier.Email, &supplier.Address,
				&supplier.Status, &supplier.CreatedAt, &supplier.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return supplier, err
}

func (r *postgresRepository) ListSupplierSources(ctx context.Context, tenantID, supplierID string, limit, offset int) ([]SupplierSource, error) {
	out := make([]SupplierSource, 0)
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE tenant_id=$1 AND id=$2)`, tenantID, supplierID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		rows, err := tx.Query(ctx, `SELECT sv.supplier_id, sv.variant_id, p.name, v.name, v.sku, p.status,
			sv.supplier_sku, sv.cost_minor, sv.created_at, sv.updated_at
			FROM supplier_variants sv
			JOIN product_variants v ON v.tenant_id=sv.tenant_id AND v.id=sv.variant_id
			JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			WHERE sv.tenant_id=$1 AND sv.supplier_id=$2
			ORDER BY lower(p.name), lower(v.name), v.id LIMIT $3 OFFSET $4`, tenantID, supplierID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var source SupplierSource
			if err := rows.Scan(&source.SupplierID, &source.VariantID, &source.ProductName, &source.VariantName,
				&source.SKU, &source.ProductStatus, &source.SupplierSKU, &source.CostMinor,
				&source.CreatedAt, &source.UpdatedAt); err != nil {
				return err
			}
			out = append(out, source)
		}
		return rows.Err()
	})
	return out, err
}

func (r *postgresRepository) SaveSupplierSource(ctx context.Context, tenantID, supplierID string, in SupplierSourceInput) (*SupplierSource, error) {
	source := &SupplierSource{SupplierID: supplierID, VariantID: in.VariantID, SupplierSKU: in.SupplierSKU, CostMinor: in.CostMinor}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var supplierStatus string
		err := tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, supplierID).Scan(&supplierStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if supplierStatus == "archived" {
			return ErrSupplierArchived
		}
		err = tx.QueryRow(ctx, `SELECT p.name, v.name, v.sku, p.status
			FROM product_variants v JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			WHERE v.tenant_id=$1 AND v.id=$2 FOR UPDATE OF p`, tenantID, in.VariantID).
			Scan(&source.ProductName, &source.VariantName, &source.SKU, &source.ProductStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if source.ProductStatus == "archived" {
			return ErrArchived
		}
		return tx.QueryRow(ctx, `INSERT INTO supplier_variants (tenant_id, supplier_id, variant_id, supplier_sku, cost_minor)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (tenant_id, supplier_id, variant_id) DO UPDATE
			SET supplier_sku=EXCLUDED.supplier_sku, cost_minor=EXCLUDED.cost_minor
			RETURNING created_at, updated_at`, tenantID, supplierID, in.VariantID, in.SupplierSKU, in.CostMinor).
			Scan(&source.CreatedAt, &source.UpdatedAt)
	})
	return source, err
}
