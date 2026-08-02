package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"candella-ecosystem/backend/pkg/config"
)

// googleOAuth wraps the Google OAuth2 config. When credentials aren't configured
// it is present but disabled, so the endpoints return a clear "not configured"
// error instead of failing obscurely.
type googleOAuth struct {
	cfg     *oauth2.Config
	enabled bool
	client  *http.Client
}

func newGoogleOAuth(c config.Config) *googleOAuth {
	if !c.GoogleEnabled() {
		return &googleOAuth{enabled: false}
	}
	return &googleOAuth{
		enabled: true,
		client:  &http.Client{Timeout: 10 * time.Second},
		cfg: &oauth2.Config{
			ClientID:     c.GoogleClientID,
			ClientSecret: c.GoogleClientSecret,
			RedirectURL:  c.GoogleRedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

// googleUser is the subset of Google's userinfo response we use.
type googleUser struct {
	Sub           string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
}

func (g *googleOAuth) authCodeURL(state string) string {
	return g.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// exchange trades the authorization code for the user's Google profile.
func (g *googleOAuth) exchange(ctx context.Context, code string) (*googleUser, error) {
	token, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google: exchange: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google: userinfo status %d", resp.StatusCode)
	}

	var u googleUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("google: decode userinfo: %w", err)
	}
	return &u, nil
}
