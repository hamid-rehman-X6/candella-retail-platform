package register

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/registers", h.ListRegisters)
		g.Post("/registers", h.CreateRegister)
		g.Get("/shifts", h.List)
		g.Post("/shifts", h.Open)
		g.Get("/shifts/{shiftID}", h.Get)
		g.Post("/shifts/{shiftID}/close", h.Close)
	})
}
