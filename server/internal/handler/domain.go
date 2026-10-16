package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shinsaber/colec/internal/domains"
)

// ListDomains answers GET /domains (admin): the domain packages built into the
// server and their state on the instance.
func (h *Handler) ListDomains(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListDomains(r.Context())
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, list)
}

// InstallDomain answers POST /domains/{id}/install (admin): installs or updates
// the package by merge (types changed on the instance keep their changes);
// dry_run=true reports without writing. After a real install, the other
// installed domains are merged again (relations whose ends were missing).
func (h *Handler) InstallDomain(w http.ResponseWriter, r *http.Request) {
	pkg, err := domains.Get(chi.URLParam(r, "id"))
	if err != nil {
		respondInternal(w, r, err, "cannot read domain packages")
		return
	}
	if pkg == nil {
		respondError(w, http.StatusNotFound, "not_found", "unknown domain")
		return
	}
	dryRun := r.URL.Query().Get("dry_run") == "true"
	rep, err := h.Store.InstallDomain(r.Context(), pkg, dryRun)
	if err != nil {
		handleStoreError(w, r, err)
		return
	}
	if rep.Applied {
		if err := h.Store.RefreshInstalledDomains(r.Context(), pkg.ID); err != nil {
			respondInternal(w, r, err, "cannot refresh installed domains")
			return
		}
	}
	status := http.StatusOK
	if len(rep.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	respondJSON(w, status, rep)
}
