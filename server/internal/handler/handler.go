package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/shinsaber/colec/config"
	"github.com/shinsaber/colec/internal/middleware"
	"github.com/shinsaber/colec/internal/model"
	"github.com/shinsaber/colec/internal/store"
	"github.com/shinsaber/colec/internal/validator"
)

type Handler struct {
	Store  store.Store
	Auth   *middleware.Auth
	Config *config.Config
}

func New(s store.Store, auth *middleware.Auth, cfg *config.Config) *Handler {
	return &Handler{Store: s, Auth: auth, Config: cfg}
}

// Router builds the chi router with all routes.
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()

	// Global middlewares (must be before routes)
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.RequestLogger)
	r.Use(chimiddleware.Recoverer)
	if h.Config.TrustedProxy {
		// Only behind nginx (TRUSTED_PROXY=true): otherwise X-Real-IP is client-controlled.
		r.Use(middleware.TrustedProxyRealIP)
	}

	// Public: liveness/readiness (compose healthcheck, monitoring)
	r.Get("/api/v1/healthz", h.Healthz)

	// Public, rate-limited per IP (credential stuffing, token guessing)
	authLimiter := middleware.NewRateLimiter(h.Config.LoginRateLimit, time.Minute)
	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Handler)
		r.Post("/api/v1/auth/login", h.Login)
		r.Post("/api/v1/auth/refresh", h.Refresh)
		r.Post("/api/v1/auth/logout", h.Logout)
	})

	// Serve uploaded files (public — img tags can't send Authorization headers)
	uploadsPrefix := "/api/v1/uploads/"
	uploadsDir := http.Dir(h.Config.UploadDir)
	r.Get(uploadsPrefix+"*", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, uploadsPrefix)
		// Prevent directory traversal
		if strings.Contains(name, "..") || strings.Contains(name, "/") {
			http.NotFound(w, r)
			return
		}
		// Only serve files with allowed extensions
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			// ok
		default:
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(string(uploadsDir), name)) //nolint:gosec // name sans "/" ni "..", extension en liste blanche (ci-dessus)
	})

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(h.Auth.Authenticate)

		r.Get("/api/v1/auth/me", h.GetCurrentUser)

		// Read endpoints (all authenticated users)
		r.Get("/api/v1/item-types", h.ListItemTypes)
		r.Get("/api/v1/item-types/{id}", h.GetItemType)
		r.Get("/api/v1/relation-types", h.ListRelationTypes)
		r.Get("/api/v1/relation-types/{id}", h.GetRelationType)
		r.Get("/api/v1/collection-types", h.ListCollectionTypes)
		r.Get("/api/v1/collection-types/{id}", h.GetCollectionType)
		r.Get("/api/v1/items", h.ListItems)
		r.Get("/api/v1/items/{id}", h.GetItem)
		r.Get("/api/v1/items/{id}/relations", h.ListItemRelations)
		r.Get("/api/v1/items/{id}/graph", h.GetItemGraph)
		r.Get("/api/v1/items/{id}/collections", h.ListItemCollections)
		r.Get("/api/v1/relations/{id}", h.GetRelation)
		r.Get("/api/v1/tags", h.ListTags)
		r.Get("/api/v1/stats", h.GetStats)

		// User collection endpoints
		r.Get("/api/v1/collections", h.ListCollections)
		r.Post("/api/v1/collections", h.CreateCollection)
		r.Get("/api/v1/collections/{id}", h.GetCollection)
		r.Put("/api/v1/collections/{id}", h.UpdateCollection)
		r.Delete("/api/v1/collections/{id}", h.DeleteCollection)
		r.Get("/api/v1/collections/{id}/items", h.ListCollectionItems)
		r.Post("/api/v1/collections/{id}/items", h.AddItemToCollection)
		r.Put("/api/v1/collections/{collectionId}/items/{itemId}", h.UpdateCollectionItem)
		r.Delete("/api/v1/collections/{collectionId}/items/{itemId}", h.RemoveItemFromCollection)

		// Admin-only endpoints
		r.Group(func(r chi.Router) {
			r.Use(h.Auth.RequireAdmin)

			r.Post("/api/v1/item-types", h.CreateItemType)
			r.Put("/api/v1/item-types/{id}", h.UpdateItemType)
			r.Delete("/api/v1/item-types/{id}", h.DeleteItemType)

			r.Post("/api/v1/relation-types", h.CreateRelationType)
			r.Put("/api/v1/relation-types/{id}", h.UpdateRelationType)
			r.Delete("/api/v1/relation-types/{id}", h.DeleteRelationType)

			r.Post("/api/v1/collection-types", h.CreateCollectionType)
			r.Put("/api/v1/collection-types/{id}", h.UpdateCollectionType)
			r.Delete("/api/v1/collection-types/{id}", h.DeleteCollectionType)

			r.Post("/api/v1/items", h.CreateItem)
			r.Put("/api/v1/items/{id}", h.UpdateItem)
			r.Delete("/api/v1/items/{id}", h.DeleteItem)

			r.Post("/api/v1/relations", h.CreateRelation)
			r.Put("/api/v1/relations/{id}", h.UpdateRelation)
			r.Delete("/api/v1/relations/{id}", h.DeleteRelation)

			r.Post("/api/v1/uploads", h.UploadFile)
		})
	})

	return r
}

// --- Helpers ---

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, model.APIError{Error: code, Message: message})
}

func respondValidationError(w http.ResponseWriter, message string, details any) {
	respondJSON(w, http.StatusUnprocessableEntity, model.APIError{
		Error:   "validation_error",
		Message: message,
		Details: details,
	})
}

// maxJSONBody caps the size of JSON request bodies (uploads have their own limit).
const maxJSONBody = 1 << 20 // 1 MiB

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}

// respondDecodeError maps a decodeJSON failure to 413 (body too large) or 400.
func respondDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respondError(w, http.StatusRequestEntityTooLarge, "payload_too_large",
			fmt.Sprintf("Corps de requête trop volumineux (max %d octets)", tooLarge.Limit))
		return
	}
	respondError(w, http.StatusBadRequest, "bad_request", "JSON invalide")
}

// hasAttributes reports whether a JSON `attributes` field was actually provided
// (absent and `null` are both treated as missing).
func hasAttributes(attrs json.RawMessage) bool {
	trimmed := bytes.TrimSpace(attrs)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}

// sortFields are the only sortable columns of the listing endpoints.
var sortFields = map[string]bool{"created_at": true, "updated_at": true}

// parseSort validates the `sort` query parameter (optional `-` prefix). An
// unknown field answers 400 and returns ok=false.
func parseSort(w http.ResponseWriter, r *http.Request) (string, bool) {
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		return "", true
	}
	if !sortFields[strings.TrimPrefix(sort, "-")] {
		respondError(w, http.StatusBadRequest, "bad_request",
			"sort invalide : valeurs acceptées created_at, updated_at (préfixe - pour descendant)")
		return "", false
	}
	return sort, true
}

// checkAttributesSchema rejects an invalid attributes_schema with a 422 and reports
// whether the caller may continue.
func checkAttributesSchema(w http.ResponseWriter, schema json.RawMessage) bool {
	if err := validator.CheckSchema(schema); err != nil {
		respondValidationError(w, "Le JSON Schema des attributs est invalide", map[string]any{
			"fields": map[string]string{"attributes_schema": err.Error()},
		})
		return false
	}
	return true
}

// respondMissingAttributes emits the 422 used when a PUT omits `attributes`.
func respondMissingAttributes(w http.ResponseWriter) {
	respondValidationError(w, "Le champ attributes est requis", map[string]any{
		"fields": map[string]string{"attributes": "required — ce champ est obligatoire"},
	})
}

func parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "UUID invalide: "+param)
		return uuid.Nil, false
	}
	return id, true
}

func parsePagination(r *http.Request) model.Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return model.Pagination{Page: page, PerPage: perPage}
}

const (
	defaultPerPage  = 20
	maxPerPage      = 100
	maxSearchLength = 200
)

// parseOptionalUUID reads an optional UUID query parameter. An absent parameter
// yields (nil, true); an invalid one answers 400 and yields (nil, false).
func parseOptionalUUID(w http.ResponseWriter, r *http.Request, param string) (*uuid.UUID, bool) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "UUID invalide: "+param)
		return nil, false
	}
	return &id, true
}

func containsUUID(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func handleStoreError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "Ressource non trouvée")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "conflict", "Conflit — ressource existante")
	default:
		respondInternal(w, r, err, "Erreur interne")
	}
}

// respondInternal logs the underlying error (with the request id) and answers 500
// with a generic message: the cause never reaches the client.
func respondInternal(w http.ResponseWriter, r *http.Request, err error, message string) {
	slog.Error("internal error",
		"err", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", chimiddleware.GetReqID(r.Context()),
	)
	respondError(w, http.StatusInternalServerError, "internal", message)
}
