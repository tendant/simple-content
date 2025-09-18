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

// ObjectMetadataRepository is an in-memory implementation of the ObjectMetadataRepository interface
type ObjectMetadataRepository struct {
	mu       sync.RWMutex
	metadata map[uuid.UUID]*domain.ObjectMetadata
}

// NewObjectMetadataRepository creates a new in-memory object metadata repository
func NewObjectMetadataRepository() repository.ObjectMetadataRepository {
	return &ObjectMetadataRepository{
		metadata: make(map[uuid.UUID]*domain.ObjectMetadata),
	}
}

// Set sets metadata for an object
func (r *ObjectMetadataRepository) Set(ctx context.Context, metadata *domain.ObjectMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata[metadata.ObjectID] = metadata
	return nil
}

// Get retrieves metadata for an object
func (r *ObjectMetadataRepository) Get(ctx context.Context, objectID uuid.UUID) (*domain.ObjectMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[objectID]
	if !exists {
		return nil, errors.New("metadata not found for object")
	}

	return metadata, nil
}
