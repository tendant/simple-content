package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// Repository implements simplecontent.Repository using in-memory storage
type Repository struct {
	mu                sync.RWMutex
	contents          map[uuid.UUID]*simplecontent.Content
	contentMetadata   map[uuid.UUID]*simplecontent.ContentMetadata
	objects           map[uuid.UUID]*simplecontent.Object
	objectMetadata    map[uuid.UUID]*simplecontent.ObjectMetadata
	derivedContents   map[uuid.UUID]*simplecontent.DerivedContent
	objectsByContent  map[uuid.UUID][]uuid.UUID // content_id -> []object_id
	objectsByKey      map[string]uuid.UUID      // "backend:key" -> object_id
}

// New creates a new in-memory repository
func New() simplecontent.Repository {
	return &Repository{
		contents:          make(map[uuid.UUID]*simplecontent.Content),
		contentMetadata:   make(map[uuid.UUID]*simplecontent.ContentMetadata),
		objects:           make(map[uuid.UUID]*simplecontent.Object),
		objectMetadata:    make(map[uuid.UUID]*simplecontent.ObjectMetadata),
		derivedContents:   make(map[uuid.UUID]*simplecontent.DerivedContent),
		objectsByContent:  make(map[uuid.UUID][]uuid.UUID),
		objectsByKey:      make(map[string]uuid.UUID),
	}
}

// Content operations

func (r *Repository) CreateContent(ctx context.Context, content *simplecontent.Content) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Create a copy to avoid external modifications
	contentCopy := *content
	r.contents[content.ID] = &contentCopy
	
	return nil
}

func (r *Repository) GetContent(ctx context.Context, id uuid.UUID) (*simplecontent.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	content, exists := r.contents[id]
	if !exists {
		return nil, simplecontent.ErrContentNotFound
	}

	if content.DeletedAt != nil {
		return nil, simplecontent.ErrContentNotFound
	}
	// Return a copy to prevent external modifications
	contentCopy := *content
	return &contentCopy, nil
}

func (r *Repository) UpdateContent(ctx context.Context, content *simplecontent.Content) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.contents[content.ID]; !exists {
		return simplecontent.ErrContentNotFound
	}
	
	// Create a copy to avoid external modifications
	contentCopy := *content
	r.contents[content.ID] = &contentCopy
	
	return nil
}

func (r *Repository) DeleteContent(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	c, exists := r.contents[id]
	if !exists {
		return simplecontent.ErrContentNotFound
	}
	
	now := time.Now()
	c.Status = string(simplecontent.ContentStatusDeleted)
	c.DeletedAt = &now
	c.UpdatedAt = now
	return nil
}

func (r *Repository) ListContent(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*simplecontent.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []*simplecontent.Content
    for _, content := range r.contents {
        if content.OwnerID == ownerID && content.TenantID == tenantID && content.DeletedAt == nil {
            contentCopy := *content
            result = append(result, &contentCopy)
        }
    }
	
	// Sort by created_at descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	
	return result, nil
}

// Content metadata operations

func (r *Repository) SetContentMetadata(ctx context.Context, metadata *simplecontent.ContentMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Verify content exists
	if _, exists := r.contents[metadata.ContentID]; !exists {
		return simplecontent.ErrContentNotFound
	}
	
	// Create a copy to avoid external modifications
	metadataCopy := *metadata
	if metadataCopy.CreatedAt.IsZero() {
		metadataCopy.CreatedAt = time.Now()
	}
	metadataCopy.UpdatedAt = time.Now()
	
	r.contentMetadata[metadata.ContentID] = &metadataCopy
	
	return nil
}

func (r *Repository) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*simplecontent.ContentMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	metadata, exists := r.contentMetadata[contentID]
	if !exists {
		return nil, fmt.Errorf("content metadata not found for content %s", contentID)
	}
	
	// Return a copy to prevent external modifications
	metadataCopy := *metadata
	return &metadataCopy, nil
}

// Object operations

func (r *Repository) CreateObject(ctx context.Context, object *simplecontent.Object) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Verify content exists
	if _, exists := r.contents[object.ContentID]; !exists {
		return simplecontent.ErrContentNotFound
	}
	
	// Create a copy to avoid external modifications
	objectCopy := *object
	r.objects[object.ID] = &objectCopy
	
	// Update content mapping
	r.objectsByContent[object.ContentID] = append(r.objectsByContent[object.ContentID], object.ID)
	
	// Update key mapping
	keyStr := fmt.Sprintf("%s:%s", object.StorageBackendName, object.ObjectKey)
	r.objectsByKey[keyStr] = object.ID
	
	return nil
}

func (r *Repository) GetObject(ctx context.Context, id uuid.UUID) (*simplecontent.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	object, exists := r.objects[id]
	if !exists {
		return nil, simplecontent.ErrObjectNotFound
	}

	if object.DeletedAt != nil {
		return nil, simplecontent.ErrObjectNotFound
	}
	// Return a copy to prevent external modifications
	objectCopy := *object
	return &objectCopy, nil
}

func (r *Repository) GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*simplecontent.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	objectIDs, exists := r.objectsByContent[contentID]
	if !exists {
		return []*simplecontent.Object{}, nil
	}
	
	var result []*simplecontent.Object
    for _, objectID := range objectIDs {
        if object, exists := r.objects[objectID]; exists {
            if object.DeletedAt == nil {
                objectCopy := *object
                result = append(result, &objectCopy)
            }
        }
    }
	
	// Sort by version descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version > result[j].Version
	})
	
	return result, nil
}

func (r *Repository) GetObjectByObjectKeyAndStorageBackendName(ctx context.Context, objectKey, storageBackendName string) (*simplecontent.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	keyStr := fmt.Sprintf("%s:%s", storageBackendName, objectKey)
	objectID, exists := r.objectsByKey[keyStr]
	if !exists {
		return nil, simplecontent.ErrObjectNotFound
	}
	
    object, exists := r.objects[objectID]
    if !exists {
        return nil, simplecontent.ErrObjectNotFound
    }
    if object.DeletedAt != nil {
        return nil, simplecontent.ErrObjectNotFound
    }
    // Return a copy to prevent external modifications
    objectCopy := *object
    return &objectCopy, nil
}

func (r *Repository) UpdateObject(ctx context.Context, object *simplecontent.Object) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.objects[object.ID]; !exists {
		return simplecontent.ErrObjectNotFound
	}
	
	// Create a copy to avoid external modifications
	objectCopy := *object
	r.objects[object.ID] = &objectCopy
	
	// Update key mapping if changed
	keyStr := fmt.Sprintf("%s:%s", object.StorageBackendName, object.ObjectKey)
	r.objectsByKey[keyStr] = object.ID
	
	return nil
}

func (r *Repository) DeleteObject(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	object, exists := r.objects[id]
	if !exists {
		return simplecontent.ErrObjectNotFound
	}
	
    now := time.Now()
    object.Status = string(simplecontent.ObjectStatusDeleted)
    object.DeletedAt = &now
    object.UpdatedAt = now
    return nil
}

// Object metadata operations

func (r *Repository) SetObjectMetadata(ctx context.Context, metadata *simplecontent.ObjectMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Verify object exists
	if _, exists := r.objects[metadata.ObjectID]; !exists {
		return simplecontent.ErrObjectNotFound
	}
	
	// Create a copy to avoid external modifications
	metadataCopy := *metadata
	if metadataCopy.CreatedAt.IsZero() {
		metadataCopy.CreatedAt = time.Now()
	}
	metadataCopy.UpdatedAt = time.Now()
	
	r.objectMetadata[metadata.ObjectID] = &metadataCopy
	
	return nil
}

func (r *Repository) GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (*simplecontent.ObjectMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	metadata, exists := r.objectMetadata[objectID]
	if !exists {
		return nil, fmt.Errorf("object metadata not found for object %s", objectID)
	}
	
	// Return a copy to prevent external modifications
	metadataCopy := *metadata
	return &metadataCopy, nil
}

// Derived content operations

func (r *Repository) CreateDerivedContentRelationship(ctx context.Context, params simplecontent.CreateDerivedContentParams) (*simplecontent.DerivedContent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Verify parent and derived content exist
	if _, exists := r.contents[params.ParentID]; !exists {
		return nil, simplecontent.ErrContentNotFound
	}
	if _, exists := r.contents[params.DerivedContentID]; !exists {
		return nil, simplecontent.ErrContentNotFound
	}
	
	now := time.Now()
    derived := &simplecontent.DerivedContent{
        ParentID:           params.ParentID,
        ContentID:          params.DerivedContentID,
        DerivationType:     params.DerivationType,
        DerivationParams:   params.DerivationParams,
        ProcessingMetadata: params.ProcessingMetadata,
        CreatedAt:          now,
        UpdatedAt:          now,
        Status:             string(simplecontent.ContentStatusCreated),
    }
	
	r.derivedContents[params.DerivedContentID] = derived
	
	// Return a copy
	derivedCopy := *derived
	return &derivedCopy, nil
}

func (r *Repository) ListDerivedContent(ctx context.Context, params simplecontent.ListDerivedContentParams) ([]*simplecontent.DerivedContent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []*simplecontent.DerivedContent
	for _, derived := range r.derivedContents {
		match := true
		
		if params.ParentID != nil && derived.ParentID != *params.ParentID {
			match = false
		}
		
		if params.DerivationType != nil && derived.DerivationType != *params.DerivationType {
			match = false
		}
		
		if match {
			derivedCopy := *derived
			result = append(result, &derivedCopy)
		}
	}
	
	// Sort by created_at descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	
	// Apply limit and offset
	if params.Offset != nil && *params.Offset > 0 {
		if *params.Offset >= len(result) {
			return []*simplecontent.DerivedContent{}, nil
		}
		result = result[*params.Offset:]
	}
	
	if params.Limit != nil && *params.Limit > 0 && *params.Limit < len(result) {
		result = result[:*params.Limit]
	}
	
	return result, nil
}

func (r *Repository) GetDerivedRelationshipByContentID(ctx context.Context, contentID uuid.UUID) (*simplecontent.DerivedContent, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    dc, exists := r.derivedContents[contentID]
    if !exists {
        return nil, fmt.Errorf("derived relationship not found for content %s", contentID)
    }
    copy := *dc
    return &copy, nil
}
