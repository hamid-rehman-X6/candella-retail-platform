package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"candella-ecosystem/backend/pkg/id"
)

// repoPostgres is the Postgres-backed Repository. It is the only place in the auth
// domain that writes SQL. Row IDs are generated here (prefixed NanoIDs, per the
// schema design) and assigned back onto the passed structs.
type repoPostgres struct {
	db *pgxpool.Pool
}

// NewPostgresRepository wires the Repository to a pgx pool.
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &repoPostgres{db: db}
}

// mapErr converts pgx's "no rows" into the domain ErrNotFound.
func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// ----- Users --------------------------------------------------------------

func (r *repoPostgres) CreateUser(ctx context.Context, u *User) error {
	u.ID = id.New(id.User)
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, email_verified_at)
		VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.PasswordHash, u.FullName, u.EmailVerifiedAt)
	return err
}

const userColumns = `id, email, password_hash, full_name, status, email_verified_at, mfa_enabled, last_login_at, created_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Status,
		&u.EmailVerifiedAt, &u.MFAEnabled, &u.LastLoginAt, &u.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *repoPostgres) GetUserByID(ctx context.Context, id string) (*User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *repoPostgres) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1 AND deleted_at IS NULL`, email))
}

func (r *repoPostgres) SetEmailVerified(ctx context.Context, userID string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET email_verified_at = $2 WHERE id = $1`, userID, at)
	return err
}

func (r *repoPostgres) UpdateLastLogin(ctx context.Context, userID string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET last_login_at = $2 WHERE id = $1`, userID, at)
	return err
}

func (r *repoPostgres) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2 WHERE id = $1`, userID, passwordHash)
	return err
}

func (r *repoPostgres) SetMFAEnabled(ctx context.Context, userID string, enabled bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET mfa_enabled = $2 WHERE id = $1`, userID, enabled)
	return err
}

// ----- Verification codes --------------------------------------------------

func (r *repoPostgres) DeleteVerificationCodes(ctx context.Context, userID, purpose string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM verification_codes WHERE user_id = $1 AND purpose = $2`, userID, purpose)
	return err
}

func (r *repoPostgres) CreateVerificationCode(ctx context.Context, vc *VerificationCode) error {
	vc.ID = id.New(id.VerificationCode)
	_, err := r.db.Exec(ctx, `
		INSERT INTO verification_codes (id, user_id, purpose, code_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		vc.ID, vc.UserID, vc.Purpose, vc.CodeHash, vc.ExpiresAt)
	return err
}

func (r *repoPostgres) GetActiveVerificationCode(ctx context.Context, userID, purpose string) (*VerificationCode, error) {
	var vc VerificationCode
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, purpose, code_hash, attempts, expires_at
		FROM verification_codes
		WHERE user_id = $1 AND purpose = $2 AND consumed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1`, userID, purpose).
		Scan(&vc.ID, &vc.UserID, &vc.Purpose, &vc.CodeHash, &vc.Attempts, &vc.ExpiresAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &vc, nil
}

func (r *repoPostgres) IncrementVerificationAttempts(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE verification_codes SET attempts = attempts + 1 WHERE id = $1`, id)
	return err
}

func (r *repoPostgres) ConsumeVerificationCode(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE verification_codes SET consumed_at = $2 WHERE id = $1`, id, at)
	return err
}

// ----- Sessions ------------------------------------------------------------

func (r *repoPostgres) CreateSession(ctx context.Context, s *Session) error {
	s.ID = id.New(id.Session)
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		s.ID, s.UserID, s.TokenHash, nil, nil, s.ExpiresAt)
	return err
}

func (r *repoPostgres) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	var s Session
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM sessions WHERE token_hash = $1`, tokenHash).
		Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &s, nil
}

func (r *repoPostgres) RevokeSession(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, id, at)
	return err
}

func (r *repoPostgres) RevokeAllUserSessions(ctx context.Context, userID string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, at)
	return err
}

// ----- Password reset ------------------------------------------------------

func (r *repoPostgres) CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error {
	t.ID = id.New(id.PasswordResetTok)
	_, err := r.db.Exec(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`,
		t.ID, t.UserID, t.TokenHash, t.ExpiresAt)
	return err
}

func (r *repoPostgres) GetPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	var t PasswordResetToken
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at
		FROM password_reset_tokens
		WHERE token_hash = $1 AND consumed_at IS NULL`, tokenHash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &t, nil
}

func (r *repoPostgres) ConsumePasswordResetToken(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE password_reset_tokens SET consumed_at = $2 WHERE id = $1`, id, at)
	return err
}

// ----- MFA -----------------------------------------------------------------

func (r *repoPostgres) UpsertUserMFASecret(ctx context.Context, userID string, encryptedSecret []byte) error {
	// Replace any prior (unconfirmed) secret and reset enabled_at until re-confirmed.
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_mfa (user_id, totp_secret_encrypted, enabled_at)
		VALUES ($1, $2, NULL)
		ON CONFLICT (user_id)
		DO UPDATE SET totp_secret_encrypted = EXCLUDED.totp_secret_encrypted, enabled_at = NULL`,
		userID, encryptedSecret)
	return err
}

func (r *repoPostgres) GetUserMFA(ctx context.Context, userID string) (*UserMFA, error) {
	var m UserMFA
	err := r.db.QueryRow(ctx, `
		SELECT user_id, totp_secret_encrypted, enabled_at
		FROM user_mfa WHERE user_id = $1`, userID).
		Scan(&m.UserID, &m.TOTPSecretEncrypted, &m.EnabledAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func (r *repoPostgres) EnableUserMFA(ctx context.Context, userID string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_mfa SET enabled_at = $2 WHERE user_id = $1`, userID, at)
	return err
}

func (r *repoPostgres) DisableUserMFA(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_mfa WHERE user_id = $1`, userID)
	return err
}

func (r *repoPostgres) ReplaceBackupCodes(ctx context.Context, userID string, codeHashes []string) error {
	// Do it atomically so a user never ends up with a half-written code set.
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM mfa_backup_codes WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for _, h := range codeHashes {
			if _, err := tx.Exec(ctx,
				`INSERT INTO mfa_backup_codes (id, user_id, code_hash) VALUES ($1, $2, $3)`,
				id.New(id.MFABackupCode), userID, h); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repoPostgres) GetUnusedBackupCodeByHash(ctx context.Context, userID, codeHash string) (*BackupCode, error) {
	var b BackupCode
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, code_hash FROM mfa_backup_codes
		WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL`, userID, codeHash).
		Scan(&b.ID, &b.UserID, &b.CodeHash)
	if err != nil {
		return nil, mapErr(err)
	}
	return &b, nil
}

func (r *repoPostgres) ConsumeBackupCode(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE mfa_backup_codes SET used_at = $2 WHERE id = $1`, id, at)
	return err
}

// ----- OAuth ---------------------------------------------------------------

func (r *repoPostgres) GetOAuthAccount(ctx context.Context, provider, providerAccountID string) (*OAuthAccount, error) {
	var a OAuthAccount
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_account_id, email
		FROM oauth_accounts WHERE provider = $1 AND provider_account_id = $2`,
		provider, providerAccountID).
		Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderAccountID, &a.Email)
	if err != nil {
		return nil, mapErr(err)
	}
	return &a, nil
}

func (r *repoPostgres) CreateOAuthAccount(ctx context.Context, a *OAuthAccount) error {
	a.ID = id.New(id.OAuthAccount)
	_, err := r.db.Exec(ctx, `
		INSERT INTO oauth_accounts (id, user_id, provider, provider_account_id, email)
		VALUES ($1, $2, $3, $4, $5)`,
		a.ID, a.UserID, a.Provider, a.ProviderAccountID, a.Email)
	return err
}

// ----- Audit ---------------------------------------------------------------

func (r *repoPostgres) InsertAuditLog(ctx context.Context, e *AuditEntry) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, action, entity_type, entity_id, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id.New(id.AuditLog), e.UserID, e.Action, e.EntityType, e.EntityID, e.IPAddress)
	return err
}
