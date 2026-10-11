package model

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string `json:"token"`         // access token JWT (courte durée)
	RefreshToken string `json:"refresh_token"` // token opaque, rotation à chaque utilisation
	ExpiresIn    int    `json:"expires_in"`    // durée de vie de `token` en secondes
	User         User   `json:"user"`
}

// RefreshRequest is the body of POST /auth/refresh and POST /auth/logout.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// ItemType
// ---------------------------------------------------------------------------

type ItemType struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ItemTypeCreate struct {
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema"`
}

type ItemTypeUpdate struct {
	Name             *string         `json:"name,omitempty"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema,omitempty"`
}

// ---------------------------------------------------------------------------
// RelationType
// ---------------------------------------------------------------------------

type RelationType struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	Description       *string         `json:"description,omitempty"`
	SourceItemTypeIDs []uuid.UUID     `json:"source_item_type_ids"`
	TargetItemTypeIDs []uuid.UUID     `json:"target_item_type_ids"`
	SourceItemTypes   []ItemType      `json:"source_item_types,omitempty"`
	TargetItemTypes   []ItemType      `json:"target_item_types,omitempty"`
	AttributesSchema  json.RawMessage `json:"attributes_schema"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type RelationTypeCreate struct {
	Name              string          `json:"name"`
	Description       *string         `json:"description,omitempty"`
	SourceItemTypeIDs []uuid.UUID     `json:"source_item_type_ids"`
	TargetItemTypeIDs []uuid.UUID     `json:"target_item_type_ids"`
	AttributesSchema  json.RawMessage `json:"attributes_schema,omitempty"`
}

type RelationTypeUpdate struct {
	Name              *string         `json:"name,omitempty"`
	Description       *string         `json:"description,omitempty"`
	SourceItemTypeIDs []uuid.UUID     `json:"source_item_type_ids,omitempty"`
	TargetItemTypeIDs []uuid.UUID     `json:"target_item_type_ids,omitempty"`
	AttributesSchema  json.RawMessage `json:"attributes_schema,omitempty"`
}

// ---------------------------------------------------------------------------
// CollectionType
// ---------------------------------------------------------------------------

type CollectionType struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type CollectionTypeCreate struct {
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema,omitempty"`
}

type CollectionTypeUpdate struct {
	Name             *string         `json:"name,omitempty"`
	Description      *string         `json:"description,omitempty"`
	AttributesSchema json.RawMessage `json:"attributes_schema,omitempty"`
}

// ---------------------------------------------------------------------------
// Item
// ---------------------------------------------------------------------------

type Item struct {
	ID         uuid.UUID       `json:"id"`
	ItemTypeID uuid.UUID       `json:"item_type_id"`
	ItemType   *ItemType       `json:"item_type,omitempty"`
	Attributes json.RawMessage `json:"attributes"`
	Tags       []Tag           `json:"tags,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type ItemCreate struct {
	ItemTypeID uuid.UUID       `json:"item_type_id"`
	Attributes json.RawMessage `json:"attributes"`
	Tags       []string        `json:"tags,omitempty"`
}

type ItemUpdate struct {
	Attributes json.RawMessage `json:"attributes,omitempty"`
	Tags       *[]string       `json:"tags,omitempty"`
}

// ---------------------------------------------------------------------------
// ItemRelation
// ---------------------------------------------------------------------------

type ItemRelation struct {
	ID             uuid.UUID       `json:"id"`
	RelationTypeID uuid.UUID       `json:"relation_type_id"`
	RelationType   *RelationType   `json:"relation_type,omitempty"`
	SourceItemID   uuid.UUID       `json:"source_item_id"`
	SourceItem     *Item           `json:"source_item,omitempty"`
	TargetItemID   uuid.UUID       `json:"target_item_id"`
	TargetItem     *Item           `json:"target_item,omitempty"`
	Attributes     json.RawMessage `json:"attributes"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ItemRelationCreate struct {
	RelationTypeID uuid.UUID       `json:"relation_type_id"`
	SourceItemID   uuid.UUID       `json:"source_item_id"`
	TargetItemID   uuid.UUID       `json:"target_item_id"`
	Attributes     json.RawMessage `json:"attributes,omitempty"`
}

type ItemRelationUpdate struct {
	Attributes json.RawMessage `json:"attributes"`
}

// ---------------------------------------------------------------------------
// Collection
// ---------------------------------------------------------------------------

type Collection struct {
	ID               uuid.UUID       `json:"id"`
	UserID           uuid.UUID       `json:"user_id"`
	CollectionTypeID uuid.UUID       `json:"collection_type_id"`
	CollectionType   *CollectionType `json:"collection_type,omitempty"`
	Name             string          `json:"name"`
	Description      *string         `json:"description,omitempty"`
	ItemCount        int             `json:"item_count"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type CollectionCreate struct {
	CollectionTypeID uuid.UUID `json:"collection_type_id"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
}

type CollectionUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ---------------------------------------------------------------------------
// CollectionItem
// ---------------------------------------------------------------------------

type CollectionItem struct {
	ID           uuid.UUID       `json:"id"`
	CollectionID uuid.UUID       `json:"collection_id"`
	ItemID       uuid.UUID       `json:"item_id"`
	Item         *Item           `json:"item,omitempty"`
	Attributes   json.RawMessage `json:"attributes"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CollectionItemAdd struct {
	ItemID     uuid.UUID       `json:"item_id"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

type CollectionItemUpdate struct {
	Attributes json.RawMessage `json:"attributes"`
}

// ItemCollectionMembership represents a collection that contains a given item,
// along with the contextual attributes of the membership.
type ItemCollectionMembership struct {
	Collection Collection      `json:"collection"`
	Attributes json.RawMessage `json:"attributes"`
}

// ---------------------------------------------------------------------------
// Tag
// ---------------------------------------------------------------------------

type Tag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ---------------------------------------------------------------------------
// Graph
// ---------------------------------------------------------------------------

type GraphNode struct {
	ID         uuid.UUID       `json:"id"`
	Type       string          `json:"type"`
	Label      string          `json:"label"`
	Attributes json.RawMessage `json:"attributes"`
	Depth      int             `json:"depth"`
}

type GraphEdge struct {
	SourceID     uuid.UUID       `json:"source_id"`
	TargetID     uuid.UUID       `json:"target_id"`
	RelationType string          `json:"relation_type"`
	Attributes   json.RawMessage `json:"attributes"`
}

type GraphResponse struct {
	Root  GraphNode   `json:"root"`
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

type ItemsByType struct {
	TypeName string `json:"type_name"`
	Count    int    `json:"count"`
}

type Stats struct {
	ItemsByType      []ItemsByType `json:"items_by_type"`
	TotalItems       int           `json:"total_items"`
	TotalCollections int           `json:"total_collections"`
	TotalRelations   int           `json:"total_relations"`
}

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

type Pagination struct {
	Page    int
	PerPage int
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PerPage
}

type PaginatedResponse struct {
	Data       any `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

func NewPaginatedResponse(data any, total int, pg Pagination) PaginatedResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pg.PerPage)))
	return PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       pg.Page,
		PerPage:    pg.PerPage,
		TotalPages: totalPages,
	}
}

// ---------------------------------------------------------------------------
// API Error
// ---------------------------------------------------------------------------

type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
