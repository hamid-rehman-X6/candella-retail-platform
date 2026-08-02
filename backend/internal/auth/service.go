package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"candella-ecosystem/backend/pkg/config"
	"candella-ecosystem/backend/pkg/crypto"
	"candella-ecosystem/backend/pkg/mailer"
)

const (
	purposeEmailVerify = "email_verify"
	purposeMFALogin    = "mfa_login" // challenge purpose between password and 2FA step
	maxCodeAttempts    = 5
	emailCodeDigits    = 6
	backupCodeCount    = 10
)

// Service holds the auth business logic. It depends on the Repository and Mailer
// interfaces (not concrete types), so it is fully unit-testable with fakes.
type Service struct {
	repo         Repository
	mailer       mailer.Mailer
	hasher       *Hasher
	enc          *crypto.Encryptor // nil when TOTP_ENCRYPTION_KEY is not configured
	cfg          config.Config
	log          *slog.Logger
	challengeKey []byte // HMAC key for stateless mfa/oauth challenges (per-process)
	google       *googleOAuth
}

// NewService constructs the auth service. enc may be nil (2FA setup will then be
// rejected); google may be nil (Google sign-in is then reported as not configured).
func NewService(
	repo Repository,
	m mailer.Mailer,
	cfg config.Config,
	log *slog.Logger,
	enc *crypto.Encryptor,
	challengeKey []byte,
) *Service {
	return &Service{
		repo:         repo,
		mailer:       m,
		hasher:       NewHasher(cfg.Argon2MemoryKB, cfg.Argon2Time, cfg.Argon2Threads),
		enc:          enc,
		cfg:          cfg,
		log:          log,
		challengeKey: challengeKey,
		google:       newGoogleOAuth(cfg),
	}
}

// Register creates an unverified account and emails a 6-digit verification code.
func (s *Service) Register(ctx context.Context, req registerRequest) (*User, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	if _, err := s.repo.GetUserByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	u := &User{Email: req.Email, PasswordHash: &hash, FullName: req.FullName}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	if err := s.issueEmailCode(ctx, u); err != nil {
		return nil, err
	}
	s.audit(ctx, &u.ID, "user.registered", "user", u.ID)
	return u, nil
}

// VerifyEmail checks the 6-digit code, marks the email verified, and starts a
// session (so the user is logged in immediately after verifying).
func (s *Service) VerifyEmail(ctx context.Context, req emailCodeRequest) (*loginResult, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrCodeInvalid
		}
		return nil, err
	}

	if err := s.checkCode(ctx, user.ID, purposeEmailVerify, req.Code); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.repo.SetEmailVerified(ctx, user.ID, now); err != nil {
		return nil, err
	}
	s.audit(ctx, &user.ID, "user.email_verified", "user", user.ID)

	raw, expires, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.EmailVerifiedAt = &now
	return &loginResult{RawToken: raw, Expires: expires, User: user}, nil
}

// ResendVerification issues a fresh code. It never reveals whether the account
// exists (or is already verified) — the handler always returns a generic 200.
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if user.IsEmailVerified() {
		return nil
	}
	return s.issueEmailCode(ctx, user)
}

// Login verifies the password and either starts a session or, when the account
// has 2FA enabled, returns a short-lived mfaToken for the second step.
func (s *Service) Login(ctx context.Context, req loginRequest) (*loginResult, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, err
	}
	// OAuth-only accounts have no password.
	if user.PasswordHash == nil {
		return nil, ErrInvalidCredential
	}
	ok, err := VerifyPassword(req.Password, *user.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredential
	}
	if !user.IsEmailVerified() {
		return nil, ErrEmailNotVerified
	}

	if user.MFAEnabled {
		token := signChallenge(s.challengeKey, user.ID, purposeMFALogin, 5*time.Minute)
		return &loginResult{MFARequired: true, MFAToken: token}, nil
	}

	return s.startSession(ctx, user)
}

// Logout revokes the session behind the given raw cookie token.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	sess, err := s.repo.GetSessionByTokenHash(ctx, crypto.HashToken(rawToken))
	if err != nil {
		return nil // already gone — treat logout as idempotent
	}
	return s.repo.RevokeSession(ctx, sess.ID, time.Now())
}

// Authenticate resolves a raw session token to its user (used by middleware).
// Any problem (missing/expired/revoked session) is reported as ErrUnauthenticated.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (*User, error) {
	if rawToken == "" {
		return nil, ErrUnauthenticated
	}
	sess, err := s.repo.GetSessionByTokenHash(ctx, crypto.HashToken(rawToken))
	if err != nil {
		return nil, ErrUnauthenticated
	}
	if sess.RevokedAt != nil || time.Now().After(sess.ExpiresAt) {
		return nil, ErrUnauthenticated
	}
	user, err := s.repo.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, ErrUnauthenticated
	}
	return user, nil
}

// ---- internal helpers -----------------------------------------------------

// startSession creates a session and records the login. Shared by password login
// and the 2FA / OAuth completion paths.
func (s *Service) startSession(ctx context.Context, user *User) (*loginResult, error) {
	raw, expires, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	_ = s.repo.UpdateLastLogin(ctx, user.ID, time.Now())
	s.audit(ctx, &user.ID, "user.login", "user", user.ID)
	return &loginResult{RawToken: raw, Expires: expires, User: user}, nil
}

func (s *Service) createSession(ctx context.Context, userID string) (string, time.Time, error) {
	raw, err := randomToken(sessionTokenBytes)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(s.cfg.SessionTTL)
	sess := &Session{UserID: userID, TokenHash: crypto.HashToken(raw), ExpiresAt: expires}
	if err := s.repo.CreateSession(ctx, sess); err != nil {
		return "", time.Time{}, err
	}
	return raw, expires, nil
}

// issueEmailCode replaces any prior codes, stores a fresh hashed code, and emails it.
func (s *Service) issueEmailCode(ctx context.Context, user *User) error {
	if err := s.repo.DeleteVerificationCodes(ctx, user.ID, purposeEmailVerify); err != nil {
		return err
	}
	code, err := numericCode(emailCodeDigits)
	if err != nil {
		return err
	}
	vc := &VerificationCode{
		UserID:    user.ID,
		Purpose:   purposeEmailVerify,
		CodeHash:  crypto.HashToken(code),
		ExpiresAt: time.Now().Add(s.cfg.EmailCodeTTL),
	}
	if err := s.repo.CreateVerificationCode(ctx, vc); err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Email{
		To:      user.Email,
		Subject: "Your Candella verification code",
		Text:    "Your verification code is " + code + ". It expires in 10 minutes.",
	})
}

// checkCode validates a one-time code against the active row, enforcing expiry and
// an attempt limit. On success it consumes the code.
func (s *Service) checkCode(ctx context.Context, userID, purpose, code string) error {
	vc, err := s.repo.GetActiveVerificationCode(ctx, userID, purpose)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrCodeInvalid
		}
		return err
	}
	if time.Now().After(vc.ExpiresAt) {
		return ErrCodeExpired
	}
	if vc.Attempts >= maxCodeAttempts {
		return ErrTooManyAttempts
	}
	if !crypto.ConstantTimeEquals(vc.CodeHash, crypto.HashToken(code)) {
		_ = s.repo.IncrementVerificationAttempts(ctx, vc.ID)
		return ErrCodeInvalid
	}
	return s.repo.ConsumeVerificationCode(ctx, vc.ID, time.Now())
}

// audit writes a best-effort audit log entry; failures are logged, not returned.
func (s *Service) audit(ctx context.Context, userID *string, action, entityType, entityID string) {
	if err := s.repo.InsertAuditLog(ctx, &AuditEntry{
		UserID: userID, Action: action, EntityType: entityType, EntityID: entityID,
	}); err != nil {
		s.log.Warn("audit log failed", "action", action, "error", err)
	}
}
