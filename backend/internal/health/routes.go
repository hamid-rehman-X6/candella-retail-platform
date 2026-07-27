package health

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/health", Check)
}
