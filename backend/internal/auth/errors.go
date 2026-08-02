package auth

import "net/http"

// Error is a typed domain error carrying the HTTP status and the stable,
// machine-readable code the frontend switches on (e.g. "invalid_credentials").
// Handlers translate these directly into the JSON error envelope; any non-*Error
// value is treated as an unexpected 500.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func newError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Predefined domain errors. Messages are safe to show end users and deliberately
// avoid leaking whether an account exists (see the login / forgot-password flows).
var (
	ErrInvalidInput      = newError(http.StatusBadRequest, "invalid_input", "Some fields are missing or invalid.")
	ErrInvalidEmail      = newError(http.StatusBadRequest, "invalid_email", "Enter a valid email address.")
	ErrWeakPassword      = newError(http.StatusBadRequest, "weak_password", "Choose a stronger password (8+ chars, mixed types).")
	ErrEmailTaken        = newError(http.StatusConflict, "email_taken", "An account with this email already exists.")
	ErrInvalidCredential = newError(http.StatusUnauthorized, "invalid_credentials", "Incorrect email or password.")
	ErrEmailNotVerified  = newError(http.StatusForbidden, "email_not_verified", "Please verify your email before signing in.")
	ErrCodeInvalid       = newError(http.StatusBadRequest, "code_invalid", "That code isn't right. Check it and try again.")
	ErrCodeExpired       = newError(http.StatusBadRequest, "code_expired", "That code has expired. Request a new one.")
	ErrTooManyAttempts   = newError(http.StatusTooManyRequests, "too_many_attempts", "Too many attempts. Request a new code.")
	ErrUnauthenticated   = newError(http.StatusUnauthorized, "unauthenticated", "You need to sign in to continue.")
	ErrTokenInvalid      = newError(http.StatusBadRequest, "token_invalid", "This link is invalid or has expired.")
	ErrMFARequired       = newError(http.StatusUnauthorized, "mfa_required", "Two-factor authentication is required.")
	ErrMFANotConfigured  = newError(http.StatusBadRequest, "mfa_not_configured", "Two-factor authentication isn't set up.")
	ErrMFAUnavailable    = newError(http.StatusServiceUnavailable, "mfa_unavailable", "Two-factor authentication isn't available right now.")
	ErrMFAAlreadyEnabled = newError(http.StatusConflict, "mfa_already_enabled", "Two-factor authentication is already enabled.")
	ErrGoogleDisabled    = newError(http.StatusBadRequest, "google_not_configured", "Google sign-in is not configured.")
	ErrRateLimited       = newError(http.StatusTooManyRequests, "rate_limited", "Too many requests. Please slow down and try again.")
)
