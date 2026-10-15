package sales

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ReturnInput takes goods back from one sale. Cash is refunded from the open
// shift on RegisterID, in the store where the sale was made.
type ReturnInput struct {
	StoreID        string     `json:"storeId"`
	RegisterID     string     `json:"registerId"`
	IdempotencyKey string     `json:"idempotencyKey"`
	Reason         string     `json:"reason"`
	Items          []CartItem `json:"items"`
}

type ReturnItem struct {
	VariantID       string `json:"variantId"`
	ProductName     string `json:"productName"`
	VariantName     string `json:"variantName"`
	SKU             string `json:"sku"`
	Quantity        int64  `json:"quantity"`
	UnitPriceMinor  int64  `json:"unitPriceMinor"`
	LineRefundMinor int64  `json:"lineRefundMinor"`
}

type Return struct {
	ID            string       `json:"id"`
	ReturnNumber  string       `json:"returnNumber"`
	OrderID       string       `json:"orderId"`
	OrderNumber   string       `json:"orderNumber"`
	StoreID       string       `json:"storeId"`
	StoreName     string       `json:"storeName"`
	RegisterName  string       `json:"registerName"`
	ShiftID       string       `json:"shiftId"`
	Reason        string       `json:"reason"`
	RefundMethod  string       `json:"refundMethod"`
	RefundMinor   int64        `json:"refundMinor"`
	CurrencyCode  string       `json:"currencyCode"`
	ProcessedBy   string       `json:"processedBy"`
	ProcessorName string       `json:"processorName"`
	Items         []ReturnItem `json:"items"`
	CreatedAt     time.Time    `json:"createdAt"`
}

// ReturnSummary is how a return appears on its sale's receipt.
type ReturnSummary struct {
	ID            string    `json:"id"`
	ReturnNumber  string    `json:"returnNumber"`
	RefundMinor   int64     `json:"refundMinor"`
	ProcessorName string    `json:"processorName"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"createdAt"`
}

var (
	ErrReturnInvalid        = &Error{http.StatusBadRequest, "invalid_return", "Choose at least one item and a quantity to return."}
	ErrReturnReason         = &Error{http.StatusBadRequest, "return_reason_required", "Enter the reason for this return (3 to 500 characters)."}
	ErrReturnItemNotInSale  = &Error{http.StatusBadRequest, "return_item_not_in_sale", "An item to return was not part of this sale."}
	ErrReturnKeyReuse       = &Error{http.StatusConflict, "return_key_reused", "This return was already submitted with different details. Start the return again."}
	ErrReturnNotFound       = &Error{http.StatusNotFound, "return_not_found", "This return was not found in this workspace."}
	ErrReturnForbidden      = &Error{http.StatusForbidden, "insufficient_role", "Only workspace owners and admins can process returns and refunds."}
	ErrReturnSaleWithoutPay = &Error{http.StatusConflict, "sale_not_refundable", "This sale has no cash payment to refund."}
)

// canRefund keeps cash refunds, the easiest way to take money out of a drawer,
// with the people answerable for the drawer.
func canRefund(role string) bool { return role == "owner" || role == "admin" }

func (s *Service) Return(ctx context.Context, tenantID, userID, processorName, orderID string, in ReturnInput) (*Return, bool, error) {
	if !validOrderID(orderID) {
		return nil, false, ErrNotFound
	}
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Reason = strings.TrimSpace(in.Reason)
	processorName = strings.TrimSpace(processorName)
	if len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 100 ||
		processorName == "" || len(processorName) > 200 ||
		len(in.Items) == 0 || len(in.Items) > 100 {
		return nil, false, ErrReturnInvalid
	}
	if len(in.Reason) < 3 || len(in.Reason) > 500 {
		return nil, false, ErrReturnReason
	}
	seen := make(map[string]bool, len(in.Items))
	for i := range in.Items {
		item := &in.Items[i]
		item.VariantID = strings.TrimSpace(item.VariantID)
		if !strings.HasPrefix(item.VariantID, "var_") || len(item.VariantID) != 16 ||
			item.Quantity <= 0 || item.Quantity > 1_000_000 || seen[item.VariantID] {
			return nil, false, ErrReturnInvalid
		}
		seen[item.VariantID] = true
	}
	// Same lock order as checkout, so returns and sales never deadlock on stock.
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].VariantID < in.Items[j].VariantID })
	return s.repo.CreateReturn(ctx, tenantID, userID, processorName, orderID, in)
}

func (s *Service) ReturnReceipt(ctx context.Context, tenantID, returnID string) (*Return, error) {
	if !strings.HasPrefix(returnID, "ret_") || len(returnID) != 16 {
		return nil, ErrReturnNotFound
	}
	return s.repo.FindReturn(ctx, tenantID, returnID)
}

func validOrderID(orderID string) bool {
	return strings.HasPrefix(orderID, "ord_") && len(orderID) == 16
}
