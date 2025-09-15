// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
)

// StorageBackendRepository is an in-memory implementation of the StorageBackendRepository interface
type StorageBackendRepository struct {
	mu       sync.RWMutex
	backends map[string]*domain.StorageBackend
}

// NewStorageBackendRepository creates a new in-memory storage backend repository
func NewStorageBackendRepository() repository.StorageBackendRepository {
	return &StorageBackendRepository{
		backends: make(map[string]*domain.StorageBackend),
	}
}

// Create adds a new storage backend to the repository
func (r *StorageBackendRepository) Create(ctx context.Context, backend *domain.StorageBackend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[backend.Name]; exists {
		return errors.New("storage backend with this name already exists")
	}

	r.backends[backend.Name] = backend
	return nil
}

// Get retrieves a storage backend by name
func (r *StorageBackendRepository) Get(ctx context.Context, name string) (*domain.StorageBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, exists := r.backends[name]
	if !exists {
		return nil, errors.New("storage backend not found")
	}

	return backend, nil
}

// Update updates an existing storage backend
func (r *StorageBackendRepository) Update(ctx context.Context, backend *domain.StorageBackend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[backend.Name]; !exists {
		return errors.New("storage backend not found")
	}

	r.backends[backend.Name] = backend
	return nil
}

// Delete removes a storage backend by name
func (r *StorageBackendRepository) Delete(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[name]; !exists {
		return errors.New("storage backend not found")
	}

	delete(r.backends, name)
	return nil
}

// List retrieves all storage backends
func (r *StorageBackendRepository) List(ctx context.Context) ([]*domain.StorageBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.StorageBackend
	for _, backend := range r.backends {
		result = append(result, backend)
	}

	return result, nil
}
