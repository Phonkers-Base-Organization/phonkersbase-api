package domain

import (
	"encoding/json"
	"time"
)

// Entity types recorded in the change history.
const (
	EntityTypeArtist       = "artist"
	EntityTypeLabel        = "label"
	EntityTypeSource       = "source"
	EntityTypeOrganisation = "organisation"
)

// ValidEntityType reports whether t is a recognized entity type.
func ValidEntityType(t string) bool {
	switch t {
	case EntityTypeArtist, EntityTypeLabel, EntityTypeSource, EntityTypeOrganisation:
		return true
	}
	return false
}

// Actions recorded in the change history.
const (
	ChangeActionCreate = "create"
	ChangeActionUpdate = "update"
	ChangeActionDelete = "delete"
)

// ValidChangeAction reports whether a is a recognized change action.
func ValidChangeAction(a string) bool {
	switch a {
	case ChangeActionCreate, ChangeActionUpdate, ChangeActionDelete:
		return true
	}
	return false
}

type ChangeEntry struct {
	ID             string          `json:"id"`
	EntityType     string          `json:"entityType"`
	EntityID       string          `json:"entityId"`
	EntityName     string          `json:"entityName"`
	Action         string          `json:"action"`
	EditorID       string          `json:"editorId"`
	EditorUsername string          `json:"editorUsername"`
	OldData        json.RawMessage `json:"oldData"`
	NewData        json.RawMessage `json:"newData"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type PaginatedChanges struct {
	Items []ChangeEntry `json:"items"`
	Info  Pagination    `json:"info"`
}

type ListChangesParams struct {
	EntityType string
	EntityID   string
	Action     string
	Editor     string
	Search     string
	Limit      int
	Offset     int
}
