# Enhanced Derived Content Filtering

This document shows how to enhance the existing derived content filtering capabilities to support both derivation type and variant filtering, providing more precise control over which derived content is retrieved.

## Current State

The existing `ListDerivedContentParams` has limited filtering:

```go
type ListDerivedContentParams struct {
    ParentID       *uuid.UUID  // Filter by parent content
    DerivationType *string     // Filter by single derivation type
    Limit          *int        // Pagination limit
    Offset         *int        // Pagination offset
}
```

**Limitations:**
- Only supports single derivation type filtering
- No variant filtering capability
- Limited flexibility for complex queries

## Enhanced Filtering Implementation

### 1. Enhanced Parameter Structure

```go
// Enhanced ListDerivedContentParams with comprehensive filtering
type ListDerivedContentParams struct {
    // Parent filtering
    ParentID  *uuid.UUID   `json:"parent_id,omitempty"`
    ParentIDs []uuid.UUID  `json:"parent_ids,omitempty"` // Support multiple parents

    // Derivation type filtering (user-facing categories)
    DerivationType  *string   `json:"derivation_type,omitempty"`  // Single type
    DerivationTypes []string  `json:"derivation_types,omitempty"` // Multiple types

    // Variant filtering (specific implementations)
    Variant  *string   `json:"variant,omitempty"`  // Single variant
    Variants []string  `json:"variants,omitempty"` // Multiple variants

    // Combined filtering for advanced use cases
    TypeVariantPairs []TypeVariantPair `json:"type_variant_pairs,omitempty"`

    // Content status filtering
    ContentStatus  *string   `json:"content_status,omitempty"`
    ContentStatuses []string `json:"content_statuses,omitempty"`

    // Temporal filtering
    CreatedAfter  *time.Time `json:"created_after,omitempty"`
    CreatedBefore *time.Time `json:"created_before,omitempty"`
    UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
    UpdatedBefore *time.Time `json:"updated_before,omitempty"`

    // Sorting and pagination
    SortBy    *string `json:"sort_by,omitempty"`    // "created_at", "updated_at", "name"
    SortOrder *string `json:"sort_order,omitempty"` // "asc", "desc" (default: "desc")
    Limit     *int    `json:"limit,omitempty"`
    Offset    *int    `json:"offset,omitempty"`

    // Advanced options
    IncludeDeleted bool `json:"include_deleted,omitempty"` // Include soft-deleted items
}

// TypeVariantPair allows precise filtering by type+variant combinations
type TypeVariantPair struct {
    DerivationType string `json:"derivation_type"`
    Variant        string `json:"variant"`
}
```

### 2. Service Interface Enhancement

```go
// Add to Service interface
type Service interface {
    // ... existing methods ...

    // Enhanced derived content queries
    ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
    CountDerivedContent(ctx context.Context, params ListDerivedContentParams) (int64, error)

    // Convenience methods for common filtering patterns
    ListDerivedByTypeAndVariant(ctx context.Context, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error)
    ListDerivedByVariants(ctx context.Context, parentID uuid.UUID, variants []string) ([]*DerivedContent, error)
    ListDerivedByTypes(ctx context.Context, parentID uuid.UUID, derivationTypes []string) ([]*DerivedContent, error)
}
```

### 3. Repository Implementation

#### Memory Repository Enhancement

```go
func (r *Repository) ListDerivedContent(ctx context.Context, params simplecontent.ListDerivedContentParams) ([]*simplecontent.DerivedContent, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var result []*simplecontent.DerivedContent
    for _, derived := range r.derivedContents {
        if r.matchesFilters(derived, params) {
            derivedCopy := *derived
            result = append(result, &derivedCopy)
        }
    }

    // Apply sorting
    r.sortDerivedContent(result, params)

    // Apply pagination
    return r.paginateDerivedContent(result, params), nil
}

func (r *Repository) matchesFilters(derived *simplecontent.DerivedContent, params simplecontent.ListDerivedContentParams) bool {
    // Parent ID filtering
    if params.ParentID != nil && derived.ParentID != *params.ParentID {
        return false
    }

    if len(params.ParentIDs) > 0 {
        found := false
        for _, parentID := range params.ParentIDs {
            if derived.ParentID == parentID {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }

    // Derivation type filtering
    if params.DerivationType != nil && derived.DerivationType != *params.DerivationType {
        return false
    }

    if len(params.DerivationTypes) > 0 {
        found := false
        for _, derivationType := range params.DerivationTypes {
            if derived.DerivationType == derivationType {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }

    // Variant filtering - NEW FEATURE
    if params.Variant != nil {
        // Extract variant from derived content (stored in DerivationType field in some implementations)
        // or from ProcessingMetadata, depending on your storage strategy
        derivedVariant := r.extractVariant(derived)
        if derivedVariant != *params.Variant {
            return false
        }
    }

    if len(params.Variants) > 0 {
        derivedVariant := r.extractVariant(derived)
        found := false
        for _, variant := range params.Variants {
            if derivedVariant == variant {
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
        derivedVariant := r.extractVariant(derived)
        found := false
        for _, pair := range params.TypeVariantPairs {
            if derived.DerivationType == pair.DerivationType && derivedVariant == pair.Variant {
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

    if params.UpdatedAfter != nil && derived.UpdatedAt.Before(*params.UpdatedAfter) {
        return false
    }

    if params.UpdatedBefore != nil && derived.UpdatedAt.After(*params.UpdatedBefore) {
        return false
    }

    return true
}

func (r *Repository) extractVariant(derived *simplecontent.DerivedContent) string {
    // Strategy 1: Variant stored in ProcessingMetadata
    if variant, exists := derived.ProcessingMetadata["variant"]; exists {
        if variantStr, ok := variant.(string); ok {
            return variantStr
        }
    }

    // Strategy 2: Variant stored in DerivationParams
    if variant, exists := derived.DerivationParams["variant"]; exists {
        if variantStr, ok := variant.(string); ok {
            return variantStr
        }
    }

    // Strategy 3: Infer variant from DerivationType (if it follows pattern like "thumbnail_256")
    parts := strings.Split(derived.DerivationType, "_")
    if len(parts) > 1 {
        return derived.DerivationType // Return full string as variant
    }

    return derived.DerivationType // Fallback
}

func (r *Repository) sortDerivedContent(result []*simplecontent.DerivedContent, params simplecontent.ListDerivedContentParams) {
    sortBy := "created_at"
    if params.SortBy != nil {
        sortBy = *params.SortBy
    }

    sortOrder := "desc"
    if params.SortOrder != nil {
        sortOrder = *params.SortOrder
    }

    sort.Slice(result, func(i, j int) bool {
        switch sortBy {
        case "created_at":
            if sortOrder == "asc" {
                return result[i].CreatedAt.Before(result[j].CreatedAt)
            }
            return result[i].CreatedAt.After(result[j].CreatedAt)
        case "updated_at":
            if sortOrder == "asc" {
                return result[i].UpdatedAt.Before(result[j].UpdatedAt)
            }
            return result[i].UpdatedAt.After(result[j].UpdatedAt)
        default:
            return result[i].CreatedAt.After(result[j].CreatedAt)
        }
    })
}

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
```

#### PostgreSQL Repository Enhancement

```go
func (r *Repository) ListDerivedContent(ctx context.Context, params simplecontent.ListDerivedContentParams) ([]*simplecontent.DerivedContent, error) {
    query, args := r.buildDerivedContentQuery(params)

    rows, err := r.pool.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query derived content: %w", err)
    }
    defer rows.Close()

    var result []*simplecontent.DerivedContent
    for rows.Next() {
        derived := &simplecontent.DerivedContent{}
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
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan derived content: %w", err)
        }
        result = append(result, derived)
    }

    return result, rows.Err()
}

func (r *Repository) buildDerivedContentQuery(params simplecontent.ListDerivedContentParams) (string, []interface{}) {
    query := `
        SELECT cd.parent_id, cd.content_id, cd.variant as derivation_type,
               cd.derivation_params, cd.processing_metadata,
               cd.created_at, cd.updated_at, c.document_type, c.status
        FROM content_derived cd
        JOIN content c ON cd.content_id = c.id
        WHERE cd.deleted_at IS NULL AND c.deleted_at IS NULL
    `

    var conditions []string
    var args []interface{}
    argCount := 0

    // Parent ID filtering
    if params.ParentID != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.parent_id = $%d", argCount))
        args = append(args, *params.ParentID)
    }

    if len(params.ParentIDs) > 0 {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.parent_id = ANY($%d)", argCount))
        args = append(args, pq.Array(params.ParentIDs))
    }

    // Derivation type filtering
    if params.DerivationType != nil {
        argCount++
        // Assuming derivation type is stored in the content table
        conditions = append(conditions, fmt.Sprintf("c.derivation_type = $%d", argCount))
        args = append(args, *params.DerivationType)
    }

    if len(params.DerivationTypes) > 0 {
        argCount++
        conditions = append(conditions, fmt.Sprintf("c.derivation_type = ANY($%d)", argCount))
        args = append(args, pq.Array(params.DerivationTypes))
    }

    // Variant filtering - NEW FEATURE
    if params.Variant != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.variant = $%d", argCount))
        args = append(args, *params.Variant)
    }

    if len(params.Variants) > 0 {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.variant = ANY($%d)", argCount))
        args = append(args, pq.Array(params.Variants))
    }

    // Type+Variant pair filtering
    if len(params.TypeVariantPairs) > 0 {
        pairConditions := make([]string, len(params.TypeVariantPairs))
        for i, pair := range params.TypeVariantPairs {
            argCount += 2
            pairConditions[i] = fmt.Sprintf("(c.derivation_type = $%d AND cd.variant = $%d)", argCount-1, argCount)
            args = append(args, pair.DerivationType, pair.Variant)
        }
        conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(pairConditions, " OR ")))
    }

    // Content status filtering
    if params.ContentStatus != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("c.status = $%d", argCount))
        args = append(args, *params.ContentStatus)
    }

    if len(params.ContentStatuses) > 0 {
        argCount++
        conditions = append(conditions, fmt.Sprintf("c.status = ANY($%d)", argCount))
        args = append(args, pq.Array(params.ContentStatuses))
    }

    // Temporal filtering
    if params.CreatedAfter != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.created_at >= $%d", argCount))
        args = append(args, *params.CreatedAfter)
    }

    if params.CreatedBefore != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.created_at <= $%d", argCount))
        args = append(args, *params.CreatedBefore)
    }

    if params.UpdatedAfter != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.updated_at >= $%d", argCount))
        args = append(args, *params.UpdatedAfter)
    }

    if params.UpdatedBefore != nil {
        argCount++
        conditions = append(conditions, fmt.Sprintf("cd.updated_at <= $%d", argCount))
        args = append(args, *params.UpdatedBefore)
    }

    // Add conditions to query
    if len(conditions) > 0 {
        query += " AND " + strings.Join(conditions, " AND ")
    }

    // Sorting
    sortBy := "cd.created_at"
    if params.SortBy != nil {
        switch *params.SortBy {
        case "created_at":
            sortBy = "cd.created_at"
        case "updated_at":
            sortBy = "cd.updated_at"
        case "name":
            sortBy = "c.name"
        }
    }

    sortOrder := "DESC"
    if params.SortOrder != nil && strings.ToUpper(*params.SortOrder) == "ASC" {
        sortOrder = "ASC"
    }

    query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

    // Pagination
    if params.Limit != nil {
        argCount++
        query += fmt.Sprintf(" LIMIT $%d", argCount)
        args = append(args, *params.Limit)
    }

    if params.Offset != nil {
        argCount++
        query += fmt.Sprintf(" OFFSET $%d", argCount)
        args = append(args, *params.Offset)
    }

    return query, args
}
```

### 4. Service Implementation

```go
// Enhanced service methods
func (s *service) ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error) {
    return s.repository.ListDerivedContent(ctx, params)
}

func (s *service) CountDerivedContent(ctx context.Context, params ListDerivedContentParams) (int64, error) {
    // Remove pagination for counting
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

func (s *service) ListDerivedByTypes(ctx context.Context, parentID uuid.UUID, derivationTypes []string) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID:        &parentID,
        DerivationTypes: derivationTypes,
    }
    return s.ListDerivedContentWithFilters(ctx, params)
}
```

### 5. HTTP API Enhancement

```go
// Enhanced HTTP endpoints
func (s *HTTPServer) handleListDerivedContentAdvanced(w http.ResponseWriter, r *http.Request) {
    params := parseAdvancedDerivedParams(r)

    results, err := s.service.ListDerivedContentWithFilters(r.Context(), params)
    if err != nil {
        writeServiceError(w, err)
        return
    }

    // Get count for pagination info
    count, err := s.service.CountDerivedContent(r.Context(), params)
    if err != nil {
        log.Printf("Warning: failed to get count: %v", err)
        count = int64(len(results))
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "results": results,
        "count":   count,
        "limit":   params.Limit,
        "offset":  params.Offset,
    })
}

func parseAdvancedDerivedParams(r *http.Request) ListDerivedContentParams {
    params := ListDerivedContentParams{}
    query := r.URL.Query()

    // Parent ID filtering
    if parentIDStr := query.Get("parent_id"); parentIDStr != "" {
        if parentID, err := uuid.Parse(parentIDStr); err == nil {
            params.ParentID = &parentID
        }
    }

    if parentIDsStr := query.Get("parent_ids"); parentIDsStr != "" {
        parentIDStrs := strings.Split(parentIDsStr, ",")
        var parentIDs []uuid.UUID
        for _, idStr := range parentIDStrs {
            if id, err := uuid.Parse(strings.TrimSpace(idStr)); err == nil {
                parentIDs = append(parentIDs, id)
            }
        }
        if len(parentIDs) > 0 {
            params.ParentIDs = parentIDs
        }
    }

    // Derivation type filtering
    if derivationType := query.Get("derivation_type"); derivationType != "" {
        params.DerivationType = &derivationType
    }

    if derivationTypesStr := query.Get("derivation_types"); derivationTypesStr != "" {
        types := strings.Split(derivationTypesStr, ",")
        for i, t := range types {
            types[i] = strings.TrimSpace(t)
        }
        params.DerivationTypes = types
    }

    // Variant filtering - NEW FEATURE
    if variant := query.Get("variant"); variant != "" {
        params.Variant = &variant
    }

    if variantsStr := query.Get("variants"); variantsStr != "" {
        variants := strings.Split(variantsStr, ",")
        for i, v := range variants {
            variants[i] = strings.TrimSpace(v)
        }
        params.Variants = variants
    }

    // Type+Variant pairs
    if pairsStr := query.Get("type_variant_pairs"); pairsStr != "" {
        pairs := strings.Split(pairsStr, ",")
        var typeVariantPairs []TypeVariantPair
        for _, pair := range pairs {
            parts := strings.Split(strings.TrimSpace(pair), ":")
            if len(parts) == 2 {
                typeVariantPairs = append(typeVariantPairs, TypeVariantPair{
                    DerivationType: strings.TrimSpace(parts[0]),
                    Variant:        strings.TrimSpace(parts[1]),
                })
            }
        }
        params.TypeVariantPairs = typeVariantPairs
    }

    // Temporal filtering
    if createdAfterStr := query.Get("created_after"); createdAfterStr != "" {
        if t, err := time.Parse(time.RFC3339, createdAfterStr); err == nil {
            params.CreatedAfter = &t
        }
    }

    if createdBeforeStr := query.Get("created_before"); createdBeforeStr != "" {
        if t, err := time.Parse(time.RFC3339, createdBeforeStr); err == nil {
            params.CreatedBefore = &t
        }
    }

    // Sorting
    if sortBy := query.Get("sort_by"); sortBy != "" {
        params.SortBy = &sortBy
    }

    if sortOrder := query.Get("sort_order"); sortOrder != "" {
        params.SortOrder = &sortOrder
    }

    // Pagination
    if limitStr := query.Get("limit"); limitStr != "" {
        if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
            params.Limit = &limit
        }
    }

    if offsetStr := query.Get("offset"); offsetStr != "" {
        if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
            params.Offset = &offset
        }
    }

    return params
}
```

## Usage Examples

### 1. Filter by Single Derivation Type and Variant

```go
// Get thumbnail_256 variants only
params := ListDerivedContentParams{
    ParentID:       &parentID,
    DerivationType: stringPtr("thumbnail"),
    Variant:        stringPtr("thumbnail_256"),
}
results, err := svc.ListDerivedContentWithFilters(ctx, params)
```

```bash
# HTTP API
curl "http://localhost:8080/api/v1/derived-content?parent_id=uuid&derivation_type=thumbnail&variant=thumbnail_256"
```

### 2. Filter by Multiple Variants

```go
// Get all thumbnail variants
params := ListDerivedContentParams{
    ParentID: &parentID,
    Variants: []string{"thumbnail_128", "thumbnail_256", "thumbnail_512"},
}
results, err := svc.ListDerivedContentWithFilters(ctx, params)
```

```bash
# HTTP API
curl "http://localhost:8080/api/v1/derived-content?parent_id=uuid&variants=thumbnail_128,thumbnail_256,thumbnail_512"
```

### 3. Filter by Multiple Derivation Types

```go
// Get thumbnails and previews
params := ListDerivedContentParams{
    ParentID:        &parentID,
    DerivationTypes: []string{"thumbnail", "preview"},
}
results, err := svc.ListDerivedContentWithFilters(ctx, params)
```

```bash
# HTTP API
curl "http://localhost:8080/api/v1/derived-content?parent_id=uuid&derivation_types=thumbnail,preview"
```

### 4. Complex Type+Variant Combinations

```go
// Get specific type+variant combinations
params := ListDerivedContentParams{
    ParentID: &parentID,
    TypeVariantPairs: []TypeVariantPair{
        {DerivationType: "thumbnail", Variant: "thumbnail_256"},
        {DerivationType: "preview", Variant: "preview_web"},
    },
}
results, err := svc.ListDerivedContentWithFilters(ctx, params)
```

```bash
# HTTP API
curl "http://localhost:8080/api/v1/derived-content?parent_id=uuid&type_variant_pairs=thumbnail:thumbnail_256,preview:preview_web"
```

### 5. Temporal and Status Filtering

```go
// Get recent thumbnails that are uploaded
now := time.Now()
yesterday := now.Add(-24 * time.Hour)
uploaded := "uploaded"

params := ListDerivedContentParams{
    ParentID:       &parentID,
    DerivationType: stringPtr("thumbnail"),
    CreatedAfter:   &yesterday,
    ContentStatus:  &uploaded,
    SortBy:         stringPtr("created_at"),
    SortOrder:      stringPtr("desc"),
    Limit:          intPtr(10),
}
results, err := svc.ListDerivedContentWithFilters(ctx, params)
```

```bash
# HTTP API
curl "http://localhost:8080/api/v1/derived-content?parent_id=uuid&derivation_type=thumbnail&created_after=2024-01-01T00:00:00Z&content_status=uploaded&sort_by=created_at&limit=10"
```

### 6. Convenience Methods

```go
// Simple convenience methods
thumbnails, err := svc.ListDerivedByTypeAndVariant(ctx, parentID, "thumbnail", "thumbnail_256")

allThumbnails, err := svc.ListDerivedByVariants(ctx, parentID, []string{
    "thumbnail_128", "thumbnail_256", "thumbnail_512",
})

mediaContent, err := svc.ListDerivedByTypes(ctx, parentID, []string{
    "thumbnail", "preview", "transcode",
})
```

## Performance Considerations

### 1. Database Indexes

```sql
-- Recommended indexes for efficient filtering
CREATE INDEX idx_content_derived_parent_type ON content_derived(parent_id, derivation_type);
CREATE INDEX idx_content_derived_parent_variant ON content_derived(parent_id, variant);
CREATE INDEX idx_content_derived_type_variant ON content_derived(derivation_type, variant);
CREATE INDEX idx_content_derived_created_at ON content_derived(created_at);
CREATE INDEX idx_content_derived_status ON content_derived(status);

-- Composite index for common filtering patterns
CREATE INDEX idx_content_derived_parent_type_variant ON content_derived(parent_id, derivation_type, variant);
```

### 2. Query Optimization

```go
// Use prepared statements for repeated queries
func (r *Repository) prepareStatements() error {
    r.stmtByTypeVariant, err = r.pool.Prepare(ctx, "by_type_variant", `
        SELECT cd.parent_id, cd.content_id, cd.variant, cd.derivation_params, cd.processing_metadata, cd.created_at, cd.updated_at, c.document_type, c.status
        FROM content_derived cd
        JOIN content c ON cd.content_id = c.id
        WHERE cd.parent_id = $1 AND c.derivation_type = $2 AND cd.variant = $3
        AND cd.deleted_at IS NULL AND c.deleted_at IS NULL
        ORDER BY cd.created_at DESC
    `)
    return err
}
```

### 3. Caching Strategy

```go
// Cache frequent filter combinations
type CachedDerivedService struct {
    service Service
    cache   Cache
}

func (c *CachedDerivedService) ListDerivedContentWithFilters(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error) {
    // Create cache key from params
    key := fmt.Sprintf("derived:%s", hashParams(params))

    if cached := c.cache.Get(key); cached != nil {
        return cached.([]*DerivedContent), nil
    }

    results, err := c.service.ListDerivedContentWithFilters(ctx, params)
    if err != nil {
        return nil, err
    }

    // Cache for 5 minutes
    c.cache.Set(key, results, 5*time.Minute)
    return results, nil
}
```

## Migration Strategy

### 1. Backward Compatibility

```go
// Maintain backward compatibility with existing API
func (s *service) ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error) {
    params := ListDerivedContentParams{
        ParentID: &parentID,
    }
    return s.ListDerivedContentWithFilters(ctx, params)
}
```

### 2. Gradual Rollout

1. **Phase 1**: Add new filtering parameters as optional
2. **Phase 2**: Update clients to use enhanced filtering
3. **Phase 3**: Deprecate old simple methods (with grace period)
4. **Phase 4**: Remove deprecated methods

### 3. Testing Strategy

```go
func TestEnhancedDerivedFiltering(t *testing.T) {
    tests := []struct {
        name     string
        params   ListDerivedContentParams
        expected int
    }{
        {
            name: "filter by single variant",
            params: ListDerivedContentParams{
                ParentID: &parentID,
                Variant:  stringPtr("thumbnail_256"),
            },
            expected: 1,
        },
        {
            name: "filter by multiple variants",
            params: ListDerivedContentParams{
                ParentID: &parentID,
                Variants: []string{"thumbnail_128", "thumbnail_256"},
            },
            expected: 2,
        },
        {
            name: "filter by type and variant combination",
            params: ListDerivedContentParams{
                ParentID: &parentID,
                TypeVariantPairs: []TypeVariantPair{
                    {DerivationType: "thumbnail", Variant: "thumbnail_256"},
                },
            },
            expected: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            results, err := svc.ListDerivedContentWithFilters(ctx, tt.params)
            require.NoError(t, err)
            assert.Len(t, results, tt.expected)
        })
    }
}
```

This enhanced filtering system provides comprehensive control over derived content queries while maintaining backward compatibility and performance considerations.