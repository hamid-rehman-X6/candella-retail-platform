package audit

import (
	"context"
	"net/http"

	appmiddleware "candella-ecosystem/backend/internal/middleware"
)

// RequestInfo is the transport detail an audit entry needs. Capturing it once in
// middleware means service code never has to be handed an *http.Request just to
// be able to log where an action came from.
type RequestInfo struct {
	IP        string
	UserAgent string
}

type reqCtxKey struct{}

// Middleware records the client IP and user agent for the rest of the request.
// Mount it after the ClientIP middleware, so the audited IP is the same one the
// rate limiter sees and forwarding headers are only honoured from trusted proxies.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := RequestInfo{IP: clientIP(r), UserAgent: r.UserAgent()}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), reqCtxKey{}, info)))
	})
}

// RequestFromContext returns the captured request info, or a zero value when the
// call didn't originate from an HTTP request (background jobs, CLI commands).
func RequestFromContext(ctx context.Context) RequestInfo {
	info, _ := ctx.Value(reqCtxKey{}).(RequestInfo)
	return info
}

// clientIP returns a bare address, or "" (stored as NULL), since
// audit_logs.ip_address is an INET column and rejects a port or an IPv6 zone.
func clientIP(r *http.Request) string {
	ip := appmiddleware.ClientIPOf(r).WithZone("")
	if !ip.IsValid() {
		return ""
	}
	return ip.String()
}
