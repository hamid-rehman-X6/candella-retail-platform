package sales

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"candella-ecosystem/backend/internal/store"
)

func (r *postgresRepository) DailyReport(ctx context.Context, tenantID string, filter ReportFilter) (*DailyReport, error) {
	report := &DailyReport{FromDate: filter.FromDate, ToDate: filter.ToDate,
		Days: []ReportDay{}, TopProducts: []TopProduct{}, StockMovements: []MovementTotal{}}
	// Each section must see the same committed snapshot, even while a cashier
	// completes a sale or a return during report generation.
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant_id', $1, true)`, tenantID); err != nil {
			return err
		}
		var currency, timezone string
		if err := tx.QueryRow(ctx, `SELECT currency_code, timezone FROM tenants WHERE id=$1`, tenantID).Scan(&currency, &timezone); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SET LOCAL ROLE candella_catalog`); err != nil {
			return err
		}
		location, err := time.LoadLocation(timezone)
		if err != nil {
			return fmt.Errorf("workspace timezone %q: %w", timezone, err)
		}
		selected, err := store.Resolve(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		report.StoreID, report.StoreName = selected.ID, selected.Name
		report.CurrencyCode, report.Timezone = currency, timezone
		start, err := time.ParseInLocation("2006-01-02", filter.FromDate, location)
		if err != nil {
			return ErrInvalidReport
		}
		endDay, err := time.ParseInLocation("2006-01-02", filter.ToDate, location)
		if err != nil {
			return ErrInvalidReport
		}
		end := endDay.AddDate(0, 0, 1)
		index := map[string]int{}
		for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
			key := day.Format("2006-01-02")
			index[key] = len(report.Days)
			report.Days = append(report.Days, ReportDay{Date: key})
		}
		rows, err := tx.Query(ctx, `SELECT to_char(o.created_at AT TIME ZONE $4, 'YYYY-MM-DD') AS business_day,
		  count(*), COALESCE(sum(p.amount_minor),0)::bigint, COALESCE(sum(o.tax_minor),0)::bigint, COALESCE(sum(o.discount_minor),0)::bigint FROM orders o
		  JOIN payments p ON p.tenant_id=o.tenant_id AND p.order_id=o.id AND p.method='cash' AND p.status='completed'
		  WHERE o.tenant_id=$1 AND o.store_id=$2 AND o.status='completed'
		    AND o.created_at >= $3 AND o.created_at < $5
		  GROUP BY business_day ORDER BY business_day`, tenantID, selected.ID, start, timezone, end)
		if err != nil {
			return err
		}
		for rows.Next() {
			var key string
			var count, gross, taxMinor, discountMinor int64
			if err := rows.Scan(&key, &count, &gross, &taxMinor, &discountMinor); err != nil {
				rows.Close()
				return err
			}
			if i, ok := index[key]; ok {
				report.Days[i].SalesCount, report.Days[i].GrossCashMinor, report.Days[i].TaxMinor = count, gross, taxMinor
				report.Days[i].DiscountMinor = discountMinor
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		rows, err = tx.Query(ctx, `SELECT to_char(created_at AT TIME ZONE $4, 'YYYY-MM-DD') AS business_day,
		  count(*), COALESCE(sum(refund_minor),0)::bigint, COALESCE(sum(tax_minor),0)::bigint FROM returns
		  WHERE tenant_id=$1 AND store_id=$2 AND refund_method='cash'
		    AND created_at >= $3 AND created_at < $5
		  GROUP BY business_day ORDER BY business_day`, tenantID, selected.ID, start, timezone, end)
		if err != nil {
			return err
		}
		for rows.Next() {
			var key string
			var count, refunds, refundedTax int64
			if err := rows.Scan(&key, &count, &refunds, &refundedTax); err != nil {
				rows.Close()
				return err
			}
			if i, ok := index[key]; ok {
				report.Days[i].RefundsCount, report.Days[i].RefundsMinor, report.Days[i].RefundedTaxMinor = count, refunds, refundedTax
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for i := range report.Days {
			day := &report.Days[i]
			day.NetSalesMinor = day.GrossCashMinor - day.RefundsMinor
			day.NetTaxMinor = day.TaxMinor - day.RefundedTaxMinor
			report.Totals.TaxMinor += day.TaxMinor
			report.Totals.RefundedTaxMinor += day.RefundedTaxMinor
			report.Totals.NetTaxMinor += day.NetTaxMinor
			report.Totals.DiscountMinor += day.DiscountMinor
			report.Totals.SalesCount += day.SalesCount
			report.Totals.GrossCashMinor += day.GrossCashMinor
			report.Totals.RefundsCount += day.RefundsCount
			report.Totals.RefundsMinor += day.RefundsMinor
			report.Totals.NetSalesMinor += day.NetSalesMinor
		}
		rows, err = tx.Query(ctx, `SELECT v.product_id, min(oi.product_name_snapshot),
		  COALESCE(sum(oi.quantity),0)::bigint, COALESCE(sum(oi.line_total_minor),0)::bigint
		  FROM orders o JOIN payments p ON p.tenant_id=o.tenant_id AND p.order_id=o.id
		    AND p.method='cash' AND p.status='completed'
		  JOIN order_items oi ON oi.tenant_id=o.tenant_id AND oi.order_id=o.id
		  JOIN product_variants v ON v.tenant_id=oi.tenant_id AND v.id=oi.variant_id
		  WHERE o.tenant_id=$1 AND o.store_id=$2 AND o.status='completed'
		    AND o.created_at >= $3 AND o.created_at < $4
		  GROUP BY v.product_id ORDER BY sum(oi.line_total_minor) DESC, v.product_id LIMIT 10`,
			tenantID, selected.ID, start, end)
		if err != nil {
			return err
		}
		for rows.Next() {
			var product TopProduct
			if err := rows.Scan(&product.ProductID, &product.ProductName, &product.UnitsSold, &product.GrossSalesMinor); err != nil {
				rows.Close()
				return err
			}
			report.TopProducts = append(report.TopProducts, product)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		rows, err = tx.Query(ctx, `SELECT kind,count(*),COALESCE(sum(greatest(delta,0)),0)::bigint,
		  COALESCE(sum(greatest(-delta,0)),0)::bigint,COALESCE(sum(delta),0)::bigint
		  FROM stock_movements WHERE tenant_id=$1 AND store_id=$2 AND created_at >= $3 AND created_at < $4
		  GROUP BY kind ORDER BY kind`, tenantID, selected.ID, start, end)
		if err != nil {
			return err
		}
		for rows.Next() {
			var movement MovementTotal
			if err := rows.Scan(&movement.Kind, &movement.Movements, &movement.UnitsIn, &movement.UnitsOut, &movement.NetUnits); err != nil {
				rows.Close()
				return err
			}
			report.StockMovements = append(report.StockMovements, movement)
		}
		err = rows.Err()
		rows.Close()
		return err
	})
	return report, err
}
