package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/repository"
	"github.com/tendant/simple-content/internal/storage"
	"github.com/tendant/simple-content/pkg/model"
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
) (*model.Object, error) {
	// Verify storage backend exists
	_, err := s.GetBackend(storageBackendName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	objectID := uuid.New()
	objectKey := fmt.Sprintf("%s/%s", contentID, objectID)

	object := &model.Object{
		ID:                 objectID,
		ContentID:          contentID,
		StorageBackendName: storageBackendName,
		Version:            version,
		ObjectKey:          objectKey,
		Status:             model.ObjectStatusCreated,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.objectRepo.Create(ctx, object); err != nil {
		return nil, err
	}

	// Set initial metadata
	objectMetadata := &model.ObjectMetadata{
		ObjectID: objectID,
		Metadata: map[string]interface{}{
			"mime_type": object.ObjectType,
			"file_name": object.FileName,
		},
	}
	if err := s.objectMetadataRepo.Set(ctx, objectMetadata); err != nil {
		return nil, err
	}

	return object, nil
}

// GetObject retrieves an object by ID
func (s *ObjectService) GetObject(ctx context.Context, id uuid.UUID) (*model.Object, error) {
	return s.objectRepo.Get(ctx, id)
}

// GetObjectsByContentID retrieves objects by content ID
func (s *ObjectService) GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*model.Object, error) {
	return s.objectRepo.GetByContentID(ctx, contentID)
}

// UpdateObject updates an object
func (s *ObjectService) UpdateObject(ctx context.Context, object *model.Object) error {
	object.UpdatedAt = time.Now()
	return s.objectRepo.Update(ctx, object)
}

// DeleteObject deletes an object
func (s *ObjectService) DeleteObject(ctx context.Context, id uuid.UUID) error {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return err
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

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		slog.Error("Failed to get backend", "err", err)
		return err
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
	objectMetaData := &model.ObjectMetadata{
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
	object.Status = model.ObjectStatusUploaded
	object.UpdatedAt = updatedTime
	return s.objectRepo.Update(ctx, object)
}

// DownloadObject downloads an object
func (s *ObjectService) DownloadObject(ctx context.Context, id uuid.UUID) (io.ReadCloser, error) {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return nil, err
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

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return "", err
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

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return "", err
	}

	// Get the download URL
	// FIXME: download filename
	return backend.GetDownloadURL(ctx, object.ObjectKey, object.FileName)
}

// GetPreviewURL gets a URL for previewing an object
func (s *ObjectService) GetPreviewURL(ctx context.Context, id uuid.UUID) (string, error) {
	object, err := s.objectRepo.Get(ctx, id)
	if err != nil {
		return "", err
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return "", err
	}

	// Get the preview URL
	return backend.GetPreviewURL(ctx, object.ObjectKey)
}

// SetObjectMetadata sets metadata for an object
func (s *ObjectService) SetObjectMetadata(ctx context.Context, objectID uuid.UUID, metadata map[string]interface{}) error {
	// Verify object exists
	if _, err := s.objectRepo.Get(ctx, objectID); err != nil {
		return err
	}

	// Create or update the object metadata
	objectMetadata := &model.ObjectMetadata{
		ObjectID:  objectID,
		UpdatedAt: time.Now().UTC(),
	}
	if _, ok := metadata["etag"]; ok {
		if etag, ok := metadata["etag"].(string); ok {
			objectMetadata.ETag = etag
		}
	}
	if _, ok := metadata["size_bytes"]; ok {
		if size_bytes, ok := metadata["size_bytes"].(int64); ok {
			objectMetadata.SizeBytes = size_bytes
		}
	}
	if _, ok := metadata["mime_type"]; ok {
		if mime_type, ok := metadata["mime_type"].(string); ok {
			objectMetadata.MimeType = mime_type
		}
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

// GetObjectMetaFromStorage retrieves metadata for an object directly from the storage backend
// This is useful when the upload happens client-side and we need to update our metadata
func (s *ObjectService) GetObjectMetaFromStorage(ctx context.Context, objectID uuid.UUID) (*storage.ObjectMeta, error) {
	// Get the object
	object, err := s.objectRepo.Get(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}

	// Get object meta from storage backend
	objectMeta, err := backend.GetObjectMeta(ctx, object.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object meta from storage: %w", err)
	}

	return objectMeta, nil
}

// UpdateObjectMetaFromStorage updates object metadata using information retrieved from the storage backend
// This is useful after a client-side upload to update our metadata and object status
func (s *ObjectService) UpdateObjectMetaFromStorage(ctx context.Context, objectID uuid.UUID) (*model.ObjectMetadata, error) {
	// Get the object
	object, err := s.objectRepo.Get(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// Get object meta from storage
	objectMeta, err := s.GetObjectMetaFromStorage(ctx, objectID)
	if err != nil {
		return nil, err
	}

	// Update object metadata
	updatedTime := time.Now().UTC()
	metadata := make(map[string]interface{}, len(objectMeta.Metadata))
	for k, v := range objectMeta.Metadata {
		metadata[k] = v
	}
	objectMetaData := &model.ObjectMetadata{
		ObjectID:  object.ID,
		ETag:      objectMeta.ETag,
		SizeBytes: objectMeta.Size,
		MimeType:  objectMeta.ContentType,
		UpdatedAt: updatedTime,
		CreatedAt: object.CreatedAt,
		Metadata:  metadata,
	}
	if err := s.objectMetadataRepo.Set(ctx, objectMetaData); err != nil {
		return nil, fmt.Errorf("failed to update object metadata: %w", err)
	}

	// Update object status
	object.Status = model.ObjectStatusUploaded
	object.UpdatedAt = updatedTime
	return objectMetaData, s.objectRepo.Update(ctx, object)
}
