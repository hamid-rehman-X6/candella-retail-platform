package sales

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candella-ecosystem/backend/internal/tenant"
)

type reportRepo struct {
	Repository
	called bool
}

func (r *reportRepo) DailyReport(_ context.Context, _ string, _ ReportFilter) (*DailyReport, error) {
	r.called = true
	return &DailyReport{}, nil
}

func TestReportDateValidation(t *testing.T) {
	for _, filter := range []ReportFilter{
		{FromDate: "2026-02-30", ToDate: "2026-03-01"},
		{FromDate: "2026-10-21", ToDate: "2026-10-20"},
		{FromDate: "2026-01-01", ToDate: "2027-01-02"},
		{FromDate: "2026-1-01", ToDate: "2026-01-02"},
	} {
		repo := &reportRepo{}
		if _, err := NewService(repo).DailyReport(context.Background(), "tnt_123456789012", filter); err != ErrInvalidReport || repo.called {
			t.Fatalf("invalid filter %+v reached repo: %v", filter, err)
		}
	}
	repo := &reportRepo{}
	if _, err := NewService(repo).DailyReport(context.Background(), "tnt_123456789012", ReportFilter{"2026-10-20", "2026-10-20"}); err != nil || !repo.called {
		t.Fatalf("valid report: %v", err)
	}
}

func TestReportEndpointsRejectMembers(t *testing.T) {
	h := &Handler{}
	for _, invoke := range []func(http.ResponseWriter, *http.Request){h.DailyReport, h.DailyReportCSV} {
		req := httptest.NewRequest(http.MethodGet, "/reports/daily?storeId=str_123456789012&from=2026-10-20&to=2026-10-20", nil)
		req = req.WithContext(tenant.WithContext(req.Context(), &tenant.Context{TenantID: "tnt_123456789012", Role: "member"}))
		w := httptest.NewRecorder()
		invoke(w, req)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "insufficient_role") {
			t.Fatalf("member report response: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestReportCSVEscapesSpreadsheetFormula(t *testing.T) {
	report := &DailyReport{StoreName: "=HYPERLINK(\"bad\")", FromDate: "2026-10-20", ToDate: "2026-10-20", Days: []ReportDay{{Date: "2026-10-20", GrossCashMinor: 15000}}, TopProducts: []TopProduct{{ProductName: "+SUM(1,2)", UnitsSold: 1, GrossSalesMinor: 15000}}}
	data, err := reportCSV(report)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	storeSafe, productSafe := false, false
	for _, row := range rows {
		for _, cell := range row {
			if cell == "'=HYPERLINK(\"bad\")" {
				storeSafe = true
			}
			if cell == "'+SUM(1,2)" {
				productSafe = true
			}
		}
	}
	if !storeSafe || !productSafe {
		t.Fatalf("unsafe CSV: %v", rows)
	}
}
