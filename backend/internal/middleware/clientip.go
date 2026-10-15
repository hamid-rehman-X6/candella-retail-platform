package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// ParseTrustedProxies parses TRUSTED_PROXIES: a comma-separated list of the
// reverse proxies in front of this server, as IPs or CIDRs. A bare IP trusts
// only that address. An empty value means no proxy is trusted.
func ParseTrustedProxies(value string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, fmt.Errorf("TRUSTED_PROXIES: %q is not a valid CIDR", entry)
			}
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(entry)
		if err != nil || addr.Zone() != "" {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %q is not a valid IP address", entry)
		}
		addr = addr.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes, nil
}

// ClientIP resolves the client IP once per request; read it with ClientIPOf.
//
// Forwarding headers are only as honest as whoever set them, so they are read
// only when the TCP peer is one of the trusted proxies. Anyone else gets their
// connection address, whatever headers they send — otherwise a client could
// pick a fresh IP per request and escape the sign-in rate limit, or write any
// IP it likes into the audit log.
//
// Behind trusted proxies, X-Forwarded-For is walked right to left, skipping
// trusted hops; the first untrusted address is the client. If the chain holds
// nothing usable, the proxy's own address is kept rather than trusting an
// earlier, client-written entry.
func ClientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	if len(trusted) == 0 {
		return chimiddleware.ClientIPFromRemoteAddr
	}
	cidrs := make([]string, len(trusted))
	for i, prefix := range trusted {
		cidrs[i] = prefix.String()
	}
	fromXFF := chimiddleware.ClientIPFromXFF(cidrs...)

	return func(next http.Handler) http.Handler {
		viaProxy := fromXFF(next)
		return chimiddleware.ClientIPFromRemoteAddr(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer := chimiddleware.GetClientIPAddr(r.Context()).WithZone("")
			if peer.IsValid() && containsAddr(trusted, peer) {
				viaProxy.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

// ClientIPOf returns the client IP resolved by ClientIP. Without that
// middleware (unit tests, handlers called directly) it falls back to the
// connection address. The zero Addr means no valid IP was available.
func ClientIPOf(r *http.Request) netip.Addr {
	if ip := chimiddleware.GetClientIPAddr(r.Context()); ip.IsValid() {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, _ := netip.ParseAddr(host)
	return ip.Unmap()
}

func containsAddr(prefixes []netip.Prefix, ip netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}
