package tax

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Rate struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RateBps      int64     `json:"rateBps"`
	Inclusive    bool      `json:"inclusive"`
	Status       string    `json:"status"`
	ProductCount int64     `json:"productCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

type RateInput struct {
	Name      string `json:"name"`
	RateBps   int64  `json:"rateBps"`
	Inclusive bool   `json:"inclusive"`
}

type RateUpdate struct {
	Name      *string `json:"name"`
	RateBps   *int64  `json:"rateBps"`
	Inclusive *bool   `json:"inclusive"`
	Status    *string `json:"status"`
}

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid   = &Error{http.StatusBadRequest, "invalid_tax_rate", "Enter a name (1 to 60 characters) and a rate between 0% and 100%."}
	ErrNotFound  = &Error{http.StatusNotFound, "tax_rate_not_found", "This tax rate was not found in this workspace."}
	ErrConflict  = &Error{http.StatusConflict, "tax_rate_conflict", "An active tax rate with this name already exists."}
	ErrForbidden = &Error{http.StatusForbidden, "insufficient_role", "Only workspace owners and admins can change tax rates."}
)

type Repository interface {
	List(ctx context.Context, tenantID string, includeArchived bool) ([]Rate, error)
	Create(ctx context.Context, tenantID string, in RateInput) (*Rate, error)
	Update(ctx context.Context, tenantID, rateID string, in RateUpdate) (*Rate, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func manages(role string) bool { return role == "owner" || role == "admin" }

func validRate(bps int64) bool { return bps >= 0 && bps <= 10000 }

func (s *Service) List(ctx context.Context, tenantID string, includeArchived bool) ([]Rate, error) {
	return s.repo.List(ctx, tenantID, includeArchived)
}

func (s *Service) Create(ctx context.Context, tenantID, role string, in RateInput) (*Rate, error) {
	if !manages(role) {
		return nil, ErrForbidden
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 60 || !validRate(in.RateBps) {
		return nil, ErrInvalid
	}
	return s.repo.Create(ctx, tenantID, in)
}

func (s *Service) Update(ctx context.Context, tenantID, role, rateID string, in RateUpdate) (*Rate, error) {
	if !manages(role) {
		return nil, ErrForbidden
	}
	if !strings.HasPrefix(rateID, "tax_") || len(rateID) != 16 {
		return nil, ErrNotFound
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if len(name) < 1 || len(name) > 60 {
			return nil, ErrInvalid
		}
		in.Name = &name
	}
	if (in.RateBps != nil && !validRate(*in.RateBps)) ||
		(in.Status != nil && *in.Status != "active" && *in.Status != "archived") {
		return nil, ErrInvalid
	}
	return s.repo.Update(ctx, tenantID, rateID, in)
}

func AsError(err error) (*Error, bool) {
	var e *Error
	return e, errors.As(err, &e)
}
