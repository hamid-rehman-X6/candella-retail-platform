package sales

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
)

const maxSaleMinor int64 = 1_000_000_000_000

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid    = &Error{http.StatusBadRequest, "invalid_checkout", "Check the cart and cash amount, then try again."}
	ErrNotFound   = &Error{http.StatusNotFound, "sale_not_found", "This sale was not found in this workspace."}
	ErrArchived   = &Error{http.StatusConflict, "product_unavailable", "A product in the cart is archived. Remove it and try again."}
	ErrKeyReuse   = &Error{http.StatusConflict, "checkout_key_reused", "This checkout was already submitted with different details. Start a new sale."}
	ErrCashShort  = &Error{http.StatusConflict, "cash_too_low", "Cash received is less than the current sale total. Check the updated prices and enter enough cash."}
	ErrTotalLarge = &Error{http.StatusBadRequest, "sale_too_large", "Sale total is too large. Reduce the cart size."}
)

type Repository interface {
	Create(context.Context, string, string, CheckoutInput) (*Sale, bool, error)
	Find(context.Context, string, string) (*Sale, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Checkout(ctx context.Context, tenantID, userID string, in CheckoutInput) (*Sale, bool, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 100 ||
		len(in.Items) == 0 || len(in.Items) > 100 ||
		in.CashReceivedMinor < 0 || in.CashReceivedMinor > maxSaleMinor {
		return nil, false, ErrInvalid
	}
	seen := make(map[string]bool, len(in.Items))
	for i := range in.Items {
		item := &in.Items[i]
		item.VariantID = strings.TrimSpace(item.VariantID)
		if !strings.HasPrefix(item.VariantID, "var_") || len(item.VariantID) != 16 ||
			item.Quantity <= 0 || item.Quantity > 1_000_000 || seen[item.VariantID] {
			return nil, false, ErrInvalid
		}
		seen[item.VariantID] = true
	}
	// Every sale locks stock in the same order, avoiding cart-order deadlocks.
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].VariantID < in.Items[j].VariantID })
	return s.repo.Create(ctx, tenantID, userID, in)
}

func (s *Service) Receipt(ctx context.Context, tenantID, orderID string) (*Sale, error) {
	if !strings.HasPrefix(orderID, "ord_") || len(orderID) != 16 {
		return nil, ErrNotFound
	}
	return s.repo.Find(ctx, tenantID, orderID)
}

func addLine(total, price, quantity int64) (int64, int64, error) {
	if price < 0 || price > maxSaleMinor || quantity <= 0 || price > maxSaleMinor/quantity {
		return 0, 0, ErrTotalLarge
	}
	line := price * quantity
	if line > maxSaleMinor-total {
		return 0, 0, ErrTotalLarge
	}
	return line, total + line, nil
}

func asSaleError(err error) (*Error, bool) {
	var e *Error
	return e, errors.As(err, &e)
}
