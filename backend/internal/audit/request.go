package audit

import (
	"context"
	"net/http"
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
// Mount it early — after chi's RealIP, so IP reflects any trusted proxy headers.
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

// clientIP strips the port that RemoteAddr carries, since audit_logs.ip_address
// is an INET column and would reject "127.0.0.1:54321".
func clientIP(r *http.Request) string {
	addr := r.RemoteAddr
	// Walk back to the last colon: IPv6 addresses contain several.
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return trimBrackets(addr[:i])
		}
	}
	return trimBrackets(addr)
}

// trimBrackets removes the [] that wrap an IPv6 host in "[::1]:8080".
func trimBrackets(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}
	return host
}
