// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/tendant/simple-content/internal/storage"
)

// FSBackend is a file system implementation of the storage.Backend interface
type FSBackend struct {
	mu        sync.RWMutex
	baseDir   string
	urlPrefix string
}

// Config options for the file system backend
type Config struct {
	BaseDir   string // Base directory for storing files
	URLPrefix string // Optional URL prefix for download/upload URLs
}

// NewFSBackend creates a new file system storage backend
func NewFSBackend(config Config) (storage.Backend, error) {
	// Validate and create base directory if it doesn't exist
	if config.BaseDir == "" {
		return nil, errors.New("base directory is required")
	}

	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &FSBackend{
		baseDir:   config.BaseDir,
		urlPrefix: config.URLPrefix,
	}, nil
}

// GetObjectMeta retrieves metadata for an object in the file system
func (b *FSBackend) GetObjectMeta(ctx context.Context, objectKey string) (*storage.ObjectMeta, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	filePath := filepath.Join(b.baseDir, objectKey)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, errors.New("object not found")
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	meta := &storage.ObjectMeta{
		Key:      objectKey,
		Size:     info.Size(),
		Metadata: make(map[string]string),
	}

	return meta, nil
}

// GetUploadURL returns a URL for uploading content
// For file system, this could be a local file:// URL or an API endpoint
func (b *FSBackend) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	if b.urlPrefix == "" {
		return "", errors.New("direct upload required for file system backend")
	}
	return fmt.Sprintf("%s/upload/%s", b.urlPrefix, objectKey), nil
}

// Upload uploads content directly to the file system
func (b *FSBackend) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	filePath := filepath.Join(b.baseDir, objectKey)

	// Create directory structure if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data from reader to file
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (b *FSBackend) UploadWithParams(ctx context.Context, reader io.Reader, params storage.UploadParams) error {
	return b.Upload(ctx, params.ObjectKey, reader)
}

// GetDownloadURL returns a URL for downloading content
func (b *FSBackend) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {
	if b.urlPrefix == "" {
		return "", errors.New("direct download required for file system backend")
	}
	return fmt.Sprintf("%s/download/%s", b.urlPrefix, objectKey), nil
}

// GetReviewURL returns a URL for reviewing content
func (b *FSBackend) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
	if b.urlPrefix == "" {
		return "", errors.New("direct preview required for file system backend")
	}
	return fmt.Sprintf("%s/preview/%s", b.urlPrefix, objectKey), nil
}

// Download downloads content directly from the file system
func (b *FSBackend) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	filePath := filepath.Join(b.baseDir, objectKey)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, errors.New("object not found")
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete deletes content from the file system
func (b *FSBackend) Delete(ctx context.Context, objectKey string) error {
	filePath := filepath.Join(b.baseDir, objectKey)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.New("object not found")
	}

	// Delete file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
