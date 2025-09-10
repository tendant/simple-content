package simplecontent

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Service defines the main interface for the simple-content library
type Service interface {
	// Content operations
	CreateContent(ctx context.Context, req CreateContentRequest) (*Content, error)
	CreateDerivedContent(ctx context.Context, req CreateDerivedContentRequest) (*Content, error)
	GetContent(ctx context.Context, id uuid.UUID) (*Content, error)
	UpdateContent(ctx context.Context, req UpdateContentRequest) error
	DeleteContent(ctx context.Context, id uuid.UUID) error
	ListContent(ctx context.Context, req ListContentRequest) ([]*Content, error)
	
	// Content metadata operations
	SetContentMetadata(ctx context.Context, req SetContentMetadataRequest) error
	GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*ContentMetadata, error)
	
	// Object operations
	CreateObject(ctx context.Context, req CreateObjectRequest) (*Object, error)
	GetObject(ctx context.Context, id uuid.UUID) (*Object, error)
	GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*Object, error)
	UpdateObject(ctx context.Context, object *Object) error
	DeleteObject(ctx context.Context, id uuid.UUID) error
	
	// Object upload/download operations
	UploadObject(ctx context.Context, id uuid.UUID, reader io.Reader) error
	UploadObjectWithMetadata(ctx context.Context, reader io.Reader, req UploadObjectWithMetadataRequest) error
	DownloadObject(ctx context.Context, id uuid.UUID) (io.ReadCloser, error)
	GetUploadURL(ctx context.Context, id uuid.UUID) (string, error)
	GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error)
	GetPreviewURL(ctx context.Context, id uuid.UUID) (string, error)
	
	// Object metadata operations
	SetObjectMetadata(ctx context.Context, objectID uuid.UUID, metadata map[string]interface{}) error
	GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (map[string]interface{}, error)
    UpdateObjectMetaFromStorage(ctx context.Context, objectID uuid.UUID) (*ObjectMetadata, error)
	
	// Storage backend operations
    RegisterBackend(name string, backend BlobStore)
    GetBackend(name string) (BlobStore, error)

    // Derived content relationship helpers
    GetDerivedRelationshipByContentID(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error)
    ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error)
}
