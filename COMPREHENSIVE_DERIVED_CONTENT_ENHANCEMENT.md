# Comprehensive Derived Content Enhancement: Interface Design & URL Integration

## Executive Summary

This document combines interface enhancement recommendations with URL integration strategy for the simple-content library's derived content functionality. The approach provides **enhanced filtering capabilities with built-in URL support** while maintaining full backward compatibility.

## Key Architectural Decisions

### 1. **Interface Strategy: Extend Existing Interface**
✅ **Recommended**: Enhance existing `ListDerivedContentParams` and service methods
❌ **Not Recommended**: Create separate new interfaces

### 2. **Data Structure Strategy: Enhanced Single Struct**
✅ **Recommended**: Add URL fields to `DerivedContent` with `db:"-"` tags
❌ **Not Recommended**: Maintain separate `DerivedContent` and `DerivedContentItem` structs

## Current Architecture Analysis

### Existing Interface Structure
```go
// Current ListDerivedContentParams (interfaces.go:125-131)
type ListDerivedContentParams struct {
    ParentID       *uuid.UUID
    DerivationType *string
    Limit          *int
    Offset         *int
}

// Current DerivedContent struct (types.go:71-83)
type DerivedContent struct {
    ParentID           uuid.UUID
    ContentID          uuid.UUID
    DerivationType     string
    DerivationParams   map[string]interface{}
    ProcessingMetadata map[string]interface{}
    CreatedAt          time.Time
    UpdatedAt          time.Time
    DocumentType       string
    Status             string
}
```

### Key Architectural Patterns Identified
1. **Parameter Objects Pattern**: Uses dedicated parameter structs (`ListDerivedContentParams`)
2. **Functional Options Pattern**: Service construction uses `WithRepository()`, `WithBlobStore()` etc.
3. **Interface Segregation**: Clean separation between `Service`, `Repository`, and `BlobStore` interfaces
4. **Backward Compatibility**: Simple methods like `ListDerivedByParent()` wrap complex repository calls
5. **Progressive Enhancement**: Existing codebase shows progression from simple to complex patterns

## Comprehensive Enhancement Strategy

### Phase 1: Enhanced Data Structure with URLs

```go
// Enhanced DerivedContent - single struct with advanced capabilities
type DerivedContent struct {
    // Persisted fields (unchanged for backward compatibility)
    ParentID           uuid.UUID              `json:"parent_id" db:"parent_id"`
    ContentID          uuid.UUID              `json:"content_id" db:"content_id"`
    DerivationType     string                 `json:"derivation_type" db:"derivation_type"`
    DerivationParams   map[string]interface{} `json:"derivation_params" db:"derivation_params"`
    ProcessingMetadata map[string]interface{} `json:"processing_metadata" db:"processing_metadata"`
    CreatedAt          time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
    DocumentType       string                 `json:"document_type" db:"document_type"`
    Status             string                 `json:"status" db:"status"`

    // NEW: Computed fields (not persisted - db:"-" tag)
    DownloadURL        string                 `json:"download_url,omitempty" db:"-"`
    PreviewURL         string                 `json:"preview_url,omitempty" db:"-"`
    ThumbnailURL       string                 `json:"thumbnail_url,omitempty" db:"-"`
    Variant            string                 `json:"variant,omitempty" db:"-"`

    // NEW: Optional enhanced data (not persisted)
    Objects            []*Object              `json:"objects,omitempty" db:"-"`
    Metadata           *ContentMetadata       `json:"metadata,omitempty" db:"-"`
    ParentContent      *Content               `json:"parent_content,omitempty" db:"-"`
}
```

### Phase 2: Enhanced Parameter Structure (Backward Compatible)

```go
// Enhanced ListDerivedContentParams - backward compatible extension
type ListDerivedContentParams struct {
    // Existing fields (no breaking changes)
    ParentID       *uuid.UUID  `json:"parent_id,omitempty"`
    DerivationType *string     `json:"derivation_type,omitempty"`
    Limit          *int        `json:"limit,omitempty"`
    Offset         *int        `json:"offset,omitempty"`

    // NEW: Advanced filtering fields
    ParentIDs        []uuid.UUID          `json:"parent_ids,omitempty"`
    DerivationTypes  []string             `json:"derivation_types,omitempty"`
    Variant          *string              `json:"variant,omitempty"`
    Variants         []string             `json:"variants,omitempty"`
    TypeVariantPairs []TypeVariantPair    `json:"type_variant_pairs,omitempty"`
    ContentStatus    *string              `json:"content_status,omitempty"`
    CreatedAfter     *time.Time           `json:"created_after,omitempty"`
    CreatedBefore    *time.Time           `json:"created_before,omitempty"`
    SortBy           *string              `json:"sort_by,omitempty"`

    // NEW: URL and metadata inclusion options
    IncludeURLs      bool                 `json:"include_urls"`
    IncludeObjects   bool                 `json:"include_objects"`
    IncludeMetadata  bool                 `json:"include_metadata"`
}

// Supporting types
type TypeVariantPair struct {
    DerivationType string `json:"derivation_type"`
    Variant        string `json:"variant"`
}
```

**Backward Compatibility Guarantee:**
```go
// Existing code continues to work unchanged
params := ListDerivedContentParams{
    ParentID: &parentID,
    DerivationType: stringPtr("thumbnail"),
    Limit: intPtr(10),
}
// No breaking changes - all existing fields remain optional
// New fields default to zero values (false for bools, nil for slices/pointers)
```

### Phase 3: Enhanced Service Interface

```go
// Service interface - add new methods without breaking existing ones
type Service interface {
    // EXISTING methods (unchanged - maintain backward compatibility)
    ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error)

    // NEW: Enhanced filtering methods with URL support
    ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
    CountDerivedContent(ctx context.Context, params ListDerivedContentParams) (int64, error)

    // NEW: URL-enabled convenience methods
    ListDerivedByTypeAndVariant(ctx context.Context, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error)
    ListDerivedByVariants(ctx context.Context, parentID uuid.UUID, variants []string) ([]*DerivedContent, error)
    GetThumbnailsBySize(ctx context.Context, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error)
    GetRecentDerived(ctx context.Context, parentID uuid.UUID, since time.Time) ([]*DerivedContent, error)

    // NEW: URL-specific methods
    ListDerivedContentWithURLs(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
    GetDerivedContentWithURLs(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error)
}
```

## Implementation Details

### Repository Layer Enhancement

**Database Tag Usage**: The `db:"-"` tags ensure URLs aren't persisted:

```go
func (r *Repository) ListDerivedContent(ctx context.Context, params simplecontent.ListDerivedContentParams) ([]*simplecontent.DerivedContent, error) {
    query, args := r.buildEnhancedQuery(params)

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query derived content: %w", err)
    }
    defer rows.Close()

    var result []*simplecontent.DerivedContent
    for rows.Next() {
        derived := &simplecontent.DerivedContent{}

        // Only scan persisted fields - URLs remain empty until populated by service
        err := rows.Scan(
            &derived.ParentID,
            &derived.ContentID,
            &derived.DerivationType,
            &derived.DerivationParams,
            &derived.ProcessingMetadata,
            &derived.CreatedAt,
            &derived.UpdatedAt,
            &derived.DocumentType,
            &derived.Status,
            // URLs are NOT scanned - they remain empty
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan derived content: %w", err)
        }
        result = append(result, derived)
    }

    return result, nil
}

func (r *Repository) buildEnhancedQuery(params simplecontent.ListDerivedContentParams) (string, []interface{}) {
    query := `
        SELECT cd.parent_id, cd.content_id, cd.derivation_type, cd.variant,
               cd.derivation_params, cd.processing_metadata, cd.created_at, cd.updated_at,
               cd.document_type, cd.status
        FROM content_derived cd
        WHERE cd.deleted_at IS NULL
    `

    var args []interface{}
    argIndex := 1

    // Backward compatible filtering
    if params.ParentID != nil {
        query += fmt.Sprintf(" AND cd.parent_id = $%d", argIndex)
        args = append(args, *params.ParentID)
        argIndex++
    }
    if params.DerivationType != nil {
        query += fmt.Sprintf(" AND cd.derivation_type = $%d", argIndex)
        args = append(args, *params.DerivationType)
        argIndex++
    }

    // NEW: Enhanced filtering
    if len(params.ParentIDs) > 0 {
        placeholders := make([]string, len(params.ParentIDs))
        for i, parentID := range params.ParentIDs {
            placeholders[i] = fmt.Sprintf("$%d", argIndex)
            args = append(args, parentID)
            argIndex++
        }
        query += fmt.Sprintf(" AND cd.parent_id IN (%s)", strings.Join(placeholders, ","))
    }

    if len(params.DerivationTypes) > 0 {
        placeholders := make([]string, len(params.DerivationTypes))
        for i, dtype := range params.DerivationTypes {
            placeholders[i] = fmt.Sprintf("$%d", argIndex)
            args = append(args, dtype)
            argIndex++
        }
        query += fmt.Sprintf(" AND cd.derivation_type IN (%s)", strings.Join(placeholders, ","))
    }

    if params.Variant != nil {
        query += fmt.Sprintf(" AND cd.variant = $%d", argIndex)
        args = append(args, *params.Variant)
        argIndex++
    }

    if len(params.Variants) > 0 {
        placeholders := make([]string, len(params.Variants))
        for i, variant := range params.Variants {
            placeholders[i] = fmt.Sprintf("$%d", argIndex)
            args = append(args, variant)
            argIndex++
        }
        query += fmt.Sprintf(" AND cd.variant IN (%s)", strings.Join(placeholders, ","))
    }

    // Temporal filtering
    if params.CreatedAfter != nil {
        query += fmt.Sprintf(" AND cd.created_at > $%d", argIndex)
        args = append(args, *params.CreatedAfter)
        argIndex++
    }

    if params.CreatedBefore != nil {
        query += fmt.Sprintf(" AND cd.created_at < $%d", argIndex)
        args = append(args, *params.CreatedBefore)
        argIndex++
    }

    // Sorting
    switch params.SortBy {
    case nil, "":
        query += " ORDER BY cd.created_at DESC"
    case "created_at_asc":
        query += " ORDER BY cd.created_at ASC"
    case "created_at_desc":
        query += " ORDER BY cd.created_at DESC"
    case "type_variant":
        query += " ORDER BY cd.derivation_type, cd.variant"
    default:
        query += " ORDER BY cd.created_at DESC"
    }

    // Pagination
    if params.Limit != nil {
        query += fmt.Sprintf(" LIMIT $%d", argIndex)
        args = append(args, *params.Limit)
        argIndex++
    }
    if params.Offset != nil {
        query += fmt.Sprintf(" OFFSET $%d", argIndex)
        args = append(args, *params.Offset)
        argIndex++
    }

    return query, args
}
```

### Service Layer Enhancement with URL Population

```go
// Enhanced service method that populates URLs and metadata when requested
func (s *service) ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error) {
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
                log.Printf("Failed to enhance derived content %s: %v", d.ContentID, err)
            }
        }
    }

    return derived, nil
}

func (s *service) enhanceDerivedContent(ctx context.Context, derived *DerivedContent, params ListDerivedContentParams) error {
    // Extract variant from processing metadata or params
    derived.Variant = extractVariant(derived)

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

// Convenience methods
func (s *service) ListDerivedContentWithURLs(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error) {
    params.IncludeURLs = true
    return s.ListDerivedContentWithFilters(ctx, params)
}

func (s *service) GetThumbnailsBySize(ctx context.Context, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error) {
    variants := make([]string, len(sizes))
    for i, size := range sizes {
        variants[i] = fmt.Sprintf("thumbnail_%s", size)
    }

    params := ListDerivedContentParams{
        ParentID:       &parentID,
        DerivationType: stringPtr("thumbnail"),
        Variants:       variants,
        IncludeURLs:    true, // Always include URLs for thumbnails
    }
    return s.ListDerivedContentWithFilters(ctx, params)
}

// Keep existing simple method unchanged for backward compatibility
func (s *service) ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID: &parentID,
        // IncludeURLs: false (default) - maintains existing behavior
    }
    return s.repository.ListDerivedContent(ctx, params)
}
```

## Usage Examples

### Backward Compatibility (unchanged behavior)
```go
// Existing code works exactly as before
derived, err := service.ListDerivedByParent(ctx, parentID)
// derived[0].DownloadURL == "" (empty)
// derived[0].PreviewURL == ""   (empty)
// derived[0].Variant == ""      (empty)
```

### Enhanced Filtering with URLs
```go
// Get thumbnails with URLs and metadata
params := ListDerivedContentParams{
    ParentID:        &parentID,
    DerivationType:  stringPtr("thumbnail"),
    Variants:        []string{"thumbnail_256", "thumbnail_512"},
    IncludeURLs:     true,
    IncludeMetadata: true,
}
thumbnails, err := service.ListDerivedContentWithFilters(ctx, params)

// Now URLs and metadata are available
for _, thumb := range thumbnails {
    fmt.Printf("Variant: %s\n", thumb.Variant)           // "thumbnail_256"
    fmt.Printf("Thumbnail URL: %s\n", thumb.ThumbnailURL) // "https://..."
    fmt.Printf("Download URL: %s\n", thumb.DownloadURL)   // "https://..."
    fmt.Printf("File Size: %d\n", thumb.Metadata.FileSize) // 15420
}
```

### Convenience Methods
```go
// Get all thumbnail sizes with URLs
thumbnails, err := service.GetThumbnailsBySize(ctx, parentID, []string{"128", "256", "512"})

// Create URL map by size
thumbnailURLs := make(map[string]string)
for _, thumb := range thumbnails {
    size := strings.TrimPrefix(thumb.Variant, "thumbnail_")
    thumbnailURLs[size] = thumb.ThumbnailURL
}
// Result: {"128": "https://...", "256": "https://...", "512": "https://..."}
```

### Advanced Filtering Combinations
```go
// Complex filtering with temporal constraints and URLs
yesterday := time.Now().Add(-24 * time.Hour)
params := ListDerivedContentParams{
    ParentIDs:       []uuid.UUID{parent1ID, parent2ID, parent3ID},
    DerivationTypes: []string{"thumbnail", "preview"},
    CreatedAfter:    &yesterday,
    IncludeURLs:     true,
    SortBy:          stringPtr("created_at_desc"),
    Limit:           intPtr(20),
}
recent, err := service.ListDerivedContentWithFilters(ctx, params)
```

## Benefits of Combined Approach

### ✅ **Unified Experience**
```go
// One struct, one interface for all use cases
var derived []*DerivedContent

// Basic usage (fast)
derived = service.ListDerivedByParent(ctx, parentID)

// Enhanced usage (complete)
derived = service.ListDerivedContentWithURLs(ctx, ListDerivedContentParams{
    ParentID: &parentID,
    IncludeURLs: true,
})
```

### ✅ **Performance Control**
```go
// Fast: No URL generation overhead
basic := service.ListDerivedByParent(ctx, parentID)

// Selective enhancement: Only when needed
withURLs := service.ListDerivedContentWithURLs(ctx, ListDerivedContentParams{
    ParentID: &parentID,
    IncludeURLs: true, // URLs generated on-demand
})
```

### ✅ **Type Safety & Consistency**
```go
// Same type everywhere - no conversion needed
func displayThumbnails(items []*DerivedContent) {
    for _, item := range items {
        // Can access both basic fields AND URLs in same struct
        fmt.Printf("Type: %s, Variant: %s, URL: %s\n",
                   item.DerivationType, item.Variant, item.ThumbnailURL)
    }
}
```

### ✅ **JSON API Consistency**
```json
{
  "parent_id": "uuid",
  "content_id": "uuid",
  "derivation_type": "thumbnail",
  "variant": "thumbnail_256",
  "download_url": "https://...",      // Present when IncludeURLs=true
  "preview_url": "https://...",       // Present when IncludeURLs=true
  "thumbnail_url": "https://...",     // Present when IncludeURLs=true
  "metadata": {                       // Present when IncludeMetadata=true
    "file_size": 15420,
    "mime_type": "image/jpeg"
  },
  "objects": [...]                    // Present when IncludeObjects=true
}
```

## Database Optimizations

### Recommended Indexes
```sql
-- Enhanced indexes for new filtering capabilities
CREATE INDEX idx_content_derived_parent_variant ON content_derived(parent_id, variant) WHERE deleted_at IS NULL;
CREATE INDEX idx_content_derived_type_variant ON content_derived(derivation_type, variant) WHERE deleted_at IS NULL;
CREATE INDEX idx_content_derived_composite ON content_derived(parent_id, derivation_type, variant, created_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_content_derived_temporal ON content_derived(created_at) WHERE deleted_at IS NULL;

-- Support for multi-parent queries
CREATE INDEX idx_content_derived_multi_parent ON content_derived(parent_id, created_at) WHERE deleted_at IS NULL;
```

## Migration Path

### Phase 1: Core Enhancement (Week 1-2)
1. Add URL fields to `DerivedContent` struct with `db:"-"` tags
2. Extend `ListDerivedContentParams` with new optional fields
3. Update repository implementations for enhanced filtering
4. Add comprehensive tests ensuring backward compatibility

### Phase 2: Service Layer Enhancement (Week 3)
1. Add new service methods (`ListDerivedContentWithFilters`, `ListDerivedContentWithURLs`)
2. Implement URL population logic with performance optimization
3. Add convenience methods for common use cases
4. Keep existing methods unchanged

### Phase 3: Documentation and Examples (Week 4)
1. Update API documentation with new capabilities
2. Create migration guide showing progression from basic to advanced usage
3. Add comprehensive examples demonstrating URL integration
4. Performance benchmarks comparing basic vs enhanced operations

## Performance Considerations

### URL Generation Cost Analysis
- **Memory backend**: No cost (URLs not supported)
- **Filesystem backend**: Minimal cost (URL construction only)
- **S3 backend**: Moderate cost (presigned URL generation)

### Optimization Strategies

**Batch URL Generation:**
```go
func (s *service) populateURLsBatch(ctx context.Context, derived []*DerivedContent) error {
    // Group by storage backend for efficiency
    backendGroups := make(map[string][]*DerivedContent)

    for _, d := range derived {
        if len(d.Objects) > 0 {
            backend := d.Objects[0].StorageBackendName
            backendGroups[backend] = append(backendGroups[backend], d)
        }
    }

    // Generate URLs per backend to optimize backend-specific operations
    for backendName, group := range backendGroups {
        backend, _ := s.GetBackend(backendName)
        for _, d := range group {
            s.populateURLsForDerived(ctx, d, backend)
        }
    }

    return nil
}
```

**Caching Strategy:**
```go
// Cache frequently requested thumbnail URLs
type CachedDerivedContentService struct {
    service *service
    cache   Cache
    ttl     time.Duration
}

func (c *CachedDerivedContentService) GetThumbnailsBySize(ctx context.Context, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error) {
    key := fmt.Sprintf("thumbnails:%s:%v", parentID, sizes)

    if cached := c.cache.Get(key); cached != nil {
        return cached.([]*DerivedContent), nil
    }

    result, err := c.service.GetThumbnailsBySize(ctx, parentID, sizes)
    if err != nil {
        return nil, err
    }

    c.cache.Set(key, result, c.ttl)
    return result, nil
}
```

## Risk Mitigation

### Testing Strategy
1. **Backward Compatibility Tests**: Comprehensive test suite ensuring existing functionality unchanged
2. **Enhanced Functionality Tests**: Full test coverage for new filtering and URL capabilities
3. **Performance Tests**: Benchmarks comparing basic vs enhanced operations
4. **Integration Tests**: Real database scenarios with various storage backends

### Deployment Strategy
1. **Feature Flags**: Optional enhanced filtering and URL generation behind configuration
2. **Gradual Rollout**: Progressive enablement of features
3. **Monitoring**: Track usage patterns, performance impact, and error rates
4. **Rollback Plan**: Quick rollback capability if issues arise

## Conclusion

This comprehensive enhancement strategy provides:

- ✅ **Zero Breaking Changes**: All existing code continues working unchanged
- ✅ **Enhanced Filtering**: Advanced querying capabilities with type, variant, and temporal filters
- ✅ **Integrated URL Support**: Thumbnail, preview, and download URLs available on-demand
- ✅ **Performance Optimization**: URLs and metadata populated only when explicitly requested
- ✅ **Single Source of Truth**: One enhanced struct for all derived content operations
- ✅ **Natural Evolution**: Follows existing architectural patterns and principles
- ✅ **Production Ready**: Database optimizations, caching strategies, and risk mitigation

The enhanced `DerivedContent` struct with integrated URL support and advanced filtering capabilities provides a powerful, flexible foundation for content management while maintaining the simplicity that existing users depend on. This approach aligns perfectly with the simple-content library's architectural principles: **clean interfaces, progressive enhancement, and backward compatibility**.