package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

// Catalog queries switch to a non-superuser role so RLS also applies when the
// local Docker connection uses PostgreSQL's superuser account.
func withCatalogTx(ctx context.Context, db *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	return database.WithTenantTx(ctx, db, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx)
	})
}

func (r *postgresRepository) ListProducts(ctx context.Context, tenantID, search string, limit, offset int) ([]Product, error) {
	out := make([]Product, 0)
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT p.id, p.name, p.description, p.status, p.created_at,
			       v.id, v.name, v.sku, v.barcode, v.price_minor, COALESCE(s.quantity, 0)
			FROM (SELECT p.* FROM products p WHERE p.tenant_id = $1
			      AND ($2 = '' OR p.name ILIKE '%' || $2 || '%'
			        OR EXISTS (SELECT 1 FROM product_variants sv WHERE sv.tenant_id = p.tenant_id
			                   AND sv.product_id = p.id AND (sv.sku ILIKE '%' || $2 || '%' OR sv.barcode ILIKE '%' || $2 || '%')))
			      ORDER BY p.created_at DESC, p.id DESC LIMIT $3 OFFSET $4) p
			JOIN product_variants v ON v.tenant_id = p.tenant_id AND v.product_id = p.id
			LEFT JOIN stores st ON st.tenant_id = p.tenant_id AND st.is_default
			LEFT JOIN stock_levels s ON s.tenant_id = p.tenant_id AND s.variant_id = v.id AND s.store_id = st.id
			ORDER BY p.created_at DESC, p.id DESC, v.created_at, v.id`, tenantID, search, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		index := make(map[string]int)
		for rows.Next() {
			var p Product
			var v Variant
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedAt,
				&v.ID, &v.Name, &v.SKU, &v.Barcode, &v.PriceMinor, &v.Quantity); err != nil {
				return err
			}
			v.ProductID = p.ID
			if i, ok := index[p.ID]; ok {
				out[i].Variants = append(out[i].Variants, v)
			} else {
				p.Variants = []Variant{v}
				index[p.ID] = len(out)
				out = append(out, p)
			}
		}
		return rows.Err()
	})
	return out, err
}

// skuForVariant encodes the case-sensitive variant ID suffix as uppercase hex.
// The encoding is injective, so case-insensitive SKU uniqueness holds for IDs.
func skuForVariant(variantID string) string { return fmt.Sprintf("SKU-%X", variantID[4:]) }

func (r *postgresRepository) CreateProduct(ctx context.Context, tenantID, userID string, in CreateProductInput) (*Product, error) {
	p := &Product{ID: id.New(id.Product), Name: in.Name, Description: in.Description, Status: "active", Variants: []Variant{}}
	v := Variant{ID: id.New(id.ProductVariant), ProductID: p.ID, Name: in.Variant.Name, SKU: in.Variant.SKU, PriceMinor: in.Variant.PriceMinor}
	if v.SKU == "" {
		v.SKU = skuForVariant(v.ID)
	}
	if in.Variant.Barcode != "" {
		v.Barcode = &in.Variant.Barcode
	}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO products (id, tenant_id, name, description) VALUES ($1,$2,$3,$4) RETURNING created_at`,
			p.ID, tenantID, p.Name, p.Description).Scan(&p.CreatedAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO product_variants (id, tenant_id, product_id, name, sku, barcode, price_minor)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, v.ID, tenantID, p.ID, v.Name, v.SKU, v.Barcode, v.PriceMinor)
		if err != nil || in.InitialQuantity == nil || *in.InitialQuantity == 0 {
			return err
		}
		storeID, err := defaultStore(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		v.Quantity = *in.InitialQuantity
		if _, err := tx.Exec(ctx, `INSERT INTO stock_levels (id, tenant_id, store_id, variant_id, quantity)
			VALUES ($1,$2,$3,$4,$5)`, id.New(id.StockLevel), tenantID, storeID, v.ID, v.Quantity); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO stock_movements (id, tenant_id, store_id, variant_id, kind,
			delta, quantity_before, quantity_after, note, created_by)
			VALUES ($1,$2,$3,$4,'receive',$5,0,$5,'Opening stock at product creation',$6)`,
			id.New(id.StockMovement), tenantID, storeID, v.ID, v.Quantity, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	p.Variants = []Variant{v}
	return p, nil
}

func (r *postgresRepository) UpdateProduct(ctx context.Context, tenantID, productID string, in UpdateProductInput) (*Product, error) {
	var p Product
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE products SET name = COALESCE($3, name), description = COALESCE($4, description),
			status = COALESCE($5, status) WHERE tenant_id = $1 AND id = $2
			RETURNING id, name, description, status, created_at`, tenantID, productID, in.Name, in.Description, in.Status).
			Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT v.id, v.name, v.sku, v.barcode, v.price_minor, COALESCE(s.quantity, 0)
			FROM product_variants v LEFT JOIN stores st ON st.tenant_id=v.tenant_id AND st.is_default
			LEFT JOIN stock_levels s ON s.tenant_id=v.tenant_id AND s.variant_id=v.id AND s.store_id=st.id
			WHERE v.tenant_id=$1 AND v.product_id=$2 ORDER BY v.created_at, v.id`, tenantID, productID)
		if err != nil {
			return err
		}
		defer rows.Close()
		p.Variants = []Variant{}
		for rows.Next() {
			var v Variant
			v.ProductID = productID
			if err := rows.Scan(&v.ID, &v.Name, &v.SKU, &v.Barcode, &v.PriceMinor, &v.Quantity); err != nil {
				return err
			}
			p.Variants = append(p.Variants, v)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postgresRepository) UpdateVariant(ctx context.Context, tenantID, variantID string, in UpdateVariantInput) (*Variant, error) {
	v := &Variant{ID: variantID}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var status string
		err := tx.QueryRow(ctx, `SELECT p.status FROM product_variants v JOIN products p
			ON p.tenant_id=v.tenant_id AND p.id=v.product_id WHERE v.tenant_id=$1 AND v.id=$2 FOR UPDATE OF p`, tenantID, variantID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "archived" {
			return ErrArchived
		}
		err = tx.QueryRow(ctx, `UPDATE product_variants SET name=COALESCE($3,name), sku=COALESCE($4,sku),
			barcode=CASE WHEN $5::boolean THEN NULLIF($6::text, '') ELSE barcode END,
			price_minor=COALESCE($7,price_minor) WHERE tenant_id=$1 AND id=$2
			RETURNING product_id, name, sku, barcode, price_minor`, tenantID, variantID, in.Name, in.SKU,
			in.Barcode != nil, in.Barcode, in.PriceMinor).
			Scan(&v.ProductID, &v.Name, &v.SKU, &v.Barcode, &v.PriceMinor)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT COALESCE(s.quantity,0) FROM stores st LEFT JOIN stock_levels s
			ON s.tenant_id=st.tenant_id AND s.store_id=st.id AND s.variant_id=$2
			WHERE st.tenant_id=$1 AND st.is_default`, tenantID, variantID).Scan(&v.Quantity)
	})
	return v, err
}

func (r *postgresRepository) AddVariant(ctx context.Context, tenantID, productID string, in CreateVariantInput) (*Variant, error) {
	v := &Variant{ID: id.New(id.ProductVariant), ProductID: productID, Name: in.Name, SKU: in.SKU, PriceMinor: in.PriceMinor}
	if v.SKU == "" {
		v.SKU = skuForVariant(v.ID)
	}
	if in.Barcode != "" {
		v.Barcode = &in.Barcode
	}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM products WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, productID).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if status == "archived" {
			return ErrArchived
		}
		_, err := tx.Exec(ctx, `INSERT INTO product_variants (id, tenant_id, product_id, name, sku, barcode, price_minor)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, v.ID, tenantID, productID, v.Name, v.SKU, v.Barcode, v.PriceMinor)
		return err
	})
	return v, err
}

func defaultStore(ctx context.Context, tx pgx.Tx, tenantID string) (string, error) {
	var storeID string
	err := tx.QueryRow(ctx, `SELECT id FROM stores WHERE tenant_id=$1 AND is_default`, tenantID).Scan(&storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("default store missing for tenant %s", tenantID)
	}
	return storeID, err
}

func (r *postgresRepository) Summary(ctx context.Context, tenantID string) (*Summary, error) {
	s := &Summary{}
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		storeID, err := defaultStore(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id=$1 AND status='active'`, tenantID).Scan(&s.ProductCount); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT count(*), COALESCE(sum(COALESCE(sl.quantity, 0)), 0),
			count(*) FILTER (WHERE COALESCE(sl.quantity, 0)=0)
			FROM product_variants v JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			LEFT JOIN stock_levels sl ON sl.tenant_id=v.tenant_id AND sl.variant_id=v.id AND sl.store_id=$2
			WHERE v.tenant_id=$1 AND p.status='active'`, tenantID, storeID).
			Scan(&s.VariantCount, &s.UnitsOnHand, &s.OutOfStock)
	})
	return s, err
}

func (r *postgresRepository) ListStock(ctx context.Context, tenantID, search string, limit, offset int) ([]StockItem, string, error) {
	out := make([]StockItem, 0)
	var storeID string
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var err error
		storeID, err = defaultStore(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT v.id, p.id, p.name, v.name, v.sku, v.barcode, v.price_minor,
			COALESCE(s.quantity, 0) FROM product_variants v
			JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			LEFT JOIN stock_levels s ON s.tenant_id=v.tenant_id AND s.variant_id=v.id AND s.store_id=$2
			WHERE v.tenant_id=$1 AND p.status='active' AND ($3='' OR p.name ILIKE '%' || $3 || '%'
			OR v.name ILIKE '%' || $3 || '%' OR v.sku ILIKE '%' || $3 || '%' OR v.barcode ILIKE '%' || $3 || '%')
			ORDER BY p.name, v.name, v.id LIMIT $4 OFFSET $5`, tenantID, storeID, search, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item StockItem
			if err := rows.Scan(&item.VariantID, &item.ProductID, &item.ProductName, &item.VariantName,
				&item.SKU, &item.Barcode, &item.PriceMinor, &item.Quantity); err != nil {
				return err
			}
			out = append(out, item)
		}
		return rows.Err()
	})
	return out, storeID, err
}

func movementRequestHash(userID string, in MovementInput) string {
	payload, _ := json.Marshal(struct {
		UserID, VariantID, Kind, Note string
		Quantity, CountedQuantity     *int64
	}{userID, in.VariantID, in.Kind, in.Note, in.Quantity, in.CountedQuantity})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (r *postgresRepository) RecordMovement(ctx context.Context, tenantID, userID string, in MovementInput) (*Movement, bool, error) {
	m := &Movement{ID: id.New(id.StockMovement), VariantID: in.VariantID, Kind: in.Kind, Note: in.Note}
	created := false
	hash := movementRequestHash(userID, in)
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// Serialize matching retries before reading or changing the balance.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, tenantID, in.IdempotencyKey); err != nil {
			return err
		}
		var existingHash string
		err := tx.QueryRow(ctx, `SELECT m.id, m.request_hash, m.variant_id, p.name, v.name, v.sku, m.kind,
			m.delta, m.quantity_before, m.quantity_after, m.note, m.created_at
			FROM stock_movements m
			JOIN product_variants v ON v.tenant_id=m.tenant_id AND v.id=m.variant_id
			JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			WHERE m.tenant_id=$1 AND m.idempotency_key=$2`, tenantID, in.IdempotencyKey).
			Scan(&m.ID, &existingHash, &m.VariantID, &m.ProductName, &m.VariantName, &m.SKU,
				&m.Kind, &m.Delta, &m.QuantityBefore, &m.QuantityAfter, &m.Note, &m.CreatedAt)
		if err == nil {
			if existingHash != hash {
				return ErrMovementKeyReuse
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		storeID, err := defaultStore(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		var status string
		err = tx.QueryRow(ctx, `SELECT p.name, v.name, v.sku, p.status FROM product_variants v
			JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			WHERE v.tenant_id=$1 AND v.id=$2 FOR UPDATE OF p`, tenantID, in.VariantID).
			Scan(&m.ProductName, &m.VariantName, &m.SKU, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "archived" {
			return ErrArchived
		}
		_, err = tx.Exec(ctx, `INSERT INTO stock_levels (id, tenant_id, store_id, variant_id, quantity)
			VALUES ($1,$2,$3,$4,0) ON CONFLICT (tenant_id, store_id, variant_id) DO NOTHING`,
			id.New(id.StockLevel), tenantID, storeID, in.VariantID)
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3 FOR UPDATE`,
			tenantID, storeID, in.VariantID).Scan(&m.QuantityBefore)
		if err != nil {
			return err
		}
		if in.Kind == "receive" {
			m.QuantityAfter = m.QuantityBefore + *in.Quantity
			if m.QuantityAfter > 1_000_000_000 {
				return ErrInvalid
			}
		} else {
			m.QuantityAfter = *in.CountedQuantity
		}
		m.Delta = m.QuantityAfter - m.QuantityBefore
		_, err = tx.Exec(ctx, `UPDATE stock_levels SET quantity=$4 WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`,
			tenantID, storeID, in.VariantID, m.QuantityAfter)
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `INSERT INTO stock_movements (id, tenant_id, store_id, variant_id, kind, delta,
			quantity_before, quantity_after, note, created_by, idempotency_key, request_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING created_at`, m.ID, tenantID, storeID, in.VariantID, in.Kind, m.Delta,
			m.QuantityBefore, m.QuantityAfter, in.Note, userID, in.IdempotencyKey, hash).Scan(&m.CreatedAt)
		if err == nil {
			created = true
		}
		return err
	})
	return m, created, err
}

func (r *postgresRepository) ListMovements(ctx context.Context, tenantID string, limit, offset int) ([]Movement, error) {
	out := make([]Movement, 0)
	err := withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT m.id, m.order_id, m.variant_id, p.name, v.name, v.sku, m.kind, m.delta,
			m.quantity_before, m.quantity_after, m.note, m.created_at FROM stock_movements m
			JOIN product_variants v ON v.tenant_id=m.tenant_id AND v.id=m.variant_id
			JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			WHERE m.tenant_id=$1 ORDER BY m.created_at DESC, m.id DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m Movement
			if err := rows.Scan(&m.ID, &m.OrderID, &m.VariantID, &m.ProductName, &m.VariantName, &m.SKU, &m.Kind,
				&m.Delta, &m.QuantityBefore, &m.QuantityAfter, &m.Note, &m.CreatedAt); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}
