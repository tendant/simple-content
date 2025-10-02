// Deprecated: This package is deprecated as of 2025-10-01 and will be removed in 3 months.
// Please migrate to github.com/tendant/simple-content/pkg/simplecontent which provides:
//   - Unified content operations (UploadContent, UploadDerivedContent)
//   - Better error handling with typed errors
//   - Cleaner API with fewer steps
//   - Full documentation in pkg/simplecontent/service.go
// See MIGRATION_FROM_LEGACY.md for migration guide.
package service

import (
	"context"
	"time"

	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
)

// StorageBackendService handles storage backend operations
type StorageBackendService struct {
	storageBackendRepo repository.StorageBackendRepository
}

// NewStorageBackendService creates a new storage backend service
func NewStorageBackendService(
	storageBackendRepo repository.StorageBackendRepository,
) *StorageBackendService {
	return &StorageBackendService{
		storageBackendRepo: storageBackendRepo,
	}
}

// CreateStorageBackend creates a new storage backend
func (s *StorageBackendService) CreateStorageBackend(
	ctx context.Context,
	name string,
	backendType string,
	config map[string]interface{},
) (*domain.StorageBackend, error) {
	now := time.Now()
	backend := &domain.StorageBackend{
		Name:      name,
		Type:      backendType,
		Config:    config,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.storageBackendRepo.Create(ctx, backend); err != nil {
		return nil, err
	}

	return backend, nil
}

// GetStorageBackend retrieves a storage backend by name
func (s *StorageBackendService) GetStorageBackend(ctx context.Context, name string) (*domain.StorageBackend, error) {
	return s.storageBackendRepo.Get(ctx, name)
}

// UpdateStorageBackend updates a storage backend
func (s *StorageBackendService) UpdateStorageBackend(ctx context.Context, backend *domain.StorageBackend) error {
	backend.UpdatedAt = time.Now()
	return s.storageBackendRepo.Update(ctx, backend)
}

// DeleteStorageBackend deletes a storage backend
func (s *StorageBackendService) DeleteStorageBackend(ctx context.Context, name string) error {
	return s.storageBackendRepo.Delete(ctx, name)
}

// ListStorageBackends lists all storage backends
func (s *StorageBackendService) ListStorageBackends(ctx context.Context) ([]*domain.StorageBackend, error) {
	return s.storageBackendRepo.List(ctx)
}
