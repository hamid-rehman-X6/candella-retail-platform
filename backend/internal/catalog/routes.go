package catalog

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Routes(r chi.Router, h *Handler, requireAuth, requireTenant func(http.Handler) http.Handler) {
	r.Group(func(g chi.Router) {
		g.Use(requireAuth)
		g.Use(requireTenant)
		g.Get("/products", h.ListProducts)
		g.Get("/catalog/summary", h.Summary)
		g.Post("/products", h.CreateProduct)
		g.Patch("/products/{productID}", h.UpdateProduct)
		g.Post("/products/{productID}/variants", h.AddVariant)
		g.Patch("/products/variants/{variantID}", h.UpdateVariant)
		g.Get("/inventory", h.ListStock)
		g.Post("/inventory/movements", h.RecordMovement)
		g.Get("/inventory/movements", h.ListMovements)
	})
}
