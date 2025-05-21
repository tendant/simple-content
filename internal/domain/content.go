package domain

import (
	"time"

	"github.com/google/uuid"
)

// Content represents a logical content entity
type Content struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	OwnerID   uuid.UUID `json:"owner_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Status    string    `json:"status"`
}

// ContentMetadata represents custom metadata for a content
type ContentMetadata struct {
	ContentID uuid.UUID              `json:"content_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}
