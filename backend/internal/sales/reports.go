package sales

import (
	"context"
	"net/http"
	"time"
)

type ReportFilter struct {
	FromDate string
	ToDate   string
}

type ReportDay struct {
	Date           string `json:"date"`
	SalesCount     int64  `json:"salesCount"`
	GrossCashMinor int64  `json:"grossCashMinor"`
	RefundsCount   int64  `json:"refundsCount"`
	RefundsMinor   int64  `json:"refundsMinor"`
	NetSalesMinor  int64  `json:"netSalesMinor"`
	// Tax charged on the day's sales, minus the tax handed back in refunds.
	TaxMinor         int64 `json:"taxMinor"`
	RefundedTaxMinor int64 `json:"refundedTaxMinor"`
	NetTaxMinor      int64 `json:"netTaxMinor"`
}

type TopProduct struct {
	ProductID       string `json:"productId"`
	ProductName     string `json:"productName"`
	UnitsSold       int64  `json:"unitsSold"`
	GrossSalesMinor int64  `json:"grossSalesMinor"`
}

type MovementTotal struct {
	Kind      string `json:"kind"`
	Movements int64  `json:"movements"`
	UnitsIn   int64  `json:"unitsIn"`
	UnitsOut  int64  `json:"unitsOut"`
	NetUnits  int64  `json:"netUnits"`
}

type DailyReport struct {
	StoreID        string          `json:"storeId"`
	StoreName      string          `json:"storeName"`
	CurrencyCode   string          `json:"currencyCode"`
	Timezone       string          `json:"timezone"`
	FromDate       string          `json:"fromDate"`
	ToDate         string          `json:"toDate"`
	Totals         ReportDay       `json:"totals"`
	Days           []ReportDay     `json:"days"`
	TopProducts    []TopProduct    `json:"topProducts"`
	StockMovements []MovementTotal `json:"stockMovements"`
}

var ErrInvalidReport = &Error{http.StatusBadRequest, "invalid_report_dates", "Choose valid dates in YYYY-MM-DD format, with a range of up to 366 days."}

func parseReportDate(value string) (time.Time, error) {
	day, err := time.Parse("2006-01-02", value)
	if err != nil || day.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidReport
	}
	return day, nil
}

func (s *Service) DailyReport(ctx context.Context, tenantID string, filter ReportFilter) (*DailyReport, error) {
	from, err := parseReportDate(filter.FromDate)
	if err != nil {
		return nil, err
	}
	to, err := parseReportDate(filter.ToDate)
	if err != nil || to.Before(from) || to.After(from.AddDate(0, 0, 365)) {
		return nil, ErrInvalidReport
	}
	return s.repo.DailyReport(ctx, tenantID, filter)
}
