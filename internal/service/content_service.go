package service

import (
	"context"
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

// CreateContent creates a new content
func (s *ContentService) CreateContent(
	ctx context.Context,
	ownerID, tenantID uuid.UUID,
) (*domain.Content, error) {
	now := time.Now()
	content := &domain.Content{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID:   ownerID,
		TenantID:  tenantID,
		Status:    "active",
	}

	if err := s.contentRepo.Create(ctx, content); err != nil {
		return nil, err
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

// SetContentMetadata sets metadata for a content
func (s *ContentService) SetContentMetadata(ctx context.Context, contentID uuid.UUID, metadata map[string]interface{}) error {
	// Verify content exists
	if _, err := s.contentRepo.Get(ctx, contentID); err != nil {
		return err
	}

	return s.metadataRepo.Set(ctx, contentID, metadata)
}

// GetContentMetadata retrieves metadata for a content
func (s *ContentService) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (map[string]interface{}, error) {
	// Verify content exists
	if _, err := s.contentRepo.Get(ctx, contentID); err != nil {
		return nil, err
	}

	return s.metadataRepo.Get(ctx, contentID)
}
