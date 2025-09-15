// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ContentStatusCreated  = "created"
	ContentStatusUploaded = "uploaded"
)

// content derivation_type
const (
	ContentDerivationTypeOriginal = "original"
	ContentDerivationTypeDerived  = "derived"
)

// content_derived derivation_typ
const (
	ContentDerivedTHUMBNAIL720 = "THUMBNAIL_720"
	ContentDerivedTHUMBNAIL480 = "THUMBNAIL_480"
	ContentDerivedTHUMBNAIL256 = "THUMBNAIL_256"
	ContentDerivedTHUMBNAIL128 = "THUMBNAIL_128"
	ContentDerivedConversion   = "CONVERSION"
)

// Content represents a logical content entity
type Content struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	OwnerID        uuid.UUID `json:"owner_id"`
	OwnerType      string    `json:"owner_type,omitempty"`
	Name           string    `json:"name,omitempty"`
	Description    string    `json:"description,omitempty"`
	DocumentType   string    `json:"document_type,omitempty"`
	Status         string    `json:"status"`
	DerivationType string    `json:"derivation_type,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DerivedContent struct {
	ParentID           uuid.UUID              `json:"parent_id"`
	ContentID          uuid.UUID              `json:"content_id"`
	DerivationType     string                 `json:"derivation_type"`
	DerivationParams   map[string]interface{} `json:"derivation_params"`
	ProcessingMetadata map[string]interface{} `json:"processing_metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	DocumentType       string                 `json:"document_type"`
	Status             string                 `json:"status"`
}

// ContentMetadata represents metadata for a content
type ContentMetadata struct {
	ContentID         uuid.UUID              `json:"content_id"`
	Tags              []string               `json:"tags,omitempty"`
	FileSize          int64                  `json:"file_size,omitempty"`
	FileName          string                 `json:"file_name,omitempty"`
	MimeType          string                 `json:"mime_type"` // MIME type
	Checksum          string                 `json:"checksum,omitempty"`
	ChecksumAlgorithm string                 `json:"checksum_algorithm,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"` // For other custom metadata
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}
