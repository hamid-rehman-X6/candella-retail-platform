package auth

import (
	"errors"
	"net/http"
	"net/url"

	"candella-ecosystem/backend/pkg/response"
)

// POST /auth/forgot-password — always a generic 200 (no account enumeration).
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"sent": true})
}

// POST /auth/reset-password — sets a new password from a reset token.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"reset": true})
}

// GET /auth/google/start — redirects the browser to Google's consent screen.
func (h *Handler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	authURL, err := h.svc.GoogleAuthURL()
	if err != nil {
		h.redirectToBridge(w, r, url.Values{
			"status": {"error"},
			"reason": {errorCode(err)},
		})
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// GET /auth/google/callback — completes OAuth and hands the outcome to the
// frontend's /auth/callback bridge page.
//
// Deliberately, the backend does not pick a destination here: it reports *what
// happened* and lets the frontend decide what to say and where to go. That's
// what gives a redirect-based flow somewhere to show success and failure.
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Google reports user-side outcomes in `error` — most importantly
	// "access_denied", which means the person pressed Cancel on the consent
	// screen. That's a choice, not a fault, and must not be shown as a failure.
	if providerErr := q.Get("error"); providerErr != "" {
		if providerErr == "access_denied" {
			h.redirectToBridge(w, r, url.Values{"status": {"cancelled"}})
		} else {
			h.redirectToBridge(w, r, url.Values{"status": {"error"}, "reason": {"google_failed"}})
		}
		return
	}

	res, err := h.svc.CompleteGoogleLogin(r.Context(), q.Get("code"), q.Get("state"))
	if err != nil {
		h.redirectToBridge(w, r, url.Values{"status": {"error"}, "reason": {errorCode(err)}})
		return
	}

	// 2FA is enabled: Google covered the first factor only. Hand the short-lived
	// challenge to the bridge page, which moves it into sessionStorage and
	// replaces the URL so it doesn't linger in the address bar.
	if res.Login.MFARequired {
		h.redirectToBridge(w, r, url.Values{
			"status":   {"mfa"},
			"mfaToken": {res.Login.MFAToken},
		})
		return
	}

	setSessionCookie(w, h.cfg, res.Login.RawToken, res.Login.Expires)
	event := "signin"
	if res.IsNewUser {
		event = "signup"
	}
	h.redirectToBridge(w, r, url.Values{"status": {"ok"}, "event": {event}})
}

// redirectToBridge sends the browser to the frontend's /auth/callback page.
// Apart from the 2FA challenge, these params carry no credentials — they only
// tell the page what to say. The page re-confirms the session against /auth/me
// rather than trusting status=ok, so a crafted link can't fake a signed-in state.
func (h *Handler) redirectToBridge(w http.ResponseWriter, r *http.Request, params url.Values) {
	http.Redirect(w, r, h.cfg.AppBaseURL+"/auth/callback?"+params.Encode(), http.StatusFound)
}

// errorCode extracts the stable machine-readable code from a domain error,
// falling back to a generic one for unexpected failures.
func errorCode(err error) string {
	var de *Error
	if errors.As(err, &de) {
		return de.Code
	}
	return "google_failed"
}
