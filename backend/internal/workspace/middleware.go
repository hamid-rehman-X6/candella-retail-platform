package workspace

import (
	"net/http"

	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/internal/tenant"
)

// TenantHeader names the header carrying the workspace a request acts within.
// The tenant is explicit on every call rather than ambient in the session: with
// ambient state, two browser tabs on two workspaces race over one value and
// writes silently land in the wrong tenant.
const TenantHeader = "X-Tenant-Id"

// RequireTenant resolves and verifies the tenant for a request, then puts it in
// the context for downstream handlers and repositories.
//
// The header is a *claim*. It becomes trustworthy only after GetMembership
// confirms the authenticated user is an active member of that live tenant —
// which is also why a bad ID returns 404 rather than 403: answering "forbidden"
// would confirm the tenant exists to anyone enumerating IDs.
//
// Must be mounted after RequireAuth.
func (h *Handler) RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			h.writeErr(w, ErrUnauthenticated)
			return
		}

		tenantID := r.Header.Get(TenantHeader)
		if tenantID == "" {
			h.writeErr(w, ErrTenantRequired)
			return
		}

		m, err := h.svc.ResolveMembership(r.Context(), user.ID, tenantID)
		if err != nil {
			h.writeErr(w, err)
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID: m.TenantID,
			Slug:     m.Slug,
			Role:     m.Role,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
