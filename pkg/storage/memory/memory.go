// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package memory

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/tendant/simple-content/internal/storage"
)

// MemoryBackend is an in-memory implementation of the storage.Backend interface
type MemoryBackend struct {
	mu              sync.RWMutex
	objects         map[string][]byte
	objectsMimeType map[string]string
}

// NewMemoryBackend creates a new in-memory storage backend
func NewMemoryBackend() storage.Backend {
	return &MemoryBackend{
		objects:         make(map[string][]byte),
		objectsMimeType: make(map[string]string),
	}
}

// GetObjectMeta retrieves metadata for an object in memory
func (b *MemoryBackend) GetObjectMeta(ctx context.Context, objectKey string) (*storage.ObjectMeta, error) {
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

	meta := &storage.ObjectMeta{
		Key:         objectKey,
		Size:        int64(len(data)),
		ContentType: mimeType,
		Metadata:    map[string]string{"mime_type": mimeType},
	}

	return meta, nil
}

// GetUploadURL returns a URL for uploading content
// In-memory implementation doesn't use URLs
func (b *MemoryBackend) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	return "", errors.New("direct upload required for memory backend")
}

// Upload uploads content directly
func (b *MemoryBackend) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.objects[objectKey] = data
	return nil
}

func (b *MemoryBackend) UploadWithParams(ctx context.Context, reader io.Reader, params storage.UploadParams) error {

	err := b.Upload(ctx, params.ObjectKey, reader)
	if err != nil {
		return err
	}
	b.objectsMimeType[params.ObjectKey] = params.MimeType
	return nil
}

// GetDownloadURL returns a URL for downloading content
// In-memory implementation doesn't use URLs
func (b *MemoryBackend) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {
	return "", errors.New("direct download required for memory backend")
}

// GetPreviewURL returns a URL for previewing content
func (b *MemoryBackend) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
	return "", errors.New("direct preview required for memory backend")
}

// Download downloads content directly
func (b *MemoryBackend) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, exists := b.objects[objectKey]
	if !exists {
		return nil, errors.New("object not found")
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// Delete deletes content
func (b *MemoryBackend) Delete(ctx context.Context, objectKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.objects[objectKey]; !exists {
		return errors.New("object not found")
	}

	delete(b.objects, objectKey)
	return nil
}
