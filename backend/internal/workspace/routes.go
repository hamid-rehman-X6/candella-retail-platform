package workspace

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes mounts the workspace endpoints. All require a valid session, so the
// auth middleware (passed in as requireAuth) guards the whole group.
func Routes(r chi.Router, h *Handler, requireAuth func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Post("/workspaces", h.Create)
		g.Get("/workspaces", h.List)
	})
}
