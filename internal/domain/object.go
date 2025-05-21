package domain

import (
	"time"

	"github.com/google/uuid"
)

// Object represents a physical object stored in a storage backend
type Object struct {
	ID               uuid.UUID `json:"id"`
	ContentID        uuid.UUID `json:"content_id"`
	StorageBackendID uuid.UUID `json:"storage_backend_id"`
	Version          int       `json:"version"`
	ObjectKey        string    `json:"object_key"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ObjectMetadata represents metadata about an object
type ObjectMetadata struct {
	ObjectID uuid.UUID              `json:"object_id"`
	Metadata map[string]interface{} `json:"metadata"`
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
