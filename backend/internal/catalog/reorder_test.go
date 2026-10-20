package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candella-ecosystem/backend/internal/tenant"
)

type reorderRepo struct {
	Repository
	called bool
	input  ReorderInput
}

func (r *reorderRepo) SaveReorder(_ context.Context, _, _ string, in ReorderInput) error {
	r.called = true
	r.input = in
	return nil
}

func TestSaveReorderValidation(t *testing.T) {
	storeID, variantID := "str_123456789012", "var_123456789012"
	bad := []ReorderInput{
		{StoreID: "", Threshold: ptr(1)},
		{StoreID: storeID, Threshold: ptr(-1)},
		{StoreID: storeID, Threshold: ptr(1_000_000_001)},
		{StoreID: storeID, PreferredSupplierID: stringPtr("sup_123456789012")},
		{StoreID: storeID, Threshold: ptr(1), PreferredSupplierID: stringPtr("other")},
	}
	for _, input := range bad {
		repo := &reorderRepo{}
		if err := NewService(repo).SaveReorder(context.Background(), "ten_123456789012", variantID, input); !errors.Is(err, ErrInvalidReorder) || repo.called {
			t.Fatalf("invalid setting %+v reached repo: %v", input, err)
		}
	}
	for _, threshold := range []*int64{ptr(0), ptr(1), nil} {
		repo := &reorderRepo{}
		if err := NewService(repo).SaveReorder(context.Background(), "ten_123456789012", variantID, ReorderInput{StoreID: storeID, Threshold: threshold}); err != nil || !repo.called {
			t.Fatalf("valid threshold %v: %v", threshold, err)
		}
	}
}
func stringPtr(s string) *string { return &s }

func TestReorderEndpointsRejectMembers(t *testing.T) {
	h := &Handler{}
	for _, invoke := range []func(http.ResponseWriter, *http.Request){h.ListLowStock, h.SaveReorder} {
		req := httptest.NewRequest(http.MethodGet, "/inventory/low-stock", strings.NewReader(`{}`))
		req = req.WithContext(tenant.WithContext(req.Context(), &tenant.Context{TenantID: "tnt_123456789012", Role: "member"}))
		w := httptest.NewRecorder()
		invoke(w, req)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "insufficient_role") {
			t.Fatalf("member reorder response: %d %s", w.Code, w.Body.String())
		}
	}
}
