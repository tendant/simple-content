package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
)

// ContentRepository is an in-memory implementation of the ContentRepository interface
type ContentRepository struct {
	mu       sync.RWMutex
	contents map[uuid.UUID]*domain.Content
}

// NewContentRepository creates a new in-memory content repository
func NewContentRepository() repository.ContentRepository {
	return &ContentRepository{
		contents: make(map[uuid.UUID]*domain.Content),
	}
}

// Create adds a new content to the repository
func (r *ContentRepository) Create(ctx context.Context, content *domain.Content) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.contents[content.ID]; exists {
		return errors.New("content already exists")
	}

	r.contents[content.ID] = content
	return nil
}

// Get retrieves a content by ID
func (r *ContentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	content, exists := r.contents[id]
	if !exists {
		return nil, errors.New("content not found")
	}

	return content, nil
}

// Update updates an existing content
func (r *ContentRepository) Update(ctx context.Context, content *domain.Content) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.contents[content.ID]; !exists {
		return errors.New("content not found")
	}

	r.contents[content.ID] = content
	return nil
}

// Delete removes a content by ID
func (r *ContentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.contents[id]; !exists {
		return errors.New("content not found")
	}

	delete(r.contents, id)
	return nil
}

// List retrieves contents by owner ID and tenant ID
func (r *ContentRepository) List(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*domain.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Validate that at least one of ownerID or tenantID is provided
	if ownerID == uuid.Nil && tenantID == uuid.Nil {
		return nil, nil
	}
	var result []*domain.Content
	for _, content := range r.contents {
		if (ownerID == uuid.Nil || content.OwnerID == ownerID) &&
			(tenantID == uuid.Nil || content.TenantID == tenantID) {
			result = append(result, content)
		}
	}

	return result, nil
}

// GetByParentID retrieves all content directly derived from a specific parent
// Note: This method is now a stub as ParentID has been removed from Content struct
// The relationship between content items is now tracked in ContentDerived table
func (r *ContentRepository) GetByParentID(ctx context.Context, parentID uuid.UUID) ([]*domain.Content, error) {
	// This is now a stub method that returns an empty slice
	// In a real implementation, this would query the ContentDerived table
	return []*domain.Content{}, nil
}

// GetDerivedContentTree retrieves the entire tree of derived content up to maxDepth
// Note: This method is now a stub as ParentID has been removed from Content struct
// The relationship between content items is now tracked in ContentDerived table
func (r *ContentRepository) GetDerivedContentTree(ctx context.Context, rootID uuid.UUID, maxDepth int) ([]*domain.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get the root content
	rootContent, exists := r.contents[rootID]
	if !exists {
		return nil, errors.New("root content not found")
	}

	// This is now a stub method that returns only the root content
	// In a real implementation, this would query the ContentDerived table
	return []*domain.Content{rootContent}, nil
}
