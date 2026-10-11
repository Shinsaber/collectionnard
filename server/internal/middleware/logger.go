package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger logs one structured line per request (method, path, status,
// size, duration, client IP, request id). It expects chi's RequestID
// middleware to run before it.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		status := ww.Status()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		slog.Log(r.Context(), level, "http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", ww.BytesWritten(),
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
			"ip", clientIP(r),
			"request_id", chimiddleware.GetReqID(r.Context()),
		)
	})
}
