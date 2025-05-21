package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
)

// StorageBackendRepository is an in-memory implementation of the StorageBackendRepository interface
type StorageBackendRepository struct {
	mu       sync.RWMutex
	backends map[uuid.UUID]*domain.StorageBackend
}

// NewStorageBackendRepository creates a new in-memory storage backend repository
func NewStorageBackendRepository() repository.StorageBackendRepository {
	return &StorageBackendRepository{
		backends: make(map[uuid.UUID]*domain.StorageBackend),
	}
}

// Create adds a new storage backend to the repository
func (r *StorageBackendRepository) Create(ctx context.Context, backend *domain.StorageBackend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[backend.ID]; exists {
		return errors.New("storage backend already exists")
	}

	// Check for duplicate name
	for _, b := range r.backends {
		if b.Name == backend.Name {
			return errors.New("storage backend with this name already exists")
		}
	}

	r.backends[backend.ID] = backend
	return nil
}

// Get retrieves a storage backend by ID
func (r *StorageBackendRepository) Get(ctx context.Context, id uuid.UUID) (*domain.StorageBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, exists := r.backends[id]
	if !exists {
		return nil, errors.New("storage backend not found")
	}

	return backend, nil
}

// GetByName retrieves a storage backend by name
func (r *StorageBackendRepository) GetByName(ctx context.Context, name string) (*domain.StorageBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, backend := range r.backends {
		if backend.Name == name {
			return backend, nil
		}
	}

	return nil, errors.New("storage backend not found")
}

// Update updates an existing storage backend
func (r *StorageBackendRepository) Update(ctx context.Context, backend *domain.StorageBackend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[backend.ID]; !exists {
		return errors.New("storage backend not found")
	}

	// Check for duplicate name
	for id, b := range r.backends {
		if b.Name == backend.Name && id != backend.ID {
			return errors.New("storage backend with this name already exists")
		}
	}

	r.backends[backend.ID] = backend
	return nil
}

// Delete removes a storage backend by ID
func (r *StorageBackendRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[id]; !exists {
		return errors.New("storage backend not found")
	}

	delete(r.backends, id)
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
