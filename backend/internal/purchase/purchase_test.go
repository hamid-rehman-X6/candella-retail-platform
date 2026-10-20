package purchase

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candella-ecosystem/backend/internal/tenant"
)

type fakeRepository struct {
	created  CreateInput
	received ReceiveInput
	called   bool
}

func (f *fakeRepository) Create(_ context.Context, _, _ string, in CreateInput) (*Order, bool, error) {
	f.called = true
	f.created = in
	return &Order{}, true, nil
}
func (f *fakeRepository) List(context.Context, string, int, int) ([]Summary, error) {
	f.called = true
	return nil, nil
}
func (f *fakeRepository) Find(context.Context, string, string) (*Order, error) {
	f.called = true
	return nil, nil
}
func (f *fakeRepository) Approve(context.Context, string, string, string) (*Order, bool, error) {
	f.called = true
	return nil, false, nil
}
func (f *fakeRepository) Receive(_ context.Context, _, _, _ string, in ReceiveInput) (*Order, *Receipt, bool, error) {
	f.called = true
	f.received = in
	return nil, nil, false, nil
}

func TestPurchaseValidationAndLockOrder(t *testing.T) {
	f := &fakeRepository{}
	svc := NewService(f)
	valid := CreateInput{StoreID: "str_AAAAAAAAAAAA", SupplierID: "sup_AAAAAAAAAAAA", IdempotencyKey: "create-001",
		Items: []LineInput{{VariantID: "var_BBBBBBBBBBBB", Quantity: 2, UnitCostMinor: 125}, {VariantID: "var_AAAAAAAAAAAA", Quantity: 1, UnitCostMinor: 150}}}
	for _, invalid := range []CreateInput{
		{StoreID: valid.StoreID, SupplierID: valid.SupplierID, IdempotencyKey: "create-001"},
		{StoreID: valid.StoreID, SupplierID: "sup_other", IdempotencyKey: "create-001", Items: valid.Items},
		{StoreID: valid.StoreID, SupplierID: valid.SupplierID, IdempotencyKey: "create-001", Items: []LineInput{{VariantID: valid.Items[0].VariantID, Quantity: 1, UnitCostMinor: maxCostMinor + 1}}},
		{StoreID: valid.StoreID, SupplierID: valid.SupplierID, IdempotencyKey: "create-001", Items: []LineInput{valid.Items[0], valid.Items[0]}},
	} {
		if _, _, err := svc.Create(context.Background(), "tenant", "user", invalid); !errors.Is(err, ErrInvalid) || f.called {
			t.Fatalf("invalid create reached repository: %+v %v", invalid, err)
		}
	}
	if _, _, err := svc.Create(context.Background(), "tenant", "user", valid); err != nil || !f.called || f.created.Items[0].VariantID != "var_AAAAAAAAAAAA" {
		t.Fatalf("valid create: %+v %v", f.created, err)
	}
	f.called = false
	if _, _, _, err := svc.Receive(context.Background(), "tenant", "po_AAAAAAAAAAAA", "user", ReceiveInput{IdempotencyKey: "receive-001", Items: []ReceiveLineInput{{VariantID: "var_AAAAAAAAAAAA", Quantity: 0}}}); !errors.Is(err, ErrInvalid) || f.called {
		t.Fatalf("invalid receive reached repository: %v", err)
	}
	if _, _, _, err := svc.Receive(context.Background(), "tenant", "po_AAAAAAAAAAAA", "user", ReceiveInput{IdempotencyKey: "receive-001", Items: []ReceiveLineInput{{VariantID: "var_BBBBBBBBBBBB", Quantity: 2}, {VariantID: "var_AAAAAAAAAAAA", Quantity: 1}}}); err != nil || f.received.Items[0].VariantID != "var_AAAAAAAAAAAA" {
		t.Fatalf("receive order: %+v %v", f.received, err)
	}
}
func TestPurchaseRoles(t *testing.T) {
	for _, role := range []string{"owner", "admin"} {
		if !canManage(role) {
			t.Fatalf("%s cannot manage", role)
		}
	}
	for _, role := range []string{"member", "viewer", ""} {
		if canManage(role) {
			t.Fatalf("%s can manage", role)
		}
	}
	h := &Handler{}
	ctx := tenant.WithContext(context.Background(), &tenant.Context{TenantID: "tnt_test", Role: "member"})
	for _, call := range []func(http.ResponseWriter, *http.Request){h.List, h.Create, h.Find, h.Approve, h.Receive} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/purchase-orders", strings.NewReader(`{}`)).WithContext(ctx)
		call(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("member status=%d", w.Code)
		}
	}
}
