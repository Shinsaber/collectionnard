package handler

import (
	"log/slog"
	"net/http"
)

// Healthz reports whether the API can reach its database. Public, unauthenticated:
// used by the docker-compose healthcheck and by external monitoring.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.Ping(r.Context()); err != nil {
		slog.Error("healthz: database unreachable", "err", err)
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "degraded",
			"error":  "database",
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
