package auth

import (
	"net/http"

	"candella-ecosystem/backend/pkg/response"
)

// POST /auth/2fa/setup — begins enrolment, returns secret + otpauth URL (authed).
func (h *Handler) TwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	secret, url, err := h.svc.SetupTwoFactor(r.Context(), user)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"secret": secret, "otpauthUrl": url})
}

// POST /auth/2fa/enable — confirms a code, returns backup codes once (authed).
func (h *Handler) TwoFactorEnable(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	var req codeRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	codes, err := h.svc.EnableTwoFactor(r.Context(), user, req.Code)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"backupCodes": codes})
}

// POST /auth/2fa/disable — turns 2FA off after re-verifying the password (authed).
func (h *Handler) TwoFactorDisable(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		h.writeErr(w, ErrUnauthenticated)
		return
	}
	var req passwordRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	if err := h.svc.DisableTwoFactor(r.Context(), user, req.Password); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"disabled": true})
}

// POST /auth/login/verify-2fa — second login step with a TOTP code.
func (h *Handler) LoginVerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req mfaCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	res, err := h.svc.LoginVerifyTwoFactor(r.Context(), req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	setSessionCookie(w, h.cfg, res.RawToken, res.Expires)
	response.JSON(w, http.StatusOK, map[string]any{"ok": true, "user": toUserResponse(res.User)})
}

// POST /auth/login/verify-backup — second login step with a backup code.
func (h *Handler) LoginVerifyBackup(w http.ResponseWriter, r *http.Request) {
	var req mfaBackupRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeErr(w, err)
		return
	}
	res, err := h.svc.LoginVerifyBackup(r.Context(), req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	setSessionCookie(w, h.cfg, res.RawToken, res.Expires)
	response.JSON(w, http.StatusOK, map[string]any{"ok": true, "user": toUserResponse(res.User)})
}
