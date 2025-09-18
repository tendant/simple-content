// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
)

// ContentMetadataRepository is an in-memory implementation of the ContentMetadataRepository interface
type ContentMetadataRepository struct {
	mu       sync.RWMutex
	metadata map[uuid.UUID]*domain.ContentMetadata
}

// NewContentMetadataRepository creates a new in-memory content metadata repository
func NewContentMetadataRepository() repository.ContentMetadataRepository {
	return &ContentMetadataRepository{
		metadata: make(map[uuid.UUID]*domain.ContentMetadata),
	}
}

// Set sets metadata for a content
func (r *ContentMetadataRepository) Set(ctx context.Context, metadata *domain.ContentMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata[metadata.ContentID] = metadata
	return nil
}

// Get retrieves metadata for a content
func (r *ContentMetadataRepository) Get(ctx context.Context, contentID uuid.UUID) (*domain.ContentMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[contentID]
	if !exists {
		return nil, errors.New("metadata not found for content")
	}

	return metadata, nil
}
