package platform

import (
	"errors"
	"net/http"

	"candella-ecosystem/backend/internal/auth"
)

// RequirePlatformAdmin gates every console route.
//
// Two deliberate choices:
//
//   - A caller who isn't staff gets 404, not 403. A 403 confirms the console
//     exists and that these routes are worth attacking; 404 says nothing.
//   - 2FA is mandatory. Cross-tenant read access is the most valuable credential
//     in the system, so a password alone must never be enough to hold it. A real
//     admin without a second factor gets a specific, actionable error instead of
//     a silent 404, because they need to know what to fix.
//
// Must be mounted after RequireAuth.
func (h *Handler) RequirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			h.writeErr(w, ErrNotFound)
			return
		}

		admin, err := h.svc.ResolveAdmin(r.Context(), user.ID)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				h.log.Error("platform admin lookup failed", "error", err)
			}
			h.writeErr(w, ErrNotFound)
			return
		}

		if !user.MFAEnabled {
			h.writeErr(w, ErrMFARequired)
			return
		}

		// last_access_at is telemetry, not audit. Auditing every request would
		// bury the entries that matter (opening a customer's record, mutations)
		// under a row per page load — handlers record those individually.
		h.svc.TouchAccess(r.Context(), admin.ID)

		next.ServeHTTP(w, r.WithContext(WithAdmin(r.Context(), admin)))
	})
}
