// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent represents an event log for audits
type AuditEvent struct {
	ID        uuid.UUID              `json:"id"`
	ContentID uuid.UUID              `json:"content_id"`
	ObjectID  uuid.UUID              `json:"object_id"`
	ActorID   uuid.UUID              `json:"actor_id"`
	Action    string                 `json:"action"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

// AccessLog represents an optional access log
type AccessLog struct {
	ID             uuid.UUID `json:"id"`
	ContentID      uuid.UUID `json:"content_id"`
	ActorID        uuid.UUID `json:"actor_id"`
	Method         string    `json:"method"`
	StorageBackend string    `json:"storage_backend"`
	CreatedAt      time.Time `json:"created_at"`
}
