// Package promotion owns the discounts an owner or admin approves in advance,
// and the limit on manual discounts a cashier may give alone.
package promotion

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Promotion struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Kind             string    `json:"kind"` // percent | amount
	PercentBps       *int64    `json:"percentBps"`
	AmountMinor      *int64    `json:"amountMinor"`
	AppliesTo        string    `json:"appliesTo"` // order | item
	MinSubtotalMinor int64     `json:"minSubtotalMinor"`
	StartsOn         *string   `json:"startsOn"` // YYYY-MM-DD, inclusive
	EndsOn           *string   `json:"endsOn"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Input struct {
	Name             string  `json:"name"`
	Kind             string  `json:"kind"`
	PercentBps       *int64  `json:"percentBps"`
	AmountMinor      *int64  `json:"amountMinor"`
	AppliesTo        string  `json:"appliesTo"`
	MinSubtotalMinor int64   `json:"minSubtotalMinor"`
	StartsOn         *string `json:"startsOn"`
	EndsOn           *string `json:"endsOn"`
}

type Settings struct {
	// The largest manual discount a cashier may give alone; 0 = none.
	CashierLimitBps int64 `json:"cashierLimitBps"`
}

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid = &Error{http.StatusBadRequest, "invalid_promotion",
		"Enter a name, a percentage between 0.01% and 100% or an amount above zero, and an end date on or after the start date."}
	ErrNotFound  = &Error{http.StatusNotFound, "promotion_not_found", "This promotion was not found in this workspace."}
	ErrConflict  = &Error{http.StatusConflict, "promotion_conflict", "An active promotion with this name already exists."}
	ErrForbidden = &Error{http.StatusForbidden, "insufficient_role", "Only workspace owners and admins can change promotions and discount limits."}
	ErrSettings  = &Error{http.StatusBadRequest, "invalid_discount_limit", "Enter a cashier limit between 0% and 100%."}
)

type Repository interface {
	List(ctx context.Context, tenantID string, includeArchived bool) ([]Promotion, error)
	Create(ctx context.Context, tenantID string, in Input) (*Promotion, error)
	Update(ctx context.Context, tenantID, promotionID string, in Input, status *string) (*Promotion, error)
	Settings(ctx context.Context, tenantID string) (Settings, error)
	SaveSettings(ctx context.Context, tenantID string, s Settings) (Settings, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func Manages(role string) bool { return role == "owner" || role == "admin" }

func validDate(value *string) bool {
	if value == nil {
		return true
	}
	day, err := time.Parse("2006-01-02", *value)
	return err == nil && day.Format("2006-01-02") == *value
}

func normalize(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	for _, d := range []**string{&in.StartsOn, &in.EndsOn} {
		if *d != nil && strings.TrimSpace(**d) == "" {
			*d = nil
		}
	}
	switch {
	case len(in.Name) < 1 || len(in.Name) > 80,
		in.AppliesTo != "order" && in.AppliesTo != "item",
		in.MinSubtotalMinor < 0 || in.MinSubtotalMinor > 1_000_000_000_000,
		!validDate(in.StartsOn) || !validDate(in.EndsOn),
		in.StartsOn != nil && in.EndsOn != nil && *in.EndsOn < *in.StartsOn:
		return in, ErrInvalid
	}
	switch in.Kind {
	case "percent":
		if in.PercentBps == nil || *in.PercentBps < 1 || *in.PercentBps > 10000 {
			return in, ErrInvalid
		}
		in.AmountMinor = nil
	case "amount":
		if in.AmountMinor == nil || *in.AmountMinor < 1 || *in.AmountMinor > 1_000_000_000_000 {
			return in, ErrInvalid
		}
		in.PercentBps = nil
	default:
		return in, ErrInvalid
	}
	return in, nil
}

func (s *Service) List(ctx context.Context, tenantID string, includeArchived bool) ([]Promotion, error) {
	return s.repo.List(ctx, tenantID, includeArchived)
}

func (s *Service) Create(ctx context.Context, tenantID, role string, in Input) (*Promotion, error) {
	if !Manages(role) {
		return nil, ErrForbidden
	}
	in, err := normalize(in)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, tenantID, in)
}

func (s *Service) Update(ctx context.Context, tenantID, role, promotionID string, in Input, status *string) (*Promotion, error) {
	if !Manages(role) {
		return nil, ErrForbidden
	}
	if !strings.HasPrefix(promotionID, "promo_") || len(promotionID) != 18 {
		return nil, ErrNotFound
	}
	if status != nil && *status != "active" && *status != "archived" {
		return nil, ErrInvalid
	}
	in, err := normalize(in)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, tenantID, promotionID, in, status)
}

func (s *Service) Settings(ctx context.Context, tenantID string) (Settings, error) {
	return s.repo.Settings(ctx, tenantID)
}

func (s *Service) SaveSettings(ctx context.Context, tenantID, role string, in Settings) (Settings, error) {
	if !Manages(role) {
		return Settings{}, ErrForbidden
	}
	if in.CashierLimitBps < 0 || in.CashierLimitBps > 10000 {
		return Settings{}, ErrSettings
	}
	return s.repo.SaveSettings(ctx, tenantID, in)
}

func AsError(err error) (*Error, bool) {
	var e *Error
	return e, errors.As(err, &e)
}
