package sales

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"candella-ecosystem/backend/internal/register"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/tax"
	"candella-ecosystem/backend/pkg/id"
	"candella-ecosystem/backend/pkg/money"
)

func returnHash(userID, orderID string, in ReturnInput) string {
	payload, _ := json.Marshal(struct {
		UserID, OrderID, StoreID, RegisterID, Reason string
		Items                                        []CartItem
	}{userID, orderID, in.StoreID, in.RegisterID, in.Reason, in.Items})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (r *postgresRepository) CreateReturn(ctx context.Context, tenantID, userID, processorName, orderID string, in ReturnInput) (*Return, bool, error) {
	var ret *Return
	created := false
	hash := returnHash(userID, orderID, in)
	err := withSalesTx(ctx, r.db, tenantID, func(tx pgx.Tx, currency, _ string) error {
		location, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		// A retried request returns the return it already made.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext('return:' || $2))`, tenantID, in.IdempotencyKey); err != nil {
			return err
		}
		var existingID, existingHash string
		err = tx.QueryRow(ctx, `SELECT id, request_hash FROM returns WHERE tenant_id=$1 AND idempotency_key=$2`,
			tenantID, in.IdempotencyKey).Scan(&existingID, &existingHash)
		if err == nil {
			if existingHash != hash {
				return ErrReturnKeyReuse
			}
			ret, err = loadReturn(ctx, tx, tenantID, existingID)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// Returns against one sale run one at a time, so two of them can never
		// both see the same units as still returnable.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext('order-return:' || $2))`, tenantID, orderID); err != nil {
			return err
		}
		var saleStoreID, saleStoreName, orderNumber string
		err = tx.QueryRow(ctx, `SELECT o.store_id, o.store_name_snapshot, o.order_number FROM orders o
			JOIN payments p ON p.tenant_id=o.tenant_id AND p.order_id=o.id AND p.method='cash'
			WHERE o.tenant_id=$1 AND o.id=$2 AND o.status='completed'`, tenantID, orderID).
			Scan(&saleStoreID, &saleStoreName, &orderNumber)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if saleStoreID != location.ID {
			return &Error{http.StatusConflict, "return_wrong_store",
				fmt.Sprintf("This sale was made at %s. Process the return there, so stock and cash go back to the right place.", saleStoreName)}
		}
		// Taken before any stock lock, like checkout; exclusive because cash leaves the drawer.
		shiftID, err := register.LockOpenShiftForPayout(ctx, tx, tenantID, location.ID, in.RegisterID)
		if err != nil {
			return err
		}

		items := make([]ReturnItem, 0, len(in.Items))
		var refund, refundTax int64
		for _, line := range in.Items {
			var item ReturnItem
			var sold, returned, lineTotal, lineTax int64
			var inclusive *bool
			err := tx.QueryRow(ctx, `SELECT oi.product_name_snapshot, oi.variant_name_snapshot, oi.sku_snapshot,
				oi.quantity, oi.unit_price_minor, oi.line_total_minor, oi.tax_minor, oi.tax_inclusive,
				COALESCE((SELECT sum(ri.quantity) FROM return_items ri
				  WHERE ri.tenant_id=oi.tenant_id AND ri.order_id=oi.order_id AND ri.variant_id=oi.variant_id), 0)
				FROM order_items oi WHERE oi.tenant_id=$1 AND oi.order_id=$2 AND oi.variant_id=$3`,
				tenantID, orderID, line.VariantID).
				Scan(&item.ProductName, &item.VariantName, &item.SKU, &sold, &item.UnitPriceMinor,
					&lineTotal, &lineTax, &inclusive, &returned)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReturnItemNotInSale
			}
			if err != nil {
				return err
			}
			if remaining := sold - returned; line.Quantity > remaining {
				return &Error{http.StatusConflict, "return_exceeds_sale",
					fmt.Sprintf("Only %d unit(s) of %s can still be returned from this sale.", remaining, item.SKU)}
			}
			item.VariantID = line.VariantID
			item.Quantity = line.Quantity
			// The customer gets back what they paid for these units, tax
			// included, worked out from the sale's own snapshot. Shares of a
			// line always add up to the line, so a full return refunds the
			// line to the paisa however it was split.
			gross := tax.Gross(lineTotal, lineTax, inclusive != nil && *inclusive)
			item.LineRefundMinor = tax.Share(gross, sold, returned, item.Quantity)
			item.TaxMinor = tax.Share(lineTax, sold, returned, item.Quantity)
			refund += item.LineRefundMinor
			refundTax += item.TaxMinor
			items = append(items, item)
		}
		if refund <= 0 {
			return ErrReturnSaleWithoutPay
		}
		drawer, err := register.DrawerCash(ctx, tx, tenantID, shiftID)
		if err != nil {
			return err
		}
		if drawer < refund {
			return &Error{http.StatusConflict, "drawer_cash_short",
				fmt.Sprintf("The drawer should hold %s, less than the %s refund. Add cash to the drawer or use another counter.",
					money.Format(drawer, currency), money.Format(refund, currency))}
		}

		var sequence int64
		if err := tx.QueryRow(ctx, `INSERT INTO document_sequences (tenant_id, current_return_number) VALUES ($1, 1)
			ON CONFLICT (tenant_id) DO UPDATE SET current_return_number=document_sequences.current_return_number+1
			RETURNING current_return_number`, tenantID).Scan(&sequence); err != nil {
			return err
		}
		returnID := id.New(id.Return)
		returnNumber := fmt.Sprintf("RET-%06d", sequence)
		if _, err := tx.Exec(ctx, `INSERT INTO returns (id, tenant_id, store_id, order_id, register_session_id,
			return_number, idempotency_key, request_hash, reason, refund_minor, currency_code, created_by, processor_name_snapshot,
			tax_minor)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			returnID, tenantID, location.ID, orderID, shiftID, returnNumber, in.IdempotencyKey, hash,
			in.Reason, refund, currency, userID, processorName, refundTax); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := tx.Exec(ctx, `INSERT INTO return_items (id, tenant_id, return_id, order_id, variant_id,
				quantity, unit_price_minor, line_refund_minor, tax_minor) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				id.New(id.ReturnItem), tenantID, returnID, orderID, item.VariantID, item.Quantity,
				item.UnitPriceMinor, item.LineRefundMinor, item.TaxMinor); err != nil {
				return err
			}
			if err := restock(ctx, tx, tenantID, location.ID, userID, returnID, returnNumber, item); err != nil {
				return err
			}
		}
		ret, err = loadReturn(ctx, tx, tenantID, returnID)
		if err == nil {
			created = true
		}
		return err
	})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return nil, false, ErrReturnKeyReuse
		}
		return nil, false, err
	}
	return ret, created, nil
}

// restock puts returned units back on the shelf of the selling store and
// records the matching positive movement in the stock ledger.
func restock(ctx context.Context, tx pgx.Tx, tenantID, storeID, userID, returnID, returnNumber string, item ReturnItem) error {
	var before int64
	err := tx.QueryRow(ctx, `SELECT quantity FROM stock_levels WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3 FOR UPDATE`,
		tenantID, storeID, item.VariantID).Scan(&before)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := tx.Exec(ctx, `INSERT INTO stock_levels (id, tenant_id, store_id, variant_id, quantity) VALUES ($1,$2,$3,$4,$5)`,
			id.New(id.StockLevel), tenantID, storeID, item.VariantID, item.Quantity); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if _, err := tx.Exec(ctx, `UPDATE stock_levels SET quantity=quantity+$4 WHERE tenant_id=$1 AND store_id=$2 AND variant_id=$3`,
			tenantID, storeID, item.VariantID, item.Quantity); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO stock_movements (id, tenant_id, store_id, variant_id, kind, delta,
		quantity_before, quantity_after, note, created_by, return_id) VALUES ($1,$2,$3,$4,'return',$5,$6,$7,$8,$9,$10)`,
		id.New(id.StockMovement), tenantID, storeID, item.VariantID, item.Quantity, before, before+item.Quantity,
		"Return "+returnNumber, userID, returnID)
	return err
}

func (r *postgresRepository) FindReturn(ctx context.Context, tenantID, returnID string) (*Return, error) {
	var ret *Return
	err := withSalesTx(ctx, r.db, tenantID, func(tx pgx.Tx, _, _ string) error {
		var err error
		ret, err = loadReturn(ctx, tx, tenantID, returnID)
		if err != nil {
			return err
		}
		if userID, role, ok := store.Actor(ctx); ok && role == "member" {
			if _, err := store.Resolve(store.WithSelection(ctx, ret.StoreID, userID, role), tx, tenantID); err != nil {
				return ErrReturnNotFound
			}
		}
		return nil
	})
	return ret, err
}

func loadReturn(ctx context.Context, tx pgx.Tx, tenantID, returnID string) (*Return, error) {
	ret := &Return{Items: []ReturnItem{}}
	err := tx.QueryRow(ctx, `SELECT r.id, r.return_number, r.order_id, o.order_number, r.store_id, o.store_name_snapshot,
		rs.register_name_snapshot, r.register_session_id, r.reason, r.refund_method, r.refund_minor, r.tax_minor, r.currency_code,
		r.created_by, r.processor_name_snapshot, r.created_at
		FROM returns r
		JOIN orders o ON o.tenant_id=r.tenant_id AND o.id=r.order_id
		JOIN register_sessions rs ON rs.tenant_id=r.tenant_id AND rs.id=r.register_session_id
		WHERE r.tenant_id=$1 AND r.id=$2`, tenantID, returnID).
		Scan(&ret.ID, &ret.ReturnNumber, &ret.OrderID, &ret.OrderNumber, &ret.StoreID, &ret.StoreName,
			&ret.RegisterName, &ret.ShiftID, &ret.Reason, &ret.RefundMethod, &ret.RefundMinor, &ret.TaxMinor, &ret.CurrencyCode,
			&ret.ProcessedBy, &ret.ProcessorName, &ret.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReturnNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT ri.variant_id, oi.product_name_snapshot, oi.variant_name_snapshot, oi.sku_snapshot,
		ri.quantity, ri.unit_price_minor, ri.line_refund_minor, ri.tax_minor
		FROM return_items ri
		JOIN order_items oi ON oi.tenant_id=ri.tenant_id AND oi.order_id=ri.order_id AND oi.variant_id=ri.variant_id
		WHERE ri.tenant_id=$1 AND ri.return_id=$2 ORDER BY ri.variant_id`, tenantID, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ReturnItem
		if err := rows.Scan(&item.VariantID, &item.ProductName, &item.VariantName, &item.SKU,
			&item.Quantity, &item.UnitPriceMinor, &item.LineRefundMinor, &item.TaxMinor); err != nil {
			return nil, err
		}
		ret.Items = append(ret.Items, item)
	}
	return ret, rows.Err()
}

// saleReturns fills in what has already been taken back from a sale.
func saleReturns(ctx context.Context, tx pgx.Tx, tenantID string, sale *Sale) error {
	sale.Returns = []ReturnSummary{}
	rows, err := tx.Query(ctx, `SELECT id, return_number, refund_minor, processor_name_snapshot, reason, created_at
		FROM returns WHERE tenant_id=$1 AND order_id=$2 ORDER BY created_at, id`, tenantID, sale.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var summary ReturnSummary
		if err := rows.Scan(&summary.ID, &summary.ReturnNumber, &summary.RefundMinor, &summary.ProcessorName,
			&summary.Reason, &summary.CreatedAt); err != nil {
			return err
		}
		sale.RefundedMinor += summary.RefundMinor
		sale.Returns = append(sale.Returns, summary)
	}
	return rows.Err()
}
