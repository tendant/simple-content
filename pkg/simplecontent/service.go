package simplecontent

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Service defines the main interface for the simple-content library.
// This interface focuses on content operations and hides storage implementation details.
type Service interface {
	// Content operations
	CreateContent(ctx context.Context, req CreateContentRequest) (*Content, error)
	GetContent(ctx context.Context, id uuid.UUID) (*Content, error)
	UpdateContent(ctx context.Context, req UpdateContentRequest) error
	DeleteContent(ctx context.Context, id uuid.UUID) error
	ListContent(ctx context.Context, req ListContentRequest) ([]*Content, error)

	// Unified content upload operations (replaces object-based workflow)
	UploadContent(ctx context.Context, req UploadContentRequest) (*Content, error)
	UploadDerivedContent(ctx context.Context, req UploadDerivedContentRequest) (*Content, error)

	// Async workflow support: upload object for existing content
	UploadObjectForContent(ctx context.Context, req UploadObjectForContentRequest) (*Object, error)

	// Content data access
	DownloadContent(ctx context.Context, contentID uuid.UUID) (io.ReadCloser, error)

	// Content metadata operations
	SetContentMetadata(ctx context.Context, req SetContentMetadataRequest) error
	GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*ContentMetadata, error)

	// Status management operations
	UpdateContentStatus(ctx context.Context, id uuid.UUID, newStatus ContentStatus) error
	UpdateObjectStatus(ctx context.Context, id uuid.UUID, newStatus ObjectStatus) error
	GetContentByStatus(ctx context.Context, status ContentStatus) ([]*Content, error)
	GetObjectsByStatus(ctx context.Context, status ObjectStatus) ([]*Object, error)

	// Object query operations
	GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*Object, error)

	// Storage backend operations
	RegisterBackend(name string, backend BlobStore)
	GetBackend(name string) (BlobStore, error)

	// Derived content operations
	CreateDerivedContent(ctx context.Context, req CreateDerivedContentRequest) (*Content, error)
	GetDerivedRelationship(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error)
	ListDerivedContent(ctx context.Context, options ...ListDerivedContentOption) ([]*DerivedContent, error)

	// Content details operations (unified interface for clients)
	GetContentDetails(ctx context.Context, contentID uuid.UUID, options ...ContentDetailsOption) (*ContentDetails, error)
	GetContentDetailsBatch(ctx context.Context, contentIDs []uuid.UUID, options ...ContentDetailsOption) (map[uuid.UUID]*ContentDetails, error)
}

// StorageService defines operations for advanced users who need direct object access.
// This is an internal interface for storage implementation details.
type StorageService interface {
	// Object operations (internal use only)
	CreateObject(ctx context.Context, req CreateObjectRequest) (*Object, error)
	GetObject(ctx context.Context, id uuid.UUID) (*Object, error)
	GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*Object, error)
	UpdateObject(ctx context.Context, object *Object) error
	DeleteObject(ctx context.Context, id uuid.UUID) error

	// Object upload/download operations (internal use only)
	UploadObject(ctx context.Context, req UploadObjectRequest) error
	DownloadObject(ctx context.Context, objectID uuid.UUID) (io.ReadCloser, error)
	GetUploadURL(ctx context.Context, objectID uuid.UUID) (string, error)
	GetDownloadURL(ctx context.Context, objectID uuid.UUID) (string, error)
	GetPreviewURL(ctx context.Context, objectID uuid.UUID) (string, error)

	// Object metadata operations (internal use only)
	SetObjectMetadata(ctx context.Context, objectID uuid.UUID, metadata map[string]interface{}) error
	GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (map[string]interface{}, error)
	UpdateObjectMetaFromStorage(ctx context.Context, objectID uuid.UUID) (*ObjectMetadata, error)
}
