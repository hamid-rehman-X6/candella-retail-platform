package auth

import "net/http"

// RequireAuth is middleware that rejects requests without a valid session cookie
// and, on success, injects the authenticated user into the request context.
//
// It lives in the auth package (rather than internal/middleware) because it needs
// the auth Service; keeping it here avoids an auth ↔ middleware import cycle.
func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.svc.Authenticate(r.Context(), h.rawSessionToken(r))
		if err != nil {
			h.writeErr(w, ErrUnauthenticated)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}
