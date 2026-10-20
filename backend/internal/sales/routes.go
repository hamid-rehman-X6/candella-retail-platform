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
		g.Get("/sales", h.Search)
		g.Get("/sales/{orderID}", h.Receipt)
		g.Get("/reports/daily", h.DailyReport)
		g.Get("/reports/daily.csv", h.DailyReportCSV)
		g.Post("/sales/{orderID}/returns", h.Return)
		g.Get("/returns/{returnID}", h.ReturnReceipt)
	})
}
