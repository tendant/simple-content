package simplecontent

import (
    "context"
    "fmt"
    "io"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent/objectkey"
    "github.com/tendant/simple-content/pkg/simplecontent/urlstrategy"
)

// service implements both the Service and StorageService interfaces
type service struct {
	repository Repository
	blobStores map[string]BlobStore
	eventSink  EventSink
	previewer  Previewer
	keyGenerator objectkey.Generator
	urlStrategy urlstrategy.URLStrategy // Pluggable URL generation strategy
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

// WithObjectKeyGenerator sets the object key generator for the service
func WithObjectKeyGenerator(generator objectkey.Generator) Option {
	return func(s *service) {
		s.keyGenerator = generator
	}
}

// WithURLStrategy sets the URL generation strategy for the service
func WithURLStrategy(strategy urlstrategy.URLStrategy) Option {
	return func(s *service) {
		s.urlStrategy = strategy
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

	// Set default key generator if none provided
	if s.keyGenerator == nil {
		s.keyGenerator = objectkey.NewRecommendedGenerator()
	}

	// Set default URL strategy if none provided
	if s.urlStrategy == nil {
		s.urlStrategy = urlstrategy.NewDefaultStrategy("/api/v1")
	}

	return s, nil
}

// NewStorageService creates a new service instance that implements StorageService for advanced object operations
func NewStorageService(options ...Option) (StorageService, error) {
	s := &service{
		blobStores: make(map[string]BlobStore),
	}

	for _, option := range options {
		option(s)
	}

	if s.repository == nil {
		return nil, fmt.Errorf("repository is required")
	}

	// Set default key generator if none provided
	if s.keyGenerator == nil {
		s.keyGenerator = objectkey.NewRecommendedGenerator()
	}

	// Set default URL strategy if none provided
	if s.urlStrategy == nil {
		s.urlStrategy = urlstrategy.NewDefaultStrategy("/api/v1")
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
        DerivationType:     req.DerivationType,                 // Store the derivation type (e.g., "thumbnail")
        Variant:            string(NormalizeVariant(variant)),  // Store the specific variant (e.g., "thumbnail_256")
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

// Unified content upload operations

func (s *service) UploadContent(ctx context.Context, req UploadContentRequest) (*Content, error) {
	// Step 1: Create the content
	now := time.Now().UTC()
	content := &Content{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		OwnerID:      req.OwnerID,
		Name:         req.Name,
		Description:  req.Description,
		DocumentType: req.DocumentType,
		Status:       string(ContentStatusCreated),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repository.CreateContent(ctx, content); err != nil {
		return nil, &ContentError{
			ContentID: content.ID,
			Op:        "upload_create",
			Err:       err,
		}
	}

	// Step 2: Determine storage backend
	storageBackend := req.StorageBackendName
	if storageBackend == "" {
		// Use first available backend as default
		for name := range s.blobStores {
			storageBackend = name
			break
		}
	}
	if storageBackend == "" {
		return nil, fmt.Errorf("no storage backend available")
	}

	// Step 3: Create the object
	objectID := uuid.New()
	objectKey := fmt.Sprintf("%s/%s", content.ID.String(), objectID.String())

	object := &Object{
		ID:                 objectID,
		ContentID:          content.ID,
		ObjectKey:          objectKey,
		StorageBackendName: storageBackend,
		Version:            1,
		Status:             string(ObjectStatusCreated),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repository.CreateObject(ctx, object); err != nil {
		return nil, &ObjectError{
			ObjectID: objectID,
			Op:       "upload_create_object",
			Err:      err,
		}
	}

	// Step 4: Upload the data
	backend, err := s.GetBackend(storageBackend)
	if err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "upload_get_backend", Err: err}
	}

	// Upload with metadata if provided
	if req.DocumentType != "" || req.FileName != "" {
		uploadParams := UploadParams{
			ObjectKey: objectKey,
			MimeType:  req.DocumentType,
		}
		if err := backend.UploadWithParams(ctx, req.Reader, uploadParams); err != nil {
			return nil, &ObjectError{ObjectID: objectID, Op: "upload_data", Err: err}
		}
	} else {
		// Simple upload without metadata
		if err := backend.Upload(ctx, objectKey, req.Reader); err != nil {
			return nil, &ObjectError{ObjectID: objectID, Op: "upload_data", Err: err}
		}
	}

	// Step 5: Update object status
	object.Status = string(ObjectStatusUploaded)
	if err := s.repository.UpdateObject(ctx, object); err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "upload_update_status", Err: err}
	}

	// Step 6: Create content metadata if provided
	if req.FileName != "" || req.FileSize > 0 || len(req.Tags) > 0 || len(req.CustomMetadata) > 0 {
		metadata := &ContentMetadata{
			ContentID: content.ID,
			FileName:  req.FileName,
			FileSize:  req.FileSize,
			MimeType:  req.DocumentType,
			Tags:      req.Tags,
			Metadata:  req.CustomMetadata,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if metadata.Metadata == nil {
			metadata.Metadata = make(map[string]interface{})
		}

		if err := s.repository.SetContentMetadata(ctx, metadata); err != nil {
			// Log warning but don't fail - content was uploaded successfully
		}
	}

	// Step 7: Update content status to uploaded
	content.Status = string(ContentStatusUploaded)
	content.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateContent(ctx, content); err != nil {
		// Log warning but don't fail - content was uploaded successfully
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ContentCreated(ctx, content); err != nil {
			// Log error but don't fail the operation
		}
	}

	return content, nil
}

func (s *service) UploadDerivedContent(ctx context.Context, req UploadDerivedContentRequest) (*Content, error) {
	// Step 1: Verify parent content exists
	_, err := s.repository.GetContent(ctx, req.ParentID)
	if err != nil {
		return nil, fmt.Errorf("parent content not found: %w", err)
	}

	// Step 2: Infer derivation_type from variant if missing
	derivationType := req.DerivationType
	if derivationType == "" && req.Variant != "" {
		derivationType = DerivationTypeFromVariant(req.Variant)
	}

	// Step 3: Create derived content
	now := time.Now().UTC()
	content := &Content{
		ID:             uuid.New(),
		TenantID:       req.TenantID,
		OwnerID:        req.OwnerID,
		Status:         string(ContentStatusCreated),
		DerivationType: NormalizeDerivationType(derivationType),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repository.CreateContent(ctx, content); err != nil {
		return nil, &ContentError{
			ContentID: content.ID,
			Op:        "upload_derived_create",
			Err:       err,
		}
	}

	// Step 4: Create derived content relationship
	_, err = s.repository.CreateDerivedContentRelationship(ctx, CreateDerivedContentParams{
		ParentID:           req.ParentID,
		DerivedContentID:   content.ID,
		DerivationType:     derivationType,
		Variant:            req.Variant,
		DerivationParams:   req.Metadata,
		ProcessingMetadata: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create derived content relationship: %w", err)
	}

	// Step 5: Determine storage backend
	storageBackend := req.StorageBackendName
	if storageBackend == "" {
		// Use first available backend as default
		for name := range s.blobStores {
			storageBackend = name
			break
		}
	}
	if storageBackend == "" {
		return nil, fmt.Errorf("no storage backend available")
	}

	// Step 6: Create the object
	objectID := uuid.New()

	// Generate object key using the configured generator
	objectKey := s.generateDerivedObjectKey(content.ID, objectID, req.ParentID, derivationType, req.Variant, content)

	object := &Object{
		ID:                 objectID,
		ContentID:          content.ID,
		ObjectKey:          objectKey,
		StorageBackendName: storageBackend,
		Version:            1,
		Status:             string(ObjectStatusCreated),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repository.CreateObject(ctx, object); err != nil {
		return nil, &ObjectError{
			ObjectID: objectID,
			Op:       "upload_derived_create_object",
			Err:      err,
		}
	}

	// Step 7: Upload the data
	backend, err := s.GetBackend(storageBackend)
	if err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "upload_derived_get_backend", Err: err}
	}

	// Simple upload for derived content
	if err := backend.Upload(ctx, objectKey, req.Reader); err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "upload_derived_data", Err: err}
	}

	// Step 8: Update object status
	object.Status = string(ObjectStatusUploaded)
	if err := s.repository.UpdateObject(ctx, object); err != nil {
		return nil, &ObjectError{ObjectID: objectID, Op: "upload_derived_update_status", Err: err}
	}

	// Step 9: Create content metadata if provided
	if req.FileName != "" || req.FileSize > 0 || len(req.Tags) > 0 {
		metadata := &ContentMetadata{
			ContentID: content.ID,
			FileName:  req.FileName,
			FileSize:  req.FileSize,
			Tags:      req.Tags,
			Metadata:  req.Metadata,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if metadata.Metadata == nil {
			metadata.Metadata = make(map[string]interface{})
		}

		if err := s.repository.SetContentMetadata(ctx, metadata); err != nil {
			// Log warning but don't fail - content was uploaded successfully
		}
	}

	// Step 10: Update content status to uploaded
	content.Status = string(ContentStatusUploaded)
	content.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateContent(ctx, content); err != nil {
		// Log warning but don't fail - content was uploaded successfully
	}

	// Fire event
	if s.eventSink != nil {
		if err := s.eventSink.ContentCreated(ctx, content); err != nil {
			// Log error but don't fail the operation
		}
	}

	return content, nil
}

func (s *service) DownloadContent(ctx context.Context, contentID uuid.UUID) (io.ReadCloser, error) {
	// Get objects for this content
	objects, err := s.repository.GetObjectsByContentID(ctx, contentID)
	if err != nil {
		return nil, &ContentError{
			ContentID: contentID,
			Op:        "download_get_objects",
			Err:       err,
		}
	}

	if len(objects) == 0 {
		return nil, &ContentError{
			ContentID: contentID,
			Op:        "download",
			Err:       fmt.Errorf("no objects found for content"),
		}
	}

	// Use the first uploaded object
	var targetObject *Object
	for _, obj := range objects {
		if obj.Status == string(ObjectStatusUploaded) {
			targetObject = obj
			break
		}
	}

	if targetObject == nil {
		return nil, &ContentError{
			ContentID: contentID,
			Op:        "download",
			Err:       fmt.Errorf("no uploaded objects found for content"),
		}
	}

	// Download from storage
	backend, err := s.GetBackend(targetObject.StorageBackendName)
	if err != nil {
		return nil, &ObjectError{ObjectID: targetObject.ID, Op: "download_get_backend", Err: err}
	}

	return backend.Download(ctx, targetObject.ObjectKey)
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

func (s *service) UploadObject(ctx context.Context, req UploadObjectRequest) error {
	object, err := s.repository.GetObject(ctx, req.ObjectID)
	if err != nil {
		return &ObjectError{ObjectID: req.ObjectID, Op: "upload", Err: err}
	}

	// Get the backend implementation
	backend, err := s.GetBackend(object.StorageBackendName)
	if err != nil {
		return &ObjectError{ObjectID: req.ObjectID, Op: "upload", Err: err}
	}

	// Upload the object with or without metadata
	if req.MimeType != "" {
		// Upload with metadata
		uploadParams := UploadParams{
			ObjectKey: object.ObjectKey,
			MimeType:  req.MimeType,
		}

		if err := backend.UploadWithParams(ctx, req.Reader, uploadParams); err != nil {
			return &StorageError{
				Backend: object.StorageBackendName,
				Key:     object.ObjectKey,
				Op:      "upload_with_params",
				Err:     err,
			}
		}
	} else {
		// Simple upload without metadata
		if err := backend.Upload(ctx, object.ObjectKey, req.Reader); err != nil {
			return &StorageError{
				Backend: object.StorageBackendName,
				Key:     object.ObjectKey,
				Op:      "upload",
				Err:     err,
			}
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

// GetContentDetails returns all details for a content including URLs and metadata.
// This provides the simplest interface for clients to get everything they need in one call.
func (s *service) GetContentDetails(ctx context.Context, contentID uuid.UUID, options ...ContentDetailsOption) (*ContentDetails, error) {
	// Apply options
	cfg := &ContentDetailsConfig{}
	for _, opt := range options {
		opt(cfg)
	}
	// Initialize the result
	result := &ContentDetails{
		ID:         contentID.String(),
		Thumbnails: make(map[string]string),
		Previews:   make(map[string]string),
		Transcodes: make(map[string]string),
		Ready:      true, // Assume ready unless we find incomplete content
	}

	// Get the content to check if it exists and get its status
	content, err := s.repository.GetContent(ctx, contentID)
	if err != nil {
		return nil, &ContentError{ContentID: contentID, Op: "get_content_details", Err: err}
	}

	// Check if content is ready
	if content.Status != "created" && content.Status != "active" {
		result.Ready = false
	}

	// Get primary objects for this content (for download/preview URLs)
	objects, err := s.repository.GetObjectsByContentID(ctx, contentID)
	if err != nil {
		return nil, &ContentError{ContentID: contentID, Op: "get_content_details", Err: err}
	}

	// Generate download and preview URLs from primary object using URL strategy
	if len(objects) > 0 && s.urlStrategy != nil {
		primaryObject := objects[0] // Use first object as primary

		// Generate download URL using URL strategy
		if downloadURL, err := s.urlStrategy.GenerateDownloadURL(ctx, contentID, primaryObject.ObjectKey, primaryObject.StorageBackendName); err == nil {
			result.Download = downloadURL
		}

		// Generate preview URL using URL strategy
		if previewURL, err := s.urlStrategy.GeneratePreviewURL(ctx, contentID, primaryObject.ObjectKey, primaryObject.StorageBackendName); err == nil {
			result.Preview = previewURL
		}

		// Generate upload URL if requested
		if cfg.IncludeUploadURL {
			if uploadURL, err := s.urlStrategy.GenerateUploadURL(ctx, contentID, primaryObject.ObjectKey, primaryObject.StorageBackendName); err == nil {
				result.Upload = uploadURL
				// Set expiry time if upload URL was generated
				if cfg.URLExpiryTime > 0 {
					expiryTime := time.Now().Add(time.Duration(cfg.URLExpiryTime) * time.Second)
					result.ExpiresAt = &expiryTime
				}
			}
		}
	}

	// Get all derived content with URLs
	derivedContent, err := s.ListDerivedContent(ctx, WithParentID(contentID), WithURLs())
	if err != nil {
		return nil, &ContentError{ContentID: contentID, Op: "get_content_details", Err: err}
	}

	// Organize derived content URLs by type
	for _, derived := range derivedContent {
		// Extract variant without prefix (e.g., "256" from "thumbnail_256")
		variant := derived.Variant
		if idx := strings.LastIndex(variant, "_"); idx >= 0 {
			variant = variant[idx+1:]
		}

		// Organize by derivation type
		switch derived.DerivationType {
		case "thumbnail":
			if derived.DownloadURL != "" {
				result.Thumbnails[variant] = derived.DownloadURL
				// Set primary thumbnail (prefer first one found)
				if result.Thumbnail == "" {
					result.Thumbnail = derived.DownloadURL
				}
			}
		case "preview":
			if derived.DownloadURL != "" {
				result.Previews[variant] = derived.DownloadURL
				// Set primary preview (prefer first one found, but keep original if exists)
				if result.Preview == "" {
					result.Preview = derived.DownloadURL
				}
			}
		case "transcode":
			if derived.DownloadURL != "" {
				result.Transcodes[variant] = derived.DownloadURL
			}
		}

		// If any derived content is not ready, mark the whole result as not ready
		if derived.Status != "created" && derived.Status != "active" {
			result.Ready = false
		}
	}

	// Add content timestamps
	result.CreatedAt = content.CreatedAt
	result.UpdatedAt = content.UpdatedAt

	// Get content metadata if available
	if metadata, err := s.repository.GetContentMetadata(ctx, contentID); err == nil {
		result.FileName = metadata.FileName
		result.FileSize = metadata.FileSize
		result.MimeType = metadata.MimeType
		result.Tags = metadata.Tags
		result.Checksum = metadata.Checksum
	}

	return result, nil
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
	// Convert ContentMetadata to KeyMetadata
	var keyMetadata *objectkey.KeyMetadata
	if contentMetadata != nil {
		keyMetadata = &objectkey.KeyMetadata{
			FileName:    contentMetadata.FileName,
			ContentType: contentMetadata.MimeType,
			IsOriginal:  true, // Default to original, will be overridden for derived content
		}
	}

	return s.keyGenerator.GenerateKey(contentID, objectID, keyMetadata)
}

func (s *service) generateDerivedObjectKey(contentID, objectID, parentContentID uuid.UUID, derivationType, variant string, content *Content) string {
	// Convert Content and metadata to KeyMetadata for derived content
	keyMetadata := &objectkey.KeyMetadata{
		IsOriginal:      false,
		DerivationType:  derivationType,
		Variant:         variant,
		ParentContentID: parentContentID,
	}

	if content != nil {
		keyMetadata.TenantID = content.TenantID.String()
		keyMetadata.OwnerID = content.OwnerID.String()
	}

	return s.keyGenerator.GenerateKey(contentID, objectID, keyMetadata)
}

func (s *service) updateObjectFromStorage(ctx context.Context, objectID uuid.UUID) error {
    _, err := s.UpdateObjectMetaFromStorage(ctx, objectID)
    return err
}

// Derived content helpers
func (s *service) GetDerivedRelationship(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error) {
    return s.repository.GetDerivedRelationshipByContentID(ctx, contentID)
}

func (s *service) ListDerivedContent(ctx context.Context, options ...ListDerivedContentOption) ([]*DerivedContent, error) {
    // Build params from options
    params := ListDerivedContentParams{}
    for _, option := range options {
        option(&params)
    }

    // Get base derived content from repository
    derived, err := s.repository.ListDerivedContent(ctx, params)
    if err != nil {
        return nil, err
    }

    // Enhance with URLs, objects, and metadata if requested
    if params.IncludeURLs || params.IncludeObjects || params.IncludeMetadata {
        for _, d := range derived {
            if err := s.enhanceDerivedContent(ctx, d, params); err != nil {
                // Log error but don't fail entire operation
                // Note: In production, you might want to use a proper logger
                fmt.Printf("Failed to enhance derived content %s: %v\n", d.ContentID, err)
            }
        }
    }

    return derived, nil
}


// Helper methods for enhancement

func (s *service) enhanceDerivedContent(ctx context.Context, derived *DerivedContent, params ListDerivedContentParams) error {
    // Note: Variant is now persisted, no need to extract it

    // Include objects if requested
    if params.IncludeObjects {
        objects, err := s.repository.GetObjectsByContentID(ctx, derived.ContentID)
        if err == nil {
            derived.Objects = objects
        }
    }

    // Include metadata if requested
    if params.IncludeMetadata {
        metadata, err := s.repository.GetContentMetadata(ctx, derived.ContentID)
        if err == nil {
            derived.Metadata = metadata
        }
    }

    // Include URLs if requested
    if params.IncludeURLs {
        if err := s.populateURLs(ctx, derived); err != nil {
            return err
        }
    }

    return nil
}

func (s *service) populateURLs(ctx context.Context, derived *DerivedContent) error {
    // Get objects for this content (use cached objects if already loaded)
    var objects []*Object
    if len(derived.Objects) > 0 {
        objects = derived.Objects
    } else {
        var err error
        objects, err = s.repository.GetObjectsByContentID(ctx, derived.ContentID)
        if err != nil || len(objects) == 0 {
            return err
        }
    }

    // Use first object (usually there's only one per derived content)
    obj := objects[0]

    // Generate URLs
    if downloadURL, err := s.GetDownloadURL(ctx, obj.ID); err == nil {
        derived.DownloadURL = downloadURL
    }

    if previewURL, err := s.GetPreviewURL(ctx, obj.ID); err == nil {
        derived.PreviewURL = previewURL
    }

    // For thumbnails, use preview URL as thumbnail URL
    if derived.DerivationType == "thumbnail" {
        derived.ThumbnailURL = derived.PreviewURL
    }

    return nil
}

func (s *service) extractVariant(derived *DerivedContent) string {
    // Strategy 1: Use persisted Variant field (NEW - highest priority)
    if derived.Variant != "" {
        return derived.Variant
    }

    // Strategy 2: ProcessingMetadata (backward compatibility)
    if variant, exists := derived.ProcessingMetadata["variant"]; exists {
        if variantStr, ok := variant.(string); ok {
            return variantStr
        }
    }

    // Strategy 3: DerivationParams (backward compatibility)
    if variant, exists := derived.DerivationParams["variant"]; exists {
        if variantStr, ok := variant.(string); ok {
            return variantStr
        }
    }

    // Strategy 4: Parse DerivationType (legacy support)
    if derived.DerivationType != "" {
        // If derivation type contains underscore, assume it includes variant
        if derived.DerivationType != "thumbnail" && derived.DerivationType != "preview" && derived.DerivationType != "transcode" {
            return derived.DerivationType
        }
    }

    // Strategy 5: Fallback to derivation type
    return derived.DerivationType
}

