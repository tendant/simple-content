// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ObjectStatusCreated    = "created"
	ObjectStatusUploading  = "uploading"
	ObjectStatusUploaded   = "uploaded"
	ObjectStatusProcessing = "processing"
	ObjectStatusProcessed  = "processed"
	ObjectStatusFailed     = "failed"
	ObjectStatusDeleted    = "deleted"
)

// Object represents a physical object stored in a storage backend
type Object struct {
	ID                 uuid.UUID `json:"id"`
	ContentID          uuid.UUID `json:"content_id"`
	StorageBackendName string    `json:"storage_backend_name"`
	StorageClass       string    `json:"storage_class,omitempty"`
	ObjectKey          string    `json:"object_key"`
	FileName           string    `json:"file_name,omitempty"`
	Version            int       `json:"version"` // Used for compatibility with version_id field
	ObjectType         string    `json:"object_type,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
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
	PreviewURL  string    `json:"preview_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
