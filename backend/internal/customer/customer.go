package customer

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Input struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type UpdateInput struct {
	Name   *string `json:"name"`
	Phone  *string `json:"phone"`
	Email  *string `json:"email"`
	Status *string `json:"status"`
}

type SaleSummary struct {
	ID              string    `json:"id"`
	OrderNumber     string    `json:"orderNumber"`
	StoreName       string    `json:"storeName"`
	CurrencyCode    string    `json:"currencyCode"`
	GrandTotalMinor int64     `json:"grandTotalMinor"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid       = &Error{http.StatusBadRequest, "invalid_customer", "Enter a name and at least one valid phone or email."}
	ErrNotFound      = &Error{http.StatusNotFound, "customer_not_found", "Customer not found in this workspace."}
	ErrDuplicate     = &Error{http.StatusConflict, "customer_duplicate", "An active customer already uses this phone or email."}
	ErrInvalidSearch = &Error{http.StatusBadRequest, "invalid_customer_search", "Search text is too long."}
)

type Repository interface {
	List(context.Context, string, string, bool, int, int) ([]Customer, error)
	Create(context.Context, string, Input) (*Customer, error)
	Update(context.Context, string, string, UpdateInput) (*Customer, error)
	History(context.Context, string, string, int, int) ([]SaleSummary, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func normalizePhone(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '-', '(', ')':
			return -1
		}
		return r
	}, strings.TrimSpace(value))
}
func validPhone(value string) bool {
	if value == "" {
		return true
	}
	digits := strings.TrimPrefix(value, "+")
	if digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func validEmail(value string) bool {
	if value == "" {
		return true
	}
	a, err := mail.ParseAddress(value)
	return err == nil && a.Address == value && !strings.ContainsAny(value, " \t\r\n")
}
func validContact(name, phone, email string) bool {
	return name != "" && len(name) <= 160 && len(phone) <= 40 && len(email) <= 254 &&
		(phone != "" || email != "") && validPhone(phone) && validEmail(email)
}
func normalizeError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" &&
		(pg.ConstraintName == "customers_active_phone_idx" || pg.ConstraintName == "customers_active_email_idx") {
		return ErrDuplicate
	}
	return err
}
func page(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
func (s *Service) List(ctx context.Context, tenantID, search string, full bool, limit, offset int) ([]Customer, error) {
	search = strings.TrimSpace(search)
	if len(search) > 160 {
		return nil, ErrInvalidSearch
	}
	limit, offset = page(limit, offset)
	return s.repo.List(ctx, tenantID, search, full, limit, offset)
}
func (s *Service) Create(ctx context.Context, tenantID string, in Input) (*Customer, error) {
	in.Name, in.Phone, in.Email = strings.TrimSpace(in.Name), normalizePhone(in.Phone), normalizeEmail(in.Email)
	if !validContact(in.Name, in.Phone, in.Email) {
		return nil, ErrInvalid
	}
	out, err := s.repo.Create(ctx, tenantID, in)
	return out, normalizeError(err)
}
func (s *Service) Update(ctx context.Context, tenantID, customerID string, in UpdateInput) (*Customer, error) {
	if in.Name == nil && in.Phone == nil && in.Email == nil && in.Status == nil {
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
		v := normalizePhone(*in.Phone)
		if len(v) > 40 || !validPhone(v) {
			return nil, ErrInvalid
		}
		in.Phone = &v
	}
	if in.Email != nil {
		v := normalizeEmail(*in.Email)
		if len(v) > 254 || !validEmail(v) {
			return nil, ErrInvalid
		}
		in.Email = &v
	}
	if in.Status != nil && *in.Status != "active" && *in.Status != "archived" {
		return nil, ErrInvalid
	}
	out, err := s.repo.Update(ctx, tenantID, customerID, in)
	return out, normalizeError(err)
}
func (s *Service) History(ctx context.Context, tenantID, customerID string, limit, offset int) ([]SaleSummary, error) {
	limit, offset = page(limit, offset)
	return s.repo.History(ctx, tenantID, customerID, limit, offset)
}
