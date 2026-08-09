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

// googleResult is the outcome of a completed Google sign-in. IsNewUser lets the
// caller tell a first-ever sign-up apart from a return visit, so the frontend
// can greet the two differently.
type googleResult struct {
	Login     *loginResult
	IsNewUser bool
}

// CompleteGoogleLogin validates the callback, resolves (or creates+links) the
// Candella user for the Google identity, and starts a session — unless the
// account has 2FA enabled, in which case it returns an mfaToken instead.
func (s *Service) CompleteGoogleLogin(ctx context.Context, code, state string) (*googleResult, error) {
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
	// Google's own verification is the only thing that makes this email safe to
	// use as an account identifier — an unverified one may belong to someone else.
	if profile.Email == "" || !profile.VerifiedEmail {
		return nil, ErrGoogleEmailUnverified
	}

	// 1. Already linked to a Google account → log that user in.
	if acc, err := s.repo.GetOAuthAccount(ctx, providerGoogle, profile.Sub); err == nil {
		user, err := s.repo.GetUserByID(ctx, acc.UserID)
		if err != nil {
			return nil, err
		}
		return s.finishGoogleLogin(ctx, user, false)
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// 2. A local account with the same email exists → link Google to it, but only
	//    if that account has itself proven ownership of the address. Linking into
	//    an unverified account would let an attacker pre-register someone's email
	//    and silently inherit them when they later sign in with Google
	//    (account pre-hijacking).
	if user, err := s.repo.GetUserByEmail(ctx, profile.Email); err == nil {
		if !user.IsEmailVerified() {
			return nil, ErrGoogleLinkBlocked
		}
		if err := s.linkGoogle(ctx, user.ID, profile); err != nil {
			return nil, err
		}
		s.audit(ctx, &user.ID, "user.linked_google", "user", user.ID)
		return s.finishGoogleLogin(ctx, user, false)
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
	return s.finishGoogleLogin(ctx, user, true)
}

// finishGoogleLogin starts the session — unless the account has 2FA enabled.
// Google proves only the first factor, so a user who set up an authenticator
// still owes us a code, exactly as password login requires. (A brand-new
// account can't have 2FA yet, so the sign-up path always lands in a session.)
func (s *Service) finishGoogleLogin(ctx context.Context, user *User, isNew bool) (*googleResult, error) {
	if user.MFAEnabled {
		token := signChallenge(s.challengeKey, user.ID, purposeMFALogin, 5*time.Minute)
		return &googleResult{
			Login:     &loginResult{MFARequired: true, MFAToken: token},
			IsNewUser: isNew,
		}, nil
	}

	res, err := s.startSession(ctx, user)
	if err != nil {
		return nil, err
	}
	return &googleResult{Login: res, IsNewUser: isNew}, nil
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
