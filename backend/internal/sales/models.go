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
	Status            string     `json:"status"`
	CurrencyCode      string     `json:"currencyCode"`
	Items             []SaleItem `json:"items"`
	SubtotalMinor     int64      `json:"subtotalMinor"`
	GrandTotalMinor   int64      `json:"grandTotalMinor"`
	CashReceivedMinor int64      `json:"cashReceivedMinor"`
	ChangeMinor       int64      `json:"changeMinor"`
	CreatedAt         time.Time  `json:"createdAt"`
}
