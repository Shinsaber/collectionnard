package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shinsaber/colec/internal/middleware"
)

func TestTrustedProxyRealIP(t *testing.T) {
	cases := []struct {
		name, header, remote, want string
	}{
		{"valid header", "203.0.113.9", "10.0.0.1:4321", "203.0.113.9:0"},
		{"ipv6 header", "2001:db8::1", "10.0.0.1:4321", "[2001:db8::1]:0"},
		{"missing header", "", "10.0.0.1:4321", "10.0.0.1:4321"},
		{"garbage header", "not-an-ip", "10.0.0.1:4321", "10.0.0.1:4321"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ""
			h := middleware.TrustedProxyRealIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.RemoteAddr
			}))
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remote
			if tc.header != "" {
				req.Header.Set("X-Real-IP", tc.header)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tc.want {
				t.Errorf("RemoteAddr = %q, want %q", got, tc.want)
			}
		})
	}
}
