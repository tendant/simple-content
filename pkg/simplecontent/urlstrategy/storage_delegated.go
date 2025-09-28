package urlstrategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// BlobStore interface for URL generation (to avoid circular imports)
type BlobStore interface {
	GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error)
	GetPreviewURL(ctx context.Context, objectKey string) (string, error)
	GetUploadURL(ctx context.Context, objectKey string) (string, error)
}

// StorageDelegatedStrategy delegates URL generation to the storage backends
// This maintains backward compatibility with existing storage backend URL generation
type StorageDelegatedStrategy struct {
	BlobStores map[string]BlobStore
}

// NewStorageDelegatedStrategy creates a new storage-delegated URL strategy
func NewStorageDelegatedStrategy(blobStores map[string]BlobStore) *StorageDelegatedStrategy {
	return &StorageDelegatedStrategy{
		BlobStores: blobStores,
	}
}

// GenerateDownloadURL delegates to the storage backend's GetDownloadURL method
func (s *StorageDelegatedStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Delegate to storage backend (current behavior)
	return backend.GetDownloadURL(ctx, objectKey, "")
}

// GeneratePreviewURL delegates to the storage backend's GetPreviewURL method
func (s *StorageDelegatedStrategy) GeneratePreviewURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Delegate to storage backend (current behavior)
	return backend.GetPreviewURL(ctx, objectKey)
}

// GenerateUploadURL delegates to the storage backend's GetUploadURL method
func (s *StorageDelegatedStrategy) GenerateUploadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Delegate to storage backend (current behavior)
	return backend.GetUploadURL(ctx, objectKey)
}

// Enhanced methods with metadata
func (s *StorageDelegatedStrategy) GenerateDownloadURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Use filename from metadata if available
	filename := ""
	if metadata != nil {
		filename = metadata.FileName
	}

	// Delegate to storage backend with filename
	return backend.GetDownloadURL(ctx, objectKey, filename)
}

// GeneratePreviewURLWithMetadata creates preview URLs with metadata
func (s *StorageDelegatedStrategy) GeneratePreviewURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	// Preview URLs typically don't need metadata in current storage backend interface
	return s.GeneratePreviewURL(ctx, contentID, objectKey, storageBackend)
}