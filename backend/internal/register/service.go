package register

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

const maxCashMinor int64 = 1_000_000_000_000

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalid          = &Error{http.StatusBadRequest, "invalid_shift", "Check the cash amount and try again."}
	ErrInvalidName      = &Error{http.StatusBadRequest, "invalid_register", "Enter a counter name between 1 and 60 characters."}
	ErrRegisterRequired = &Error{http.StatusBadRequest, "register_required", "Select a counter before taking cash."}
	ErrRegisterNotFound = &Error{http.StatusNotFound, "register_not_found", "This counter was not found in the selected store."}
	ErrRegisterConflict = &Error{http.StatusConflict, "register_conflict", "A counter with this name already exists in this store."}
	ErrNotFound         = &Error{http.StatusNotFound, "shift_not_found", "This shift was not found in this workspace."}
	ErrAlreadyOpen      = &Error{http.StatusConflict, "shift_already_open", "This counter already has an open shift. Close it before opening a new one."}
	ErrNoOpenShift      = &Error{http.StatusConflict, "shift_required", "Open a shift on this counter before taking cash."}
	ErrClosed           = &Error{http.StatusConflict, "shift_closed", "This shift is already closed."}
	ErrNotYours         = &Error{http.StatusForbidden, "shift_not_yours", "Only the cashier who opened this shift, or a workspace owner or admin, can close it."}
	ErrReasonRequired   = &Error{http.StatusBadRequest, "variance_reason_required", "Counted cash differs from expected cash. Enter a reason for the difference (at least 3 characters)."}
	ErrExpectedChanged  = &Error{http.StatusConflict, "shift_changed", "A sale was completed while you were closing. Check the updated expected cash and count again."}
	ErrForbidden        = &Error{http.StatusForbidden, "insufficient_role", "Only workspace owners and admins can add counters."}
	ErrInvalidFilter    = &Error{http.StatusBadRequest, "invalid_shift_filter", "Check the shift filters and try again."}
)

type Repository interface {
	ListRegisters(ctx context.Context, tenantID, storeID string) ([]Register, error)
	CreateRegister(ctx context.Context, tenantID string, in CreateRegisterInput) (*Register, error)
	Open(ctx context.Context, tenantID string, actor Actor, in OpenInput) (*Shift, error)
	Get(ctx context.Context, tenantID string, actor Actor, shiftID string) (*Shift, error)
	Close(ctx context.Context, tenantID string, actor Actor, shiftID string, in CloseInput) (*Shift, error)
	List(ctx context.Context, tenantID string, actor Actor, filter ShiftFilter) ([]Shift, bool, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func validID(value, prefix string) bool {
	return strings.HasPrefix(value, prefix+"_") && len(value) == len(prefix)+13
}

func (s *Service) ListRegisters(ctx context.Context, tenantID, storeID string) ([]Register, error) {
	return s.repo.ListRegisters(ctx, tenantID, storeID)
}

func (s *Service) CreateRegister(ctx context.Context, tenantID string, actor Actor, in CreateRegisterInput) (*Register, error) {
	if !actor.manages() {
		return nil, ErrForbidden
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 60 {
		return nil, ErrInvalidName
	}
	return s.repo.CreateRegister(ctx, tenantID, in)
}

func (s *Service) Open(ctx context.Context, tenantID string, actor Actor, in OpenInput) (*Shift, error) {
	actor.Name = strings.TrimSpace(actor.Name)
	if in.RegisterID == "" {
		return nil, ErrRegisterRequired
	}
	if !validID(in.RegisterID, "reg") {
		return nil, ErrRegisterNotFound
	}
	if in.OpeningCashMinor < 0 || in.OpeningCashMinor > maxCashMinor || actor.Name == "" || len(actor.Name) > 200 {
		return nil, ErrInvalid
	}
	return s.repo.Open(ctx, tenantID, actor, in)
}

func (s *Service) Get(ctx context.Context, tenantID string, actor Actor, shiftID string) (*Shift, error) {
	if !validID(shiftID, "shift") {
		return nil, ErrNotFound
	}
	return s.repo.Get(ctx, tenantID, actor, shiftID)
}

func (s *Service) Close(ctx context.Context, tenantID string, actor Actor, shiftID string, in CloseInput) (*Shift, error) {
	actor.Name = strings.TrimSpace(actor.Name)
	if !validID(shiftID, "shift") {
		return nil, ErrNotFound
	}
	in.VarianceReason = strings.TrimSpace(in.VarianceReason)
	if in.CountedCashMinor < 0 || in.CountedCashMinor > maxCashMinor ||
		in.ExpectedCashMinor < 0 || len(in.VarianceReason) > 500 ||
		actor.Name == "" || len(actor.Name) > 200 {
		return nil, ErrInvalid
	}
	return s.repo.Close(ctx, tenantID, actor, shiftID, in)
}

func (s *Service) List(ctx context.Context, tenantID string, actor Actor, filter ShiftFilter) ([]Shift, bool, error) {
	if filter.Status != "" && filter.Status != "open" && filter.Status != "closed" || filter.Offset < 0 {
		return nil, false, ErrInvalidFilter
	}
	if filter.Limit <= 0 {
		filter.Limit = 25
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repo.List(ctx, tenantID, actor, filter)
}

// closingFigures applies the closing rule: the counted cash is compared with
// the expected cash, and any difference must be explained.
func closingFigures(openingMinor, cashSalesMinor int64, in CloseInput) (expected, variance int64, err error) {
	expected = openingMinor + cashSalesMinor
	if in.ExpectedCashMinor != expected {
		return 0, 0, ErrExpectedChanged
	}
	variance = in.CountedCashMinor - expected
	if variance != 0 && len(in.VarianceReason) < 3 {
		return 0, 0, ErrReasonRequired
	}
	return expected, variance, nil
}

func AsError(err error) (*Error, bool) {
	var e *Error
	return e, errors.As(err, &e)
}
