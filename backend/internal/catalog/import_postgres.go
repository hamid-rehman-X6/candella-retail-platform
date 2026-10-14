package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/id"
)

type importPostgres struct{ db *pgxpool.Pool }

func NewImportPostgresRepository(db *pgxpool.Pool) ImportRepository { return &importPostgres{db: db} }

type importCandidate struct {
	name, variantName, sku, barcode string
	price, quantity                 int64
	productID, variantID            string
}

type existingImportVariant struct{ variantID, productID, sku, barcode, status string }

func (r *importPostgres) currency(ctx context.Context, tenantID string) (string, error) {
	var currency string
	err := r.db.QueryRow(ctx, `SELECT currency_code FROM tenants WHERE id=$1 AND deleted_at IS NULL`, tenantID).Scan(&currency)
	return strings.TrimSpace(currency), err
}

func (r *importPostgres) PreviewImport(ctx context.Context, tenantID, mode string, rows []ImportRow) (*ImportPreview, error) {
	currency, err := r.currency(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var preview *ImportPreview
	err = withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		var err error
		preview, _, err = evaluateImport(ctx, tx, tenantID, currency, mode, rows)
		return err
	})
	return preview, err
}

func evaluateImport(ctx context.Context, tx pgx.Tx, tenantID, currency, mode string, rows []ImportRow) (*ImportPreview, []importCandidate, error) {
	preview := &ImportPreview{Rows: make([]ImportRowResult, 0, len(rows))}
	candidates := make([]importCandidate, len(rows))
	skus, barcodes, names := []string{}, []string{}, []string{}
	for _, row := range rows {
		if name := strings.ToLower(strings.TrimSpace(row.ProductName)); name != "" {
			names = append(names, name)
		}
		if sku := strings.ToUpper(strings.TrimSpace(row.SKU)); sku != "" {
			skus = append(skus, sku)
		}
		if barcode := strings.TrimSpace(row.Barcode); barcode != "" {
			barcodes = append(barcodes, barcode)
		}
	}
	bySKU, byBarcode := map[string]existingImportVariant{}, map[string]existingImportVariant{}
	lookup, err := tx.Query(ctx, `SELECT v.id,v.product_id,v.sku,COALESCE(v.barcode,''),p.status
		FROM product_variants v JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
		WHERE v.tenant_id=$1 AND (upper(v.sku)=ANY($2::text[]) OR v.barcode=ANY($3::text[]))`, tenantID, skus, barcodes)
	if err != nil {
		return nil, nil, err
	}
	for lookup.Next() {
		var match existingImportVariant
		if err := lookup.Scan(&match.variantID, &match.productID, &match.sku, &match.barcode, &match.status); err != nil {
			lookup.Close()
			return nil, nil, err
		}
		bySKU[strings.ToUpper(match.sku)] = match
		if match.barcode != "" {
			byBarcode[match.barcode] = match
		}
	}
	if err := lookup.Err(); err != nil {
		lookup.Close()
		return nil, nil, err
	}
	lookup.Close()
	existingNames := map[string]bool{}
	if mode == "create" {
		products, err := tx.Query(ctx, `SELECT lower(name) FROM products WHERE tenant_id=$1 AND lower(name)=ANY($2::text[])`, tenantID, names)
		if err != nil {
			return nil, nil, err
		}
		for products.Next() {
			var name string
			if err := products.Scan(&name); err != nil {
				products.Close()
				return nil, nil, err
			}
			existingNames[name] = true
		}
		if err := products.Err(); err != nil {
			products.Close()
			return nil, nil, err
		}
		products.Close()
	}
	seenSKU, seenBarcode, seenVariant, productNames := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]string{}
	for index, row := range rows {
		candidate := importCandidate{name: strings.TrimSpace(row.ProductName), variantName: strings.TrimSpace(row.VariantName),
			sku: strings.ToUpper(strings.TrimSpace(row.SKU)), barcode: strings.TrimSpace(row.Barcode)}
		if candidate.variantName == "" {
			candidate.variantName = "Standard"
		}
		result := ImportRowResult{Line: row.Line, ProductName: candidate.name, SKU: candidate.sku, Action: mode, Errors: []string{}}
		if candidate.name == "" || len(candidate.name) > 160 {
			result.Errors = append(result.Errors, "Product name must be 1–160 characters.")
		}
		if len(candidate.variantName) > 100 {
			result.Errors = append(result.Errors, "Variant name is too long.")
		}
		if len(candidate.sku) > 80 {
			result.Errors = append(result.Errors, "SKU is too long.")
		}
		if len(candidate.barcode) > 80 {
			result.Errors = append(result.Errors, "Barcode is too long.")
		}
		if price, err := parseImportMoney(row.SellingPrice, currency); err == nil {
			candidate.price = price
		} else {
			result.Errors = append(result.Errors, "Selling price is invalid for "+currency+".")
		}
		if quantity, err := strconv.ParseInt(row.OpeningQuantity, 10, 64); err == nil && quantity >= 0 && quantity <= 1_000_000_000 {
			candidate.quantity = quantity
		} else {
			result.Errors = append(result.Errors, "Opening quantity must be a whole number from 0 to 1,000,000,000.")
		}
		if candidate.sku != "" {
			if seenSKU[candidate.sku] {
				result.Errors = append(result.Errors, "Duplicate SKU in this file.")
			}
			seenSKU[candidate.sku] = true
		}
		if candidate.barcode != "" {
			if seenBarcode[candidate.barcode] {
				result.Errors = append(result.Errors, "Duplicate barcode in this file.")
			}
			seenBarcode[candidate.barcode] = true
		}
		if mode == "create" {
			if existingNames[strings.ToLower(candidate.name)] {
				result.Errors = append(result.Errors, "Product name already exists in this workspace. Use Update existing or rename it.")
			}
			if candidate.sku != "" {
				if _, found := bySKU[candidate.sku]; found {
					result.Errors = append(result.Errors, "SKU already belongs to a product in this workspace.")
				}
			}
			if candidate.barcode != "" {
				if _, found := byBarcode[candidate.barcode]; found {
					result.Errors = append(result.Errors, "Barcode already belongs to a product in this workspace.")
				}
			}
			variantKey := strings.ToLower(candidate.name) + "\x00" + strings.ToLower(candidate.variantName)
			if seenVariant[variantKey] {
				result.Errors = append(result.Errors, "This product and variant name appears more than once in the file.")
			}
			seenVariant[variantKey] = true
		} else {
			if candidate.sku == "" {
				result.Errors = append(result.Errors, "SKU is required to update an existing variant.")
			}
			match, found := bySKU[candidate.sku]
			if candidate.sku != "" && !found {
				result.Errors = append(result.Errors, "SKU does not exist in this workspace.")
			}
			if found {
				candidate.productID, candidate.variantID = match.productID, match.variantID
				if match.status != "active" {
					result.Errors = append(result.Errors, "Archived products cannot be updated by import.")
				}
				if oldName, seen := productNames[match.productID]; seen && oldName != candidate.name {
					result.Errors = append(result.Errors, "Variants of one product must use the same product name.")
				}
				productNames[match.productID] = candidate.name
			}
			if candidate.barcode != "" {
				if barcodeOwner, used := byBarcode[candidate.barcode]; used && barcodeOwner.variantID != candidate.variantID {
					result.Errors = append(result.Errors, "Barcode belongs to another variant.")
				}
			}
			if candidate.quantity != 0 {
				result.Errors = append(result.Errors, "Update existing does not change stock. Use Inventory for stock changes.")
			}
		}
		if len(result.Errors) > 0 {
			result.Action = "invalid"
			preview.InvalidCount++
		} else if mode == "create" {
			preview.CreateCount++
		} else {
			preview.UpdateCount++
		}
		preview.Rows = append(preview.Rows, result)
		candidates[index] = candidate
	}
	return preview, candidates, nil
}

func (r *importPostgres) ConfirmImport(ctx context.Context, tenantID, userID, mode, key, hash string, rows []ImportRow) (*ImportResult, error) {
	currency, err := r.currency(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := &ImportResult{}
	err = withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1),hashtext($2))`, tenantID, key); err != nil {
			return err
		}
		var oldHash string
		var oldRows []byte
		err := tx.QueryRow(ctx, `SELECT request_hash,created_count,updated_count,result_rows FROM catalog_import_batches
			WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, key).Scan(&oldHash, &result.Created, &result.Updated, &oldRows)
		if err == nil {
			if oldHash != hash {
				return ErrImportKeyReuse
			}
			if err := json.Unmarshal(oldRows, &result.Rows); err != nil {
				return err
			}
			result.Replayed = true
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		preview, candidates, err := evaluateImport(ctx, tx, tenantID, currency, mode, rows)
		if err != nil {
			return err
		}
		if preview.InvalidCount != 0 {
			return ErrImportChanged
		}
		if mode == "create" {
			storeID, err := defaultStore(ctx, tx, tenantID)
			if err != nil {
				return err
			}
			productIDs := map[string]string{}
			for index, row := range candidates {
				productKey := strings.ToLower(row.name)
				productID := productIDs[productKey]
				if productID == "" {
					productID = id.New(id.Product)
					if _, err := tx.Exec(ctx, `INSERT INTO products (id,tenant_id,name) VALUES ($1,$2,$3)`, productID, tenantID, row.name); err != nil {
						return err
					}
					productIDs[productKey] = productID
				}
				variantID := id.New(id.ProductVariant)
				sku := row.sku
				if sku == "" {
					sku = skuForVariant(variantID)
				}
				preview.Rows[index].SKU = sku
				var barcode *string
				if row.barcode != "" {
					barcode = &row.barcode
				}
				if _, err := tx.Exec(ctx, `INSERT INTO product_variants (id,tenant_id,product_id,name,sku,barcode,price_minor)
					VALUES ($1,$2,$3,$4,$5,$6,$7)`, variantID, tenantID, productID, row.variantName, sku, barcode, row.price); err != nil {
					return err
				}
				if row.quantity > 0 {
					if _, err := tx.Exec(ctx, `INSERT INTO stock_levels (id,tenant_id,store_id,variant_id,quantity) VALUES ($1,$2,$3,$4,$5)`, id.New(id.StockLevel), tenantID, storeID, variantID, row.quantity); err != nil {
						return err
					}
					if _, err := tx.Exec(ctx, `INSERT INTO stock_movements (id,tenant_id,store_id,variant_id,kind,delta,quantity_before,quantity_after,note,created_by)
						VALUES ($1,$2,$3,$4,'receive',$5,0,$5,'Opening stock from CSV import',$6)`, id.New(id.StockMovement), tenantID, storeID, variantID, row.quantity, userID); err != nil {
						return err
					}
				}
				result.Created++
			}
		} else {
			for _, row := range candidates {
				if _, err := tx.Exec(ctx, `UPDATE products SET name=$3 WHERE tenant_id=$1 AND id=$2`, tenantID, row.productID, row.name); err != nil {
					return err
				}
				var barcode *string
				if row.barcode != "" {
					barcode = &row.barcode
				}
				if _, err := tx.Exec(ctx, `UPDATE product_variants SET name=$3, barcode=$4, price_minor=$5
					WHERE tenant_id=$1 AND id=$2`, tenantID, row.variantID, row.variantName, barcode, row.price); err != nil {
					return err
				}
				result.Updated++
			}
		}
		result.Rows = preview.Rows
		rowJSON, err := json.Marshal(result.Rows)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO catalog_import_batches
			(id,tenant_id,idempotency_key,request_hash,mode,created_count,updated_count,result_rows,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id.New("cimp"), tenantID, key, hash, mode, result.Created, result.Updated, rowJSON, userID)
		return err
	})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return nil, ErrImportChanged
		}
		return nil, err
	}
	return result, nil
}

func spreadsheetSafe(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", []rune(trimmed)[0]) {
		return "'" + value
	}
	return value
}

func (r *importPostgres) ExportProducts(ctx context.Context, tenantID string) ([][]string, error) {
	currency, err := r.currency(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := [][]string{importColumns}
	err = withCatalogTx(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT p.name,v.name,v.sku,COALESCE(v.barcode,''),v.price_minor,COALESCE(s.quantity,0)
			FROM product_variants v JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
			LEFT JOIN stores st ON st.tenant_id=v.tenant_id AND st.is_default
			LEFT JOIN stock_levels s ON s.tenant_id=v.tenant_id AND s.store_id=st.id AND s.variant_id=v.id
			WHERE v.tenant_id=$1 ORDER BY p.name,v.name,v.id LIMIT 50001`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var name, variant, sku, barcode string
			var price, quantity int64
			if err := rows.Scan(&name, &variant, &sku, &barcode, &price, &quantity); err != nil {
				return err
			}
			if len(result) > 50000 {
				return ErrImportFile
			}
			result = append(result, []string{spreadsheetSafe(name), spreadsheetSafe(variant), spreadsheetSafe(sku), spreadsheetSafe(barcode), formatImportMoney(price, currency), strconv.FormatInt(quantity, 10)})
		}
		return rows.Err()
	})
	return result, err
}
