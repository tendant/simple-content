// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	model "github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
	"github.com/tendant/simple-content/internal/storage"
	"github.com/tendant/simple-content/pkg/utils"
)

// ObjectService handles object-related operations
type ObjectService struct {
	objectRepo          repository.ObjectRepository
	objectMetadataRepo  repository.ObjectMetadataRepository
	contentRepo         repository.ContentRepository
	contentMetadataRepo repository.ContentMetadataRepository
	backends            map[string]storage.Backend
}

// NewObjectService creates a new object service
func NewObjectService(
	objectRepo repository.ObjectRepository,
	objectMetadataRepo repository.ObjectMetadataRepository,
	contentRepo repository.ContentRepository,
	contentMetadataRepo repository.ContentMetadataRepository,
) *ObjectService {
	return &ObjectService{
		objectRepo:          objectRepo,
		objectMetadataRepo:  objectMetadataRepo,
		contentRepo:         contentRepo,
		contentMetadataRepo: contentMetadataRepo,
		backends:            make(map[string]storage.Backend),
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

// CreateObjectParams contains parameters for creating an object
type CreateObjectParams struct {
	ContentID          uuid.UUID
	StorageBackendName string
	Version            int
	ObjectKey          string
}

// CreateObject creates a new object for a content
func (s *ObjectService) CreateObject(
	ctx context.Context,
	params CreateObjectParams,
) (*model.Object, error) {
	// Verify storage backend exists
	_, err := s.GetBackend(params.StorageBackendName)
	if err != nil {
		return nil, err
	}

	// Get content metadata (optional)
	var contentMetadata *model.ContentMetadata
	contentMetadata, err = s.contentMetadataRepo.Get(ctx, params.ContentID)
	if err != nil {
		// Log the error but continue without metadata
		slog.Warn("Warning: %v", "err", err)
		// Don't return error, just proceed with nil metadata
	}

	now := time.Now().UTC()
	objectID := uuid.New()

	// Use provided ObjectKey if available, otherwise generate one
	objectKey := params.ObjectKey
	if objectKey == "" {
		objectKey = GenerateObjectKey(params.ContentID, objectID, contentMetadata)
	}

	object := &model.Object{
		ID:                 objectID,
		ContentID:          params.ContentID,
		StorageBackendName: params.StorageBackendName,
		Version:            params.Version,
		ObjectKey:          objectKey,
		Status:             model.ObjectStatusCreated,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Add metadata-derived fields if available
	if contentMetadata != nil {
		object.ObjectType = contentMetadata.MimeType
		object.FileName = contentMetadata.FileName
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

type UploadObjectWithMetadataParams struct {
	ObjectID uuid.UUID
	MimeType string
}

func (s *ObjectService) UploadObjectWithMetadata(ctx context.Context, reader io.Reader, params UploadObjectWithMetadataParams) error {
	object, err := s.objectRepo.Get(ctx, params.ObjectID)
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

	// Detect mimeType if not provided
	mimeType := params.MimeType
	if mimeType == "" {
		buffer := make([]byte, 512)
		var buf bytes.Buffer
		teeReader := io.TeeReader(reader, &buf)
		n, err := teeReader.Read(buffer)
		if err != nil && err != io.EOF {
			slog.Error("Failed to read file for MIME type detection", "err", err)
			return fmt.Errorf("failed to read for MIME detection: %w", err)
		}
		mimeType = http.DetectContentType(buffer[:n])
		reader = io.MultiReader(bytes.NewReader(buffer[:n]), reader)
	}

	// Upload the object
	err = backend.UploadWithParams(ctx, reader, storage.UploadParams{
		ObjectKey: object.ObjectKey,
		MimeType:  mimeType,
	})
	if err != nil {
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

	// Get the download URL with sanitized filename
	sanitizedFilename := utils.SanitizeFilename(object.FileName)
	return backend.GetDownloadURL(ctx, object.ObjectKey, sanitizedFilename)
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

	// Get existing metadata or create new
	var objectMetadata *model.ObjectMetadata
	existing, err := s.objectMetadataRepo.Get(ctx, objectID)
	if err == nil {
		// Update existing metadata
		objectMetadata = existing
		// If Metadata is nil, initialize it
		if objectMetadata.Metadata == nil {
			objectMetadata.Metadata = make(map[string]interface{})
		}
	} else {
		// Create new metadata
		objectMetadata = &model.ObjectMetadata{
			ObjectID:  objectID,
			CreatedAt: time.Now().UTC(),
			Metadata:  make(map[string]interface{}),
		}
	}

	// Update the timestamp
	objectMetadata.UpdatedAt = time.Now().UTC()

	// Extract specific fields and also store in the Metadata map
	if _, ok := metadata["etag"]; ok {
		if etag, ok := metadata["etag"].(string); ok {
			objectMetadata.ETag = etag
			objectMetadata.Metadata["etag"] = etag
		}
	}
	if _, ok := metadata["size_bytes"]; ok {
		if size_bytes, ok := metadata["size_bytes"].(int64); ok {
			objectMetadata.SizeBytes = size_bytes
			objectMetadata.Metadata["size_bytes"] = size_bytes
		}
	}
	if _, ok := metadata["mime_type"]; ok {
		if mime_type, ok := metadata["mime_type"].(string); ok {
			objectMetadata.MimeType = mime_type
			objectMetadata.Metadata["mime_type"] = mime_type
		}
	}

	// Copy all other metadata fields
	for k, v := range metadata {
		if k != "etag" && k != "size_bytes" && k != "mime_type" {
			objectMetadata.Metadata[k] = v
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

// GetObjectMetaFromStorageByObjectKeyAndStorageBackendName retrieves metadata for an object directly from the storage backend
// This is useful when the upload happens client-side and we need to update our metadata
func (s *ObjectService) GetObjectMetaFromStorageByObjectKeyAndStorageBackendName(ctx context.Context, objectKey string, storageBackendName string) (*storage.ObjectMeta, error) {

	// Get the backend implementation
	backend, err := s.GetBackend(storageBackendName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}

	// Get object meta from storage backend
	objectMeta, err := backend.GetObjectMeta(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object meta from storage: %w", err)
	}

	return objectMeta, nil
}

// GetObjectByObjectKeyAndStorageBackendNameParams contains parameters for looking up object by object key and storage backend name
type GetObjectByObjectKeyAndStorageBackendNameParams struct {
	StorageBackendName string
	ObjectKey          string
}

// GetObjectByObjectKeyAndStorageBackendName looks for object by object_key and storage_backend_name and returns the object UUID if found, nil if not exists
func (s *ObjectService) GetObjectByObjectKeyAndStorageBackendName(ctx context.Context, params GetObjectByObjectKeyAndStorageBackendNameParams) (uuid.UUID, error) {

	object, err := s.objectRepo.GetByObjectKeyAndStorageBackendName(ctx, params.ObjectKey, params.StorageBackendName)
	if err != nil {
		if strings.Contains(err.Error(), "object not found") {
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("failed to get object by key and storage backend name: %w", err)
	}
	return object.ID, nil
}

// GenerateObjectKey creates an object key based on content ID, object ID and content metadata
func GenerateObjectKey(contentID, objectID uuid.UUID, contentMetadata *model.ContentMetadata) string {
	if contentMetadata != nil && contentMetadata.FileName != "" {
		return fmt.Sprintf("C/%s/%s/%s", contentID, objectID, contentMetadata.FileName)
	}
	return fmt.Sprintf("C/%s/%s", contentID, objectID)
}

// GetLatestVersionObject returns the object with the highest version number from a slice of objects.
// If there are multiple objects with the same highest version, it returns the first one found.
func GetLatestVersionObject(objects []*model.Object) *model.Object {
	if len(objects) == 0 {
		return nil
	}

	latestObject := objects[0]
	for _, obj := range objects[1:] {
		if obj.Version > latestObject.Version {
			latestObject = obj
		}
	}

	return latestObject
}

// UpdateObjectMetaFromStorage updates object metadata using information retrieved from the storage backend
// This is useful after a client-side upload to update our metadata and object status
func (s *ObjectService) UpdateObjectMetaFromStorage(ctx context.Context, objectID uuid.UUID) (model.ObjectMetadata, error) {
	// Get the object
	object, err := s.objectRepo.Get(ctx, objectID)
	if err != nil {
		return model.ObjectMetadata{}, fmt.Errorf("failed to get object: %w", err)
	}

	// Get object meta from storage
	objectMeta, err := s.GetObjectMetaFromStorage(ctx, objectID)
	if err != nil {
		return model.ObjectMetadata{}, err
	}

	// Update object metadata
	updatedTime := time.Now().UTC()
	metadata := make(map[string]interface{}, len(objectMeta.Metadata))
	for k, v := range objectMeta.Metadata {
		metadata[k] = v
	}
	objectMetaData := model.ObjectMetadata{
		ObjectID:  object.ID,
		ETag:      objectMeta.ETag,
		SizeBytes: objectMeta.Size,
		MimeType:  objectMeta.ContentType,
		UpdatedAt: updatedTime,
		CreatedAt: object.CreatedAt,
		Metadata:  metadata,
	}
	if err := s.objectMetadataRepo.Set(ctx, &objectMetaData); err != nil {
		return model.ObjectMetadata{}, fmt.Errorf("failed to update object metadata: %w", err)
	}

	// Update object status
	object.Status = model.ObjectStatusUploaded
	object.UpdatedAt = updatedTime
	if err := s.objectRepo.Update(ctx, object); err != nil {
		return model.ObjectMetadata{}, fmt.Errorf("failed to update object: %w", err)
	}
	return objectMetaData, nil
}
