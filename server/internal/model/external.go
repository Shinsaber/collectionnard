package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// ExternalWorkHit is a work found by a connector (GET /external/{connector}/works):
// a Wikidata work (books) or a MusicBrainz release group (music).
type ExternalWorkHit struct {
	ID             string     `json:"id"` // identifier at the source: the `work` of a selection
	WikidataID     string     `json:"wikidata_id,omitempty"`
	Label          string     `json:"label"`
	Description    string     `json:"description,omitempty"`
	ItemTypeID     uuid.UUID  `json:"item_type_id"`
	StartYear      *int       `json:"start_year,omitempty"`
	ExistingItemID *uuid.UUID `json:"existing_item_id,omitempty"`
	SourceURL      string     `json:"source_url"`
}

// ExternalEditionHit is an edition found by a connector (GET /external/{connector}/editions):
// a BnF record (multi-volume set, or for a single-volume novel the book itself)
// or a MusicBrainz release.
type ExternalEditionHit struct {
	ID         string   `json:"id"` // identifier at the source: one of the `editions` of a selection
	BnfArk     string   `json:"bnf_ark,omitempty"`
	Title      string   `json:"title"`
	Publishers []string `json:"publishers"`
	StartYear  *int     `json:"start_year,omitempty"`
	Language   string   `json:"language,omitempty"`   // ISO 639-1 (translations)
	Collection string   `json:"collection,omitempty"` // editorial collection and number: « Folio 2 »
	ISBN       string   `json:"isbn,omitempty"`       // single-volume edition only
	// Music: credited artist, media (« 2×12" Vinyl »), country, barcode, catalogue number.
	Artist         string     `json:"artist,omitempty"`
	Format         string     `json:"format,omitempty"`
	Country        string     `json:"country,omitempty"`
	Barcode        string     `json:"barcode,omitempty"`
	CatalogNumber  string     `json:"catalog_number,omitempty"`
	ExistingItemID *uuid.UUID `json:"existing_item_id,omitempty"`
	SourceURL      string     `json:"source_url"`
	// MatchedVolume: search by ISBN only — the volume carrying this ISBN.
	MatchedVolume *ExternalVolumeHit `json:"matched_volume,omitempty"`
}

// ExternalEditionQuery is the search of GET /external/{connector}/editions: exactly one
// of a title, a normalised ISBN, a normalised barcode (EAN / UPC) or a work (its
// editions, when the connector can list them).
type ExternalEditionQuery struct {
	Title, ISBN, Barcode, Work string
}

type ExternalVolumeHit struct {
	BnfArk string `json:"bnf_ark"`
	Number *int   `json:"number,omitempty"`
	Title  string `json:"title"`
	ISBN   string `json:"isbn,omitempty"`
}

// ExternalImportRequest is the selection of POST /external/{connector}/preview
// and /import: a work and / or editions (with their volumes or tracks).
type ExternalImportRequest struct {
	Work     string   `json:"work,omitempty"`     // Wikidata QID, MusicBrainz release group
	Editions []string `json:"editions,omitempty"` // BnF ARKs, MusicBrainz releases
	// Selected lists the node keys to write (import only).
	Selected []string `json:"selected,omitempty"`
	// Overrides replaces the attributes of selected nodes (corrections made in
	// the preview, import only): node key → complete attributes.
	Overrides map[string]json.RawMessage `json:"overrides,omitempty"`
	// Title of the proposal (POST /external/{connector}/propose only, P4-3b).
	Title string `json:"title,omitempty"`
}

// Roles of the nodes of a plan (vocabulary of the connector, translated by the client).
const (
	NodeWork      = "work"
	NodeUniverse  = "universe"
	NodePerson    = "person"
	NodeEdition   = "edition"
	NodePublisher = "publisher"
	NodeVolume    = "volume"
	NodeSeason    = "season" // a season of a series, with its episodes (role volume)
)

// Status of a node against the catalogue.
const (
	NodeNew       = "new"       // no matching item: will be created
	NodeUpdate    = "update"    // matching item with empty fields the source fills
	NodeUnchanged = "unchanged" // matching item, nothing to add
)

// Notes explaining why a node is not selected by default.
const (
	NoteReprint  = "reprint"   // another record of the edition has the same number
	NoteNoNumber = "no_number" // volume without a number in the BnF record
)

// ExternalPlan is what the sources propose for a selection, matched against
// the catalogue (POST /external/bd-manga/preview).
type ExternalPlan struct {
	RetrievedAt string             `json:"retrieved_at"`
	Nodes       []ExternalNode     `json:"nodes"`
	Relations   []ExternalRelation `json:"relations"`
}

// ExternalNode is an item to create or complete. Attributes are those that
// would be written: the existing values, completed by the source.
type ExternalNode struct {
	Key            string          `json:"key"`
	Role           string          `json:"role"`
	ItemTypeID     uuid.UUID       `json:"item_type_id"`
	Attributes     json.RawMessage `json:"attributes"`
	ExistingItemID *uuid.UUID      `json:"existing_item_id,omitempty"`
	Status         string          `json:"status"`
	Selected       bool            `json:"selected"`
	Parent         string          `json:"parent,omitempty"`
	SourceURL      string          `json:"source_url"`
	Note           string          `json:"note,omitempty"`
}

// ExternalRelation links two nodes (by key).
type ExternalRelation struct {
	RelationTypeID uuid.UUID       `json:"relation_type_id"`
	From           string          `json:"from"`
	To             string          `json:"to"`
	Attributes     json.RawMessage `json:"attributes"`
	ExistingID     *uuid.UUID      `json:"existing_id,omitempty"`
}

// ExternalImportReport is the import report plus the item id of every written
// node (meaningful once applied).
type ExternalImportReport struct {
	ImportReport
	ItemIDs map[string]uuid.UUID `json:"item_ids"`
}

// ConnectorInfo describes a connector (GET /external/connectors): its path
// segment and the domain package holding the types it writes.
type ConnectorInfo struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	// EditionsByWork: editions can be listed from a work (editions?work=).
	EditionsByWork bool `json:"editions_by_work"`
	// EditionCode: the code editions are searched by, « isbn » or « barcode ».
	EditionCode string `json:"edition_code"`
}
