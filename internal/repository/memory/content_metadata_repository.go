package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/repository"
)

// ContentMetadataRepository is an in-memory implementation of the ContentMetadataRepository interface
type ContentMetadataRepository struct {
	mu       sync.RWMutex
	metadata map[uuid.UUID]map[string]interface{}
}

// NewContentMetadataRepository creates a new in-memory content metadata repository
func NewContentMetadataRepository() repository.ContentMetadataRepository {
	return &ContentMetadataRepository{
		metadata: make(map[uuid.UUID]map[string]interface{}),
	}
}

// Set sets metadata for a content
func (r *ContentMetadataRepository) Set(ctx context.Context, contentID uuid.UUID, metadata map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata[contentID] = metadata
	return nil
}

// Get retrieves metadata for a content
func (r *ContentMetadataRepository) Get(ctx context.Context, contentID uuid.UUID) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[contentID]
	if !exists {
		return nil, errors.New("metadata not found for content")
	}

	return metadata, nil
}
