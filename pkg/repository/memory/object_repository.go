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

// ObjectRepository is an in-memory implementation of the ObjectRepository interface
type ObjectRepository struct {
	mu      sync.RWMutex
	objects map[uuid.UUID]*domain.Object
}

// NewObjectRepository creates a new in-memory object repository
func NewObjectRepository() repository.ObjectRepository {
	return &ObjectRepository{
		objects: make(map[uuid.UUID]*domain.Object),
	}
}

// Create adds a new object to the repository
func (r *ObjectRepository) Create(ctx context.Context, object *domain.Object) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.objects[object.ID]; exists {
		return errors.New("object already exists")
	}

	r.objects[object.ID] = object
	return nil
}

// Get retrieves an object by ID
func (r *ObjectRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	object, exists := r.objects[id]
	if !exists {
		return nil, errors.New("object not found")
	}

	return object, nil
}

// GetByContentID retrieves objects by content ID
func (r *ObjectRepository) GetByContentID(ctx context.Context, contentID uuid.UUID) ([]*domain.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Object
	for _, object := range r.objects {
		if object.ContentID == contentID {
			result = append(result, object)
		}
	}

	return result, nil
}

// GetByObjectKeyAndStorageBackend retrieves a non-deleted object by object key and storage backend name
func (r *ObjectRepository) GetByObjectKeyAndStorageBackendName(ctx context.Context, objectKey string, storageBackendName string) (*domain.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, object := range r.objects {
		// Check if object matches both object key and storage backend name
		if object.ObjectKey == objectKey && object.StorageBackendName == storageBackendName {
			return object, nil
		}
	}

	return nil, errors.New("object not found")
}

// Update updates an existing object
func (r *ObjectRepository) Update(ctx context.Context, object *domain.Object) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.objects[object.ID]; !exists {
		return errors.New("object not found")
	}

	r.objects[object.ID] = object
	return nil
}

// Delete removes an object by ID
func (r *ObjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.objects[id]; !exists {
		return errors.New("object not found")
	}

	delete(r.objects, id)
	return nil
}
