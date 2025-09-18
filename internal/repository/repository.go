// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
)

// ListDerivedContentParams defines parameters for listing derived content
type ListDerivedContentParams struct {
	ParentIDs      []uuid.UUID
	TenantID       uuid.UUID
	DerivationType []string
}

// GetDerivedContentByLevelParams defines parameters for getting derived content by level
type GetDerivedContentByLevelParams struct {
	RootID   uuid.UUID // The root content ID to start from
	Level    int       // The level of derivation (0 = root, 1 = direct children, etc.)
	TenantID uuid.UUID // Optional tenant filter
	MaxDepth int       // Optional max depth to search (default is 10)
}

// ContentWithParent represents a content item with its parent ID
type ContentWithParent struct {
	Content  *domain.Content // The content item
	ParentID uuid.UUID       // ID of the parent content (nil for root content)
	Level    int             // Level in the derivation hierarchy
}

// CreateDerivedContentParams defines parameters for creating derived content
type CreateDerivedContentParams struct {
	ParentID           uuid.UUID
	DerivedContentID   uuid.UUID
	DerivationParams   map[string]interface{}
	ProcessingMetadata map[string]interface{}
	DerivationType     string
}

// DeleteDerivedContentParams defines parameters for deleting derived content
type DeleteDerivedContentParams struct {
	ParentID         uuid.UUID
	DerivedContentID uuid.UUID
}

// ContentRepository defines the interface for content operations
type ContentRepository interface {
	Create(ctx context.Context, content *domain.Content) error
	Get(ctx context.Context, id uuid.UUID) (*domain.Content, error)
	Update(ctx context.Context, content *domain.Content) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*domain.Content, error)
	ListDerivedContent(ctx context.Context, params ListDerivedContentParams) ([]*domain.DerivedContent, error)
	GetDerivedContentByLevel(ctx context.Context, params GetDerivedContentByLevelParams) ([]ContentWithParent, error)
	CreateDerivedContentRelationship(ctx context.Context, params CreateDerivedContentParams) (domain.DerivedContent, error)
	DeleteDerivedContentRelationship(ctx context.Context, params DeleteDerivedContentParams) error
}

// ContentMetadataRepository defines the interface for content metadata operations
type ContentMetadataRepository interface {
	Set(ctx context.Context, metadata *domain.ContentMetadata) error
	Get(ctx context.Context, contentID uuid.UUID) (*domain.ContentMetadata, error)
}

// ObjectRepository defines the interface for object operations
type ObjectRepository interface {
	Create(ctx context.Context, object *domain.Object) error
	Get(ctx context.Context, id uuid.UUID) (*domain.Object, error)
	GetByContentID(ctx context.Context, contentID uuid.UUID) ([]*domain.Object, error)
	GetByObjectKeyAndStorageBackendName(ctx context.Context, objectKey string, storageBackendName string) (*domain.Object, error)
	Update(ctx context.Context, object *domain.Object) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ObjectMetadataRepository defines the interface for object metadata operations
type ObjectMetadataRepository interface {
	Set(ctx context.Context, metadata *domain.ObjectMetadata) error
	Get(ctx context.Context, objectID uuid.UUID) (*domain.ObjectMetadata, error)
}

// StorageBackendRepository defines the interface for storage backend operations
type StorageBackendRepository interface {
	Create(ctx context.Context, backend *domain.StorageBackend) error
	Get(ctx context.Context, name string) (*domain.StorageBackend, error)
	Update(ctx context.Context, backend *domain.StorageBackend) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*domain.StorageBackend, error)
}

// AuditEventRepository defines the interface for audit event operations
type AuditEventRepository interface {
	Create(ctx context.Context, event *domain.AuditEvent) error
	Get(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
	List(ctx context.Context, contentID, objectID, actorID uuid.UUID) ([]*domain.AuditEvent, error)
}

// AccessLogRepository defines the interface for access log operations
type AccessLogRepository interface {
	Create(ctx context.Context, log *domain.AccessLog) error
	List(ctx context.Context, contentID, actorID uuid.UUID) ([]*domain.AccessLog, error)
}
