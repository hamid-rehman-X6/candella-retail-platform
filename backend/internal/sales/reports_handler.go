package sales

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/store"
	"candella-ecosystem/backend/internal/tenant"
	"candella-ecosystem/backend/pkg/response"
)

func (h *Handler) report(w http.ResponseWriter, r *http.Request) *DailyReport {
	t, _ := tenant.FromContext(r.Context())
	if t.Role != "owner" && t.Role != "admin" {
		response.Error(w, http.StatusForbidden, "insufficient_role", "Ask a workspace owner or admin for reports access.")
		return nil
	}
	u, _ := auth.UserFromContext(r.Context())
	storeID := r.URL.Query().Get("storeId")
	if storeID == "" {
		h.writeErr(w, store.ErrSelection)
		return nil
	}
	filter := ReportFilter{FromDate: r.URL.Query().Get("from"), ToDate: r.URL.Query().Get("to")}
	ctx := store.WithSelection(r.Context(), storeID, u.ID, t.Role)
	report, err := h.svc.DailyReport(ctx, t.TenantID, filter)
	if err != nil {
		h.writeErr(w, err)
		return nil
	}
	return report
}

func (h *Handler) DailyReport(w http.ResponseWriter, r *http.Request) {
	report := h.report(w, r)
	if report == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.JSON(w, http.StatusOK, map[string]any{"report": report})
}

func (h *Handler) DailyReportCSV(w http.ResponseWriter, r *http.Request) {
	report := h.report(w, r)
	if report == nil {
		return
	}
	data, err := reportCSV(report)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="candella-report-`+report.FromDate+`-to-`+report.ToDate+`.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// Spreadsheet software treats these leading characters as formulas. Prefix
// operator-entered text so opening a report cannot execute a CSV formula.
func safeCSV(value string) string {
	if strings.HasPrefix(value, "=") || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") ||
		strings.HasPrefix(value, "@") || strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") {
		return "'" + value
	}
	return value
}

func reportCSV(report *DailyReport) ([]byte, error) {
	var buffer bytes.Buffer
	w := csv.NewWriter(&buffer)
	write := func(values ...string) error { return w.Write(values) }
	n := func(value int64) string { return strconv.FormatInt(value, 10) }
	if err := write("Candella daily report", report.FromDate, "to", report.ToDate); err != nil {
		return nil, err
	}
	if err := write("Store", safeCSV(report.StoreName), "Currency", report.CurrencyCode, "Timezone", report.Timezone); err != nil {
		return nil, err
	}
	if err := write("Amounts are integer minor units; net = gross cash - cash refunds"); err != nil {
		return nil, err
	}
	if err := write("Date", "Sales count", "Gross cash minor", "Refund count", "Cash refunds minor", "Net sales minor"); err != nil {
		return nil, err
	}
	for _, day := range report.Days {
		if err := write(day.Date, n(day.SalesCount), n(day.GrossCashMinor), n(day.RefundsCount), n(day.RefundsMinor), n(day.NetSalesMinor)); err != nil {
			return nil, err
		}
	}
	if err := write("TOTAL", n(report.Totals.SalesCount), n(report.Totals.GrossCashMinor), n(report.Totals.RefundsCount), n(report.Totals.RefundsMinor), n(report.Totals.NetSalesMinor)); err != nil {
		return nil, err
	}
	if err := write(); err != nil {
		return nil, err
	}
	if err := write("Top products by gross sales"); err != nil {
		return nil, err
	}
	if err := write("Product", "Units sold", "Gross sales minor"); err != nil {
		return nil, err
	}
	for _, product := range report.TopProducts {
		if err := write(safeCSV(product.ProductName), n(product.UnitsSold), n(product.GrossSalesMinor)); err != nil {
			return nil, err
		}
	}
	if err := write(); err != nil {
		return nil, err
	}
	if err := write("Stock movement totals"); err != nil {
		return nil, err
	}
	if err := write("Type", "Movements", "Units in", "Units out", "Net units"); err != nil {
		return nil, err
	}
	for _, movement := range report.StockMovements {
		if err := write(movement.Kind, n(movement.Movements), n(movement.UnitsIn), n(movement.UnitsOut), n(movement.NetUnits)); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buffer.Bytes(), w.Error()
}
