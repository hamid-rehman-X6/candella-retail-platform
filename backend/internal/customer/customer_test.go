package customer

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
	created Input
	updated UpdateInput
	called  bool
}

func (f *fakeRepository) List(context.Context, string, string, bool, int, int) ([]Customer, error) {
	f.called = true
	return []Customer{}, nil
}
func (f *fakeRepository) Create(_ context.Context, _ string, in Input) (*Customer, error) {
	f.called = true
	f.created = in
	return &Customer{Name: in.Name, Phone: in.Phone, Email: in.Email}, nil
}
func (f *fakeRepository) Update(_ context.Context, _, _ string, in UpdateInput) (*Customer, error) {
	f.called = true
	f.updated = in
	return &Customer{}, nil
}
func (f *fakeRepository) History(context.Context, string, string, int, int) ([]SaleSummary, error) {
	f.called = true
	return []SaleSummary{}, nil
}

func TestCustomerContactValidationAndNormalization(t *testing.T) {
	f := &fakeRepository{}
	s := NewService(f)
	for _, in := range []Input{{Name: "N"}, {Name: "N", Email: "not an email"}, {Name: "", Phone: "123"}} {
		if _, err := s.Create(context.Background(), "tenant", in); !errors.Is(err, ErrInvalid) || f.called {
			t.Fatalf("invalid input reached repository: %+v, %v", in, err)
		}
	}
	out, err := s.Create(context.Background(), "tenant", Input{Name: " Ada ", Phone: " 03-00 ", Email: " ADA@Example.com "})
	if err != nil || out.Name != "Ada" || f.created.Email != "ada@example.com" || f.created.Phone != "0300" {
		t.Fatalf("normalization: %+v %+v %v", out, f.created, err)
	}
	f.called = false
	empty := ""
	if _, err := s.Update(context.Background(), "tenant", "cus_x", UpdateInput{Name: &empty}); !errors.Is(err, ErrInvalid) || f.called {
		t.Fatalf("invalid update reached repository: %v", err)
	}
}

func TestCustomerRoleRestrictions(t *testing.T) {
	for _, role := range []string{"owner", "admin"} {
		if !canRead(role) || !canManage(role) {
			t.Fatalf("%s should manage", role)
		}
	}
	if !canRead("member") || canManage("member") || canRead("viewer") {
		t.Fatal("customer role policy changed")
	}
	h := &Handler{}
	ctx := tenant.WithContext(context.Background(), &tenant.Context{TenantID: "tnt_test", Role: "member"})
	for _, tc := range []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"create", h.Create}, {"update", h.Update}, {"history", h.History},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(`{}`)).WithContext(ctx)
			tc.call(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
}
