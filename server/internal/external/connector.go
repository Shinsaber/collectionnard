package external

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

// Connector proposes catalogue items from external sources for one kind of
// collection (BD / manga, novels…). Every connector follows the same path: a
// work searched by title, its editions by title or ISBN / EAN, a preview
// matched against the catalogue, then the catalogue import of the selection.
type Connector interface {
	// ID is the connector path segment (/external/{id}/…).
	ID() string
	// Domain is the domain package holding the types the connector writes.
	Domain() string
	SearchWorks(ctx context.Context, text string) ([]model.ExternalWorkHit, error)
	// SearchEditions searches by title, normalised ISBN / EAN or Wikidata work (one of them).
	SearchEditions(ctx context.Context, q model.ExternalEditionQuery) ([]model.ExternalEditionHit, error)
	Preview(ctx context.Context, req model.ExternalImportRequest) (*model.ExternalPlan, error)
	// Build returns the selection in the catalogue import format, and the
	// catalogue id of each proposed node key.
	Build(ctx context.Context, req model.ExternalImportRequest) (*model.CatalogExport, map[string]uuid.UUID, error)
}

// UnavailableError: the catalogue lacks types the connector writes (domain
// package not installed, or a type created by hand under another name).
type UnavailableError struct {
	Domain  string
	Missing []string
}

func (e *UnavailableError) Error() string {
	return "types missing from the catalogue (domain " + e.Domain + "): " + strings.Join(e.Missing, ", ")
}

// InputError is an invalid request (unknown key, malformed identifier…).
type InputError string

func (e InputError) Error() string { return string(e) }
