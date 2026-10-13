package sales

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Post("/sales", h.Checkout)
		g.Get("/sales/{orderID}", h.Receipt)
	})
}
