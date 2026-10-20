package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"candella-ecosystem/backend/internal/store"
)

func (r *postgresRepository) ListLowStock(ctx context.Context, tenantID string, limit, offset int) ([]ReorderItem, error) {
	out := []ReorderItem{}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT v.id,p.id,p.name,v.name,v.sku,v.barcode,v.price_minor,
		  COALESCE(sl.quantity,0),rs.threshold,COALESCE(ps.id,fs.id),COALESCE(ps.name,fs.name),ps.id,
		  rs.threshold+1-COALESCE(sl.quantity,0) FROM reorder_settings rs
		  JOIN product_variants v ON v.tenant_id=rs.tenant_id AND v.id=rs.variant_id
		  JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id AND p.status='active'
		  LEFT JOIN stock_levels sl ON sl.tenant_id=rs.tenant_id AND sl.store_id=rs.store_id AND sl.variant_id=rs.variant_id
		  LEFT JOIN suppliers ps ON ps.tenant_id=rs.tenant_id AND ps.id=rs.preferred_supplier_id AND ps.status='active'
		  LEFT JOIN LATERAL (SELECT supplier.id,supplier.name FROM supplier_variants sv
		    JOIN suppliers supplier ON supplier.tenant_id=sv.tenant_id AND supplier.id=sv.supplier_id
		    WHERE sv.tenant_id=v.tenant_id AND sv.variant_id=v.id AND supplier.status='active'
		    ORDER BY (sv.cost_minor IS NULL),sv.cost_minor,lower(supplier.name),supplier.id LIMIT 1) fs ON true
		  WHERE rs.tenant_id=$1 AND rs.store_id=$2 AND COALESCE(sl.quantity,0)<=rs.threshold
		  ORDER BY COALESCE(sl.quantity,0), lower(p.name),lower(v.name),v.id LIMIT $3 OFFSET $4`,
			tenantID, location.ID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item ReorderItem
			if err := rows.Scan(&item.VariantID, &item.ProductID, &item.ProductName, &item.VariantName,
				&item.SKU, &item.Barcode, &item.PriceMinor, &item.Quantity, &item.ReorderThreshold,
				&item.PreferredSupplierID, &item.PreferredSupplierName, &item.ConfiguredSupplierID, &item.SuggestedQuantity); err != nil {
				return err
			}
			out = append(out, item)
		}
		return rows.Err()
	})
	return out, err
}

func (r *postgresRepository) SaveReorder(ctx context.Context, tenantID, variantID string, in ReorderInput) error {
	return withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if location.ID != in.StoreID {
			return store.ErrSelection
		}
		var status string
		err = tx.QueryRow(ctx, `SELECT p.status FROM product_variants v
		  JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
		  WHERE v.tenant_id=$1 AND v.id=$2 FOR SHARE OF p,v`, tenantID, variantID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if in.Threshold != nil && status != "active" {
			return ErrArchived
		}
		if in.Threshold == nil {
			_, err = tx.Exec(ctx, `DELETE FROM reorder_settings WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`, tenantID, in.StoreID, variantID)
			return err
		}
		if in.PreferredSupplierID != nil {
			var supplierStatus string
			err = tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE tenant_id=$1 AND id=$2 FOR SHARE`, tenantID, *in.PreferredSupplierID).Scan(&supplierStatus)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSupplierNotActive
			}
			if err != nil {
				return err
			}
			if supplierStatus != "active" {
				return ErrSupplierNotActive
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO reorder_settings (tenant_id,store_id,variant_id,threshold,preferred_supplier_id)
		  VALUES ($1,$2,$3,$4,$5) ON CONFLICT (tenant_id,store_id,variant_id)
		  DO UPDATE SET threshold=EXCLUDED.threshold,preferred_supplier_id=EXCLUDED.preferred_supplier_id,updated_at=now()`,
			tenantID, in.StoreID, variantID, *in.Threshold, in.PreferredSupplierID)
		return err
	})
}
