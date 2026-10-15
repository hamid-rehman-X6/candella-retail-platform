// Package register owns cash registers (the counters in a store) and the shifts
// worked on them. A cash sale is only taken into an open shift, so closing a
// shift can account for every rupee that entered its drawer.
package register

import "time"

type Register struct {
	ID          string        `json:"id"`
	StoreID     string        `json:"storeId"`
	Name        string        `json:"name"`
	OpenSession *ShiftSummary `json:"openSession"`
}

// ShiftSummary is what any cashier at the counter may see about its open
// shift: enough to know whose drawer it is, but not the running cash total.
type ShiftSummary struct {
	ID         string    `json:"id"`
	OpenedBy   string    `json:"openedBy"`
	OpenerName string    `json:"openerName"`
	OpenedAt   time.Time `json:"openedAt"`
}

type Shift struct {
	ID               string     `json:"id"`
	StoreID          string     `json:"storeId"`
	StoreName        string     `json:"storeName"`
	RegisterID       string     `json:"registerId"`
	RegisterName     string     `json:"registerName"`
	Status           string     `json:"status"`
	CurrencyCode     string     `json:"currencyCode"`
	OpenedBy         string     `json:"openedBy"`
	OpenerName       string     `json:"openerName"`
	OpenedAt         time.Time  `json:"openedAt"`
	OpeningCashMinor int64      `json:"openingCashMinor"`
	ClosedBy         *string    `json:"closedBy"`
	CloserName       *string    `json:"closerName"`
	ClosedAt         *time.Time `json:"closedAt"`
	// For an open shift these are live; for a closed one they are what was
	// recorded at closing and never change.
	SalesCount        int64   `json:"salesCount"`
	CashSalesMinor    int64   `json:"cashSalesMinor"`
	RefundsCount      int64   `json:"refundsCount"`
	CashRefundsMinor  int64   `json:"cashRefundsMinor"`
	ExpectedCashMinor int64   `json:"expectedCashMinor"`
	CountedCashMinor  *int64  `json:"countedCashMinor"`
	VarianceMinor     *int64  `json:"varianceMinor"`
	VarianceReason    *string `json:"varianceReason"`
}

type CreateRegisterInput struct {
	StoreID string `json:"storeId"`
	Name    string `json:"name"`
}

type OpenInput struct {
	StoreID          string `json:"storeId"`
	RegisterID       string `json:"registerId"`
	OpeningCashMinor int64  `json:"openingCashMinor"`
}

type CloseInput struct {
	CountedCashMinor int64 `json:"countedCashMinor"`
	// The expected cash the closer was shown. If a sale lands in between, the
	// close is refused so nobody signs off a figure they did not see.
	ExpectedCashMinor int64  `json:"expectedCashMinor"`
	VarianceReason    string `json:"varianceReason"`
}

type ShiftFilter struct {
	StoreID string
	Status  string
	Limit   int
	Offset  int
}

// Actor is who is asking, for the rules that depend on it: cashiers see and
// close only their own shifts; owners and admins manage every shift.
type Actor struct {
	UserID string
	Name   string
	Role   string
}

func (a Actor) manages() bool { return a.Role == "owner" || a.Role == "admin" }
