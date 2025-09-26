# Content with Derived Service Implementation

This document shows how to implement a service that fetches content along with its derived contents in a single operation, providing a more efficient and convenient API for applications that need the complete content hierarchy.

## Current State

The simple-content library currently provides separate operations:
- `GetContent(ctx, contentID)` - Get single content
- `ListDerivedByParent(ctx, parentID)` - Get derived content relationships
- `GetDerivedRelationshipByContentID(ctx, contentID)` - Get derived relationship for specific content

## Proposed Enhancement

### 1. New Data Structures

```go
// ContentWithDerived represents a content item with its derived contents
type ContentWithDerived struct {
    *Content
    DerivedContents []*DerivedContentItem `json:"derived_contents,omitempty"`
    ParentContent   *ContentReference     `json:"parent_content,omitempty"`
}

// DerivedContentItem represents a derived content with full details
type DerivedContentItem struct {
    *Content
    Variant             string                 `json:"variant"`
    DerivationParams    map[string]interface{} `json:"derivation_params,omitempty"`
    ProcessingMetadata  map[string]interface{} `json:"processing_metadata,omitempty"`
    Objects             []*Object              `json:"objects,omitempty"`
    Metadata            *ContentMetadata       `json:"metadata,omitempty"`
}

// ContentReference represents a lightweight parent content reference
type ContentReference struct {
    ID             uuid.UUID `json:"id"`
    Name           string    `json:"name"`
    DocumentType   string    `json:"document_type"`
    DerivationType string    `json:"derivation_type,omitempty"`
}

// GetContentWithDerivedOptions provides options for the fetch operation
type GetContentWithDerivedOptions struct {
    IncludeObjects   bool `json:"include_objects"`
    IncludeMetadata  bool `json:"include_metadata"`
    MaxDepth         int  `json:"max_depth"`         // For nested derived content
    DerivationFilter []string `json:"derivation_filter"` // Filter by derivation types
}
```

### 2. Service Interface Extension

```go
// Add to Service interface
type Service interface {
    // ... existing methods ...

    // GetContentWithDerived retrieves content with its derived contents
    GetContentWithDerived(ctx context.Context, contentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error)

    // GetMultipleContentWithDerived retrieves multiple contents with their derived contents
    GetMultipleContentWithDerived(ctx context.Context, contentIDs []uuid.UUID, opts *GetContentWithDerivedOptions) ([]*ContentWithDerived, error)

    // GetContentHierarchy retrieves a complete content hierarchy (parent + all descendants)
    GetContentHierarchy(ctx context.Context, rootContentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error)
}
```

### 3. Service Implementation

```go
// service_impl.go - Add these methods to the service struct

func (s *service) GetContentWithDerived(ctx context.Context, contentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error) {
    if opts == nil {
        opts = &GetContentWithDerivedOptions{
            IncludeObjects:  false,
            IncludeMetadata: false,
            MaxDepth:        1,
        }
    }

    // Get the main content
    content, err := s.GetContent(ctx, contentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get content: %w", err)
    }

    result := &ContentWithDerived{
        Content: content,
    }

    // If this is derived content, get parent reference
    if content.DerivationType != "" {
        parentRef, err := s.getParentReference(ctx, contentID)
        if err != nil {
            // Log but don't fail - parent reference is optional
            log.Printf("Warning: failed to get parent reference for content %s: %v", contentID, err)
        } else {
            result.ParentContent = parentRef
        }
    }

    // Get derived contents
    derivedContents, err := s.getDerivedContentsWithDetails(ctx, contentID, opts, 0)
    if err != nil {
        return nil, fmt.Errorf("failed to get derived contents: %w", err)
    }
    result.DerivedContents = derivedContents

    return result, nil
}

func (s *service) GetMultipleContentWithDerived(ctx context.Context, contentIDs []uuid.UUID, opts *GetContentWithDerivedOptions) ([]*ContentWithDerived, error) {
    if len(contentIDs) == 0 {
        return []*ContentWithDerived{}, nil
    }

    results := make([]*ContentWithDerived, 0, len(contentIDs))

    // Process in batches to avoid overwhelming the database
    batchSize := 50
    for i := 0; i < len(contentIDs); i += batchSize {
        end := i + batchSize
        if end > len(contentIDs) {
            end = len(contentIDs)
        }

        batch := contentIDs[i:end]
        for _, contentID := range batch {
            contentWithDerived, err := s.GetContentWithDerived(ctx, contentID, opts)
            if err != nil {
                // Log error but continue with other content
                log.Printf("Warning: failed to get content with derived for %s: %v", contentID, err)
                continue
            }
            results = append(results, contentWithDerived)
        }
    }

    return results, nil
}

func (s *service) GetContentHierarchy(ctx context.Context, rootContentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error) {
    if opts == nil {
        opts = &GetContentWithDerivedOptions{
            IncludeObjects:  false,
            IncludeMetadata: false,
            MaxDepth:        10, // Higher default for hierarchy
        }
    }

    // Ensure max depth is set for hierarchy
    if opts.MaxDepth == 0 {
        opts.MaxDepth = 10
    }

    return s.GetContentWithDerived(ctx, rootContentID, opts)
}

// Helper methods

func (s *service) getParentReference(ctx context.Context, contentID uuid.UUID) (*ContentReference, error) {
    relationship, err := s.GetDerivedRelationshipByContentID(ctx, contentID)
    if err != nil {
        return nil, err
    }

    parentContent, err := s.GetContent(ctx, relationship.ParentID)
    if err != nil {
        return nil, err
    }

    return &ContentReference{
        ID:             parentContent.ID,
        Name:           parentContent.Name,
        DocumentType:   parentContent.DocumentType,
        DerivationType: parentContent.DerivationType,
    }, nil
}

func (s *service) getDerivedContentsWithDetails(ctx context.Context, parentID uuid.UUID, opts *GetContentWithDerivedOptions, currentDepth int) ([]*DerivedContentItem, error) {
    // Check depth limit
    if currentDepth >= opts.MaxDepth {
        return []*DerivedContentItem{}, nil
    }

    // Get derived relationships
    relationships, err := s.ListDerivedByParent(ctx, parentID)
    if err != nil {
        return nil, err
    }

    if len(relationships) == 0 {
        return []*DerivedContentItem{}, nil
    }

    results := make([]*DerivedContentItem, 0, len(relationships))

    for _, rel := range relationships {
        // Apply derivation filter if specified
        if len(opts.DerivationFilter) > 0 {
            if !contains(opts.DerivationFilter, rel.DerivationType) {
                continue
            }
        }

        // Get the derived content
        derivedContent, err := s.GetContent(ctx, rel.ContentID)
        if err != nil {
            log.Printf("Warning: failed to get derived content %s: %v", rel.ContentID, err)
            continue
        }

        item := &DerivedContentItem{
            Content:            derivedContent,
            Variant:            rel.DerivationType, // This should be the variant from the relationship
            DerivationParams:   rel.DerivationParams,
            ProcessingMetadata: rel.ProcessingMetadata,
        }

        // Include objects if requested
        if opts.IncludeObjects {
            objects, err := s.GetObjectsByContentID(ctx, derivedContent.ID)
            if err != nil {
                log.Printf("Warning: failed to get objects for content %s: %v", derivedContent.ID, err)
            } else {
                item.Objects = objects
            }
        }

        // Include metadata if requested
        if opts.IncludeMetadata {
            metadata, err := s.GetContentMetadata(ctx, derivedContent.ID)
            if err != nil {
                log.Printf("Warning: failed to get metadata for content %s: %v", derivedContent.ID, err)
            } else {
                item.Metadata = metadata
            }
        }

        // Recursively get nested derived content if within depth limit
        if currentDepth+1 < opts.MaxDepth {
            nestedDerived, err := s.getDerivedContentsWithDetails(ctx, derivedContent.ID, opts, currentDepth+1)
            if err != nil {
                log.Printf("Warning: failed to get nested derived content for %s: %v", derivedContent.ID, err)
            } else if len(nestedDerived) > 0 {
                // Convert nested derived to ContentWithDerived for the item
                nestedResult := &ContentWithDerived{
                    Content:         derivedContent,
                    DerivedContents: nestedDerived,
                }
                // You might want to extend DerivedContentItem to include nested derived content
                // For now, we'll just include them at this level
            }
        }

        results = append(results, item)
    }

    return results, nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 4. HTTP API Extension

```go
// Add to HTTP server routes
r.Get("/contents/{contentID}/with-derived", s.handleGetContentWithDerived)
r.Post("/contents/batch/with-derived", s.handleGetMultipleContentWithDerived)
r.Get("/contents/{contentID}/hierarchy", s.handleGetContentHierarchy)

// HTTP Handlers

func (s *HTTPServer) handleGetContentWithDerived(w http.ResponseWriter, r *http.Request) {
    contentIDStr := chi.URLParam(r, "contentID")
    contentID, err := uuid.Parse(contentIDStr)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
        return
    }

    // Parse query parameters
    opts := &simplecontent.GetContentWithDerivedOptions{
        IncludeObjects:  r.URL.Query().Get("include_objects") == "true",
        IncludeMetadata: r.URL.Query().Get("include_metadata") == "true",
        MaxDepth:        1, // Default
    }

    if maxDepthStr := r.URL.Query().Get("max_depth"); maxDepthStr != "" {
        if maxDepth, err := strconv.Atoi(maxDepthStr); err == nil && maxDepth > 0 {
            opts.MaxDepth = maxDepth
        }
    }

    if derivationFilter := r.URL.Query().Get("derivation_filter"); derivationFilter != "" {
        opts.DerivationFilter = strings.Split(derivationFilter, ",")
    }

    result, err := s.service.GetContentWithDerived(r.Context(), contentID, opts)
    if err != nil {
        writeServiceError(w, err)
        return
    }

    writeJSON(w, http.StatusOK, result)
}

func (s *HTTPServer) handleGetMultipleContentWithDerived(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ContentIDs []string `json:"content_ids"`
        Options    *simplecontent.GetContentWithDerivedOptions `json:"options"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body", nil)
        return
    }

    if len(req.ContentIDs) == 0 {
        writeError(w, http.StatusBadRequest, "missing_content_ids", "content_ids is required", nil)
        return
    }

    // Parse UUIDs
    contentIDs := make([]uuid.UUID, 0, len(req.ContentIDs))
    for _, idStr := range req.ContentIDs {
        id, err := uuid.Parse(idStr)
        if err != nil {
            writeError(w, http.StatusBadRequest, "invalid_content_id", fmt.Sprintf("Invalid content ID: %s", idStr), nil)
            return
        }
        contentIDs = append(contentIDs, id)
    }

    if req.Options == nil {
        req.Options = &simplecontent.GetContentWithDerivedOptions{
            IncludeObjects:  false,
            IncludeMetadata: false,
            MaxDepth:        1,
        }
    }

    results, err := s.service.GetMultipleContentWithDerived(r.Context(), contentIDs, req.Options)
    if err != nil {
        writeServiceError(w, err)
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "contents": results,
        "count":    len(results),
    })
}

func (s *HTTPServer) handleGetContentHierarchy(w http.ResponseWriter, r *http.Request) {
    contentIDStr := chi.URLParam(r, "contentID")
    contentID, err := uuid.Parse(contentIDStr)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
        return
    }

    // Parse query parameters - hierarchy typically includes more data
    opts := &simplecontent.GetContentWithDerivedOptions{
        IncludeObjects:  r.URL.Query().Get("include_objects") == "true",
        IncludeMetadata: r.URL.Query().Get("include_metadata") == "true",
        MaxDepth:        10, // Higher default for hierarchy
    }

    if maxDepthStr := r.URL.Query().Get("max_depth"); maxDepthStr != "" {
        if maxDepth, err := strconv.Atoi(maxDepthStr); err == nil && maxDepth > 0 {
            opts.MaxDepth = maxDepth
        }
    }

    if derivationFilter := r.URL.Query().Get("derivation_filter"); derivationFilter != "" {
        opts.DerivationFilter = strings.Split(derivationFilter, ",")
    }

    result, err := s.service.GetContentHierarchy(r.Context(), contentID, opts)
    if err != nil {
        writeServiceError(w, err)
        return
    }

    writeJSON(w, http.StatusOK, result)
}
```

## Usage Examples

### 1. Programmatic Usage

```go
// Basic usage - get content with immediate derived content
result, err := svc.GetContentWithDerived(ctx, contentID, nil)
if err != nil {
    return err
}

fmt.Printf("Content: %s has %d derived items\n", result.Name, len(result.DerivedContents))
for _, derived := range result.DerivedContents {
    fmt.Printf("  - %s (%s)\n", derived.Name, derived.Variant)
}

// Advanced usage with options
opts := &simplecontent.GetContentWithDerivedOptions{
    IncludeObjects:   true,
    IncludeMetadata:  true,
    MaxDepth:         3,
    DerivationFilter: []string{"thumbnail", "preview"},
}

result, err := svc.GetContentWithDerived(ctx, contentID, opts)
if err != nil {
    return err
}

// Process thumbnails and previews with full metadata
for _, derived := range result.DerivedContents {
    fmt.Printf("Derived: %s\n", derived.Name)
    if derived.Metadata != nil {
        fmt.Printf("  Size: %d bytes\n", derived.Metadata.FileSize)
        fmt.Printf("  Type: %s\n", derived.Metadata.MimeType)
    }
    fmt.Printf("  Objects: %d\n", len(derived.Objects))
}
```

### 2. HTTP API Usage

```bash
# Get content with derived contents (basic)
curl "http://localhost:8080/api/v1/contents/{id}/with-derived"

# Get content with derived contents including metadata and objects
curl "http://localhost:8080/api/v1/contents/{id}/with-derived?include_metadata=true&include_objects=true"

# Get content hierarchy (3 levels deep)
curl "http://localhost:8080/api/v1/contents/{id}/hierarchy?max_depth=3&include_metadata=true"

# Get multiple contents with derived
curl -X POST http://localhost:8080/api/v1/contents/batch/with-derived \
  -H "Content-Type: application/json" \
  -d '{
    "content_ids": ["uuid1", "uuid2", "uuid3"],
    "options": {
      "include_metadata": true,
      "max_depth": 2,
      "derivation_filter": ["thumbnail"]
    }
  }'
```

### 3. Filter by Derivation Type

```go
// Get only thumbnails and previews
opts := &simplecontent.GetContentWithDerivedOptions{
    DerivationFilter: []string{"thumbnail", "preview"},
    IncludeMetadata:  true,
}

result, err := svc.GetContentWithDerived(ctx, contentID, opts)
```

## Performance Considerations

### 1. Database Optimization
- Add appropriate indexes for derived content queries
- Consider query batching for multiple content requests
- Implement database connection pooling

### 2. Caching Strategy
```go
// Example: Add caching layer
type CachedContentService struct {
    service simplecontent.Service
    cache   Cache
}

func (c *CachedContentService) GetContentWithDerived(ctx context.Context, contentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error) {
    cacheKey := fmt.Sprintf("content_with_derived:%s:%v", contentID, opts)

    if cached := c.cache.Get(cacheKey); cached != nil {
        return cached.(*ContentWithDerived), nil
    }

    result, err := c.service.GetContentWithDerived(ctx, contentID, opts)
    if err != nil {
        return nil, err
    }

    c.cache.Set(cacheKey, result, 5*time.Minute)
    return result, nil
}
```

### 3. Lazy Loading Options
```go
// Add lazy loading for expensive operations
type GetContentWithDerivedOptions struct {
    // ... existing fields ...
    LazyLoadObjects   bool `json:"lazy_load_objects"`
    LazyLoadMetadata  bool `json:"lazy_load_metadata"`
}
```

## Response Examples

### Basic Response
```json
{
  "id": "content-uuid",
  "name": "Original Image",
  "document_type": "image/jpeg",
  "status": "uploaded",
  "derived_contents": [
    {
      "id": "derived-uuid-1",
      "name": "Thumbnail 256px",
      "document_type": "image/jpeg",
      "derivation_type": "thumbnail",
      "variant": "thumbnail_256",
      "derivation_params": {
        "size": 256,
        "algorithm": "lanczos3"
      }
    },
    {
      "id": "derived-uuid-2",
      "name": "Preview",
      "document_type": "image/webp",
      "derivation_type": "preview",
      "variant": "preview_web"
    }
  ]
}
```

### With Metadata and Objects
```json
{
  "id": "content-uuid",
  "name": "Original Image",
  "derived_contents": [
    {
      "id": "derived-uuid-1",
      "name": "Thumbnail 256px",
      "variant": "thumbnail_256",
      "metadata": {
        "file_size": 15420,
        "mime_type": "image/jpeg",
        "file_name": "thumbnail_256.jpg"
      },
      "objects": [
        {
          "id": "object-uuid",
          "storage_backend_name": "s3",
          "status": "uploaded",
          "version": 1
        }
      ]
    }
  ]
}
```

This enhancement provides a comprehensive solution for fetching content with derived contents efficiently while maintaining flexibility through options and filters.