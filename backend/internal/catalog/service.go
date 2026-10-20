package catalog

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid           = &Error{http.StatusBadRequest, "invalid_input", "Check the entered details and try again."}
	ErrNotFound          = &Error{http.StatusNotFound, "not_found", "That item was not found in this workspace."}
	ErrConflict          = &Error{http.StatusConflict, "duplicate_identifier", "That SKU or barcode is already used in this workspace."}
	ErrArchived          = &Error{http.StatusConflict, "product_archived", "Archived products cannot receive new variants or stock."}
	ErrMovementKeyReuse  = &Error{http.StatusConflict, "inventory_key_reused", "This stock change was already submitted with different details. Start a new stock change."}
	ErrInvalidReorder    = &Error{http.StatusBadRequest, "invalid_reorder_setting", "Enter a whole-number alert threshold from 0 to 1,000,000,000 and choose a valid supplier."}
	ErrSupplierNotActive = &Error{http.StatusConflict, "reorder_supplier_unavailable", "Choose an active supplier in this workspace."}
)

type Repository interface {
	ListProducts(context.Context, string, string, int, int) ([]Product, error)
	CreateProduct(context.Context, string, string, CreateProductInput) (*Product, error)
	UpdateProduct(context.Context, string, string, UpdateProductInput) (*Product, error)
	AddVariant(context.Context, string, string, CreateVariantInput) (*Variant, error)
	UpdateVariant(context.Context, string, string, UpdateVariantInput) (*Variant, error)
	ListStock(context.Context, string, string, int, int) ([]StockItem, string, error)
	ListLowStock(context.Context, string, int, int) ([]ReorderItem, error)
	SaveReorder(context.Context, string, string, ReorderInput) error
	Summary(context.Context, string) (*Summary, error)
	RecordMovement(context.Context, string, string, MovementInput) (*Movement, bool, error)
	ListMovements(context.Context, string, int, int) ([]Movement, error)
	ListSuppliers(context.Context, string, string, int, int) ([]Supplier, error)
	CreateSupplier(context.Context, string, SupplierInput) (*Supplier, error)
	UpdateSupplier(context.Context, string, string, SupplierUpdateInput) (*Supplier, error)
	ListSupplierSources(context.Context, string, string, int, int) ([]SupplierSource, error)
	SaveSupplierSource(context.Context, string, string, SupplierSourceInput) (*SupplierSource, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func trimVariant(v CreateVariantInput) (CreateVariantInput, error) {
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" {
		v.Name = "Standard"
	}
	v.SKU = strings.ToUpper(strings.TrimSpace(v.SKU))
	v.Barcode = strings.TrimSpace(v.Barcode)
	if len(v.Name) > 100 || len(v.SKU) > 80 || len(v.Barcode) > 80 || v.PriceMinor < 0 || v.PriceMinor > 1_000_000_000_000 {
		return v, ErrInvalid
	}
	return v, nil
}

func normalizeDBError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrConflict
	}
	return err
}

func (s *Service) ListProducts(ctx context.Context, tenantID, search string, limit, offset int) ([]Product, error) {
	return s.repo.ListProducts(ctx, tenantID, strings.TrimSpace(search), pageLimit(limit), pageOffset(offset))
}
func (s *Service) CreateProduct(ctx context.Context, tenantID, userID string, in CreateProductInput) (*Product, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" || len(in.Name) > 160 || len(in.Description) > 2000 ||
		(in.InitialQuantity != nil && (*in.InitialQuantity < 0 || *in.InitialQuantity > 1_000_000_000)) {
		return nil, ErrInvalid
	}
	v, err := trimVariant(in.Variant)
	if err != nil {
		return nil, err
	}
	in.Variant = v
	p, err := s.repo.CreateProduct(ctx, tenantID, userID, in)
	return p, normalizeDBError(err)
}
func (s *Service) UpdateProduct(ctx context.Context, tenantID, productID string, in UpdateProductInput) (*Product, error) {
	if in.Name == nil && in.Description == nil && in.Status == nil {
		return nil, ErrInvalid
	}
	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" || len(v) > 160 {
			return nil, ErrInvalid
		}
		in.Name = &v
	}
	if in.Description != nil {
		v := strings.TrimSpace(*in.Description)
		if len(v) > 2000 {
			return nil, ErrInvalid
		}
		in.Description = &v
	}
	if in.Status != nil && *in.Status != "active" && *in.Status != "archived" {
		return nil, ErrInvalid
	}
	return s.repo.UpdateProduct(ctx, tenantID, productID, in)
}
func (s *Service) AddVariant(ctx context.Context, tenantID, productID string, in CreateVariantInput) (*Variant, error) {
	v, err := trimVariant(in)
	if err != nil {
		return nil, err
	}
	out, err := s.repo.AddVariant(ctx, tenantID, productID, v)
	return out, normalizeDBError(err)
}
func (s *Service) UpdateVariant(ctx context.Context, tenantID, variantID string, in UpdateVariantInput) (*Variant, error) {
	if in.Name == nil && in.SKU == nil && in.Barcode == nil && in.PriceMinor == nil {
		return nil, ErrInvalid
	}
	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" || len(v) > 100 {
			return nil, ErrInvalid
		}
		in.Name = &v
	}
	if in.SKU != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.SKU))
		if v == "" || len(v) > 80 {
			return nil, ErrInvalid
		}
		in.SKU = &v
	}
	if in.Barcode != nil {
		v := strings.TrimSpace(*in.Barcode)
		if len(v) > 80 {
			return nil, ErrInvalid
		}
		in.Barcode = &v
	}
	if in.PriceMinor != nil && (*in.PriceMinor < 0 || *in.PriceMinor > 1_000_000_000_000) {
		return nil, ErrInvalid
	}
	v, err := s.repo.UpdateVariant(ctx, tenantID, variantID, in)
	return v, normalizeDBError(err)
}
func (s *Service) ListStock(ctx context.Context, tenantID, search string, limit, offset int) ([]StockItem, string, error) {
	return s.repo.ListStock(ctx, tenantID, strings.TrimSpace(search), pageLimit(limit), pageOffset(offset))
}
func (s *Service) ListLowStock(ctx context.Context, tenantID string, limit, offset int) ([]ReorderItem, error) {
	return s.repo.ListLowStock(ctx, tenantID, pageLimit(limit), pageOffset(offset))
}
func (s *Service) SaveReorder(ctx context.Context, tenantID, variantID string, in ReorderInput) error {
	in.StoreID = strings.TrimSpace(in.StoreID)
	if !strings.HasPrefix(variantID, "var_") || len(variantID) != 16 || !strings.HasPrefix(in.StoreID, "str_") || len(in.StoreID) != 16 ||
		(in.Threshold != nil && (*in.Threshold < 0 || *in.Threshold > 1_000_000_000)) {
		return ErrInvalidReorder
	}
	if in.PreferredSupplierID != nil {
		v := strings.TrimSpace(*in.PreferredSupplierID)
		if v != "" && (!strings.HasPrefix(v, "sup_") || len(v) != 16) {
			return ErrInvalidReorder
		}
		if v == "" {
			in.PreferredSupplierID = nil
		} else {
			in.PreferredSupplierID = &v
		}
	}
	if in.Threshold == nil && in.PreferredSupplierID != nil {
		return ErrInvalidReorder
	}
	return s.repo.SaveReorder(ctx, tenantID, variantID, in)
}
func (s *Service) Summary(ctx context.Context, tenantID string) (*Summary, error) {
	return s.repo.Summary(ctx, tenantID)
}
func (s *Service) RecordMovement(ctx context.Context, tenantID, userID string, in MovementInput) (*Movement, bool, error) {
	in.VariantID = strings.TrimSpace(in.VariantID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = strings.TrimSpace(in.Note)
	if in.VariantID == "" || len(in.Note) > 500 || len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 100 {
		return nil, false, ErrInvalid
	}
	switch in.Kind {
	case "receive":
		if in.Quantity == nil || *in.Quantity <= 0 || *in.Quantity > 1_000_000_000 || in.CountedQuantity != nil {
			return nil, false, ErrInvalid
		}
	case "count":
		if in.CountedQuantity == nil || *in.CountedQuantity < 0 || *in.CountedQuantity > 1_000_000_000 || in.Quantity != nil || in.Note == "" {
			return nil, false, ErrInvalid
		}
	default:
		return nil, false, ErrInvalid
	}
	return s.repo.RecordMovement(ctx, tenantID, userID, in)
}
func (s *Service) ListMovements(ctx context.Context, tenantID string, limit, offset int) ([]Movement, error) {
	return s.repo.ListMovements(ctx, tenantID, pageLimit(limit), pageOffset(offset))
}
func pageLimit(n int) int {
	if n <= 0 {
		return 50
	}
	if n > 100 {
		return 100
	}
	return n
}
func pageOffset(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
