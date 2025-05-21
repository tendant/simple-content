package domain

import (
	"time"

	"github.com/google/uuid"
)

// Content represents a logical content entity
type Content struct {
	ID              uuid.UUID  `json:"id"`
	ParentID        *uuid.UUID `json:"parent_id,omitempty"` // Reference to parent content (if any)
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	OwnerID         uuid.UUID  `json:"owner_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Status          string     `json:"status"`
	DerivationType  string     `json:"derivation_type"`  // "original" or "derived"
	DerivationLevel int        `json:"derivation_level"` // Tracks derivation depth
}

// ContentMetadata represents metadata for a content
type ContentMetadata struct {
	ContentID   uuid.UUID              `json:"content_id"`
	ContentType string                 `json:"content_type"` // MIME type
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	FileSize    int64                  `json:"file_size,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"` // For other custom metadata
}
