package auth

import "time"

// User is the domain view of a row in the users table.
type User struct {
	ID              string
	Email           string
	PasswordHash    *string // nil for OAuth-only accounts
	FullName        string
	Status          string
	EmailVerifiedAt *time.Time
	MFAEnabled      bool
	LastLoginAt     *time.Time
	CreatedAt       time.Time
}

// IsEmailVerified reports whether the user has proven ownership of their email.
func (u User) IsEmailVerified() bool { return u.EmailVerifiedAt != nil }

// Session is a row in the sessions table (the raw token lives only in the cookie).
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// VerificationCode is a hashed one-time code (e.g. email verification).
type VerificationCode struct {
	ID        string
	UserID    string
	Purpose   string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

// PasswordResetToken is a hashed, single-use reset token.
type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
}

// UserMFA holds a user's encrypted TOTP secret and enabled state.
type UserMFA struct {
	UserID              string
	TOTPSecretEncrypted []byte
	EnabledAt           *time.Time
}

// IsEnabled reports whether 2FA has been confirmed and turned on.
func (m UserMFA) IsEnabled() bool { return m.EnabledAt != nil }

// BackupCode is a single-use 2FA recovery code (stored hashed).
type BackupCode struct {
	ID       string
	UserID   string
	CodeHash string
}

// OAuthAccount links a user to an external identity provider account.
type OAuthAccount struct {
	ID                string
	UserID            string
	Provider          string
	ProviderAccountID string
	Email             *string
}

// AuditEntry is a security event to append to audit_logs.
type AuditEntry struct {
	UserID     *string
	Action     string
	EntityType string
	EntityID   string
	IPAddress  *string
}
