package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/repository"
)

// ObjectMetadataRepository is an in-memory implementation of the ObjectMetadataRepository interface
type ObjectMetadataRepository struct {
	mu       sync.RWMutex
	metadata map[uuid.UUID]map[string]interface{}
}

// NewObjectMetadataRepository creates a new in-memory object metadata repository
func NewObjectMetadataRepository() repository.ObjectMetadataRepository {
	return &ObjectMetadataRepository{
		metadata: make(map[uuid.UUID]map[string]interface{}),
	}
}

// Set sets metadata for an object
func (r *ObjectMetadataRepository) Set(ctx context.Context, objectID uuid.UUID, metadata map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.metadata[objectID] = metadata
	return nil
}

// Get retrieves metadata for an object
func (r *ObjectMetadataRepository) Get(ctx context.Context, objectID uuid.UUID) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[objectID]
	if !exists {
		return nil, errors.New("metadata not found for object")
	}

	return metadata, nil
}
