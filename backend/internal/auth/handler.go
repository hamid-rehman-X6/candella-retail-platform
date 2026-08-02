package auth

import (
	"errors"
	"net/http"

	"candella-ecosystem/backend/pkg/config"
	"candella-ecosystem/backend/pkg/response"
)

// Handler adapts HTTP requests to the auth Service. It is deliberately thin:
// decode → call service → write the JSON envelope (and set/clear cookies).
type Handler struct {
	svc *Service
	cfg config.Config
}

// NewHandler builds the auth HTTP handler.
func NewHandler(svc *Service, cfg config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

// writeErr maps a domain *Error to the error envelope; anything else is a 500.
func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	var de *Error
	if errors.As(err, &de) {
		response.Error(w, de.Status, de.Code, de.Message)
		return
	}
	response.Error(w, http.StatusInternalServerError, "internal_error", "Something went wrong. Please try again.")
}

// rawSessionToken reads the raw session token from the cookie (empty if absent).
func (h *Handler) rawSessionToken(r *http.Request) string {
	c, err := r.Cookie(h.cfg.SessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// POST /auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"userId": user.ID, "email": user.Email})
}

// POST /auth/verify-email
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req emailCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	res, err := h.svc.VerifyEmail(r.Context(), req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	setSessionCookie(w, h.cfg, res.RawToken, res.Expires)
	response.JSON(w, http.StatusOK, map[string]any{"verified": true, "user": toUserResponse(res.User)})
}

// POST /auth/resend-verification — always a generic 200 (no account enumeration).
func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req emailOnlyRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.svc.ResendVerification(r.Context(), req.Email); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"sent": true})
}

// POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if res.MFARequired {
		response.JSON(w, http.StatusOK, map[string]any{"mfaRequired": true, "mfaToken": res.MFAToken})
		return
	}
	setSessionCookie(w, h.cfg, res.RawToken, res.Expires)
	response.JSON(w, http.StatusOK, map[string]any{"mfaRequired": false, "user": toUserResponse(res.User)})
}

// POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.svc.Logout(r.Context(), h.rawSessionToken(r))
	clearSessionCookie(w, h.cfg)
	response.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /auth/me — requires the auth middleware.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"user": toUserResponse(user)})
}
