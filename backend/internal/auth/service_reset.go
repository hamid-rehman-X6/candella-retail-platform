package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"candella-ecosystem/backend/pkg/crypto"
	"candella-ecosystem/backend/pkg/mailer"
	"candella-ecosystem/backend/pkg/validate"
)

const resetTokenBytes = 32

// ForgotPassword emails a reset link if the account exists. It always succeeds
// from the caller's perspective (no account enumeration) — the handler returns a
// generic 200 regardless.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}

	raw, err := randomToken(resetTokenBytes)
	if err != nil {
		return err
	}
	prt := &PasswordResetToken{
		UserID:    user.ID,
		TokenHash: crypto.HashToken(raw),
		ExpiresAt: time.Now().Add(s.cfg.ResetTokenTTL),
	}
	if err := s.repo.CreatePasswordResetToken(ctx, prt); err != nil {
		return err
	}

	resetURL := strings.TrimRight(s.cfg.AppBaseURL, "/") + "/reset-password?token=" + raw
	message, err := mailer.PasswordResetEmail(user.Email, user.FullName, resetURL, s.cfg.ResetTokenTTL)
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, message)
}

// ResetPassword sets a new password from a valid reset token, then revokes all of
// the user's sessions (so a leaked session can't survive a reset).
func (s *Service) ResetPassword(ctx context.Context, req resetPasswordRequest) error {
	if !validate.PasswordStrong(req.Password) {
		return ErrWeakPassword
	}

	prt, err := s.repo.GetPasswordResetByTokenHash(ctx, crypto.HashToken(req.Token))
	if err != nil {
		return ErrTokenInvalid
	}
	if time.Now().After(prt.ExpiresAt) {
		return ErrTokenInvalid
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, prt.UserID, hash); err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordResetToken(ctx, prt.ID, time.Now()); err != nil {
		return err
	}
	if err := s.repo.RevokeAllUserSessions(ctx, prt.UserID, time.Now()); err != nil {
		return err
	}
	s.audit(ctx, &prt.UserID, "password.reset", "user", prt.UserID)
	return nil
}
