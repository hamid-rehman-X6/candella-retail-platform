package audit

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	appmiddleware "candella-ecosystem/backend/internal/middleware"
)

func TestMiddlewareRecordsResolvedClientIP(t *testing.T) {
	cases := []struct {
		name, trusted, peer, xff, want string
	}{
		{"spoofed header from a direct client", "", "203.0.113.7:51000", "1.2.3.4", "203.0.113.7"},
		{"client behind a trusted proxy", "10.0.0.0/8", "10.1.2.3:443", "198.51.100.20", "198.51.100.20"},
		{"IPv6 connection without port or brackets", "", "[2001:db8::7]:8080", "", "2001:db8::7"},
		{"IPv6 zone dropped for the INET column", "", "[fe80::1%eth0]:8080", "", "fe80::1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trusted, err := appmiddleware.ParseTrustedProxies(tc.trusted)
			if err != nil {
				t.Fatal(err)
			}
			var got RequestInfo
			h := appmiddleware.ClientIP(trusted)(Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = RequestFromContext(r.Context())
			})))
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = tc.peer
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got.IP != tc.want {
				t.Fatalf("audited IP = %q, want %q", got.IP, tc.want)
			}
			if _, err := netip.ParseAddr(got.IP); err != nil {
				t.Fatalf("audited IP %q would be rejected by the INET column", got.IP)
			}
		})
	}
}
