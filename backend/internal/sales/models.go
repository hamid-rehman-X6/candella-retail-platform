package sales

import "time"

type CartItem struct {
	VariantID string `json:"variantId"`
	Quantity  int64  `json:"quantity"`
	// An item promotion for this line. omitempty keeps the idempotency hash of
	// carts without one exactly as it was before discounts existed.
	PromotionID string `json:"promotionId,omitempty"`
}

// DiscountInput is an order-level discount: an approved promotion, or a
// manual percentage with a reason.
type DiscountInput struct {
	PromotionID string `json:"promotionId,omitempty"`
	PercentBps  int64  `json:"percentBps,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type CheckoutInput struct {
	StoreID           string         `json:"storeId"`
	RegisterID        string         `json:"registerId"`
	CustomerID        string         `json:"customerId"`
	IdempotencyKey    string         `json:"idempotencyKey"`
	Items             []CartItem     `json:"items"`
	CashReceivedMinor int64          `json:"cashReceivedMinor"`
	Discount          *DiscountInput `json:"discount,omitempty"`
}

type SaleItem struct {
	VariantID      string `json:"variantId"`
	ProductName    string `json:"productName"`
	VariantName    string `json:"variantName"`
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
	LineTotalMinor int64  `json:"lineTotalMinor"`
	// Units already taken back; Quantity - ReturnedQuantity can still be returned.
	ReturnedQuantity int64 `json:"returnedQuantity"`
	// The tax as charged on this line; nil fields mean the line was not taxed.
	TaxName      *string `json:"taxName"`
	TaxRateBps   *int64  `json:"taxRateBps"`
	TaxInclusive *bool   `json:"taxInclusive"`
	TaxMinor     int64   `json:"taxMinor"`
	// Discount on this line: its item promotion plus its share of any order
	// discount. Tax above was worked out after it.
	DiscountMinor int64   `json:"discountMinor"`
	PromotionName *string `json:"promotionName"`
}

type Sale struct {
	ID            string     `json:"id"`
	CustomerID    string     `json:"customerId"`
	CustomerName  string     `json:"customerName"`
	StoreID       string     `json:"storeId"`
	StoreName     string     `json:"storeName"`
	RegisterName  string     `json:"registerName"` // empty for sales made before shifts existed
	ShiftID       string     `json:"shiftId"`
	OrderNumber   string     `json:"orderNumber"`
	CashierID     string     `json:"cashierId"`
	CashierName   string     `json:"cashierName"`
	Status        string     `json:"status"`
	CurrencyCode  string     `json:"currencyCode"`
	Items         []SaleItem `json:"items"`
	SubtotalMinor int64      `json:"subtotalMinor"`
	DiscountMinor int64      `json:"discountMinor"`
	// The order-level discount as given; nil when the sale had none.
	DiscountKind       *string   `json:"discountKind"`
	DiscountName       *string   `json:"discountName"`
	DiscountPercentBps *int64    `json:"discountPercentBps"`
	DiscountReason     *string   `json:"discountReason"`
	TaxMinor           int64     `json:"taxMinor"`
	GrandTotalMinor    int64     `json:"grandTotalMinor"`
	CashReceivedMinor  int64     `json:"cashReceivedMinor"`
	ChangeMinor        int64     `json:"changeMinor"`
	CreatedAt          time.Time `json:"createdAt"`
	// Returns made against this sale, oldest first. The sale itself never changes.
	Returns       []ReturnSummary `json:"returns"`
	RefundedMinor int64           `json:"refundedMinor"`
}

type SaleSummary struct {
	ID              string    `json:"id"`
	StoreID         string    `json:"storeId"`
	StoreName       string    `json:"storeName"`
	OrderNumber     string    `json:"orderNumber"`
	CashierID       string    `json:"cashierId"`
	CashierName     string    `json:"cashierName"`
	CurrencyCode    string    `json:"currencyCode"`
	GrandTotalMinor int64     `json:"grandTotalMinor"`
	CreatedAt       time.Time `json:"createdAt"`
}

type SalesFilter struct {
	FromDate       string `json:"fromDate"`
	ToDate         string `json:"toDate"`
	OrderNumber    string `json:"orderNumber"`
	Cashier        string `json:"cashier"`
	MinAmountMinor *int64 `json:"minAmountMinor"`
	MaxAmountMinor *int64 `json:"maxAmountMinor"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
}
