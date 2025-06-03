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
}

// NewContentService creates a new content service
func NewContentService(
	contentRepo repository.ContentRepository,
	metadataRepo repository.ContentMetadataRepository,
) *ContentService {
	return &ContentService{
		contentRepo:  contentRepo,
		metadataRepo: metadataRepo,
	}
}

// CreateContentParams contains parameters for creating new content
type CreateContentParams struct {
	OwnerID      uuid.UUID
	TenantID     uuid.UUID
	Title        string
	Description  string
	DocumentType string
}

// CreateContent creates a new content
func (s *ContentService) CreateContent(
	ctx context.Context,
	params CreateContentParams,
) (*model.Content, error) {
	now := time.Now()
	content := &model.Content{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		OwnerID:        params.OwnerID,
		TenantID:       params.TenantID,
		Name:           params.Title,
		Description:    params.Description,
		DocumentType:   params.DocumentType,
		Status:         model.ContentStatusCreated,
		DerivationType: model.ContentDerivationTypeOriginal,
	}

	if err := s.contentRepo.Create(ctx, content); err != nil {
		return nil, err
	}

	return content, nil
}

// CreateDerivedContentParams contains parameters for creating derived content
type CreateDerivedContentParams struct {
	ParentID uuid.UUID
	OwnerID  uuid.UUID
	TenantID uuid.UUID
}

// CreateDerivedContent creates a new content derived from an existing content
func (s *ContentService) CreateDerivedContent(
	ctx context.Context,
	params CreateDerivedContentParams,
) (*model.Content, error) {
	// Verify parent content exists
	_, err := s.contentRepo.Get(ctx, params.ParentID)
	if err != nil {
		return nil, fmt.Errorf("parent content not found: %w", err)
	}

	// Create derived content
	now := time.Now()
	content := &model.Content{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		OwnerID:        params.OwnerID,
		TenantID:       params.TenantID,
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

// UpdateContentParams contains parameters for updating content
type UpdateContentParams struct {
	Content *model.Content
}

// UpdateContent updates a content
func (s *ContentService) UpdateContent(
	ctx context.Context,
	params UpdateContentParams,
) error {
	params.Content.UpdatedAt = time.Now()
	return s.contentRepo.Update(ctx, params.Content)
}

// DeleteContentParams contains parameters for deleting content
type DeleteContentParams struct {
	ID uuid.UUID
}

// DeleteContent deletes a content
func (s *ContentService) DeleteContent(
	ctx context.Context,
	params DeleteContentParams,
) error {
	// Delete the content
	return s.contentRepo.Delete(ctx, params.ID)
}

// ListContentParams contains parameters for listing content
type ListContentParams struct {
	OwnerID  uuid.UUID
	TenantID uuid.UUID
}

// ListContent lists all content for an owner and tenant
func (s *ContentService) ListContent(
	ctx context.Context,
	params ListContentParams,
) ([]*model.Content, error) {
	return s.contentRepo.List(ctx, params.OwnerID, params.TenantID)
}

// SetContentMetadataParams contains parameters for setting content metadata
type SetContentMetadataParams struct {
	ContentID      uuid.UUID
	ContentType    string
	Title          string
	Description    string
	Tags           []string
	FileName       string
	FileSize       int64
	CreatedBy      string
	CustomMetadata map[string]interface{}
}

// SetContentMetadata sets metadata for a content
func (s *ContentService) SetContentMetadata(
	ctx context.Context,
	params SetContentMetadataParams,
) error {
	// Verify content exists
	_, err := s.contentRepo.Get(ctx, params.ContentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}

	// Create content metadata
	contentMetadata := &model.ContentMetadata{
		ContentID: params.ContentID,
		Tags:      params.Tags,
		UpdatedAt: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}

	// Set mime type if provided in content type
	if params.ContentType != "" {
		contentMetadata.MimeType = params.ContentType
		contentMetadata.Metadata["content_type"] = params.ContentType
	}

	// Set file name if provided
	if params.FileName != "" {
		contentMetadata.FileName = params.FileName
		contentMetadata.Metadata["file_name"] = params.FileName
	}

	// Set file size if provided
	if params.FileSize > 0 {
		contentMetadata.FileSize = params.FileSize
		contentMetadata.Metadata["file_size"] = params.FileSize
	}

	// Copy custom metadata if provided
	if params.CustomMetadata != nil {
		for k, v := range params.CustomMetadata {
			contentMetadata.Metadata[k] = v
		}

		// Extract file name and mime type if present in custom metadata
		if fileName, ok := params.CustomMetadata["file_name"].(string); ok {
			contentMetadata.FileName = fileName
		}
	}

	// Add title and description to metadata if provided
	if params.Title != "" {
		contentMetadata.Metadata["title"] = params.Title
	}
	if params.Description != "" {
		contentMetadata.Metadata["description"] = params.Description
	}

	if params.CreatedBy != "" {
		contentMetadata.Metadata["created_by"] = params.CreatedBy
	}

	// Set creation time if not already set
	if contentMetadata.CreatedAt.IsZero() {
		contentMetadata.CreatedAt = time.Now().UTC()
	}

	return s.metadataRepo.Set(ctx, contentMetadata)
}

// GetContentMetadata retrieves metadata for a content
func (s *ContentService) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*model.ContentMetadata, error) {
	return s.metadataRepo.Get(ctx, contentID)
}
