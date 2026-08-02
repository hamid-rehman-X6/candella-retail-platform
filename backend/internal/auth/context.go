package auth

import "context"

type ctxKey int

const userCtxKey ctxKey = iota

// WithUser returns a copy of ctx carrying the authenticated user. The auth
// middleware calls this after validating the session cookie.
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// UserFromContext returns the authenticated user, if the request passed through
// the auth middleware.
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userCtxKey).(*User)
	return u, ok
}
