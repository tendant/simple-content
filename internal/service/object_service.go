package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
	"github.com/tendant/simple-content/internal/storage"
)

// ObjectService handles object-related operations
type ObjectService struct {
	objectRepo         repository.ObjectRepository
	objectMetadataRepo repository.ObjectMetadataRepository
	storageBackendRepo repository.StorageBackendRepository
	defaultBackend     storage.Backend
	backends           map[string]storage.Backend
}

// NewObjectService creates a new object service
func NewObjectService(
	objectRepo repository.ObjectRepository,
	objectMetadataRepo repository.ObjectMetadataRepository,
	storageBackendRepo repository.StorageBackendRepository,
	defaultBackend storage.Backend,
) *ObjectService {
	return &ObjectService{
		objectRepo:         objectRepo,
		objectMetadataRepo: objectMetadataRepo,
		storageBackendRepo: storageBackendRepo,
		defaultBackend:     defaultBackend,
		backends:           make(map[string]storage.Backend),
	}
}

// RegisterBackend registers a storage backend
func (s *ObjectService) RegisterBackend(name string, backend storage.Backend) {
	s.backends[name] = backend
}

// GetBackend returns a storage backend by name
func (s *ObjectService) GetBackend(name string) (storage.Backend, error) {
	backend, exists := s.backends[name]
	if !exists {
		return nil, fmt.Errorf("storage backend not found: %s", name)
	}
	return backend, nil
}

// CreateObject creates a new object
func (s *ObjectService) CreateObject(
	ctx context.Context,
	contentID uuid.UUID,
	storageBackendName string,
	version int,
) (*domain.Object, error) {
	// Verify storage backend exists
	storageBackend, err := s.storageBackendRepo.Get(ctx, storageBackendName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	objectID := uuid.New()
	objectKey := fmt.Sprintf("%s/%s", contentID, objectID)

	object := &domain.Object{
		ID:                 objectID,
		ContentID:          contentID,
		StorageBackendName: storageBackendName,
		Version:            version,
		ObjectKey:          objectKey,
		Status:             domain.ObjectStatusCreated,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.objectRepo.Create(ctx, object); err != nil {
		return nil, err
	}

	// Set initial metadata
	objectMetadata := &domain.ObjectMetadata{
		ObjectID: objectID,
		Metadata: map[string]interface{}{
			"storage_backend_type": storageBackend.Type,
			"object_type":          object.ObjectType,
			"file_name":            object.FileName,
		},
	}
	if err := s.objectMetadataRepo.Set(ctx, objectMetadata); err != nil {
		return nil, err
	}

	return object, nil
}

// GetObject retrieves an object by ID
func (s *ObjectService) GetObject(ctx context.Context, id uuid.UUID) (*domain.Object, error) {
	return s.objectRepo.Get(ctx, id)
}

// GetObjectsByContentID retrieves objects by content ID
func (s *ObjectService) GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*domain.Object, error) {
	return s.objectRepo.GetByContentID(ctx, contentID)
}

// UpdateObject updates an object
func (s *ObjectService) UpdateObject(ctx context.Context, object *domain.Object) error {
	object.UpdatedAt = time.Now()
	return s.objectRepo.Update(ctx, object)
}

// DeleteObject deletes an object
func (s *ObjectService) DeleteObject(ctx context.Context, id uuid.UUID) error {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Get the storage backend
	storageBackend, err := s.storageBackendRepo.Get(ctx, object.StorageBackendName)
	if err != nil {
		return err
	}

	// Get the backend implementation
	var backend storage.Backend
	if storageBackend.Type == "memory" {
		backend = s.defaultBackend
	} else {
		backend, err = s.GetBackend(object.StorageBackendName)
		if err != nil {
			return err
		}
	}

	// Delete the object from storage
	if err := backend.Delete(ctx, object.ObjectKey); err != nil {
		return err
	}

	// Delete the object from the repository
	return s.objectRepo.Delete(ctx, id)
}

// UploadObject uploads an object
func (s *ObjectService) UploadObject(ctx context.Context, id uuid.UUID, reader io.Reader) error {

	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		slog.Error("Failed to get object", "err", err)
		return err
	}

	// Get the storage backend
	storageBackend, err := s.storageBackendRepo.Get(ctx, object.StorageBackendName)
	if err != nil {
		slog.Error("Failed to get storage backend", "err", err)
		return err
	}

	// Get the backend implementation
	var backend storage.Backend
	if storageBackend.Type == "memory" {
		backend = s.defaultBackend
	} else {
		backend, err = s.GetBackend(object.StorageBackendName)
		if err != nil {
			slog.Error("Failed to get backend", "err", err)
			return err
		}
	}

	// Upload the object
	if err := backend.Upload(ctx, object.ObjectKey, reader); err != nil {
		slog.Error("Failed to upload object", "err", err)
		return err
	}

	// Get object meta from storage backend
	objectMeta, err := backend.GetObjectMeta(ctx, object.ObjectKey)
	if err != nil {
		slog.Error("Failed to get object meta", "err", err)
		return err
	}

	// Update object metadata
	updatedTime := time.Now().UTC()
	objectMetaData := &domain.ObjectMetadata{
		ObjectID:  object.ID,
		ETag:      objectMeta.ETag,
		SizeBytes: objectMeta.Size,
		MimeType:  objectMeta.ContentType,
		UpdatedAt: updatedTime,
	}
	if err := s.objectMetadataRepo.Set(ctx, objectMetaData); err != nil {
		slog.Error("Failed to update object metadata", "err", err)
		return err
	}

	// Update object status
	object.Status = domain.ObjectStatusUploaded
	object.UpdatedAt = updatedTime
	return s.objectRepo.Update(ctx, object)
}

// DownloadObject downloads an object
func (s *ObjectService) DownloadObject(ctx context.Context, id uuid.UUID) (io.ReadCloser, error) {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get the storage backend
	storageBackend, err := s.storageBackendRepo.Get(ctx, object.StorageBackendName)
	if err != nil {
		return nil, err
	}

	// Get the backend implementation
	var backend storage.Backend
	if storageBackend.Type == "memory" {
		backend = s.defaultBackend
	} else {
		backend, err = s.GetBackend(object.StorageBackendName)
		if err != nil {
			return nil, err
		}
	}

	// Download the object
	return backend.Download(ctx, object.ObjectKey)
}

// GetUploadURL gets a URL for uploading an object
func (s *ObjectService) GetUploadURL(ctx context.Context, id uuid.UUID) (string, error) {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return "", err
	}

	// Get the storage backend
	storageBackend, err := s.storageBackendRepo.Get(ctx, object.StorageBackendName)
	if err != nil {
		return "", err
	}

	// Get the backend implementation
	var backend storage.Backend
	if storageBackend.Type == "memory" {
		backend = s.defaultBackend
	} else {
		backend, err = s.GetBackend(object.StorageBackendName)
		if err != nil {
			return "", err
		}
	}

	// Get the upload URL
	return backend.GetUploadURL(ctx, object.ObjectKey)
}

// GetDownloadURL gets a URL for downloading an object
func (s *ObjectService) GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return "", err
	}

	// Get the storage backend
	storageBackend, err := s.storageBackendRepo.Get(ctx, object.StorageBackendName)
	if err != nil {
		return "", err
	}

	// Get the backend implementation
	var backend storage.Backend
	if storageBackend.Type == "memory" {
		backend = s.defaultBackend
	} else {
		backend, err = s.GetBackend(object.StorageBackendName)
		if err != nil {
			return "", err
		}
	}

	// Get the download URL
	// FIXME: download filename
	return backend.GetDownloadURL(ctx, object.ObjectKey, "")
}

// SetObjectMetadata sets metadata for an object
func (s *ObjectService) SetObjectMetadata(ctx context.Context, objectID uuid.UUID, metadata map[string]interface{}) error {
	// Verify object exists
	if _, err := s.objectRepo.Get(ctx, objectID); err != nil {
		return err
	}

	// Create or update the object metadata
	objectMetadata := &domain.ObjectMetadata{
		ObjectID: objectID,
		Metadata: metadata,
	}
	return s.objectMetadataRepo.Set(ctx, objectMetadata)
}

// GetObjectMetadata retrieves metadata for an object
func (s *ObjectService) GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (map[string]interface{}, error) {
	// Verify object exists
	if _, err := s.objectRepo.Get(ctx, objectID); err != nil {
		return nil, err
	}

	// Get the object metadata
	objectMetadata, err := s.objectMetadataRepo.Get(ctx, objectID)
	if err != nil {
		return nil, err
	}

	return objectMetadata.Metadata, nil
}
