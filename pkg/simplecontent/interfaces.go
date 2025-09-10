package simplecontent

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// BlobStore defines the interface for storage backends
type BlobStore interface {
	// GetUploadURL returns a URL for uploading content
	GetUploadURL(ctx context.Context, objectKey string) (string, error)

	// Upload uploads content directly
	Upload(ctx context.Context, objectKey string, reader io.Reader) error

	// UploadWithParams uploads content with additional parameters
	UploadWithParams(ctx context.Context, reader io.Reader, params UploadParams) error

	// GetDownloadURL returns a URL for downloading content
	GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error)

	// GetPreviewURL returns a URL for previewing content
	GetPreviewURL(ctx context.Context, objectKey string) (string, error)

	// Download downloads content directly
	Download(ctx context.Context, objectKey string) (io.ReadCloser, error)

	// Delete deletes content
	Delete(ctx context.Context, objectKey string) error

	// GetObjectMeta retrieves metadata for an object
	GetObjectMeta(ctx context.Context, objectKey string) (*ObjectMeta, error)
}

// Repository defines the interface for content and object persistence
type Repository interface {
	// Content operations
	CreateContent(ctx context.Context, content *Content) error
	GetContent(ctx context.Context, id uuid.UUID) (*Content, error)
	UpdateContent(ctx context.Context, content *Content) error
	DeleteContent(ctx context.Context, id uuid.UUID) error
	ListContent(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*Content, error)
	
	// Content metadata operations
	SetContentMetadata(ctx context.Context, metadata *ContentMetadata) error
	GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*ContentMetadata, error)
	
    // Derived content operations
    CreateDerivedContentRelationship(ctx context.Context, params CreateDerivedContentParams) (*DerivedContent, error)
    ListDerivedContent(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
    // GetDerivedRelationshipByContentID returns the derived-content relationship for a given derived content ID
    GetDerivedRelationshipByContentID(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error)
	
	// Object operations
	CreateObject(ctx context.Context, object *Object) error
	GetObject(ctx context.Context, id uuid.UUID) (*Object, error)
	GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*Object, error)
	GetObjectByObjectKeyAndStorageBackendName(ctx context.Context, objectKey, storageBackendName string) (*Object, error)
	UpdateObject(ctx context.Context, object *Object) error
	DeleteObject(ctx context.Context, id uuid.UUID) error
	
	// Object metadata operations
	SetObjectMetadata(ctx context.Context, metadata *ObjectMetadata) error
	GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (*ObjectMetadata, error)
}

// EventSink defines the interface for event handling
type EventSink interface {
	// ContentCreated is fired when content is created
	ContentCreated(ctx context.Context, content *Content) error
	
	// ContentUpdated is fired when content is updated
	ContentUpdated(ctx context.Context, content *Content) error
	
	// ContentDeleted is fired when content is deleted
	ContentDeleted(ctx context.Context, contentID uuid.UUID) error
	
	// ObjectCreated is fired when an object is created
	ObjectCreated(ctx context.Context, object *Object) error
	
	// ObjectUploaded is fired when an object is uploaded
	ObjectUploaded(ctx context.Context, object *Object) error
	
	// ObjectDeleted is fired when an object is deleted
	ObjectDeleted(ctx context.Context, objectID uuid.UUID) error
}

// Previewer defines the interface for content preview generation
type Previewer interface {
	// GeneratePreview generates a preview for the given object
	GeneratePreview(ctx context.Context, object *Object, blobStore BlobStore) (*ObjectPreview, error)
	
	// SupportsContent returns true if the previewer supports the given content type
	SupportsContent(mimeType string) bool
}

// ObjectMeta contains metadata about an object in storage
type ObjectMeta struct {
	Key         string
	Size        int64
	ContentType string
	UpdatedAt   time.Time
	ETag        string
	Metadata    map[string]string
}

// UploadParams contains parameters for uploading an object
type UploadParams struct {
	ObjectKey string
	MimeType  string
}

// CreateDerivedContentParams contains parameters for creating derived content relationships
type CreateDerivedContentParams struct {
	ParentID           uuid.UUID
	DerivedContentID   uuid.UUID
	DerivationType     string
	DerivationParams   map[string]interface{}
	ProcessingMetadata map[string]interface{}
}

// ListDerivedContentParams contains parameters for listing derived content
type ListDerivedContentParams struct {
	ParentID       *uuid.UUID
	DerivationType *string
	Limit          *int
	Offset         *int
}
