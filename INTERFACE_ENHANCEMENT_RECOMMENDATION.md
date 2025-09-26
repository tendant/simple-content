# Enhanced Derived Content Filtering: Interface Design Recommendation

## Executive Summary

After analyzing the current simple-content library architecture and the enhanced derived content filtering requirements, I recommend **extending the existing interface** with a gradual enhancement approach that maintains backward compatibility while providing advanced filtering capabilities.

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

// Service interface method (service.go:50)
ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error)

// Repository interface method (interfaces.go:53)
ListDerivedContent(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
```

### Key Architectural Patterns Identified
1. **Parameter Objects Pattern**: Uses dedicated parameter structs (`ListDerivedContentParams`)
2. **Functional Options Pattern**: Service construction uses `WithRepository()`, `WithBlobStore()` etc.
3. **Interface Segregation**: Clean separation between `Service`, `Repository`, and `BlobStore` interfaces
4. **Backward Compatibility**: Simple methods like `ListDerivedByParent()` wrap complex repository calls
5. **Progressive Enhancement**: Existing codebase shows progression from simple to complex patterns

## Recommended Approach: **Extend Existing Interface**

### Why Not Create a New Interface?

❌ **Against New Interface:**
- **API Fragmentation**: Two different ways to achieve similar functionality
- **Maintenance Overhead**: Duplicate functionality across interfaces
- **Developer Confusion**: Users won't know which interface to choose
- **Code Duplication**: Similar validation, error handling, and business logic
- **Breaking Library Cohesion**: Goes against single-responsibility principle

✅ **For Extending Existing Interface:**
- **Unified Experience**: Single, comprehensive interface for all derived content operations
- **Natural Evolution**: Follows existing architectural patterns
- **Backward Compatibility**: Existing code continues working unchanged
- **Progressive Disclosure**: Simple use cases remain simple, complex cases become possible
- **Easier Testing**: Single interface to mock and test

## Detailed Enhancement Strategy

### Phase 1: Enhance Parameter Structure (Backward Compatible)

```go
// Enhanced ListDerivedContentParams - backward compatible extension
type ListDerivedContentParams struct {
    // Existing fields (no breaking changes)
    ParentID       *uuid.UUID  // Keep existing
    DerivationType *string     // Keep existing
    Limit          *int        // Keep existing
    Offset         *int        // Keep existing

    // NEW: Advanced filtering fields
    ParentIDs        []uuid.UUID          // Multiple parents
    DerivationTypes  []string             // Multiple types
    Variant          *string              // Single variant filter
    Variants         []string             // Multiple variants
    TypeVariantPairs []TypeVariantPair    // Precise combinations
    ContentStatus    *string              // Status filtering
    CreatedAfter     *time.Time           // Temporal filtering
    CreatedBefore    *time.Time           // Temporal filtering
    SortBy           *string              // Sorting control
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
```

### Phase 2: Enhance Service Interface

```go
// Service interface - add new methods without breaking existing ones
type Service interface {
    // EXISTING methods (unchanged)
    ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error)

    // NEW: Enhanced filtering methods
    ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
    CountDerivedContent(ctx context.Context, params ListDerivedContentParams) (int64, error)

    // NEW: Convenience methods (optional)
    ListDerivedByTypeAndVariant(ctx context.Context, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error)
    ListDerivedByVariants(ctx context.Context, parentID uuid.UUID, variants []string) ([]*DerivedContent, error)
    GetThumbnailsBySize(ctx context.Context, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error)
    GetRecentDerived(ctx context.Context, parentID uuid.UUID, since time.Time) ([]*DerivedContent, error)
}
```

### Phase 3: Repository Layer Enhancement

```go
// Repository interface - extend existing method capabilities
type Repository interface {
    // EXISTING method signature remains unchanged but implementation becomes more powerful
    ListDerivedContent(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)

    // NEW: Optional counting method for pagination
    CountDerivedContent(ctx context.Context, params ListDerivedContentParams) (int64, error)
}
```

## Implementation Strategy

### Step 1: Extend Parameter Structure
```go
// In interfaces.go - extend existing struct
type ListDerivedContentParams struct {
    // Existing fields (unchanged)
    ParentID       *uuid.UUID
    DerivationType *string
    Limit          *int
    Offset         *int

    // New advanced fields - all optional for backward compatibility
    ParentIDs        []uuid.UUID          `json:"parent_ids,omitempty"`
    DerivationTypes  []string             `json:"derivation_types,omitempty"`
    Variant          *string              `json:"variant,omitempty"`
    Variants         []string             `json:"variants,omitempty"`
    TypeVariantPairs []TypeVariantPair    `json:"type_variant_pairs,omitempty"`
    ContentStatus    *string              `json:"content_status,omitempty"`
    CreatedAfter     *time.Time           `json:"created_after,omitempty"`
    CreatedBefore    *time.Time           `json:"created_before,omitempty"`
    SortBy           *string              `json:"sort_by,omitempty"`
}
```

### Step 2: Enhance Repository Implementations

**Memory Repository (backward compatible):**
```go
func (r *Repository) ListDerivedContent(ctx context.Context, params simplecontent.ListDerivedContentParams) ([]*simplecontent.DerivedContent, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var result []*simplecontent.DerivedContent

    for _, derived := range r.derivedContent {
        if r.matchesEnhancedFilters(derived, params) {
            result = append(result, derived)
        }
    }

    // Apply sorting if specified
    r.sortDerivedContent(result, params)

    // Apply pagination if specified
    result = r.paginateDerivedContent(result, params)

    return result, nil
}

// Enhanced filtering logic
func (r *Repository) matchesEnhancedFilters(derived *simplecontent.DerivedContent, params simplecontent.ListDerivedContentParams) bool {
    // Existing logic for backward compatibility
    if params.ParentID != nil && *params.ParentID != derived.ParentID {
        return false
    }
    if params.DerivationType != nil && *params.DerivationType != derived.DerivationType {
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

    // Temporal filtering
    if params.CreatedAfter != nil && derived.CreatedAt.Before(*params.CreatedAfter) {
        return false
    }
    if params.CreatedBefore != nil && derived.CreatedAt.After(*params.CreatedBefore) {
        return false
    }

    return true
}
```

**PostgreSQL Repository (production-ready):**
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
        err := rows.Scan(
            &derived.ParentID, &derived.ContentID,
            &derived.DerivationType, &derived.Variant,
            &derived.DerivationParams, &derived.ProcessingMetadata,
            &derived.CreatedAt, &derived.UpdatedAt,
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
               cd.derivation_params, cd.processing_metadata, cd.created_at, cd.updated_at
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

### Step 3: Enhance Service Layer

```go
// In service_impl.go - add enhanced methods while keeping existing ones
func (s *service) ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error) {
    return s.repository.ListDerivedContent(ctx, params)
}

func (s *service) CountDerivedContent(ctx context.Context, params ListDerivedContentParams) (int64, error) {
    // For counting, we temporarily remove limits and get all matching records
    countParams := params
    countParams.Limit = nil
    countParams.Offset = nil

    results, err := s.repository.ListDerivedContent(ctx, countParams)
    if err != nil {
        return 0, err
    }

    return int64(len(results)), nil
}

// Convenience methods
func (s *service) ListDerivedByTypeAndVariant(ctx context.Context, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID:       &parentID,
        DerivationType: &derivationType,
        Variant:        &variant,
    }
    return s.ListDerivedContentWithFilters(ctx, params)
}

func (s *service) ListDerivedByVariants(ctx context.Context, parentID uuid.UUID, variants []string) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID: &parentID,
        Variants: variants,
    }
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
    }
    return s.ListDerivedContentWithFilters(ctx, params)
}

func (s *service) GetRecentDerived(ctx context.Context, parentID uuid.UUID, since time.Time) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID:     &parentID,
        CreatedAfter: &since,
        SortBy:       stringPtr("created_at_desc"),
    }
    return s.ListDerivedContentWithFilters(ctx, params)
}

// Keep existing simple method unchanged for backward compatibility
func (s *service) ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID: &parentID,
    }
    return s.repository.ListDerivedContent(ctx, params)
}
```

## Migration Path

### Phase 1: Non-Breaking Enhancement (Week 1-2)
1. Extend `ListDerivedContentParams` with new optional fields
2. Update repository implementations to handle enhanced filtering
3. Add comprehensive tests for new functionality
4. Ensure all existing tests still pass

### Phase 2: Service Layer Enhancement (Week 3)
1. Add new service methods (`ListDerivedContentWithFilters`, etc.)
2. Keep existing methods unchanged
3. Add convenience methods for common use cases

### Phase 3: Documentation and Examples (Week 4)
1. Update API documentation
2. Create migration guide for users wanting enhanced functionality
3. Add examples showing progression from simple to advanced usage

## Benefits of This Approach

### ✅ **Backward Compatibility**
```go
// Existing code works unchanged
derived, err := service.ListDerivedByParent(ctx, parentID)
```

### ✅ **Progressive Enhancement**
```go
// Simple enhancement
params := ListDerivedContentParams{
    ParentID: &parentID,
    Variant:  stringPtr("thumbnail_256"),
}
derived, err := service.ListDerivedContentWithFilters(ctx, params)

// Advanced enhancement
params := ListDerivedContentParams{
    ParentID:         &parentID,
    DerivationTypes:  []string{"thumbnail", "preview"},
    Variants:         []string{"thumbnail_256", "preview_web"},
    CreatedAfter:     &yesterday,
    SortBy:           stringPtr("created_at_desc"),
    Limit:            intPtr(50),
}
derived, err := service.ListDerivedContentWithFilters(ctx, params)
```

### ✅ **Unified Interface**
- Single source of truth for derived content operations
- Consistent error handling and validation
- Shared caching and optimization opportunities

### ✅ **Natural Evolution**
- Follows existing architectural patterns
- Maintains library design principles
- Easy to understand and adopt

## Database Optimizations

### Recommended Indexes
```sql
-- Enhanced indexes for new filtering capabilities
CREATE INDEX idx_content_derived_parent_variant ON content_derived(parent_id, variant) WHERE deleted_at IS NULL;
CREATE INDEX idx_content_derived_type_variant ON content_derived(derivation_type, variant) WHERE deleted_at IS NULL;
CREATE INDEX idx_content_derived_composite ON content_derived(parent_id, derivation_type, variant, created_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_content_derived_temporal ON content_derived(created_at) WHERE deleted_at IS NULL;
```

## Risk Mitigation

### Testing Strategy
1. **Backward Compatibility Tests**: Ensure existing functionality unchanged
2. **Enhanced Functionality Tests**: Comprehensive test coverage for new features
3. **Performance Tests**: Verify no regression in existing performance
4. **Integration Tests**: Test new filtering with real database scenarios

### Deployment Strategy
1. **Feature Flags**: Optional enhanced filtering behind configuration flags
2. **Gradual Rollout**: Enable enhanced features incrementally
3. **Monitoring**: Track usage patterns and performance impact
4. **Rollback Plan**: Quick rollback to previous behavior if needed

## Conclusion

**Recommendation: Extend Existing Interface**

This approach provides:
- ✅ **Zero Breaking Changes**: Existing code continues working
- ✅ **Enhanced Capabilities**: Advanced filtering becomes possible
- ✅ **Clean Architecture**: Single, cohesive interface
- ✅ **Easy Migration**: Optional adoption of new features
- ✅ **Future-Proof**: Foundation for additional enhancements

The enhanced `ListDerivedContentParams` becomes a powerful, flexible filtering system while maintaining the simplicity and reliability that existing users depend on.

This strategy aligns perfectly with the simple-content library's architectural principles: clean interfaces, progressive enhancement, and backward compatibility.