package sales

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/internal/register"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/tax"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/id"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

// Currency comes from the authenticated workspace. All business rows are then
// accessed as candella_catalog so tenant RLS applies even with a superuser pool.
func withSalesTx(ctx context.Context, db *pgxpool.Pool, tenantID string, fn func(pgx.Tx, string, string) error) error {
	return database.WithTenantTx(ctx, db, tenantID, func(tx pgx.Tx) error {
		var currency, timezone string
		if err := tx.QueryRow(ctx, `SELECT currency_code, timezone FROM tenants WHERE id=$1`, tenantID).Scan(&currency, &timezone); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		return fn(tx, currency, timezone)
	})
}

func requestHash(userID string, in CheckoutInput) string {
	payload, _ := json.Marshal(struct {
		UserID, StoreID, RegisterID string
		CustomerID                  string `json:"CustomerID,omitempty"`
		Items                       []CartItem
		Cash                        int64
	}{userID, in.StoreID, in.RegisterID, in.CustomerID, in.Items, in.CashReceivedMinor})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (r *postgresRepository) Create(ctx context.Context, tenantID, userID, cashierName string, in CheckoutInput) (*Sale, bool, error) {
	var sale *Sale
	created := false
	hash := requestHash(userID, in)
	err := withSalesTx(ctx, r.db, tenantID, func(tx pgx.Tx, currency, _ string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		storeID := location.ID
		// Serialize requests with the same key before checking or touching stock.
		// A hash collision only makes unrelated checkouts wait; uniqueness still
		// uses the full tenant ID and key in the orders table.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, tenantID, in.IdempotencyKey); err != nil {
			return err
		}
		var existingID, existingHash string
		err = tx.QueryRow(ctx, `SELECT id, request_hash FROM orders WHERE tenant_id=$1 AND idempotency_key=$2`,
			tenantID, in.IdempotencyKey).Scan(&existingID, &existingHash)
		if err == nil {
			if existingHash != hash {
				return ErrKeyReuse
			}
			sale, err = loadSale(ctx, tx, tenantID, existingID)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var customerID any
		customerName := ""
		if in.CustomerID != "" {
			err = tx.QueryRow(ctx, `SELECT name FROM customers WHERE tenant_id=$1 AND id=$2 AND status='active' FOR SHARE`,
				tenantID, in.CustomerID).Scan(&customerName)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCustomerUnavailable
			}
			if err != nil {
				return err
			}
			customerID = in.CustomerID
		}
		// Taken before any stock lock; closing the shift waits for this sale.
		shiftID, err := register.LockOpenShift(ctx, tx, tenantID, storeID, in.RegisterID)
		if err != nil {
			return err
		}

		items := make([]SaleItem, 0, len(in.Items))
		taxRates := make([]*string, 0, len(in.Items))
		before := make([]int64, 0, len(in.Items))
		var subtotal, taxTotal, exclusiveTax int64
		for _, cart := range in.Items {
			var item SaleItem
			var status string
			var taxRateID *string
			item.VariantID = cart.VariantID
			item.Quantity = cart.Quantity
			// The rate is read with the price under the same lock, so the sale
			// charges one consistent price-and-tax pair.
			err := tx.QueryRow(ctx, `SELECT p.name, v.name, v.sku, v.price_minor, p.status,
				t.id, t.name, t.rate_bps, t.inclusive
				FROM product_variants v JOIN products p ON p.tenant_id=v.tenant_id AND p.id=v.product_id
				LEFT JOIN tax_rates t ON t.tenant_id=p.tenant_id AND t.id=p.tax_rate_id
				WHERE v.tenant_id=$1 AND v.id=$2 FOR UPDATE OF p, v`, tenantID, cart.VariantID).
				Scan(&item.ProductName, &item.VariantName, &item.SKU, &item.UnitPriceMinor, &status,
					&taxRateID, &item.TaxName, &item.TaxRateBps, &item.TaxInclusive)
			if errors.Is(err, pgx.ErrNoRows) {
				return &Error{http.StatusNotFound, "variant_not_found", "A cart item was not found in this workspace. Remove it and try again."}
			}
			if err != nil {
				return err
			}
			if status != "active" {
				return ErrArchived
			}
			var available int64
			err = tx.QueryRow(ctx, `SELECT quantity FROM stock_levels
				WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3 FOR UPDATE`, tenantID, storeID, cart.VariantID).
				Scan(&available)
			if errors.Is(err, pgx.ErrNoRows) {
				available = 0
			} else if err != nil {
				return err
			}
			if available < cart.Quantity {
				return &Error{http.StatusConflict, "insufficient_stock",
					fmt.Sprintf("Only %d unit(s) of %s are available. Reduce the quantity or receive stock.", available, item.SKU)}
			}
			item.LineTotalMinor, subtotal, err = addLine(subtotal, item.UnitPriceMinor, item.Quantity)
			if err != nil {
				return err
			}
			if item.TaxRateBps != nil {
				item.TaxMinor = tax.Line(item.LineTotalMinor, *item.TaxRateBps, *item.TaxInclusive)
				taxTotal += item.TaxMinor
				if !*item.TaxInclusive {
					exclusiveTax += item.TaxMinor
				}
			}
			items = append(items, item)
			taxRates = append(taxRates, taxRateID)
			before = append(before, available)
		}
		total := subtotal + exclusiveTax
		if total > maxSaleMinor {
			return ErrTotalLarge
		}
		if in.CashReceivedMinor < total {
			return ErrCashShort
		}

		var sequence int64
		if err := tx.QueryRow(ctx, `INSERT INTO document_sequences (tenant_id, current_number) VALUES ($1,1)
			ON CONFLICT (tenant_id) DO UPDATE SET current_number=document_sequences.current_number+1
			RETURNING current_number`, tenantID).Scan(&sequence); err != nil {
			return err
		}
		orderID := id.New(id.Order)
		orderNumber := fmt.Sprintf("POS-%06d", sequence)
		if _, err := tx.Exec(ctx, `INSERT INTO orders
			(id, tenant_id, store_id, store_name_snapshot, order_number, idempotency_key, request_hash,
			 currency_code, subtotal_minor, grand_total_minor, created_by, cashier_name_snapshot, register_session_id,
			 customer_id, customer_name_snapshot, tax_minor)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, orderID, tenantID, storeID, location.Name, orderNumber,
			in.IdempotencyKey, hash, currency, subtotal, total, userID, cashierName, shiftID, customerID, customerName, taxTotal); err != nil {
			return err
		}
		for index, item := range items {
			if _, err := tx.Exec(ctx, `INSERT INTO order_items
				(id, tenant_id, order_id, variant_id, product_name_snapshot, variant_name_snapshot,
				 sku_snapshot, quantity, unit_price_minor, line_total_minor,
				 tax_rate_id, tax_name_snapshot, tax_rate_bps, tax_inclusive, tax_minor)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, id.New(id.OrderItem), tenantID, orderID,
				item.VariantID, item.ProductName, item.VariantName, item.SKU, item.Quantity,
				item.UnitPriceMinor, item.LineTotalMinor,
				taxRates[index], item.TaxName, item.TaxRateBps, item.TaxInclusive, item.TaxMinor); err != nil {
				return err
			}
			result, err := tx.Exec(ctx, `UPDATE stock_levels SET quantity=quantity-$4
				WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3 AND quantity >= $4`,
				tenantID, storeID, item.VariantID, item.Quantity)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return &Error{http.StatusConflict, "insufficient_stock", "Stock changed during checkout. Refresh the cart and try again."}
			}
			if _, err := tx.Exec(ctx, `INSERT INTO stock_movements
				(id, tenant_id, store_id, variant_id, kind, delta, quantity_before, quantity_after,
				 note, created_by, order_id) VALUES ($1,$2,$3,$4,'sale',$5,$6,$7,'Cash sale',$8,$9)`,
				id.New(id.StockMovement), tenantID, storeID, item.VariantID, -item.Quantity,
				before[index], before[index]-item.Quantity, userID, orderID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO payments
			(id, tenant_id, order_id, method, status, amount_minor, cash_received_minor,
			 change_minor, currency_code) VALUES ($1,$2,$3,'cash','completed',$4,$5,$6,$7)`,
			id.New(id.Payment), tenantID, orderID, total, in.CashReceivedMinor,
			in.CashReceivedMinor-total, currency); err != nil {
			return err
		}
		sale, err = loadSale(ctx, tx, tenantID, orderID)
		if err == nil {
			created = true
		}
		return err
	})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return nil, false, ErrKeyReuse
		}
		return nil, false, err
	}
	return sale, created, nil
}

func (r *postgresRepository) Find(ctx context.Context, tenantID, orderID string) (*Sale, error) {
	var sale *Sale
	err := withSalesTx(ctx, r.db, tenantID, func(tx pgx.Tx, _, _ string) error {
		var err error
		sale, err = loadSale(ctx, tx, tenantID, orderID)
		if err != nil {
			return err
		}
		if userID, role, ok := store.Actor(ctx); ok && role == "member" {
			if _, err := store.Resolve(store.WithSelection(ctx, sale.StoreID, userID, role), tx, tenantID); err != nil {
				return ErrNotFound
			}
		}
		return err
	})
	return sale, err
}

func (r *postgresRepository) Search(ctx context.Context, tenantID string, filter SalesFilter) ([]SaleSummary, bool, error) {
	results := []SaleSummary{}
	hasMore := false
	actorID, role, hasActor := store.Actor(ctx)
	canViewAll := !hasActor || role != "member"
	err := withSalesTx(ctx, r.db, tenantID, func(tx pgx.Tx, _, timezone string) error {
		location, err := time.LoadLocation(timezone)
		if err != nil {
			return fmt.Errorf("workspace timezone %q: %w", timezone, err)
		}
		var from, until *time.Time
		if filter.FromDate != "" {
			day, err := time.ParseInLocation("2006-01-02", filter.FromDate, location)
			if err != nil {
				return ErrInvalidFilter
			}
			from = &day
		}
		if filter.ToDate != "" {
			day, err := time.ParseInLocation("2006-01-02", filter.ToDate, location)
			if err != nil {
				return ErrInvalidFilter
			}
			next := day.AddDate(0, 0, 1)
			until = &next
		}
		rows, err := tx.Query(ctx, `SELECT id, store_id, store_name_snapshot, order_number, created_by, cashier_name_snapshot,
			currency_code, grand_total_minor, created_at FROM orders
			WHERE tenant_id=$1 AND status='completed'
			AND ($2::timestamptz IS NULL OR created_at >= $2)
			AND ($3::timestamptz IS NULL OR created_at < $3)
			AND ($4::text = '' OR order_number ILIKE '%' || $4 || '%')
			AND ($5::text = '' OR cashier_name_snapshot ILIKE '%' || $5 || '%' OR created_by ILIKE '%' || $5 || '%')
			AND ($6::bigint IS NULL OR grand_total_minor >= $6)
			AND ($7::bigint IS NULL OR grand_total_minor <= $7)
			AND ($10::bool OR EXISTS(SELECT 1 FROM stores st WHERE st.tenant_id=orders.tenant_id AND st.id=orders.store_id
				AND (st.is_default OR EXISTS(SELECT 1 FROM store_access sa WHERE sa.tenant_id=st.tenant_id AND sa.store_id=st.id AND sa.user_id=$11))))
			ORDER BY created_at DESC, id DESC LIMIT $8 OFFSET $9`,
			tenantID, from, until, filter.OrderNumber, filter.Cashier,
			filter.MinAmountMinor, filter.MaxAmountMinor, filter.Limit+1, filter.Offset, canViewAll, actorID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var sale SaleSummary
			if err := rows.Scan(&sale.ID, &sale.StoreID, &sale.StoreName, &sale.OrderNumber, &sale.CashierID, &sale.CashierName,
				&sale.CurrencyCode, &sale.GrandTotalMinor, &sale.CreatedAt); err != nil {
				return err
			}
			if len(results) == filter.Limit {
				hasMore = true
				break
			}
			results = append(results, sale)
		}
		return rows.Err()
	})
	return results, hasMore, err
}

func loadSale(ctx context.Context, tx pgx.Tx, tenantID, orderID string) (*Sale, error) {
	sale := &Sale{Items: []SaleItem{}}
	err := tx.QueryRow(ctx, `SELECT o.id, o.store_id, o.store_name_snapshot, COALESCE(rs.register_name_snapshot, ''), COALESCE(o.register_session_id, ''),
		o.order_number, o.created_by, o.cashier_name_snapshot, o.status, o.currency_code,
		o.subtotal_minor, o.tax_minor, o.grand_total_minor, p.cash_received_minor, p.change_minor, o.created_at,
		COALESCE(o.customer_id,''), o.customer_name_snapshot
		FROM orders o JOIN payments p ON p.tenant_id=o.tenant_id AND p.order_id=o.id
		LEFT JOIN register_sessions rs ON rs.tenant_id=o.tenant_id AND rs.id=o.register_session_id
		WHERE o.tenant_id=$1 AND o.id=$2`, tenantID, orderID).
		Scan(&sale.ID, &sale.StoreID, &sale.StoreName, &sale.RegisterName, &sale.ShiftID, &sale.OrderNumber, &sale.CashierID, &sale.CashierName, &sale.Status, &sale.CurrencyCode, &sale.SubtotalMinor,
			&sale.TaxMinor, &sale.GrandTotalMinor, &sale.CashReceivedMinor, &sale.ChangeMinor, &sale.CreatedAt,
			&sale.CustomerID, &sale.CustomerName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT oi.variant_id, oi.product_name_snapshot, oi.variant_name_snapshot,
		oi.sku_snapshot, oi.quantity, oi.unit_price_minor, oi.line_total_minor,
		oi.tax_name_snapshot, oi.tax_rate_bps, oi.tax_inclusive, oi.tax_minor,
		COALESCE((SELECT sum(ri.quantity) FROM return_items ri
		  WHERE ri.tenant_id=oi.tenant_id AND ri.order_id=oi.order_id AND ri.variant_id=oi.variant_id), 0)
		FROM order_items oi
		WHERE oi.tenant_id=$1 AND oi.order_id=$2 ORDER BY oi.id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(&item.VariantID, &item.ProductName, &item.VariantName, &item.SKU,
			&item.Quantity, &item.UnitPriceMinor, &item.LineTotalMinor,
			&item.TaxName, &item.TaxRateBps, &item.TaxInclusive, &item.TaxMinor, &item.ReturnedQuantity); err != nil {
			return nil, err
		}
		sale.Items = append(sale.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return sale, saleReturns(ctx, tx, tenantID, sale)
}
