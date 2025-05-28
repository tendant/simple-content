package storage

import (
	"context"
	"io"
)

// Backend defines the interface for storage backends
type Backend interface {
	// GetUploadURL returns a URL for uploading content
	GetUploadURL(ctx context.Context, objectKey string) (string, error)

	// Upload uploads content directly
	Upload(ctx context.Context, objectKey string, reader io.Reader) error

	// GetDownloadURL returns a URL for downloading content
	GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error)

	// Download downloads content directly
	Download(ctx context.Context, objectKey string) (io.ReadCloser, error)

	// Delete deletes content
	Delete(ctx context.Context, objectKey string) error
}
