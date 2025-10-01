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
// DerivationType here represents the category (e.g., "thumbnail", "preview").
// Variant represents the specific variant (e.g., "thumbnail_256").
type DerivedContent struct {
	// Persisted fields
	ParentID           uuid.UUID              `json:"parent_id" db:"parent_id"`
	ContentID          uuid.UUID              `json:"content_id" db:"content_id"`
	DerivationType     string                 `json:"derivation_type" db:"derivation_type"`
	Variant            string                 `json:"variant" db:"variant"`                      // NEW: Specific variant (persisted)
	DerivationParams   map[string]interface{} `json:"derivation_params" db:"derivation_params"`
	ProcessingMetadata map[string]interface{} `json:"processing_metadata" db:"processing_metadata"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
	DocumentType       string                 `json:"document_type" db:"document_type"`
	Status             string                 `json:"status" db:"status"`

	// Computed fields (not persisted - populated by service layer)
	DownloadURL        string                 `json:"download_url,omitempty" db:"-"`
	PreviewURL         string                 `json:"preview_url,omitempty" db:"-"`
	ThumbnailURL       string                 `json:"thumbnail_url,omitempty" db:"-"`

	// Optional enhanced data (not persisted - populated on demand)
	Objects            []*Object              `json:"objects,omitempty" db:"-"`
	Metadata           *ContentMetadata       `json:"metadata,omitempty" db:"-"`
	ParentContent      *Content               `json:"parent_content,omitempty" db:"-"`
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

// ContentDetails represents all details for a content including URLs and metadata.
// This unified type provides everything clients need in a single call.
type ContentDetails struct {
	ID          string            `json:"id"`                        // Content ID

	// Access URLs
	Download    string            `json:"download,omitempty"`        // Primary download URL
	Upload      string            `json:"upload,omitempty"`          // Upload URL (when WithUploadAccess option used)
	Preview     string            `json:"preview,omitempty"`         // Primary preview URL
	Thumbnail   string            `json:"thumbnail,omitempty"`       // Primary thumbnail URL
	Thumbnails  map[string]string `json:"thumbnails,omitempty"`      // size -> URL (256, 512, etc.)
	Previews    map[string]string `json:"previews,omitempty"`        // variant -> URL (720p, 1080p, webm, etc.)
	Transcodes  map[string]string `json:"transcodes,omitempty"`      // format -> URL (mp3, flac, mp4, etc.)

	// File metadata
	FileName    string            `json:"file_name,omitempty"`       // Original file name
	FileSize    int64             `json:"file_size,omitempty"`       // File size in bytes
	MimeType    string            `json:"mime_type,omitempty"`       // MIME type
	Tags        []string          `json:"tags,omitempty"`            // Content tags
	Checksum    string            `json:"checksum,omitempty"`        // File checksum

	// Status and timing
	Ready       bool              `json:"ready"`                     // Are all URLs ready/available?
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`      // When URLs expire (for presigned URLs)
	CreatedAt   time.Time         `json:"created_at"`                // Content creation time
	UpdatedAt   time.Time         `json:"updated_at"`                // Content last update time
}

// ContentListFilters defines filtering options for listing content (admin operations)
type ContentListFilters struct {
	TenantID        *uuid.UUID
	TenantIDs       []uuid.UUID
	OwnerID         *uuid.UUID
	OwnerIDs        []uuid.UUID
	Status          *string
	Statuses        []string
	DerivationType  *string
	DerivationTypes []string
	DocumentType    *string
	DocumentTypes   []string
	CreatedAfter    *time.Time
	CreatedBefore   *time.Time
	UpdatedAfter    *time.Time
	UpdatedBefore   *time.Time
	Limit           *int
	Offset          *int
	SortBy          *string
	SortOrder       *string
	IncludeDeleted  bool
}

// ContentCountFilters defines filtering options for counting content
type ContentCountFilters struct {
	TenantID        *uuid.UUID
	TenantIDs       []uuid.UUID
	OwnerID         *uuid.UUID
	OwnerIDs        []uuid.UUID
	Status          *string
	Statuses        []string
	DerivationType  *string
	DerivationTypes []string
	DocumentType    *string
	DocumentTypes   []string
	CreatedAfter    *time.Time
	CreatedBefore   *time.Time
	UpdatedAfter    *time.Time
	UpdatedBefore   *time.Time
	IncludeDeleted  bool
}

// ContentStatisticsOptions defines what statistics to include
type ContentStatisticsOptions struct {
	IncludeStatusBreakdown       bool
	IncludeTenantBreakdown       bool
	IncludeDerivationBreakdown   bool
	IncludeDocumentTypeBreakdown bool
	IncludeTimeRange             bool
}

// ContentStatisticsResult contains aggregated statistics about content
type ContentStatisticsResult struct {
	TotalCount       int64
	ByStatus         map[string]int64
	ByTenant         map[string]int64
	ByDerivationType map[string]int64
	ByDocumentType   map[string]int64
	OldestContent    *time.Time
	NewestContent    *time.Time
}
