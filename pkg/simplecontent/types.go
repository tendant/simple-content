package simplecontent

import (
    "time"

    "github.com/google/uuid"
)

// ContentStatus is the domain type for content lifecycle states.
type ContentStatus string

// Content status constants (typed).
const (
    ContentStatusCreated  ContentStatus = "created"
    ContentStatusUploaded ContentStatus = "uploaded"
    ContentStatusDeleted  ContentStatus = "deleted"
)

// Content derivation type constants
const (
    ContentDerivationTypeOriginal = "original"
    ContentDerivationTypeDerived  = "derived"
)

// DerivationVariant is the specific variant within a category (e.g., "thumbnail_256").
type DerivationVariant string

// Derivation variant constants (typed).
const (
    VariantThumbnail720 DerivationVariant = "thumbnail_720"
    VariantThumbnail480 DerivationVariant = "thumbnail_480"
    VariantThumbnail256 DerivationVariant = "thumbnail_256"
    VariantThumbnail128 DerivationVariant = "thumbnail_128"
    VariantConversion   DerivationVariant = "conversion"
)

// ObjectStatus is the domain type for object lifecycle states.
type ObjectStatus string

// Object status constants (typed).
const (
    ObjectStatusCreated    ObjectStatus = "created"
    ObjectStatusUploading  ObjectStatus = "uploading"
    ObjectStatusUploaded   ObjectStatus = "uploaded"
    ObjectStatusProcessing ObjectStatus = "processing"
    ObjectStatusProcessed  ObjectStatus = "processed"
    ObjectStatusFailed     ObjectStatus = "failed"
    ObjectStatusDeleted    ObjectStatus = "deleted"
)

// Content represents a logical content entity.
//
// For derived content, the DerivationType field holds the user-facing
// derivation type (e.g., "thumbnail", "preview"). Specific variant (e.g.,
// "thumbnail_256") is tracked in the derived-content relationship.
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
    DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// DerivedContent represents content derived from a parent content.
// DerivationType here represents the specific variant (e.g., "thumbnail_256").
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
	MimeType          string                 `json:"mime_type"`
	Checksum          string                 `json:"checksum,omitempty"`
	ChecksumAlgorithm string                 `json:"checksum_algorithm,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// Object represents a physical object stored in a storage backend
type Object struct {
    ID                 uuid.UUID `json:"id"`
    ContentID          uuid.UUID `json:"content_id"`
    StorageBackendName string    `json:"storage_backend_name"`
    StorageClass       string    `json:"storage_class,omitempty"`
    ObjectKey          string    `json:"object_key"`
    FileName           string    `json:"file_name,omitempty"`
    Version            int       `json:"version"`
    ObjectType         string    `json:"object_type,omitempty"`
    Status             string    `json:"status"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
    DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

// ObjectMetadata represents metadata about an object
type ObjectMetadata struct {
	ObjectID  uuid.UUID              `json:"object_id"`
	SizeBytes int64                  `json:"size_bytes"`
	MimeType  string                 `json:"mime_type"`
	ETag      string                 `json:"etag,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ObjectPreview represents a preview generated from an object
type ObjectPreview struct {
	ID          uuid.UUID `json:"id"`
	ObjectID    uuid.UUID `json:"object_id"`
	PreviewType string    `json:"preview_type"`
	Status      string    `json:"status"`
	PreviewURL  string    `json:"preview_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// StorageBackend represents a configurable storage backend
type StorageBackend struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
