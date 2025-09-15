// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package domain

import (
	"time"
)

// StorageBackend represents a configurable storage backend
type StorageBackend struct {
	Name      string                 `json:"name"` // Primary identifier
	Type      string                 `json:"type"` // "memory", "fs", "s3", etc.
	Config    map[string]interface{} `json:"config"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
