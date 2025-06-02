package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/repository"
	"github.com/tendant/simple-content/pkg/model"
)

// ContentService handles content-related operations
type ContentService struct {
	contentRepo  repository.ContentRepository
	metadataRepo repository.ContentMetadataRepository
	objectRepo   repository.ObjectRepository
}

// NewContentService creates a new content service
func NewContentService(
	contentRepo repository.ContentRepository,
	metadataRepo repository.ContentMetadataRepository,
	objectRepo repository.ObjectRepository,
) *ContentService {
	return &ContentService{
		contentRepo:  contentRepo,
		metadataRepo: metadataRepo,
		objectRepo:   objectRepo,
	}
}

// CreateContent creates a new original content
func (s *ContentService) CreateContent(
	ctx context.Context,
	ownerID, tenantID uuid.UUID,
) (*model.Content, error) {
	now := time.Now()
	content := &model.Content{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		OwnerID:        ownerID,
		TenantID:       tenantID,
		Status:         model.ContentStatusCreated,
		DerivationType: model.ContentDerivationTypeOriginal,
	}

	if err := s.contentRepo.Create(ctx, content); err != nil {
		return nil, err
	}

	return content, nil
}

// CreateDerivedContent creates a new content derived from an existing content
func (s *ContentService) CreateDerivedContent(
	ctx context.Context,
	parentID uuid.UUID,
	ownerID, tenantID uuid.UUID,
) (*model.Content, error) {
	// Verify parent content exists
	_, err := s.contentRepo.Get(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("parent content not found: %w", err)
	}

	// Create derived content
	now := time.Now()
	content := &model.Content{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		OwnerID:        ownerID,
		TenantID:       tenantID,
		Status:         model.ContentStatusCreated,
		DerivationType: model.ContentDerivationTypeDerived,
	}

	if err := s.contentRepo.Create(ctx, content); err != nil {
		return nil, fmt.Errorf("failed to create derived content: %w", err)
	}

	// Note: Content derivation relationships will be handled by the ContentDerivedRepository
	// which will be implemented separately

	return content, nil
}

// GetContent retrieves a content by ID
func (s *ContentService) GetContent(ctx context.Context, id uuid.UUID) (*model.Content, error) {
	return s.contentRepo.Get(ctx, id)
}

// UpdateContent updates a content
func (s *ContentService) UpdateContent(ctx context.Context, content *model.Content) error {
	content.UpdatedAt = time.Now()
	return s.contentRepo.Update(ctx, content)
}

// DeleteContent deletes a content
func (s *ContentService) DeleteContent(ctx context.Context, id uuid.UUID) error {
	// Get all objects for this content
	objects, err := s.objectRepo.GetByContentID(ctx, id)
	if err != nil {
		return err
	}

	// Delete all objects
	for _, obj := range objects {
		if err := s.objectRepo.Delete(ctx, obj.ID); err != nil {
			return err
		}
	}

	// Delete the content
	return s.contentRepo.Delete(ctx, id)
}

// ListContents lists contents by owner ID and tenant ID
func (s *ContentService) ListContents(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*model.Content, error) {
	return s.contentRepo.List(ctx, ownerID, tenantID)
}

// SetContentMetadata sets metadata for a content
func (s *ContentService) SetContentMetadata(
	ctx context.Context,
	contentID uuid.UUID,
	contentType, title, description string,
	tags []string,
	fileSize int64,
	createdBy string,
	customMetadata map[string]interface{},
) error {
	// Verify content exists
	_, err := s.contentRepo.Get(ctx, contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}

	// Prepare metadata
	metadata := &model.ContentMetadata{
		ContentID: contentID,
		Tags:      tags,
		FileSize:  fileSize,
		Metadata:  make(map[string]interface{}),
	}

	// Store content type, title, description, and created by in the metadata map
	metadata.Metadata["content_type"] = contentType
	fileName := customMetadata["file_name"]
	if fileName != nil {
		metadata.FileName = fileName.(string)
	}
	mimeType := customMetadata["mime_type"]
	if mimeType != nil {
		metadata.MimeType = mimeType.(string)
	}

	// Copy custom metadata
	for k, v := range customMetadata {
		metadata.Metadata[k] = v
	}

	return s.metadataRepo.Set(ctx, metadata)
}

// GetContentMetadata retrieves metadata for a content
func (s *ContentService) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*model.ContentMetadata, error) {
	// Verify content exists
	if _, err := s.contentRepo.Get(ctx, contentID); err != nil {
		return nil, err
	}

	return s.metadataRepo.Get(ctx, contentID)
}
