package auth

import (
	"context"
	"errors"
	"time"
)

const (
	providerGoogle    = "google"
	purposeOAuthState = "oauth_state"
)

// GoogleAuthURL returns the URL to redirect the user to, plus a signed state value
// (returned embedded in the URL). Errors with ErrGoogleDisabled when unconfigured.
func (s *Service) GoogleAuthURL() (string, error) {
	if s.google == nil || !s.google.enabled {
		return "", ErrGoogleDisabled
	}
	// A signed, expiring state parameter guards against CSRF on the callback.
	state := signChallenge(s.challengeKey, providerGoogle, purposeOAuthState, 10*time.Minute)
	return s.google.authCodeURL(state), nil
}

// CompleteGoogleLogin validates the callback, resolves (or creates+links) the
// Candella user for the Google identity, and starts a session.
func (s *Service) CompleteGoogleLogin(ctx context.Context, code, state string) (*loginResult, error) {
	if s.google == nil || !s.google.enabled {
		return nil, ErrGoogleDisabled
	}
	if _, err := parseChallenge(s.challengeKey, state, purposeOAuthState); err != nil {
		return nil, ErrTokenInvalid
	}

	profile, err := s.google.exchange(ctx, code)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// 1. Already linked to a Google account → log that user in.
	if acc, err := s.repo.GetOAuthAccount(ctx, providerGoogle, profile.Sub); err == nil {
		user, err := s.repo.GetUserByID(ctx, acc.UserID)
		if err != nil {
			return nil, err
		}
		return s.startSession(ctx, user)
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// 2. A local account with the same email exists → link Google to it.
	if user, err := s.repo.GetUserByEmail(ctx, profile.Email); err == nil {
		if err := s.linkGoogle(ctx, user.ID, profile); err != nil {
			return nil, err
		}
		return s.startSession(ctx, user)
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// 3. Brand-new user — Google has already verified the email, so mark verified.
	now := time.Now()
	user := &User{
		Email:           profile.Email,
		PasswordHash:    nil, // OAuth-only account
		FullName:        fallbackName(profile.Name, profile.Email),
		EmailVerifiedAt: &now,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	if err := s.linkGoogle(ctx, user.ID, profile); err != nil {
		return nil, err
	}
	s.audit(ctx, &user.ID, "user.registered_google", "user", user.ID)
	return s.startSession(ctx, user)
}

func (s *Service) linkGoogle(ctx context.Context, userID string, profile *googleUser) error {
	email := profile.Email
	return s.repo.CreateOAuthAccount(ctx, &OAuthAccount{
		UserID:            userID,
		Provider:          providerGoogle,
		ProviderAccountID: profile.Sub,
		Email:             &email,
	})
}

// fallbackName uses the provider's display name, or the email's local part.
func fallbackName(name, email string) string {
	if name != "" {
		return name
	}
	for i, r := range email {
		if r == '@' {
			return email[:i]
		}
	}
	return email
}
