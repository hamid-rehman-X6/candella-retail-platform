package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// resolve runs a request from peer with the given headers through ClientIP and
// returns the IP the rest of the stack would see.
func resolve(t *testing.T, trusted string, peer string, headers map[string][]string) string {
	t.Helper()
	prefixes, err := ParseTrustedProxies(trusted)
	if err != nil {
		t.Fatalf("ParseTrustedProxies(%q): %v", trusted, err)
	}
	var got string
	h := ClientIP(prefixes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = ClientIPOf(r).String()
	}))
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = peer
	for name, values := range headers {
		for _, v := range values {
			req.Header.Add(name, v)
		}
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestClientIP(t *testing.T) {
	spoofed := map[string][]string{
		"X-Forwarded-For": {"1.2.3.4"},
		"X-Real-Ip":       {"5.6.7.8"},
		"True-Client-Ip":  {"9.9.9.9"},
	}
	cases := []struct {
		name    string
		trusted string
		peer    string
		headers map[string][]string
		want    string
	}{
		{"no proxy: headers ignored", "", "203.0.113.7:51000", spoofed, "203.0.113.7"},
		{"untrusted peer: headers ignored", "10.0.0.0/8", "203.0.113.7:51000", spoofed, "203.0.113.7"},
		{"dual-stack peer folds to IPv4", "", "[::ffff:203.0.113.7]:51000", nil, "203.0.113.7"},
		{"trusted proxy: client from XFF", "10.0.0.0/8", "10.1.2.3:443",
			map[string][]string{"X-Forwarded-For": {"198.51.100.20"}}, "198.51.100.20"},
		{"trusted proxy: client-written prefix skipped", "10.0.0.0/8", "10.1.2.3:443",
			map[string][]string{"X-Forwarded-For": {"1.2.3.4, 198.51.100.20"}}, "198.51.100.20"},
		{"two trusted hops skipped", "10.0.0.0/8", "10.1.2.3:443",
			map[string][]string{"X-Forwarded-For": {"198.51.100.20, 10.9.9.9"}}, "198.51.100.20"},
		{"split XFF headers merged in order", "10.0.0.0/8", "10.1.2.3:443",
			map[string][]string{"X-Forwarded-For": {"1.2.3.4", "198.51.100.20"}}, "198.51.100.20"},
		{"trusted proxy: X-Real-IP not used", "10.0.0.0/8", "10.1.2.3:443",
			map[string][]string{"X-Real-Ip": {"5.6.7.8"}}, "10.1.2.3"},
		{"trusted proxy without XFF keeps proxy IP", "10.0.0.0/8", "10.1.2.3:443", nil, "10.1.2.3"},
		{"garbage nearest the proxy fails closed", "10.0.0.0/8", "10.1.2.3:443",
			map[string][]string{"X-Forwarded-For": {"1.2.3.4, not-an-ip"}}, "10.1.2.3"},
		{"bare trusted IP", "10.1.2.3", "10.1.2.3:443",
			map[string][]string{"X-Forwarded-For": {"198.51.100.20"}}, "198.51.100.20"},
		{"bare trusted IP does not trust its neighbour", "10.1.2.3", "10.1.2.4:443",
			map[string][]string{"X-Forwarded-For": {"198.51.100.20"}}, "10.1.2.4"},
		{"IPv6 proxy", "fd00::/8", "[fd00::1]:443",
			map[string][]string{"X-Forwarded-For": {"2001:db8::42"}}, "2001:db8::42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(t, tc.trusted, tc.peer, tc.headers); got != tc.want {
				t.Fatalf("client IP = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	got, err := ParseTrustedProxies(" 10.0.0.0/8, 192.168.1.10 ,,fd00::/8, ::ffff:172.16.0.1, 10.1.2.99/24")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/8", "192.168.1.10/32", "fd00::/8", "172.16.0.1/32", "10.1.2.0/24"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != netip.MustParsePrefix(want[i]) {
			t.Fatalf("entry %d = %v, want %s", i, got[i], want[i])
		}
	}

	if got, err := ParseTrustedProxies(""); err != nil || len(got) != 0 {
		t.Fatalf("empty value = %v, %v; want no proxies", got, err)
	}

	for _, bad := range []string{"10.0.0.0/33", "localhost", "10.0.0", "fe80::1%eth0", "10.0.0.1, nope"} {
		if _, err := ParseTrustedProxies(bad); err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
			t.Errorf("ParseTrustedProxies(%q) err = %v, want a TRUSTED_PROXIES error", bad, err)
		}
	}
}

func TestClientIPOfWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8::7]:8080"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := ClientIPOf(req).String(); got != "2001:db8::7" {
		t.Fatalf("ClientIPOf = %q, want the connection address", got)
	}
}

// limited sends n requests through ClientIP and a 2-per-minute limiter and
// returns how many were refused.
func limited(t *testing.T, trusted string, requests []struct{ peer, xff string }) int {
	t.Helper()
	prefixes, err := ParseTrustedProxies(trusted)
	if err != nil {
		t.Fatal(err)
	}
	h := ClientIP(prefixes)(NewRateLimiter(2, time.Minute).Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	refused := 0
	for _, request := range requests {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = request.peer
		if request.xff != "" {
			req.Header.Set("X-Forwarded-For", request.xff)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			refused++
		}
	}
	return refused
}

func TestRateLimitCannotBeEscaped(t *testing.T) {
	t.Run("rotating a spoofed X-Forwarded-For", func(t *testing.T) {
		got := limited(t, "", []struct{ peer, xff string }{
			{"203.0.113.7:1", "1.1.1.1"}, {"203.0.113.7:2", "2.2.2.2"},
			{"203.0.113.7:3", "3.3.3.3"}, {"203.0.113.7:4", "4.4.4.4"},
		})
		if got != 2 {
			t.Fatalf("refused %d of 4, want 2: a new XFF value must not reset the limit", got)
		}
	})

	t.Run("rotating addresses inside one IPv6 /64", func(t *testing.T) {
		got := limited(t, "", []struct{ peer, xff string }{
			{"[2001:db8:1:2::1]:1", ""}, {"[2001:db8:1:2::2]:1", ""},
			{"[2001:db8:1:2:ffff::3]:1", ""},
		})
		if got != 1 {
			t.Fatalf("refused %d of 3, want 1", got)
		}
	})

	t.Run("different clients behind a trusted proxy keep their own limits", func(t *testing.T) {
		got := limited(t, "10.0.0.0/8", []struct{ peer, xff string }{
			{"10.1.2.3:1", "198.51.100.1"}, {"10.1.2.3:2", "198.51.100.1"},
			{"10.1.2.3:3", "198.51.100.2"}, {"10.1.2.3:4", "198.51.100.2"},
		})
		if got != 0 {
			t.Fatalf("refused %d, want 0", got)
		}
	})
}
