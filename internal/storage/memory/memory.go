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
	mu      sync.RWMutex
	objects map[string][]byte
}

// NewMemoryBackend creates a new in-memory storage backend
func NewMemoryBackend() storage.Backend {
	return &MemoryBackend{
		objects: make(map[string][]byte),
	}
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
