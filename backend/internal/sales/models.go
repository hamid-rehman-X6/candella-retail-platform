package sales

import "time"

type CartItem struct {
	VariantID string `json:"variantId"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutInput struct {
	IdempotencyKey    string     `json:"idempotencyKey"`
	Items             []CartItem `json:"items"`
	CashReceivedMinor int64      `json:"cashReceivedMinor"`
}

type SaleItem struct {
	VariantID      string `json:"variantId"`
	ProductName    string `json:"productName"`
	VariantName    string `json:"variantName"`
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
	LineTotalMinor int64  `json:"lineTotalMinor"`
}

type Sale struct {
	ID                string     `json:"id"`
	OrderNumber       string     `json:"orderNumber"`
	CashierID         string     `json:"cashierId"`
	CashierName       string     `json:"cashierName"`
	Status            string     `json:"status"`
	CurrencyCode      string     `json:"currencyCode"`
	Items             []SaleItem `json:"items"`
	SubtotalMinor     int64      `json:"subtotalMinor"`
	GrandTotalMinor   int64      `json:"grandTotalMinor"`
	CashReceivedMinor int64      `json:"cashReceivedMinor"`
	ChangeMinor       int64      `json:"changeMinor"`
	CreatedAt         time.Time  `json:"createdAt"`
}

type SaleSummary struct {
	ID              string    `json:"id"`
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
