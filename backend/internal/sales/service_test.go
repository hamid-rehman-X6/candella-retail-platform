package sales

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candella-ecosystem/backend/internal/tenant"
)

type fakeRepository struct {
	called bool
	input  CheckoutInput
	filter SalesFilter
}

func (f *fakeRepository) Create(_ context.Context, _, _, _ string, in CheckoutInput) (*Sale, bool, error) {
	f.called, f.input = true, in
	return &Sale{}, true, nil
}
func (f *fakeRepository) Find(context.Context, string, string) (*Sale, error) {
	return &Sale{}, nil
}
func (f *fakeRepository) Search(_ context.Context, _ string, filter SalesFilter) ([]SaleSummary, bool, error) {
	f.called, f.filter = true, filter
	return []SaleSummary{}, false, nil
}

func TestCheckoutValidationAndStableLockOrder(t *testing.T) {
	valid := CheckoutInput{IdempotencyKey: "request-123", CashReceivedMinor: 500,
		Items: []CartItem{{VariantID: "var_BBBBBBBBBBBB", Quantity: 2}, {VariantID: "var_AAAAAAAAAAAA", Quantity: 1}}}
	cases := []struct {
		name  string
		input CheckoutInput
	}{
		{"empty cart", CheckoutInput{IdempotencyKey: "request-123"}},
		{"short key", CheckoutInput{IdempotencyKey: "short", Items: valid.Items}},
		{"duplicate variant", CheckoutInput{IdempotencyKey: "request-123", Items: []CartItem{valid.Items[0], valid.Items[0]}}},
		{"zero quantity", CheckoutInput{IdempotencyKey: "request-123", Items: []CartItem{{VariantID: valid.Items[0].VariantID}}}},
		{"negative cash", CheckoutInput{IdempotencyKey: "request-123", CashReceivedMinor: -1, Items: valid.Items}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, _, err := NewService(repo).Checkout(context.Background(), "tenant", "user", "Cashier", tc.input)
			if err != ErrInvalid || repo.called {
				t.Fatalf("invalid checkout reached repository: err=%v called=%v", err, repo.called)
			}
		})
	}
	repo := &fakeRepository{}
	_, _, err := NewService(repo).Checkout(context.Background(), "tenant", "user", "Cashier", valid)
	if err != nil || !repo.called || repo.input.Items[0].VariantID != "var_AAAAAAAAAAAA" {
		t.Fatalf("valid checkout not sorted before repository: err=%v input=%+v", err, repo.input)
	}
}

func TestSearchValidatesFiltersBeforeRepository(t *testing.T) {
	for _, filter := range []SalesFilter{
		{FromDate: "2026-02-30"}, {FromDate: "2026-04-02", ToDate: "2026-04-01"},
		{Offset: -1}, {MinAmountMinor: amount(-1)},
		{MinAmountMinor: amount(200), MaxAmountMinor: amount(100)},
	} {
		repo := &fakeRepository{}
		if _, _, err := NewService(repo).Search(context.Background(), "tenant", filter); err != ErrInvalidFilter || repo.called {
			t.Fatalf("invalid filter reached repository: %+v, err=%v", filter, err)
		}
	}
	repo := &fakeRepository{}
	if _, _, err := NewService(repo).Search(context.Background(), "tenant", SalesFilter{OrderNumber: " POS-1 ", Limit: 101}); err != nil || !repo.called || repo.filter.Limit != 100 || repo.filter.OrderNumber != "POS-1" {
		t.Fatalf("valid search not normalized: %+v, err=%v", repo.filter, err)
	}
}

func amount(value int64) *int64 { return &value }

func TestLineTotalsUseIntegerMathAndBoundSize(t *testing.T) {
	line, total, err := addLine(150, 125, 3)
	if err != nil || line != 375 || total != 525 {
		t.Fatalf("line=%d total=%d err=%v", line, total, err)
	}
	if _, _, err := addLine(maxSaleMinor-1, 2, 1); err != ErrTotalLarge {
		t.Fatalf("overflow error=%v", err)
	}
	if _, _, err := addLine(0, maxSaleMinor, 2); err != ErrTotalLarge {
		t.Fatalf("multiplication overflow error=%v", err)
	}
}

func TestCheckoutRequiresWorkspaceRole(t *testing.T) {
	for _, role := range []string{"owner", "admin", "member"} {
		if !canCheckout(role) {
			t.Fatalf("role %q should be able to sell", role)
		}
	}
	for _, role := range []string{"", "viewer", "suspended"} {
		if canCheckout(role) {
			t.Fatalf("role %q should not be able to sell", role)
		}
	}
	h := &Handler{}
	r := httptest.NewRequest(http.MethodPost, "/sales", strings.NewReader(`{}`))
	r = r.WithContext(tenant.WithContext(r.Context(), &tenant.Context{TenantID: "tenant", Role: "viewer"}))
	w := httptest.NewRecorder()
	h.Checkout(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("forbidden role got HTTP %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/sales", nil).WithContext(r.Context()))
	if w.Code != http.StatusForbidden {
		t.Fatalf("forbidden search got HTTP %d", w.Code)
	}
}
