package platform

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes mounts the platform console endpoints under their own namespace.
//
// Keeping them on /platform/* rather than sprinkling admin capabilities through
// the tenant API is the point: nothing here can be reached by a customer request
// that merely forgot a scope check, and the whole surface can be firewalled,
// rate-limited or moved to its own host as one unit.
func Routes(r chi.Router, h *Handler, requireAuth func(http.Handler) http.Handler) {
	r.Route("/platform", func(p chi.Router) {
		p.Use(requireAuth)
		p.Use(h.RequirePlatformAdmin)

		p.Get("/me", h.Me)
		p.Get("/overview", h.Overview)
		p.Get("/tenants", h.ListTenants)
		p.Get("/tenants/{tenantID}", h.GetTenant)
	})
}
