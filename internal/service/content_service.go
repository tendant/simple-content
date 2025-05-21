package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
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
) (*domain.Content, error) {
	now := time.Now()
	content := &domain.Content{
		ID:              uuid.New(),
		ParentID:        nil, // Explicitly nil for original content
		CreatedAt:       now,
		UpdatedAt:       now,
		OwnerID:         ownerID,
		TenantID:        tenantID,
		Status:          "active",
		DerivationType:  "original",
		DerivationLevel: 0, // Original content has level 0
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
) (*domain.Content, error) {
	// Verify parent content exists
	parentContent, err := s.contentRepo.Get(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("parent content not found: %w", err)
	}

	// Check derivation level limit
	if parentContent.DerivationLevel >= 5 {
		return nil, errors.New("maximum derivation depth reached (limit: 5 levels)")
	}

	now := time.Now()
	content := &domain.Content{
		ID:              uuid.New(),
		ParentID:        &parentID,
		CreatedAt:       now,
		UpdatedAt:       now,
		OwnerID:         ownerID,
		TenantID:        tenantID,
		Status:          "active",
		DerivationType:  "derived",
		DerivationLevel: parentContent.DerivationLevel + 1,
	}

	if err := s.contentRepo.Create(ctx, content); err != nil {
		return nil, fmt.Errorf("failed to create derived content: %w", err)
	}

	return content, nil
}

// GetContent retrieves a content by ID
func (s *ContentService) GetContent(ctx context.Context, id uuid.UUID) (*domain.Content, error) {
	return s.contentRepo.Get(ctx, id)
}

// UpdateContent updates a content
func (s *ContentService) UpdateContent(ctx context.Context, content *domain.Content) error {
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
func (s *ContentService) ListContents(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*domain.Content, error) {
	return s.contentRepo.List(ctx, ownerID, tenantID)
}

// GetDerivedContent retrieves all content directly derived from a specific parent
func (s *ContentService) GetDerivedContent(ctx context.Context, parentID uuid.UUID) ([]*domain.Content, error) {
	return s.contentRepo.GetByParentID(ctx, parentID)
}

// GetDerivedContentTree retrieves the entire tree of derived content
func (s *ContentService) GetDerivedContentTree(ctx context.Context, rootID uuid.UUID) ([]*domain.Content, error) {
	// Use a max depth of 5 for the derivation tree
	return s.contentRepo.GetDerivedContentTree(ctx, rootID, 5)
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
	if _, err := s.contentRepo.Get(ctx, contentID); err != nil {
		return err
	}

	metadata := &domain.ContentMetadata{
		ContentID:   contentID,
		ContentType: contentType,
		Title:       title,
		Description: description,
		Tags:        tags,
		FileSize:    fileSize,
		CreatedBy:   createdBy,
		Metadata:    customMetadata,
	}

	return s.metadataRepo.Set(ctx, metadata)
}

// GetContentMetadata retrieves metadata for a content
func (s *ContentService) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*domain.ContentMetadata, error) {
	// Verify content exists
	if _, err := s.contentRepo.Get(ctx, contentID); err != nil {
		return nil, err
	}

	return s.metadataRepo.Get(ctx, contentID)
}
