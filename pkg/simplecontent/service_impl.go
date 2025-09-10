package simplecontent

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/google/uuid"
)

// service implements the Service interface
type service struct {
	repository Repository
	blobStores map[string]BlobStore
	eventSink  EventSink
	previewer  Previewer
}

// Option represents a functional option for configuring the service
type Option func(*service)

// WithRepository sets the repository for the service
func WithRepository(repo Repository) Option {
	return func(s *service) {
		s.repository = repo
	}
}

// WithBlobStore adds a blob storage backend
func WithBlobStore(name string, store BlobStore) Option {
	return func(s *service) {
		if s.blobStores == nil {
			s.blobStores = make(map[string]BlobStore)
		}
		s.blobStores[name] = store
	}
}

// WithEventSink sets the event sink for the service
func WithEventSink(sink EventSink) Option {
	return func(s *service) {
		s.eventSink = sink
	}
}

// WithPreviewer sets the previewer for the service
func WithPreviewer(previewer Previewer) Option {
	return func(s *service) {
		s.previewer = previewer
	}
}

// New creates a new service instance with the given options
func New(options ...Option) (Service, error) {
	s := &service{
		blobStores: make(map[string]BlobStore),
	}

	for _, option := range options {
		option(s)
	}

	if s.repository == nil {
		return nil, fmt.Errorf("repository is required")
	}

	return s, nil
}

// Content operations

func (s *service) CreateContent(ctx context.Context, req CreateContentRequest) (*Content, error) {
	now := time.Now().UTC()
	content := &Content{
		ID:             uuid.New(),
		TenantID:       req.TenantID,
		OwnerID:        req.OwnerID,
		Name:           req.Name,
		Description:    req.Description,
		DocumentType:   req.DocumentType,
        DerivationType: req.DerivationType,
        Status:         string(ContentStatusCreated),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repository.CreateContent(ctx, content); err != nil {
		return nil, &ContentError{
			ContentID: content.ID,
			Op:        "create",
			Err:       err,
		}
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ContentCreated(ctx, content); err != nil {
			// Log error but don't fail the operation
			// TODO: Add proper logging
		}
	}

	return content, nil
}

func (s *service) CreateDerivedContent(ctx context.Context, req CreateDerivedContentRequest) (*Content, error) {
    // Verify parent content exists
    _, err := s.repository.GetContent(ctx, req.ParentID)
    if err != nil {
        return nil, fmt.Errorf("parent content not found: %w", err)
    }

    // Infer derivation_type from variant if missing
    if req.DerivationType == "" && req.Variant != "" {
        req.DerivationType = DerivationTypeFromVariant(req.Variant)
    }

    // Create derived content
    now := time.Now().UTC()
    content := &Content{
        ID:             uuid.New(),
        TenantID:       req.TenantID,
        OwnerID:        req.OwnerID,
        Status:         string(ContentStatusCreated),
        DerivationType: NormalizeDerivationType(req.DerivationType),
        CreatedAt:      now,
        UpdatedAt:      now,
    }

	if err := s.repository.CreateContent(ctx, content); err != nil {
		return nil, &ContentError{
			ContentID: content.ID,
			Op:        "create_derived",
			Err:       err,
		}
	}

	// Create content metadata if provided
	if req.Metadata != nil {
		metadata := &ContentMetadata{
			ContentID: content.ID,
			Metadata:  req.Metadata,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.repository.SetContentMetadata(ctx, metadata); err != nil {
			return nil, fmt.Errorf("failed to create derived content metadata: %w", err)
		}
	}

    // Create derived content relationship
    // Determine variant to persist in relationship
    variant := req.Variant
    if variant == "" {
        variant = req.DerivationType
    }

    _, err = s.repository.CreateDerivedContentRelationship(ctx, CreateDerivedContentParams{
        ParentID:           req.ParentID,
        DerivedContentID:   content.ID,
        DerivationType:     string(NormalizeVariant(variant)),
        DerivationParams:   req.Metadata,
        ProcessingMetadata: nil,
    })
	if err != nil {
		return nil, fmt.Errorf("failed to create derived content relationship: %w", err)
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ContentCreated(ctx, content); err != nil {
			// Log error but don't fail the operation
		}
	}

	return content, nil
}

func (s *service) GetContent(ctx context.Context, id uuid.UUID) (*Content, error) {
	return s.repository.GetContent(ctx, id)
}

func (s *service) UpdateContent(ctx context.Context, req UpdateContentRequest) error {
	req.Content.UpdatedAt = time.Now().UTC()
	
	if err := s.repository.UpdateContent(ctx, req.Content); err != nil {
		return &ContentError{
			ContentID: req.Content.ID,
			Op:        "update",
			Err:       err,
		}
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ContentUpdated(ctx, req.Content); err != nil {
			// Log error but don't fail the operation
		}
	}

	return nil
}

func (s *service) DeleteContent(ctx context.Context, id uuid.UUID) error {
	if err := s.repository.DeleteContent(ctx, id); err != nil {
		return &ContentError{
			ContentID: id,
			Op:        "delete",
			Err:       err,
		}
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ContentDeleted(ctx, id); err != nil {
			// Log error but don't fail the operation
		}
	}

	return nil
}

func (s *service) ListContent(ctx context.Context, req ListContentRequest) ([]*Content, error) {
	return s.repository.ListContent(ctx, req.OwnerID, req.TenantID)
}

// Content metadata operations

func (s *service) SetContentMetadata(ctx context.Context, req SetContentMetadataRequest) error {
	// Verify content exists
	_, err := s.repository.GetContent(ctx, req.ContentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}

	now := time.Now().UTC()
	metadata := &ContentMetadata{
		ContentID: req.ContentID,
		Tags:      req.Tags,
		FileName:  req.FileName,
		FileSize:  req.FileSize,
		MimeType:  req.ContentType,
		Metadata:  make(map[string]interface{}),
		UpdatedAt: now,
	}

	// Set MIME type
	if req.ContentType != "" {
		metadata.Metadata["mime_type"] = req.ContentType
	}

	// Set file name
	if req.FileName != "" {
		metadata.Metadata["file_name"] = req.FileName
	}

	// Set file size
	if req.FileSize > 0 {
		metadata.Metadata["file_size"] = req.FileSize
	}

	// Copy custom metadata
	if req.CustomMetadata != nil {
		for k, v := range req.CustomMetadata {
			metadata.Metadata[k] = v
		}
	}

	// Add title and description
	if req.Title != "" {
		metadata.Metadata["title"] = req.Title
	}
	if req.Description != "" {
		metadata.Metadata["description"] = req.Description
	}
	if req.CreatedBy != "" {
		metadata.Metadata["created_by"] = req.CreatedBy
	}

	return s.repository.SetContentMetadata(ctx, metadata)
}

func (s *service) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*ContentMetadata, error) {
	return s.repository.GetContentMetadata(ctx, contentID)
}

// Object operations

func (s *service) CreateObject(ctx context.Context, req CreateObjectRequest) (*Object, error) {
	// Verify storage backend exists
	_, err := s.GetBackend(req.StorageBackendName)
	if err != nil {
		return nil, err
	}

	// Get content metadata (optional)
	var contentMetadata *ContentMetadata
	contentMetadata, err = s.repository.GetContentMetadata(ctx, req.ContentID)
	if err != nil {
		// Log the warning but continue
		contentMetadata = nil
	}

	now := time.Now().UTC()
	objectID := uuid.New()

	// Generate object key if not provided
	objectKey := req.ObjectKey
	if objectKey == "" {
		objectKey = s.generateObjectKey(req.ContentID, objectID, contentMetadata)
	}

    object := &Object{
        ID:                 objectID,
        ContentID:          req.ContentID,
        StorageBackendName: req.StorageBackendName,
        ObjectKey:          objectKey,
        Version:            req.Version,
        Status:             string(ObjectStatusCreated),
        CreatedAt:          now,
        UpdatedAt:          now,
    }

	// Add metadata-derived fields if available
	if contentMetadata != nil {
		object.ObjectType = contentMetadata.MimeType
		object.FileName = contentMetadata.FileName
	}

	if err := s.repository.CreateObject(ctx, object); err != nil {
		return nil, &ObjectError{
			ObjectID: objectID,
			Op:       "create",
			Err:      err,
		}
	}

	// Set initial metadata
	objectMetadata := &ObjectMetadata{
		ObjectID: objectID,
		Metadata: map[string]interface{}{
			"mime_type": object.ObjectType,
			"file_name": object.FileName,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repository.SetObjectMetadata(ctx, objectMetadata); err != nil {
		return nil, &ObjectError{
			ObjectID: objectID,
			Op:       "create_metadata",
			Err:      err,
		}
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ObjectCreated(ctx, object); err != nil {
			// Log error but don't fail the operation
		}
	}

	return object, nil
}

func (s *service) GetObject(ctx context.Context, id uuid.UUID) (*Object, error) {
	return s.repository.GetObject(ctx, id)
}

func (s *service) GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*Object, error) {
	return s.repository.GetObjectsByContentID(ctx, contentID)
}

func (s *service) UpdateObject(ctx context.Context, object *Object) error {
	object.UpdatedAt = time.Now().UTC()
	
	if err := s.repository.UpdateObject(ctx, object); err != nil {
		return &ObjectError{
			ObjectID: object.ID,
			Op:       "update",
			Err:      err,
		}
	}

	return nil
}

func (s *service) DeleteObject(ctx context.Context, id uuid.UUID) error {
	if err := s.repository.DeleteObject(ctx, id); err != nil {
		return &ObjectError{
			ObjectID: id,
			Op:       "delete",
			Err:      err,
		}
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ObjectDeleted(ctx, id); err != nil {
			// Log error but don't fail the operation
		}
	}

	return nil
}

// Object upload/download operations

func (s *service) UploadObject(ctx context.Context, id uuid.UUID, reader io.Reader) error {
	object, err := s.repository.GetObject(ctx, id)
	if err != nil {
		return &ObjectError{ObjectID: id, Op: "upload", Err: err}
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return &ObjectError{ObjectID: id, Op: "upload", Err: err}
	}

	// Upload the object
	if err := backend.Upload(ctx, object.ObjectKey, reader); err != nil {
		return &StorageError{
			Backend: object.StorageBackendName,
			Key:     object.ObjectKey,
			Op:      "upload",
			Err:     err,
		}
	}

	// Update object metadata from storage
	if err := s.updateObjectFromStorage(ctx, id); err != nil {
		return err
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ObjectUploaded(ctx, object); err != nil {
			// Log error but don't fail the operation
		}
	}

	return nil
}

func (s *service) UploadObjectWithMetadata(ctx context.Context, reader io.Reader, req UploadObjectWithMetadataRequest) error {
	object, err := s.repository.GetObject(ctx, req.ObjectID)
	if err != nil {
		return &ObjectError{ObjectID: req.ObjectID, Op: "upload_with_metadata", Err: err}
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return &ObjectError{ObjectID: req.ObjectID, Op: "upload_with_metadata", Err: err}
	}

	// Upload the object with parameters
	uploadParams := UploadParams{
		ObjectKey: object.ObjectKey,
		MimeType:  req.MimeType,
	}
	
	if err := backend.UploadWithParams(ctx, reader, uploadParams); err != nil {
		return &StorageError{
			Backend: object.StorageBackendName,
			Key:     object.ObjectKey,
			Op:      "upload_with_params",
			Err:     err,
		}
	}

	// Update object metadata from storage
	if err := s.updateObjectFromStorage(ctx, req.ObjectID); err != nil {
		return err
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ObjectUploaded(ctx, object); err != nil {
			// Log error but don't fail the operation
		}
	}

	return nil
}

func (s *service) DownloadObject(ctx context.Context, id uuid.UUID) (io.ReadCloser, error) {
	object, err := s.repository.GetObject(ctx, id)
	if err != nil {
		return nil, &ObjectError{ObjectID: id, Op: "download", Err: err}
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return nil, &ObjectError{ObjectID: id, Op: "download", Err: err}
	}

	// Download the object
	reader, err := backend.Download(ctx, object.ObjectKey)
	if err != nil {
		return nil, &StorageError{
			Backend: object.StorageBackendName,
			Key:     object.ObjectKey,
			Op:      "download",
			Err:     err,
		}
	}

	return reader, nil
}

func (s *service) GetUploadURL(ctx context.Context, id uuid.UUID) (string, error) {
	object, err := s.repository.GetObject(ctx, id)
	if err != nil {
		return "", &ObjectError{ObjectID: id, Op: "get_upload_url", Err: err}
	}

	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return "", &ObjectError{ObjectID: id, Op: "get_upload_url", Err: err}
	}

	return backend.GetUploadURL(ctx, object.ObjectKey)
}

func (s *service) GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
	object, err := s.repository.GetObject(ctx, id)
	if err != nil {
		return "", &ObjectError{ObjectID: id, Op: "get_download_url", Err: err}
	}

	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return "", &ObjectError{ObjectID: id, Op: "get_download_url", Err: err}
	}

	return backend.GetDownloadURL(ctx, object.ObjectKey, object.FileName)
}

func (s *service) GetPreviewURL(ctx context.Context, id uuid.UUID) (string, error) {
	object, err := s.repository.GetObject(ctx, id)
	if err != nil {
		return "", &ObjectError{ObjectID: id, Op: "get_preview_url", Err: err}
	}

	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return "", &ObjectError{ObjectID: id, Op: "get_preview_url", Err: err}
	}

	return backend.GetPreviewURL(ctx, object.ObjectKey)
}

// Object metadata operations

func (s *service) SetObjectMetadata(ctx context.Context, objectID uuid.UUID, metadata map[string]interface{}) error {
	// Verify object exists
	if _, err := s.repository.GetObject(ctx, objectID); err != nil {
		return &ObjectError{ObjectID: objectID, Op: "set_metadata", Err: err}
	}

	now := time.Now().UTC()
	objectMetadata := &ObjectMetadata{
		ObjectID:  objectID,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}

	// Get existing metadata if it exists
	if existing, err := s.repository.GetObjectMetadata(ctx, objectID); err == nil {
		objectMetadata = existing
		if objectMetadata.Metadata == nil {
			objectMetadata.Metadata = make(map[string]interface{})
		}
	} else {
		objectMetadata.CreatedAt = now
	}

	objectMetadata.UpdatedAt = now

	// Extract specific fields and update
	if etag, ok := metadata["etag"].(string); ok {
		objectMetadata.ETag = etag
	}
	if sizeBytes, ok := metadata["size_bytes"].(int64); ok {
		objectMetadata.SizeBytes = sizeBytes
	}
	if mimeType, ok := metadata["mime_type"].(string); ok {
		objectMetadata.MimeType = mimeType
	}

	// Copy all metadata
	for k, v := range metadata {
		objectMetadata.Metadata[k] = v
	}

	return s.repository.SetObjectMetadata(ctx, objectMetadata)
}

func (s *service) GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (map[string]interface{}, error) {
	// Verify object exists
	if _, err := s.repository.GetObject(ctx, objectID); err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "get_metadata", Err: err}
	}

	objectMetadata, err := s.repository.GetObjectMetadata(ctx, objectID)
	if err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "get_metadata", Err: err}
	}

	return objectMetadata.Metadata, nil
}

func (s *service) UpdateObjectMetaFromStorage(ctx context.Context, objectID uuid.UUID) (*ObjectMetadata, error) {
	// Get the object
	object, err := s.repository.GetObject(ctx, objectID)
	if err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "update_meta_from_storage", Err: err}
	}

	// Get backend
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "update_meta_from_storage", Err: err}
	}

	// Get object meta from storage
	objectMeta, err := backend.GetObjectMeta(ctx, object.ObjectKey)
	if err != nil {
		return nil, &StorageError{
			Backend: object.StorageBackendName,
			Key:     object.ObjectKey,
			Op:      "get_object_meta",
			Err:     err,
		}
	}

	// Update object metadata
	updatedTime := time.Now().UTC()
	metadata := make(map[string]interface{})
	for k, v := range objectMeta.Metadata {
		metadata[k] = v
	}

	objectMetadata := &ObjectMetadata{
		ObjectID:  objectID,
		ETag:      objectMeta.ETag,
		SizeBytes: objectMeta.Size,
		MimeType:  objectMeta.ContentType,
		UpdatedAt: updatedTime,
		CreatedAt: object.CreatedAt,
		Metadata:  metadata,
	}

	if err := s.repository.SetObjectMetadata(ctx, objectMetadata); err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "update_meta_from_storage", Err: err}
	}

	// Update object status
    object.Status = string(ObjectStatusUploaded)
	object.UpdatedAt = updatedTime
	if err := s.repository.UpdateObject(ctx, object); err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "update_meta_from_storage", Err: err}
	}

	return objectMetadata, nil
}

// Storage backend operations

func (s *service) RegisterBackend(name string, backend BlobStore) {
	s.blobStores[name] = backend
}

func (s *service) GetBackend(name string) (BlobStore, error) {
    backend, exists := s.blobStores[name]
    if !exists {
        return nil, fmt.Errorf("%w: %s", ErrStorageBackendNotFound, name)
    }
    return backend, nil
}

// Helper methods

func (s *service) generateObjectKey(contentID, objectID uuid.UUID, contentMetadata *ContentMetadata) string {
	if contentMetadata != nil && contentMetadata.FileName != "" {
		return fmt.Sprintf("C/%s/%s/%s", contentID, objectID, contentMetadata.FileName)
	}
	return fmt.Sprintf("C/%s/%s", contentID, objectID)
}

func (s *service) updateObjectFromStorage(ctx context.Context, objectID uuid.UUID) error {
    _, err := s.UpdateObjectMetaFromStorage(ctx, objectID)
    return err
}

// Derived content helpers
func (s *service) GetDerivedRelationshipByContentID(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error) {
    return s.repository.GetDerivedRelationshipByContentID(ctx, contentID)
}

func (s *service) ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID: &parentID,
    }
    return s.repository.ListDerivedContent(ctx, params)
}
