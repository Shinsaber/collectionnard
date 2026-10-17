package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/shinsaber/colec/internal/external"
	"github.com/shinsaber/colec/internal/external/bnf"
	"github.com/shinsaber/colec/internal/model"
)

// externalTimeout bounds one request to the external sources (a preview may
// chain a dozen calls to Wikidata and the BnF).
const externalTimeout = 2 * time.Minute

// externalContext extends the write deadline of the response (server-wide 30 s)
// and bounds the calls to the sources.
func externalContext(w http.ResponseWriter, r *http.Request) (context.Context, context.CancelFunc) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(externalTimeout + 10*time.Second))
	return context.WithTimeout(r.Context(), externalTimeout)
}

// ListConnectors answers GET /external/connectors (admin): the connectors and the
// domain package each one writes into.
func (h *Handler) ListConnectors(w http.ResponseWriter, r *http.Request) {
	out := make([]model.ConnectorInfo, 0, len(h.Connectors))
	for _, c := range h.Connectors {
		out = append(out, model.ConnectorInfo{ID: c.ID(), Domain: c.Domain()})
	}
	respondJSON(w, http.StatusOK, out)
}

// connector returns the connector named by the {connector} path segment (404 if unknown).
func (h *Handler) connector(w http.ResponseWriter, r *http.Request) (external.Connector, bool) {
	id := chi.URLParam(r, "connector")
	for _, c := range h.Connectors {
		if c.ID() == id {
			return c, true
		}
	}
	respondError(w, http.StatusNotFound, "not_found", "unknown connector")
	return nil, false
}

// SearchExternalWorks answers GET /external/{connector}/works?q= (admin).
func (h *Handler) SearchExternalWorks(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.connector(w, r)
	if !ok {
		return
	}
	q, ok := searchText(w, r)
	if !ok {
		return
	}
	ctx, cancel := externalContext(w, r)
	defer cancel()
	hits, err := conn.SearchWorks(ctx, q)
	if err != nil {
		respondExternalError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, hits)
}

// SearchExternalEditions answers GET /external/{connector}/editions?q=|isbn= (admin).
func (h *Handler) SearchExternalEditions(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.connector(w, r)
	if !ok {
		return
	}
	rawISBN := strings.TrimSpace(r.URL.Query().Get("isbn"))
	hasQ := strings.TrimSpace(r.URL.Query().Get("q")) != ""
	if (rawISBN == "") == !hasQ {
		respondError(w, http.StatusBadRequest, "bad_request", "exactly one of q and isbn is required")
		return
	}
	var title, isbn string
	if rawISBN != "" {
		if isbn = bnf.NormalizeISBN(rawISBN); isbn == "" {
			respondError(w, http.StatusBadRequest, "bad_request", "invalid ISBN")
			return
		}
	} else {
		var ok bool
		if title, ok = searchText(w, r); !ok {
			return
		}
	}
	ctx, cancel := externalContext(w, r)
	defer cancel()
	hits, err := conn.SearchEditions(ctx, title, isbn)
	if err != nil {
		respondExternalError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, hits)
}

// PreviewExternal answers POST /external/{connector}/preview (admin): the items
// and relations the sources propose, matched against the catalogue.
func (h *Handler) PreviewExternal(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.connector(w, r)
	if !ok {
		return
	}
	var req model.ExternalImportRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	ctx, cancel := externalContext(w, r)
	defer cancel()
	plan, err := conn.Preview(ctx, req)
	if err != nil {
		respondExternalError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, plan)
}

// ImportExternal answers POST /external/{connector}/import (admin): the selected
// nodes go through the catalogue import (same report, 200 or 422, dry_run).
func (h *Handler) ImportExternal(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.connector(w, r)
	if !ok {
		return
	}
	dryRun, ok := parseDryRun(w, r)
	if !ok {
		return
	}
	var req model.ExternalImportRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondDecodeError(w, err)
		return
	}
	ctx, cancel := externalContext(w, r)
	defer cancel()
	cat, ids, err := conn.Build(ctx, req)
	if err != nil {
		respondExternalError(w, r, err)
		return
	}
	rep, err := h.Store.ImportCatalog(ctx, cat, dryRun)
	if err != nil {
		respondInternal(w, r, err, "internal error")
		return
	}
	status := http.StatusOK
	if len(rep.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	respondJSON(w, status, model.ExternalImportReport{ImportReport: *rep, ItemIDs: ids})
}

// searchText reads the `q` parameter of a search (2 to maxSearchLength characters).
func searchText(w http.ResponseWriter, r *http.Request) (string, bool) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if n := utf8.RuneCountInString(q); n < 2 || n > maxSearchLength {
		respondError(w, http.StatusBadRequest, "bad_request", "q must be 2 to 200 characters long")
		return "", false
	}
	return q, true
}

// respondExternalError maps the errors of a connector: 400 invalid request,
// 404 unknown external entity, 409 domain types missing (details.missing), 502 source unreachable.
func respondExternalError(w http.ResponseWriter, r *http.Request, err error) {
	var input external.InputError
	var unavailable *external.UnavailableError
	switch {
	case errors.As(err, &input):
		respondError(w, http.StatusBadRequest, "bad_request", input.Error())
	case errors.As(err, &unavailable):
		respondJSON(w, http.StatusConflict, model.APIError{
			Error:   "connector_unavailable",
			Message: unavailable.Error(),
			Details: map[string]any{"missing": unavailable.Missing, "domain": unavailable.Domain},
		})
	case errors.Is(err, external.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, external.ErrUnavailable), errors.Is(err, context.DeadlineExceeded):
		respondError(w, http.StatusBadGateway, "external_unavailable", "external source unavailable: "+err.Error())
	default:
		handleStoreError(w, r, err)
	}
}
