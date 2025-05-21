package domain

import (
	"time"

	"github.com/google/uuid"
)

// StorageBackend represents a configurable storage backend
type StorageBackend struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // "memory", "fs", "s3", etc.
	Config    map[string]interface{} `json:"config"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
