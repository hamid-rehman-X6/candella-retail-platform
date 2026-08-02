package auth

import (
	"net/http"
	"strings"
	"time"

	"candella-ecosystem/backend/pkg/config"
)

// sessionTokenBytes is the entropy of the opaque session token (256 bits).
const sessionTokenBytes = 32

// setSessionCookie writes the session cookie carrying the raw (unhashed) token.
// The cookie is HttpOnly (never readable from JS) and its Secure/SameSite flags
// come from config so local http dev and HTTPS prod both work.
func setSessionCookie(w http.ResponseWriter, cfg config.Config, rawToken string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   cfg.SessionCookieSecure,
		SameSite: sameSite(cfg.SessionCookieSameSite),
	})
}

// clearSessionCookie expires the session cookie on logout.
func clearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.SessionCookieSecure,
		SameSite: sameSite(cfg.SessionCookieSameSite),
	})
}

func sameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
