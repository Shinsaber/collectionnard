package model

import "github.com/google/uuid"

// DomainFormat identifies a domain package file (server/internal/domains/packages).
const DomainFormat = "collectionnard.domain"

// DomainPackage is a set of meta-types (types of a collection domain: TV, books…)
// installable on an instance. Types keep fixed UUIDs across instances.
type DomainPackage struct {
	Format          string                 `json:"format"`
	Version         int                    `json:"version"`
	ID              string                 `json:"id"`
	PackageVersion  int                    `json:"package_version"`
	Title           map[string]string      `json:"title"`
	Description     map[string]string      `json:"description"`
	ItemTypes       []ExportItemType       `json:"item_types"`
	CollectionTypes []ExportCollectionType `json:"collection_types"`
	RelationTypes   []ExportRelationType   `json:"relation_types"`
}

// Domain install states (DomainInfo.Status).
const (
	DomainNotInstalled    = "not_installed"
	DomainUpToDate        = "up_to_date"
	DomainUpdateAvailable = "update_available"
)

// DomainInfo describes a package and its state on the instance (GET /domains).
type DomainInfo struct {
	ID               string            `json:"id"`
	Title            map[string]string `json:"title"`
	Description      map[string]string `json:"description"`
	PackageVersion   int               `json:"package_version"`
	InstalledVersion *int              `json:"installed_version,omitempty"`
	Status           string            `json:"status"`
	// ModifiedLocally: at least one installed type differs from the package version it came from.
	ModifiedLocally bool `json:"modified_locally"`
	ItemTypes       int  `json:"item_types"`
	RelationTypes   int  `json:"relation_types"`
	CollectionTypes int  `json:"collection_types"`
}

// Outcomes of a meta-type in a domain install (DomainInstallEntry.Outcome).
const (
	DomainCreated   = "created"   // absent : créé depuis le paquet
	DomainUpdated   = "updated"   // non modifié localement : remplacé par la nouvelle version
	DomainMerged    = "merged"    // modifié localement : ajouts du paquet fusionnés, modifications gardées
	DomainUnchanged = "unchanged" // rien à changer
	DomainSkipped   = "skipped"   // supprimé localement depuis l'installation, ou relation dont les types manquent : pas écrit
)

// DomainInstallEntry is the outcome for one meta-type of the package.
type DomainInstallEntry struct {
	Entity  string    `json:"entity"` // item_types | collection_types | relation_types
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Outcome string    `json:"outcome"`
	// Conflicts: paths changed both locally and in the package; the local value is kept.
	Conflicts []string `json:"conflicts,omitempty"`
	// MissingTypes: item types referenced by the package (relation ends, allowed
	// types) that are not on the instance (domain not installed): left out.
	MissingTypes []uuid.UUID `json:"missing_types,omitempty"`
}

// DomainInstallReport answers POST /domains/{id}/install.
type DomainInstallReport struct {
	DryRun      bool                 `json:"dry_run"`
	Applied     bool                 `json:"applied"`
	Domain      string               `json:"domain"`
	FromVersion *int                 `json:"from_version,omitempty"`
	ToVersion   int                  `json:"to_version"`
	Entries     []DomainInstallEntry `json:"entries"`
	Errors      []ImportError        `json:"errors"`
}
