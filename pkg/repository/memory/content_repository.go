// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
)

// ContentRepository is an in-memory implementation of the ContentRepository interface
type ContentRepository struct {
	mu               sync.RWMutex
	contents         map[uuid.UUID]*domain.Content
	derivedRelations map[uuid.UUID][]domain.DerivedContent // Map of parent ID to derived content relationships
}

// NewContentRepository creates a new in-memory content repository
func NewContentRepository() repository.ContentRepository {
	return &ContentRepository{
		contents:         make(map[uuid.UUID]*domain.Content),
		derivedRelations: make(map[uuid.UUID][]domain.DerivedContent),
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

// ListDerivedContent retrieves derived content based on the provided parameters
func (r *ContentRepository) ListDerivedContent(ctx context.Context, params repository.ListDerivedContentParams) ([]*domain.DerivedContent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.DerivedContent

	// If parent IDs are provided, filter by them
	if len(params.ParentIDs) > 0 {
		for _, parentID := range params.ParentIDs {
			relations, exists := r.derivedRelations[parentID]
			if exists {
				for i := range relations {
					// Make a copy to avoid modifying the stored data
					relation := relations[i]

					// Filter by derivation type if provided
					if len(params.DerivationType) > 0 {
						typeMatch := false
						for _, dt := range params.DerivationType {
							if relation.DerivationType == dt {
								typeMatch = true
								break
							}
						}
						if !typeMatch {
							continue
						}
					}

					// Filter by tenant ID if provided
					if params.TenantID != uuid.Nil {
						content, exists := r.contents[relation.ContentID]
						if !exists || content.TenantID != params.TenantID {
							continue
						}
					}

					result = append(result, &relation)
				}
			}
		}
	} else {
		// If no parent IDs provided, collect all derived content
		for _, relations := range r.derivedRelations {
			for i := range relations {
				relation := relations[i]

				// Apply the same filters as above
				if len(params.DerivationType) > 0 {
					typeMatch := false
					for _, dt := range params.DerivationType {
						if relation.DerivationType == dt {
							typeMatch = true
							break
						}
					}
					if !typeMatch {
						continue
					}
				}

				if params.TenantID != uuid.Nil {
					content, exists := r.contents[relation.ContentID]
					if !exists || content.TenantID != params.TenantID {
						continue
					}
				}

				result = append(result, &relation)
			}
		}
	}

	return result, nil
}

// CreateDerivedContentRelationship creates a new derived content relationship
func (r *ContentRepository) CreateDerivedContentRelationship(ctx context.Context, params repository.CreateDerivedContentParams) (domain.DerivedContent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if both parent and derived content exist
	_, exists := r.contents[params.ParentID]
	if !exists {
		return domain.DerivedContent{}, errors.New("parent content not found")
	}

	derivedContent, exists := r.contents[params.DerivedContentID]
	if !exists {
		return domain.DerivedContent{}, errors.New("derived content not found")
	}

	// Create the derived content relationship
	relationship := domain.DerivedContent{
		ParentID:           params.ParentID,
		ContentID:          params.DerivedContentID,
		DerivationType:     params.DerivationType,
		DerivationParams:   params.DerivationParams,
		ProcessingMetadata: params.ProcessingMetadata,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		DocumentType:       derivedContent.DocumentType,
		Status:             derivedContent.Status,
	}

	// Store the relationship in our in-memory map
	r.derivedRelations[params.ParentID] = append(r.derivedRelations[params.ParentID], relationship)

	return relationship, nil
}

// DeleteDerivedContent deletes a derived content
func (r *ContentRepository) DeleteDerivedContentRelationship(ctx context.Context, params repository.DeleteDerivedContentParams) error {
	// This is a stub implementation for the in-memory repository
	// In a real implementation, we would delete the relationship from the database
	return nil
}

// GetDerivedContentByLevel retrieves derived content at a specific level with parent information
func (r *ContentRepository) GetDerivedContentByLevel(ctx context.Context, params repository.GetDerivedContentByLevelParams) ([]repository.ContentWithParent, error) {
	// This is a stub implementation for the in-memory repository
	// In a real implementation, we would traverse the derivation tree to find content at the specified level
	r.mu.RLock()
	defer r.mu.RUnlock()

	// For in-memory implementation, we'll just return an empty slice
	// A proper implementation would require maintaining a graph of parent-child relationships
	return []repository.ContentWithParent{}, nil
}
