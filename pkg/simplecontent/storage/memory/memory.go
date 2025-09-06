package memory

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/tendant/simple-content/pkg/simplecontent"
)

// Backend is an in-memory implementation of the simplecontent.BlobStore interface
type Backend struct {
	mu              sync.RWMutex
	objects         map[string][]byte
	objectsMimeType map[string]string
}

// New creates a new in-memory storage backend
func New() simplecontent.BlobStore {
	return &Backend{
		objects:         make(map[string][]byte),
		objectsMimeType: make(map[string]string),
	}
}

// GetObjectMeta retrieves metadata for an object in memory
func (b *Backend) GetObjectMeta(ctx context.Context, objectKey string) (*simplecontent.ObjectMeta, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, exists := b.objects[objectKey]
	if !exists {
		return nil, errors.New("object not found")
	}
	mimeType, exists := b.objectsMimeType[objectKey]
	if !exists {
		return nil, errors.New("object not found")
	}

	meta := &simplecontent.ObjectMeta{
		Key:         objectKey,
		Size:        int64(len(data)),
		ContentType: mimeType,
		Metadata:    map[string]string{"mime_type": mimeType},
	}

	return meta, nil
}

// GetUploadURL returns a URL for uploading content
// In-memory implementation doesn't use URLs
func (b *Backend) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	return "", errors.New("direct upload required for memory backend")
}

// Upload uploads content directly
func (b *Backend) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.objects[objectKey] = data
	// Set default MIME type if not set
	if _, exists := b.objectsMimeType[objectKey]; !exists {
		b.objectsMimeType[objectKey] = "application/octet-stream"
	}
	return nil
}

// UploadWithParams uploads content with parameters
func (b *Backend) UploadWithParams(ctx context.Context, reader io.Reader, params simplecontent.UploadParams) error {
	err := b.Upload(ctx, params.ObjectKey, reader)
	if err != nil {
		return err
	}
	b.objectsMimeType[params.ObjectKey] = params.MimeType
	return nil
}

// GetDownloadURL returns a URL for downloading content
// In-memory implementation doesn't use URLs
func (b *Backend) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {
	return "", errors.New("direct download required for memory backend")
}

// GetPreviewURL returns a URL for previewing content
func (b *Backend) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
	return "", errors.New("direct preview required for memory backend")
}

// Download downloads content directly
func (b *Backend) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, exists := b.objects[objectKey]
	if !exists {
		return nil, errors.New("object not found")
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// Delete deletes content
func (b *Backend) Delete(ctx context.Context, objectKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.objects[objectKey]; !exists {
		return errors.New("object not found")
	}

	delete(b.objects, objectKey)
	return nil
}