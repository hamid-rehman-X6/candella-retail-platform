// Package catalog owns sellable products and the first stock workflow.
package catalog

import "time"

type Variant struct {
	ID         string  `json:"id"`
	ProductID  string  `json:"productId"`
	Name       string  `json:"name"`
	SKU        string  `json:"sku"`
	Barcode    *string `json:"barcode"`
	PriceMinor int64   `json:"priceMinor"`
	Quantity   int64   `json:"quantity"`
}

type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	// The rate charged on every variant; nil means the product is not taxed.
	TaxRateID   *string   `json:"taxRateId"`
	TaxRateName *string   `json:"taxRateName"`
	Variants    []Variant `json:"variants"`
	CreatedAt   time.Time `json:"createdAt"`
}

type StockItem struct {
	VariantID             string  `json:"variantId"`
	ProductID             string  `json:"productId"`
	ProductName           string  `json:"productName"`
	VariantName           string  `json:"variantName"`
	SKU                   string  `json:"sku"`
	Barcode               *string `json:"barcode"`
	PriceMinor            int64   `json:"priceMinor"`
	Quantity              int64   `json:"quantity"`
	ReorderThreshold      *int64  `json:"reorderThreshold"`
	PreferredSupplierID   *string `json:"preferredSupplierId"`
	PreferredSupplierName *string `json:"preferredSupplierName"`
	ConfiguredSupplierID  *string `json:"configuredSupplierId"`
	// The product's tax, so the till can show it before checkout confirms it.
	TaxName      *string `json:"taxName"`
	TaxRateBps   *int64  `json:"taxRateBps"`
	TaxInclusive *bool   `json:"taxInclusive"`
}

type ReorderItem struct {
	StockItem
	SuggestedQuantity int64 `json:"suggestedQuantity"`
}

type ReorderInput struct {
	StoreID             string  `json:"storeId"`
	Threshold           *int64  `json:"threshold"`
	PreferredSupplierID *string `json:"preferredSupplierId"`
}

type Summary struct {
	ProductCount int64 `json:"productCount"`
	VariantCount int64 `json:"variantCount"`
	UnitsOnHand  int64 `json:"unitsOnHand"`
	OutOfStock   int64 `json:"outOfStock"`
}

type Movement struct {
	ID              string    `json:"id"`
	StoreID         string    `json:"storeId"`
	OrderID         *string   `json:"orderId"`
	PurchaseOrderID *string   `json:"purchaseOrderId"`
	VariantID       string    `json:"variantId"`
	ProductName     string    `json:"productName"`
	VariantName     string    `json:"variantName"`
	SKU             string    `json:"sku"`
	Kind            string    `json:"kind"`
	Delta           int64     `json:"delta"`
	QuantityBefore  int64     `json:"quantityBefore"`
	QuantityAfter   int64     `json:"quantityAfter"`
	Note            string    `json:"note"`
	CreatedAt       time.Time `json:"createdAt"`
}

type CreateProductInput struct {
	StoreID         string             `json:"storeId"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Variant         CreateVariantInput `json:"variant"`
	InitialQuantity *int64             `json:"initialQuantity"`
	TaxRateID       string             `json:"taxRateId"`
}

type CreateVariantInput struct {
	Name       string `json:"name"`
	SKU        string `json:"sku"`
	Barcode    string `json:"barcode"`
	PriceMinor int64  `json:"priceMinor"`
}

type UpdateProductInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	// "" removes the product's tax.
	TaxRateID *string `json:"taxRateId"`
}

type UpdateVariantInput struct {
	Name       *string `json:"name"`
	SKU        *string `json:"sku"`
	Barcode    *string `json:"barcode"`
	PriceMinor *int64  `json:"priceMinor"`
}

type MovementInput struct {
	StoreID         string `json:"storeId"`
	IdempotencyKey  string `json:"idempotencyKey"`
	VariantID       string `json:"variantId"`
	Kind            string `json:"kind"`
	Quantity        *int64 `json:"quantity"`
	CountedQuantity *int64 `json:"countedQuantity"`
	Note            string `json:"note"`
}
