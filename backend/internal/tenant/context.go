// Package tenant carries the resolved tenant for a request.
//
// It deliberately has no dependencies: every feature slice needs to read the
// current tenant, but only the workspace slice (which owns memberships) is
// allowed to establish it. Keeping the context key here lets products, orders,
// inventory and the rest read it without importing the workspace package.
package tenant

import "context"

// Context is the tenant the current request is acting within, together with the
// authenticated user's role inside it. It is only ever created by the workspace
// middleware, after a membership has been verified — a tenant ID supplied by a
// client is a claim, never an authority.
type Context struct {
	TenantID string
	Slug     string
	Role     string // owner | admin | member
}

type ctxKey struct{}

// WithContext returns a copy of ctx carrying the resolved tenant.
func WithContext(ctx context.Context, t *Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

// FromContext returns the resolved tenant, if the request went through the
// tenant middleware.
func FromContext(ctx context.Context) (*Context, bool) {
	t, ok := ctx.Value(ctxKey{}).(*Context)
	return t, ok
}

// IDFromContext returns just the tenant ID, or "" for requests that aren't
// tenant-scoped (auth endpoints, the platform console).
func IDFromContext(ctx context.Context) string {
	if t, ok := FromContext(ctx); ok {
		return t.TenantID
	}
	return ""
}
