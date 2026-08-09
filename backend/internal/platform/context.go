package platform

import "context"

type adminCtxKey struct{}

// WithAdmin returns a copy of ctx carrying the authenticated platform admin.
func WithAdmin(ctx context.Context, a *Admin) context.Context {
	return context.WithValue(ctx, adminCtxKey{}, a)
}

// AdminFromContext returns the platform admin placed by RequirePlatformAdmin.
func AdminFromContext(ctx context.Context) (*Admin, bool) {
	a, ok := ctx.Value(adminCtxKey{}).(*Admin)
	return a, ok
}
