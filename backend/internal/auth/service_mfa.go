package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"candella-ecosystem/backend/pkg/crypto"
)

// SetupTwoFactor begins 2FA enrolment: it generates a TOTP secret, stores it
// encrypted (unconfirmed), and returns the secret + otpauth URL for the QR code.
// The user must then confirm a code via EnableTwoFactor to actually turn it on.
func (s *Service) SetupTwoFactor(ctx context.Context, user *User) (secret, otpauthURL string, err error) {
	if s.enc == nil {
		return "", "", ErrMFAUnavailable
	}
	if mfa, err := s.repo.GetUserMFA(ctx, user.ID); err == nil && mfa.IsEnabled() {
		return "", "", ErrMFAAlreadyEnabled
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return "", "", err
	}

	secret, otpauthURL, err = generateTOTPSecret(user.Email)
	if err != nil {
		return "", "", err
	}
	enc, err := s.enc.Encrypt([]byte(secret))
	if err != nil {
		return "", "", err
	}
	if err := s.repo.UpsertUserMFASecret(ctx, user.ID, enc); err != nil {
		return "", "", err
	}
	return secret, otpauthURL, nil
}

// EnableTwoFactor confirms the first TOTP code, turns 2FA on, and returns a fresh
// set of single-use backup codes (shown to the user exactly once).
func (s *Service) EnableTwoFactor(ctx context.Context, user *User, code string) ([]string, error) {
	if s.enc == nil {
		return nil, ErrMFAUnavailable
	}
	mfa, err := s.repo.GetUserMFA(ctx, user.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrMFANotConfigured
		}
		return nil, err
	}
	if mfa.IsEnabled() {
		return nil, ErrMFAAlreadyEnabled
	}

	secret, err := s.enc.Decrypt(mfa.TOTPSecretEncrypted)
	if err != nil {
		return nil, err
	}
	if !validateTOTP(code, string(secret)) {
		return nil, ErrCodeInvalid
	}

	now := time.Now()
	if err := s.repo.EnableUserMFA(ctx, user.ID, now); err != nil {
		return nil, err
	}
	if err := s.repo.SetMFAEnabled(ctx, user.ID, true); err != nil {
		return nil, err
	}

	plain, hashes, err := generateBackupCodes(backupCodeCount)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceBackupCodes(ctx, user.ID, hashes); err != nil {
		return nil, err
	}
	s.audit(ctx, &user.ID, "mfa.enabled", "user", user.ID)
	return plain, nil
}

// DisableTwoFactor turns 2FA off after re-verifying the user's password.
func (s *Service) DisableTwoFactor(ctx context.Context, user *User, password string) error {
	if user.PasswordHash == nil {
		return ErrInvalidCredential
	}
	ok, err := VerifyPassword(password, *user.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCredential
	}
	if !user.MFAEnabled {
		return ErrMFANotConfigured
	}

	if err := s.repo.ReplaceBackupCodes(ctx, user.ID, nil); err != nil {
		return err
	}
	if err := s.repo.DisableUserMFA(ctx, user.ID); err != nil {
		return err
	}
	if err := s.repo.SetMFAEnabled(ctx, user.ID, false); err != nil {
		return err
	}
	s.audit(ctx, &user.ID, "mfa.disabled", "user", user.ID)
	return nil
}

// LoginVerifyTwoFactor completes login by validating a TOTP code against the
// short-lived mfaToken issued at the password step.
func (s *Service) LoginVerifyTwoFactor(ctx context.Context, req mfaCodeRequest) (*loginResult, error) {
	user, err := s.userFromMFAToken(ctx, req.MFAToken)
	if err != nil {
		return nil, err
	}
	if s.enc == nil {
		return nil, ErrMFAUnavailable
	}
	mfa, err := s.repo.GetUserMFA(ctx, user.ID)
	if err != nil || !mfa.IsEnabled() {
		return nil, ErrMFANotConfigured
	}
	secret, err := s.enc.Decrypt(mfa.TOTPSecretEncrypted)
	if err != nil {
		return nil, err
	}
	if !validateTOTP(req.Code, string(secret)) {
		return nil, ErrCodeInvalid
	}
	return s.startSession(ctx, user)
}

// LoginVerifyBackup completes login by consuming one single-use backup code.
func (s *Service) LoginVerifyBackup(ctx context.Context, req mfaBackupRequest) (*loginResult, error) {
	user, err := s.userFromMFAToken(ctx, req.MFAToken)
	if err != nil {
		return nil, err
	}
	code := strings.ToUpper(strings.TrimSpace(req.BackupCode))
	bc, err := s.repo.GetUnusedBackupCodeByHash(ctx, user.ID, crypto.HashToken(code))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrCodeInvalid
		}
		return nil, err
	}
	if err := s.repo.ConsumeBackupCode(ctx, bc.ID, time.Now()); err != nil {
		return nil, err
	}
	s.audit(ctx, &user.ID, "mfa.backup_used", "user", user.ID)
	return s.startSession(ctx, user)
}

// userFromMFAToken validates the login challenge and loads the corresponding user.
func (s *Service) userFromMFAToken(ctx context.Context, token string) (*User, error) {
	userID, err := parseChallenge(s.challengeKey, token, purposeMFALogin)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	return user, nil
}

// generateBackupCodes returns n plaintext codes plus their hashes for storage.
func generateBackupCodes(n int) (plain []string, hashes []string, err error) {
	plain = make([]string, 0, n)
	hashes = make([]string, 0, n)
	for i := 0; i < n; i++ {
		code, err := backupCode()
		if err != nil {
			return nil, nil, err
		}
		plain = append(plain, code)
		hashes = append(hashes, crypto.HashToken(code))
	}
	return plain, hashes, nil
}
