package middleware

import (
	"net"
	"net/http"
)

// TrustedProxyRealIP replaces r.RemoteAddr with the client IP announced by a
// reverse proxy in X-Real-IP. It must only be enabled (TRUSTED_PROXY=true) when
// the API is reachable exclusively through a proxy that overwrites this header
// (web/nginx.conf does: `proxy_set_header X-Real-IP $remote_addr`). Enabled on
// a directly exposed port, it would let any client spoof its IP and bypass the
// per-IP rate limiter. chi's RealIP is deprecated for exactly that reason
// (GHSA-3fxj-6jh8-hvhx).
func TrustedProxyRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := r.Header.Get("X-Real-IP"); ip != "" && net.ParseIP(ip) != nil {
			// Keep the host:port shape expected by net.SplitHostPort callers.
			r.RemoteAddr = net.JoinHostPort(ip, "0")
		}
		next.ServeHTTP(w, r)
	})
}
