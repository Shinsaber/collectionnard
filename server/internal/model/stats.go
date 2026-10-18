package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

// NumberStat aggregates a numeric attribute of a CollectionType.
type NumberStat struct {
	Field string  `json:"field"`
	Count int     `json:"count"`
	Sum   float64 `json:"sum"`
	Avg   float64 `json:"avg"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

// ValueCount is one bar of a Distribution.
type ValueCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// Distribution counts the values of an enum or boolean attribute.
type Distribution struct {
	Field   string       `json:"field"`
	Values  []ValueCount `json:"values"`
	Missing int          `json:"missing"`
}

// MonthCount is one point of a timeline ("2026-10").
type MonthCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

// Timeline counts the values of a date attribute per month.
type Timeline struct {
	Field  string       `json:"field"`
	Months []MonthCount `json:"months"`
}

// StatsItem is what the client needs to render an item label.
type StatsItem struct {
	ID         uuid.UUID       `json:"id"`
	ItemTypeID uuid.UUID       `json:"item_type_id"`
	Attributes json.RawMessage `json:"attributes"`
}

// Completeness tells how many children of a container item (relations marked
// `x-completeness`) are in the collection.
type Completeness struct {
	Item  StatsItem `json:"item"`
	Owned int       `json:"owned"`
	Total int       `json:"total"`
}

// MaxCompleteness caps CollectionStats.Completeness.
const MaxCompleteness = 200

// CollectionStats is the response of GET /collections/{id}/stats.
type CollectionStats struct {
	CollectionID          uuid.UUID      `json:"collection_id"`
	TotalItems            int            `json:"total_items"`
	ItemsByType           []ItemsByType  `json:"items_by_type"`
	Numbers               []NumberStat   `json:"numbers"`
	Distributions         []Distribution `json:"distributions"`
	Timelines             []Timeline     `json:"timelines"`
	Added                 []MonthCount   `json:"added"`
	Completeness          []Completeness `json:"completeness"`
	CompletenessTruncated bool           `json:"completeness_truncated"`
	// ItemTypes cited by ItemsByType and Completeness (labels for anonymous readers).
	ItemTypes []ItemType `json:"item_types"`
}

// CollectionSummary is one collection in MeStats.
type CollectionSummary struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	CollectionTypeID uuid.UUID `json:"collection_type_id"`
	ItemCount        int       `json:"item_count"`
}

// MeStats is the response of GET /me/stats.
type MeStats struct {
	TotalCollections int                 `json:"total_collections"`
	TotalEntries     int                 `json:"total_entries"`
	DistinctItems    int                 `json:"distinct_items"`
	ItemsByType      []ItemsByType       `json:"items_by_type"`
	Collections      []CollectionSummary `json:"collections"`
	Added            []MonthCount        `json:"added"`
}
