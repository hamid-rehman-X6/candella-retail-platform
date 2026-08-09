package workspace

import "net/http"

// Error is a typed domain error carrying the HTTP status and a stable code the
// frontend can switch on (mirrors the auth slice's convention).
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func newError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

var (
	ErrInvalidInput    = newError(http.StatusBadRequest, "invalid_input", "Some fields are missing or invalid.")
	ErrNameRequired    = newError(http.StatusBadRequest, "name_required", "Enter a name for your workspace.")
	ErrInvalidIndustry = newError(http.StatusBadRequest, "invalid_industry", "Choose a valid industry.")
	ErrUnauthenticated = newError(http.StatusUnauthorized, "unauthenticated", "You need to sign in to continue.")

	// Tenant resolution. "Not a member" is reported as 404 rather than 403 so a
	// caller can't probe which tenant IDs exist.
	ErrTenantRequired  = newError(http.StatusBadRequest, "tenant_required", "No workspace was specified for this request.")
	ErrNoMembership    = newError(http.StatusNotFound, "workspace_not_found", "That workspace doesn't exist, or you don't have access to it.")
	ErrTenantSuspended = newError(http.StatusForbidden, "workspace_suspended", "This workspace has been suspended. Contact support to restore access.")
)
