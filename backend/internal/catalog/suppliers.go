package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Supplier struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SupplierInput struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type SupplierUpdateInput struct {
	Name    *string `json:"name"`
	Phone   *string `json:"phone"`
	Email   *string `json:"email"`
	Address *string `json:"address"`
	Status  *string `json:"status"`
}

type SupplierSource struct {
	SupplierID    string    `json:"supplierId"`
	VariantID     string    `json:"variantId"`
	ProductName   string    `json:"productName"`
	VariantName   string    `json:"variantName"`
	SKU           string    `json:"sku"`
	ProductStatus string    `json:"productStatus"`
	SupplierSKU   string    `json:"supplierSku"`
	CostMinor     *int64    `json:"costMinor"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SupplierSourceInput struct {
	VariantID   string `json:"variantId"`
	SupplierSKU string `json:"supplierSku"`
	CostMinor   *int64 `json:"costMinor"`
}

var (
	ErrSupplierDuplicate = &Error{http.StatusConflict, "supplier_duplicate", "An active supplier with this name already exists in this workspace."}
	ErrSupplierSKU       = &Error{http.StatusConflict, "supplier_sku_duplicate", "This supplier SKU is already linked to another variant."}
	ErrSupplierArchived  = &Error{http.StatusConflict, "supplier_archived", "Archived suppliers cannot be linked to products. Restore the supplier first."}
)

func validSupplierEmail(value string) bool {
	if value == "" {
		return true
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && !strings.ContainsAny(value, " \t\r\n")
}

func normalizeSupplierError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		if pg.ConstraintName == "supplier_variants_sku_idx" {
			return ErrSupplierSKU
		}
		return ErrSupplierDuplicate
	}
	return err
}

func (s *Service) ListSuppliers(ctx context.Context, tenantID, search string, limit, offset int) ([]Supplier, error) {
	return s.repo.ListSuppliers(ctx, tenantID, strings.TrimSpace(search), pageLimit(limit), pageOffset(offset))
}

func (s *Service) CreateSupplier(ctx context.Context, tenantID string, in SupplierInput) (*Supplier, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.TrimSpace(in.Email)
	in.Address = strings.TrimSpace(in.Address)
	if in.Name == "" || len(in.Name) > 160 || len(in.Phone) > 40 || len(in.Email) > 254 ||
		len(in.Address) > 500 || !validSupplierEmail(in.Email) {
		return nil, ErrInvalid
	}
	out, err := s.repo.CreateSupplier(ctx, tenantID, in)
	return out, normalizeSupplierError(err)
}

func (s *Service) UpdateSupplier(ctx context.Context, tenantID, supplierID string, in SupplierUpdateInput) (*Supplier, error) {
	if in.Name == nil && in.Phone == nil && in.Email == nil && in.Address == nil && in.Status == nil {
		return nil, ErrInvalid
	}
	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" || len(v) > 160 {
			return nil, ErrInvalid
		}
		in.Name = &v
	}
	if in.Phone != nil {
		v := strings.TrimSpace(*in.Phone)
		if len(v) > 40 {
			return nil, ErrInvalid
		}
		in.Phone = &v
	}
	if in.Email != nil {
		v := strings.TrimSpace(*in.Email)
		if len(v) > 254 || !validSupplierEmail(v) {
			return nil, ErrInvalid
		}
		in.Email = &v
	}
	if in.Address != nil {
		v := strings.TrimSpace(*in.Address)
		if len(v) > 500 {
			return nil, ErrInvalid
		}
		in.Address = &v
	}
	if in.Status != nil && *in.Status != "active" && *in.Status != "archived" {
		return nil, ErrInvalid
	}
	out, err := s.repo.UpdateSupplier(ctx, tenantID, supplierID, in)
	return out, normalizeSupplierError(err)
}

func (s *Service) ListSupplierSources(ctx context.Context, tenantID, supplierID string, limit, offset int) ([]SupplierSource, error) {
	return s.repo.ListSupplierSources(ctx, tenantID, supplierID, pageLimit(limit), pageOffset(offset))
}

func (s *Service) SaveSupplierSource(ctx context.Context, tenantID, supplierID string, in SupplierSourceInput) (*SupplierSource, error) {
	in.VariantID = strings.TrimSpace(in.VariantID)
	in.SupplierSKU = strings.TrimSpace(in.SupplierSKU)
	if in.VariantID == "" || len(in.SupplierSKU) > 80 ||
		(in.CostMinor != nil && (*in.CostMinor < 0 || *in.CostMinor > 1_000_000_000_000)) {
		return nil, ErrInvalid
	}
	out, err := s.repo.SaveSupplierSource(ctx, tenantID, supplierID, in)
	return out, normalizeSupplierError(err)
}
