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
	// Soft delete: set deleted_at timestamp, keep status at last operational state
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
	// Soft delete: set deleted_at timestamp, keep status at last operational state
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
        Variant:            params.Variant,                     // NEW: Store variant
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
		if r.matchesEnhancedFilters(derived, params) {
			derivedCopy := *derived
			result = append(result, &derivedCopy)
		}
	}

	// Apply sorting
	r.sortDerivedContent(result, params)

	// Apply pagination
	result = r.paginateDerivedContent(result, params)

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

// Enhanced filtering logic for derived content
func (r *Repository) matchesEnhancedFilters(derived *simplecontent.DerivedContent, params simplecontent.ListDerivedContentParams) bool {
	// Existing logic for backward compatibility
	if params.ParentID != nil && derived.ParentID != *params.ParentID {
		return false
	}
	if params.DerivationType != nil && derived.DerivationType != *params.DerivationType {
		return false
	}

	// NEW: Enhanced filtering logic
	if len(params.ParentIDs) > 0 {
		found := false
		for _, id := range params.ParentIDs {
			if id == derived.ParentID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(params.DerivationTypes) > 0 {
		found := false
		for _, dtype := range params.DerivationTypes {
			if dtype == derived.DerivationType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Variant filtering - new capability
	actualVariant := r.extractVariant(derived)
	if params.Variant != nil && actualVariant != *params.Variant {
		return false
	}

	if len(params.Variants) > 0 {
		found := false
		for _, variant := range params.Variants {
			if variant == actualVariant {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Type+Variant pair filtering
	if len(params.TypeVariantPairs) > 0 {
		found := false
		for _, pair := range params.TypeVariantPairs {
			if pair.DerivationType == derived.DerivationType && pair.Variant == actualVariant {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Content status filtering
	if params.ContentStatus != nil && derived.Status != *params.ContentStatus {
		return false
	}

	// Temporal filtering
	if params.CreatedAfter != nil && derived.CreatedAt.Before(*params.CreatedAfter) {
		return false
	}
	if params.CreatedBefore != nil && derived.CreatedAt.After(*params.CreatedBefore) {
		return false
	}

	return true
}

// extractVariant extracts variant from derived content using multiple strategies
func (r *Repository) extractVariant(derived *simplecontent.DerivedContent) string {
	// Strategy 1: Direct Variant field (preferred - persisted data)
	if derived.Variant != "" {
		return derived.Variant
	}

	// Strategy 2: ProcessingMetadata (fallback)
	if variant, exists := derived.ProcessingMetadata["variant"]; exists {
		if variantStr, ok := variant.(string); ok {
			return variantStr
		}
	}

	// Strategy 3: DerivationParams (fallback)
	if variant, exists := derived.DerivationParams["variant"]; exists {
		if variantStr, ok := variant.(string); ok {
			return variantStr
		}
	}

	// Strategy 4: Parse DerivationType (e.g., "thumbnail_256" -> "thumbnail_256")
	if derived.DerivationType != "" && (len(derived.DerivationType) > 0) {
		// If derivation type contains underscore, assume it includes variant
		if derived.DerivationType != "thumbnail" && derived.DerivationType != "preview" && derived.DerivationType != "transcode" {
			return derived.DerivationType
		}
	}

	// Strategy 4: Fallback to derivation type
	return derived.DerivationType
}

// sortDerivedContent applies sorting based on parameters
func (r *Repository) sortDerivedContent(result []*simplecontent.DerivedContent, params simplecontent.ListDerivedContentParams) {
	if params.SortBy == nil {
		// Default sort: created_at descending
		sort.Slice(result, func(i, j int) bool {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		})
		return
	}

	switch *params.SortBy {
	case "created_at_asc":
		sort.Slice(result, func(i, j int) bool {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		})
	case "created_at_desc":
		sort.Slice(result, func(i, j int) bool {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		})
	case "type_variant":
		sort.Slice(result, func(i, j int) bool {
			if result[i].DerivationType != result[j].DerivationType {
				return result[i].DerivationType < result[j].DerivationType
			}
			variantI := r.extractVariant(result[i])
			variantJ := r.extractVariant(result[j])
			return variantI < variantJ
		})
	default:
		// Default sort: created_at descending
		sort.Slice(result, func(i, j int) bool {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		})
	}
}

// paginateDerivedContent applies pagination based on parameters
func (r *Repository) paginateDerivedContent(result []*simplecontent.DerivedContent, params simplecontent.ListDerivedContentParams) []*simplecontent.DerivedContent {
	// Apply offset
	if params.Offset != nil && *params.Offset > 0 {
		if *params.Offset >= len(result) {
			return []*simplecontent.DerivedContent{}
		}
		result = result[*params.Offset:]
	}

	// Apply limit
	if params.Limit != nil && *params.Limit > 0 && *params.Limit < len(result) {
		result = result[:*params.Limit]
	}

	return result
}

// Admin operations - for administrative tasks without owner/tenant restrictions

func (r *Repository) ListContentWithFilters(ctx context.Context, filters simplecontent.ContentListFilters) ([]*simplecontent.Content, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*simplecontent.Content

	for _, content := range r.contents {
		// Skip deleted unless specifically requested
		if !filters.IncludeDeleted && content.DeletedAt != nil {
			continue
		}

		// Apply filters
		if filters.TenantID != nil && content.TenantID != *filters.TenantID {
			continue
		}
		if len(filters.TenantIDs) > 0 {
			found := false
			for _, tid := range filters.TenantIDs {
				if content.TenantID == tid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.OwnerID != nil && content.OwnerID != *filters.OwnerID {
			continue
		}
		if len(filters.OwnerIDs) > 0 {
			found := false
			for _, oid := range filters.OwnerIDs {
				if content.OwnerID == oid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.Status != nil && content.Status != *filters.Status {
			continue
		}
		if len(filters.Statuses) > 0 {
			found := false
			for _, s := range filters.Statuses {
				if content.Status == s {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.DerivationType != nil && content.DerivationType != *filters.DerivationType {
			continue
		}
		if len(filters.DerivationTypes) > 0 {
			found := false
			for _, dt := range filters.DerivationTypes {
				if content.DerivationType == dt {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.DocumentType != nil && content.DocumentType != *filters.DocumentType {
			continue
		}
		if len(filters.DocumentTypes) > 0 {
			found := false
			for _, docType := range filters.DocumentTypes {
				if content.DocumentType == docType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.CreatedAfter != nil && content.CreatedAt.Before(*filters.CreatedAfter) {
			continue
		}
		if filters.CreatedBefore != nil && content.CreatedAt.After(*filters.CreatedBefore) {
			continue
		}

		if filters.UpdatedAfter != nil && content.UpdatedAt.Before(*filters.UpdatedAfter) {
			continue
		}
		if filters.UpdatedBefore != nil && content.UpdatedAt.After(*filters.UpdatedBefore) {
			continue
		}

		result = append(result, content)
	}

	// Sort results
	sortBy := "created_at"
	sortOrder := "DESC"
	if filters.SortBy != nil {
		sortBy = *filters.SortBy
	}
	if filters.SortOrder != nil {
		sortOrder = *filters.SortOrder
	}

	// Simple sorting implementation
	if sortBy == "created_at" {
		if sortOrder == "ASC" {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if result[i].CreatedAt.After(result[j].CreatedAt) {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		} else {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if result[i].CreatedAt.Before(result[j].CreatedAt) {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		}
	}

	// Apply pagination
	if filters.Offset != nil && *filters.Offset > 0 {
		if *filters.Offset >= len(result) {
			return []*simplecontent.Content{}, nil
		}
		result = result[*filters.Offset:]
	}

	if filters.Limit != nil && *filters.Limit > 0 && *filters.Limit < len(result) {
		result = result[:*filters.Limit]
	}

	return result, nil
}

func (r *Repository) CountContentWithFilters(ctx context.Context, filters simplecontent.ContentCountFilters) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64

	for _, content := range r.contents {
		// Skip deleted unless specifically requested
		if !filters.IncludeDeleted && content.DeletedAt != nil {
			continue
		}

		// Apply filters (same as ListContentWithFilters but just count)
		if filters.TenantID != nil && content.TenantID != *filters.TenantID {
			continue
		}
		if len(filters.TenantIDs) > 0 {
			found := false
			for _, tid := range filters.TenantIDs {
				if content.TenantID == tid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.OwnerID != nil && content.OwnerID != *filters.OwnerID {
			continue
		}
		if len(filters.OwnerIDs) > 0 {
			found := false
			for _, oid := range filters.OwnerIDs {
				if content.OwnerID == oid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.Status != nil && content.Status != *filters.Status {
			continue
		}
		if len(filters.Statuses) > 0 {
			found := false
			for _, s := range filters.Statuses {
				if content.Status == s {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.DerivationType != nil && content.DerivationType != *filters.DerivationType {
			continue
		}
		if len(filters.DerivationTypes) > 0 {
			found := false
			for _, dt := range filters.DerivationTypes {
				if content.DerivationType == dt {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.DocumentType != nil && content.DocumentType != *filters.DocumentType {
			continue
		}
		if len(filters.DocumentTypes) > 0 {
			found := false
			for _, docType := range filters.DocumentTypes {
				if content.DocumentType == docType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.CreatedAfter != nil && content.CreatedAt.Before(*filters.CreatedAfter) {
			continue
		}
		if filters.CreatedBefore != nil && content.CreatedAt.After(*filters.CreatedBefore) {
			continue
		}

		if filters.UpdatedAfter != nil && content.UpdatedAt.Before(*filters.UpdatedAfter) {
			continue
		}
		if filters.UpdatedBefore != nil && content.UpdatedAt.After(*filters.UpdatedBefore) {
			continue
		}

		count++
	}

	return count, nil
}

func (r *Repository) GetContentStatistics(ctx context.Context, filters simplecontent.ContentCountFilters, options simplecontent.ContentStatisticsOptions) (*simplecontent.ContentStatisticsResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := &simplecontent.ContentStatisticsResult{
		ByStatus:         make(map[string]int64),
		ByTenant:         make(map[string]int64),
		ByDerivationType: make(map[string]int64),
		ByDocumentType:   make(map[string]int64),
	}

	var oldest, newest *time.Time

	for _, content := range r.contents {
		// Skip deleted unless specifically requested
		if !filters.IncludeDeleted && content.DeletedAt != nil {
			continue
		}

		// Apply filters (same logic as Count)
		if filters.TenantID != nil && content.TenantID != *filters.TenantID {
			continue
		}
		if len(filters.TenantIDs) > 0 {
			found := false
			for _, tid := range filters.TenantIDs {
				if content.TenantID == tid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.OwnerID != nil && content.OwnerID != *filters.OwnerID {
			continue
		}
		if len(filters.OwnerIDs) > 0 {
			found := false
			for _, oid := range filters.OwnerIDs {
				if content.OwnerID == oid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.Status != nil && content.Status != *filters.Status {
			continue
		}
		if len(filters.Statuses) > 0 {
			found := false
			for _, s := range filters.Statuses {
				if content.Status == s {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.DerivationType != nil && content.DerivationType != *filters.DerivationType {
			continue
		}
		if len(filters.DerivationTypes) > 0 {
			found := false
			for _, dt := range filters.DerivationTypes {
				if content.DerivationType == dt {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.DocumentType != nil && content.DocumentType != *filters.DocumentType {
			continue
		}
		if len(filters.DocumentTypes) > 0 {
			found := false
			for _, docType := range filters.DocumentTypes {
				if content.DocumentType == docType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if filters.CreatedAfter != nil && content.CreatedAt.Before(*filters.CreatedAfter) {
			continue
		}
		if filters.CreatedBefore != nil && content.CreatedAt.After(*filters.CreatedBefore) {
			continue
		}

		if filters.UpdatedAfter != nil && content.UpdatedAt.Before(*filters.UpdatedAfter) {
			continue
		}
		if filters.UpdatedBefore != nil && content.UpdatedAt.After(*filters.UpdatedBefore) {
			continue
		}

		// Count this content
		result.TotalCount++

		// Status breakdown
		if options.IncludeStatusBreakdown {
			result.ByStatus[content.Status]++
		}

		// Tenant breakdown
		if options.IncludeTenantBreakdown {
			result.ByTenant[content.TenantID.String()]++
		}

		// Derivation type breakdown
		if options.IncludeDerivationBreakdown {
			dt := content.DerivationType
			if dt == "" {
				dt = "original"
			}
			result.ByDerivationType[dt]++
		}

		// Document type breakdown
		if options.IncludeDocumentTypeBreakdown {
			docType := content.DocumentType
			if docType == "" {
				docType = "unknown"
			}
			result.ByDocumentType[docType]++
		}

		// Time range
		if options.IncludeTimeRange {
			if oldest == nil || content.CreatedAt.Before(*oldest) {
				t := content.CreatedAt
				oldest = &t
			}
			if newest == nil || content.CreatedAt.After(*newest) {
				t := content.CreatedAt
				newest = &t
			}
		}
	}

	result.OldestContent = oldest
	result.NewestContent = newest

	return result, nil
}
