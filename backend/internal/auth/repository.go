package auth

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by repository reads when no matching row exists. The
// service maps it to the appropriate domain error (often a generic one, to avoid
// revealing whether an account exists).
var ErrNotFound = errors.New("auth: not found")

// Repository is the data-access port for the auth domain. The service depends on
// this interface; the Postgres implementation (repoPostgres) is the only code that
// writes SQL. This makes the service unit-testable with an in-memory fake.
type Repository interface {
	// Users
	CreateUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	SetEmailVerified(ctx context.Context, userID string, at time.Time) error
	UpdateLastLogin(ctx context.Context, userID string, at time.Time) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	SetMFAEnabled(ctx context.Context, userID string, enabled bool) error

	// Email verification codes
	DeleteVerificationCodes(ctx context.Context, userID, purpose string) error
	CreateVerificationCode(ctx context.Context, vc *VerificationCode) error
	GetActiveVerificationCode(ctx context.Context, userID, purpose string) (*VerificationCode, error)
	IncrementVerificationAttempts(ctx context.Context, id string) error
	ConsumeVerificationCode(ctx context.Context, id string, at time.Time) error

	// Sessions
	CreateSession(ctx context.Context, s *Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	RevokeSession(ctx context.Context, id string, at time.Time) error
	RevokeAllUserSessions(ctx context.Context, userID string, at time.Time) error

	// Password reset
	CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error
	GetPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	ConsumePasswordResetToken(ctx context.Context, id string, at time.Time) error

	// MFA (TOTP + backup codes)
	UpsertUserMFASecret(ctx context.Context, userID string, encryptedSecret []byte) error
	GetUserMFA(ctx context.Context, userID string) (*UserMFA, error)
	EnableUserMFA(ctx context.Context, userID string, at time.Time) error
	DisableUserMFA(ctx context.Context, userID string) error
	ReplaceBackupCodes(ctx context.Context, userID string, codeHashes []string) error
	GetUnusedBackupCodeByHash(ctx context.Context, userID, codeHash string) (*BackupCode, error)
	ConsumeBackupCode(ctx context.Context, id string, at time.Time) error

	// OAuth
	GetOAuthAccount(ctx context.Context, provider, providerAccountID string) (*OAuthAccount, error)
	CreateOAuthAccount(ctx context.Context, a *OAuthAccount) error

	// Audit
	InsertAuditLog(ctx context.Context, e *AuditEntry) error
}
