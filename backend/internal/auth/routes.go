package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes mounts the auth endpoints under /auth. `limit` is the rate-limit
// middleware applied to the sensitive, unauthenticated endpoints; `h.RequireAuth`
// guards the endpoints that need a session.
//
// Endpoints are added per phase; this reflects Phase B (core credential auth).
func Routes(r chi.Router, h *Handler, limit func(http.Handler) http.Handler) {
	r.Route("/auth", func(a chi.Router) {
		// Rate-limited, unauthenticated endpoints.
		a.Group(func(p chi.Router) {
			p.Use(limit)
			p.Post("/register", h.Register)
			p.Post("/verify-email", h.VerifyEmail)
			p.Post("/resend-verification", h.ResendVerification)
			p.Post("/login", h.Login)
			p.Post("/login/verify-2fa", h.LoginVerifyTwoFactor)
			p.Post("/login/verify-backup", h.LoginVerifyBackup)
			p.Post("/forgot-password", h.ForgotPassword)
			p.Post("/reset-password", h.ResetPassword)
		})

		// Google OAuth is a browser redirect flow (GET), not a rate-limited API call.
		a.Get("/google/start", h.GoogleStart)
		a.Get("/google/callback", h.GoogleCallback)

		// Logout is intentionally unauthenticated + idempotent: it just clears the
		// cookie and revokes the session if one exists.
		a.Post("/logout", h.Logout)

		// Authenticated endpoints (valid session cookie required).
		a.Group(func(pr chi.Router) {
			pr.Use(h.RequireAuth)
			pr.Get("/me", h.Me)
			pr.Post("/2fa/setup", h.TwoFactorSetup)
			pr.Post("/2fa/enable", h.TwoFactorEnable)
			pr.Post("/2fa/disable", h.TwoFactorDisable)
		})
	})
}
