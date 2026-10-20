package purchase

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
)

const maxCostMinor int64 = 1_000_000_000_000

type LineInput struct {
	VariantID     string `json:"variantId"`
	Quantity      int64  `json:"quantity"`
	UnitCostMinor int64  `json:"unitCostMinor"`
}
type CreateInput struct {
	StoreID        string      `json:"storeId"`
	SupplierID     string      `json:"supplierId"`
	IdempotencyKey string      `json:"idempotencyKey"`
	Items          []LineInput `json:"items"`
}
type ReceiveLineInput struct {
	VariantID string `json:"variantId"`
	Quantity  int64  `json:"quantity"`
}
type ReceiveInput struct {
	IdempotencyKey string             `json:"idempotencyKey"`
	Items          []ReceiveLineInput `json:"items"`
}
type Item struct {
	VariantID        string `json:"variantId"`
	ProductName      string `json:"productName"`
	VariantName      string `json:"variantName"`
	SKU              string `json:"sku"`
	Quantity         int64  `json:"quantity"`
	ReceivedQuantity int64  `json:"receivedQuantity"`
	UnitCostMinor    int64  `json:"unitCostMinor"`
	LineCostMinor    int64  `json:"lineCostMinor"`
}
type ReceiptItem struct {
	VariantID     string `json:"variantId"`
	Quantity      int64  `json:"quantity"`
	UnitCostMinor int64  `json:"unitCostMinor"`
	LineCostMinor int64  `json:"lineCostMinor"`
}
type Receipt struct {
	ID              string        `json:"id"`
	PurchaseOrderID string        `json:"purchaseOrderId"`
	TotalCostMinor  int64         `json:"totalCostMinor"`
	CreatedAt       time.Time     `json:"createdAt"`
	Items           []ReceiptItem `json:"items"`
}
type Order struct {
	ID             string     `json:"id"`
	Number         string     `json:"number"`
	StoreID        string     `json:"storeId"`
	StoreName      string     `json:"storeName"`
	SupplierID     string     `json:"supplierId"`
	SupplierName   string     `json:"supplierName"`
	Status         string     `json:"status"`
	CurrencyCode   string     `json:"currencyCode"`
	TotalCostMinor int64      `json:"totalCostMinor"`
	CreatedAt      time.Time  `json:"createdAt"`
	ApprovedAt     *time.Time `json:"approvedAt"`
	Items          []Item     `json:"items"`
	Receipts       []Receipt  `json:"receipts"`
}
type Summary struct {
	ID             string    `json:"id"`
	Number         string    `json:"number"`
	StoreID        string    `json:"storeId"`
	SupplierName   string    `json:"supplierName"`
	Status         string    `json:"status"`
	CurrencyCode   string    `json:"currencyCode"`
	TotalCostMinor int64     `json:"totalCostMinor"`
	CreatedAt      time.Time `json:"createdAt"`
}
type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid             = &Error{http.StatusBadRequest, "invalid_purchase_order", "Choose a store, active supplier and products with valid quantities and costs."}
	ErrNotFound            = &Error{http.StatusNotFound, "purchase_order_not_found", "Purchase order not found in this workspace."}
	ErrSupplierUnavailable = &Error{http.StatusConflict, "supplier_unavailable", "Supplier is archived or not in this workspace."}
	ErrVariantUnavailable  = &Error{http.StatusConflict, "purchase_variant_unavailable", "A product is archived or not in this workspace."}
	ErrKeyReuse            = &Error{http.StatusConflict, "purchase_key_reused", "This request key was already used with different details. Refresh and try again."}
	ErrApprovalRequired    = &Error{http.StatusConflict, "purchase_approval_required", "Approve this purchase order before receiving stock."}
	ErrOverReceive         = &Error{http.StatusConflict, "purchase_over_received", "Received quantity exceeds the amount still ordered. Check the delivery quantities."}
	ErrFullyReceived       = &Error{http.StatusConflict, "purchase_fully_received", "This purchase order has already been fully received."}
	ErrStockLimit          = &Error{http.StatusConflict, "stock_limit", "Receiving this delivery would exceed the stock limit."}
)

type Repository interface {
	Create(context.Context, string, string, CreateInput) (*Order, bool, error)
	List(context.Context, string, int, int) ([]Summary, error)
	Find(context.Context, string, string) (*Order, error)
	Approve(context.Context, string, string, string) (*Order, bool, error)
	Receive(context.Context, string, string, string, ReceiveInput) (*Order, *Receipt, bool, error)
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo} }
func validID(value, prefix string) bool {
	return strings.HasPrefix(value, prefix+"_") && len(value) == len(prefix)+13
}
func validKey(value string) bool { return len(value) >= 8 && len(value) <= 100 }
func (s *Service) Create(ctx context.Context, tenantID, userID string, in CreateInput) (*Order, bool, error) {
	in.StoreID = strings.TrimSpace(in.StoreID)
	in.SupplierID = strings.TrimSpace(in.SupplierID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if !validID(in.StoreID, "str") || !validID(in.SupplierID, "sup") || !validKey(in.IdempotencyKey) || len(in.Items) == 0 || len(in.Items) > 100 {
		return nil, false, ErrInvalid
	}
	seen := map[string]bool{}
	var total int64
	for i := range in.Items {
		line := &in.Items[i]
		line.VariantID = strings.TrimSpace(line.VariantID)
		if !validID(line.VariantID, "var") || seen[line.VariantID] || line.Quantity <= 0 || line.Quantity > 1_000_000 || line.UnitCostMinor < 0 || line.UnitCostMinor > maxCostMinor || line.UnitCostMinor > maxCostMinor/line.Quantity || line.UnitCostMinor*line.Quantity > maxCostMinor-total {
			return nil, false, ErrInvalid
		}
		seen[line.VariantID] = true
		total += line.UnitCostMinor * line.Quantity
	}
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].VariantID < in.Items[j].VariantID })
	return s.repo.Create(ctx, tenantID, userID, in)
}
func (s *Service) List(ctx context.Context, tenantID string, limit, offset int) ([]Summary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, tenantID, limit, offset)
}
func (s *Service) Find(ctx context.Context, tenantID, orderID string) (*Order, error) {
	if !validID(orderID, "po") {
		return nil, ErrNotFound
	}
	return s.repo.Find(ctx, tenantID, orderID)
}
func (s *Service) Approve(ctx context.Context, tenantID, orderID, userID string) (*Order, bool, error) {
	if !validID(orderID, "po") {
		return nil, false, ErrNotFound
	}
	return s.repo.Approve(ctx, tenantID, orderID, userID)
}
func (s *Service) Receive(ctx context.Context, tenantID, orderID, userID string, in ReceiveInput) (*Order, *Receipt, bool, error) {
	if !validID(orderID, "po") {
		return nil, nil, false, ErrNotFound
	}
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if !validKey(in.IdempotencyKey) || len(in.Items) == 0 || len(in.Items) > 100 {
		return nil, nil, false, ErrInvalid
	}
	seen := map[string]bool{}
	for i := range in.Items {
		line := &in.Items[i]
		line.VariantID = strings.TrimSpace(line.VariantID)
		if !validID(line.VariantID, "var") || seen[line.VariantID] || line.Quantity <= 0 || line.Quantity > 1_000_000 {
			return nil, nil, false, ErrInvalid
		}
		seen[line.VariantID] = true
	}
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].VariantID < in.Items[j].VariantID })
	return s.repo.Receive(ctx, tenantID, orderID, userID, in)
}
