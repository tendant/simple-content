// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package storage

import (
	"context"
	"io"
	"time"
)

// ObjectMeta contains metadata about an object in storage
type ObjectMeta struct {
	// Key is the object key/path in the storage
	Key string
	// Size is the size of the object in bytes
	Size int64
	// ContentType is the MIME type of the object
	ContentType string
	// UpdatedAt is when the object was last modified
	UpdatedAt time.Time
	// ETag is the entity tag of the object
	ETag string
	// Metadata contains any custom metadata associated with the object
	Metadata map[string]string
}

type UploadParams struct {
	ObjectKey string
	MimeType  string
}

// Backend defines the interface for storage backends
type Backend interface {
	// GetUploadURL returns a URL for uploading content
	GetUploadURL(ctx context.Context, objectKey string) (string, error)

	// Upload uploads content directly
	Upload(ctx context.Context, objectKey string, reader io.Reader) error

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
