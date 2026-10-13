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
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Variants    []Variant `json:"variants"`
	CreatedAt   time.Time `json:"createdAt"`
}

type StockItem struct {
	VariantID   string  `json:"variantId"`
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	VariantName string  `json:"variantName"`
	SKU         string  `json:"sku"`
	Barcode     *string `json:"barcode"`
	PriceMinor  int64   `json:"priceMinor"`
	Quantity    int64   `json:"quantity"`
}

type Summary struct {
	ProductCount int64 `json:"productCount"`
	VariantCount int64 `json:"variantCount"`
	UnitsOnHand  int64 `json:"unitsOnHand"`
	OutOfStock   int64 `json:"outOfStock"`
}

type Movement struct {
	ID             string    `json:"id"`
	OrderID        *string   `json:"orderId"`
	VariantID      string    `json:"variantId"`
	ProductName    string    `json:"productName"`
	VariantName    string    `json:"variantName"`
	SKU            string    `json:"sku"`
	Kind           string    `json:"kind"`
	Delta          int64     `json:"delta"`
	QuantityBefore int64     `json:"quantityBefore"`
	QuantityAfter  int64     `json:"quantityAfter"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"createdAt"`
}

type CreateProductInput struct {
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Variant         CreateVariantInput `json:"variant"`
	InitialQuantity *int64             `json:"initialQuantity"`
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
}

type UpdateVariantInput struct {
	Name       *string `json:"name"`
	SKU        *string `json:"sku"`
	Barcode    *string `json:"barcode"`
	PriceMinor *int64  `json:"priceMinor"`
}

type MovementInput struct {
	IdempotencyKey  string `json:"idempotencyKey"`
	VariantID       string `json:"variantId"`
	Kind            string `json:"kind"`
	Quantity        *int64 `json:"quantity"`
	CountedQuantity *int64 `json:"countedQuantity"`
	Note            string `json:"note"`
}
