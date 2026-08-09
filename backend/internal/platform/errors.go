package platform

import "net/http"

// Error is a typed domain error carrying the HTTP status and a stable code
// (mirrors the auth and workspace slices' convention).
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
	// ErrNotFound is returned both for a genuinely missing record and for a
	// non-staff caller touching a console route — the console shouldn't confirm
	// its own existence to somebody probing for it.
	ErrNotFound = newError(http.StatusNotFound, "not_found", "Not found.")
	// ErrMFARequired is the one case we *do* explain: a real admin whose account
	// has no second factor, who needs to be told what to fix.
	ErrMFARequired = newError(http.StatusForbidden, "platform_mfa_required",
		"Two-factor authentication is required for platform access. Enable it on your account, then sign in again.")
	ErrForbidden = newError(http.StatusForbidden, "forbidden", "Your admin role doesn't allow this action.")
)
