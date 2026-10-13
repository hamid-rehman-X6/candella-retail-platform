package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candella-ecosystem/backend/internal/tenant"
)

type supplierStub struct {
	Repository
	created bool
	linked  bool
	input   SupplierInput
	source  SupplierSourceInput
}

func (r *supplierStub) CreateSupplier(_ context.Context, _ string, in SupplierInput) (*Supplier, error) {
	r.created, r.input = true, in
	return &Supplier{Name: in.Name}, nil
}

func (r *supplierStub) SaveSupplierSource(_ context.Context, _, _ string, in SupplierSourceInput) (*SupplierSource, error) {
	r.linked, r.source = true, in
	return &SupplierSource{VariantID: in.VariantID}, nil
}

func TestSupplierInputValidation(t *testing.T) {
	for _, in := range []SupplierInput{
		{Name: " "},
		{Name: "Acme", Email: "invalid-email"},
		{Name: "Acme", Phone: strings.Repeat("1", 41)},
		{Name: "Acme", Address: strings.Repeat("x", 501)},
	} {
		repo := &supplierStub{}
		if _, err := NewService(repo).CreateSupplier(context.Background(), "tenant", in); err != ErrInvalid || repo.created {
			t.Fatalf("invalid supplier reached repository: %+v err=%v", in, err)
		}
	}
	repo := &supplierStub{}
	if _, err := NewService(repo).CreateSupplier(context.Background(), "tenant", SupplierInput{
		Name: " Acme ", Email: " stock@acme.example ",
	}); err != nil || repo.input.Name != "Acme" || repo.input.Email != "stock@acme.example" {
		t.Fatalf("supplier was not normalized: %+v err=%v", repo.input, err)
	}
}

func TestSupplierSourceCostValidation(t *testing.T) {
	for _, cost := range []int64{-1, 1_000_000_000_001} {
		repo := &supplierStub{}
		_, err := NewService(repo).SaveSupplierSource(context.Background(), "tenant", "supplier", SupplierSourceInput{
			VariantID: "var_123456789012", CostMinor: &cost,
		})
		if err != ErrInvalid || repo.linked {
			t.Fatalf("invalid cost %d reached repository: %v", cost, err)
		}
	}
	repo := &supplierStub{}
	if _, err := NewService(repo).SaveSupplierSource(context.Background(), "tenant", "supplier", SupplierSourceInput{
		VariantID: " var_123456789012 ", SupplierSKU: " V-123 ", CostMinor: ptr(13500),
	}); err != nil || !repo.linked || repo.source.SupplierSKU != "V-123" || repo.source.VariantID != "var_123456789012" {
		t.Fatalf("source was not normalized: %+v err=%v", repo.source, err)
	}
}

func TestSupplierWritesRejectMembers(t *testing.T) {
	h := &Handler{}
	for _, invoke := range []func(http.ResponseWriter, *http.Request){h.CreateSupplier, h.UpdateSupplier, h.SaveSupplierSource} {
		req := httptest.NewRequest(http.MethodPost, "/suppliers", strings.NewReader(`{}`))
		req = req.WithContext(tenant.WithContext(req.Context(), &tenant.Context{TenantID: "tnt_123456789012", Role: "member"}))
		response := httptest.NewRecorder()
		invoke(response, req)
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "insufficient_role") {
			t.Fatalf("member write response: %d %s", response.Code, response.Body.String())
		}
	}
}
