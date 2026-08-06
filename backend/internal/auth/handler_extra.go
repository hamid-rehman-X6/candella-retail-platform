package auth

import (
	"net/http"

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
	url, err := h.svc.GoogleAuthURL()
	if err != nil {
		// Not configured: send the user back to login with a clear reason.
		http.Redirect(w, r, h.cfg.AppBaseURL+"/login?error=google_not_configured", http.StatusFound)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// GET /auth/google/callback — completes OAuth, sets the session, returns to the app.
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	res, err := h.svc.CompleteGoogleLogin(r.Context(), code, state)
	if err != nil {
		http.Redirect(w, r, h.cfg.AppBaseURL+"/login?error=google_failed", http.StatusFound)
		return
	}
	setSessionCookie(w, h.cfg, res.RawToken, res.Expires)
	http.Redirect(w, r, h.cfg.AppBaseURL+"/dashboard", http.StatusFound)
}
