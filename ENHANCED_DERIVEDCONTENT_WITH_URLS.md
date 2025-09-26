# Enhanced DerivedContent with URLs - Single Struct Approach

## Problem with Two Structs

The current approach requiring separate `DerivedContent` and `DerivedContentItem` structs creates unnecessary complexity:

```go
// Current - Two structs needed
type DerivedContent struct {
    ParentID           uuid.UUID
    ContentID          uuid.UUID
    DerivationType     string
    // ... no URLs
}

type DerivedContentItem struct {
    *simplecontent.Content
    Variant            string
    DerivationParams   map[string]interface{}
    Objects           []*simplecontent.Object
    // ... separate struct for URLs
}
```

**Problems:**
- ❌ **Duplication**: Same data represented in two places
- ❌ **Complexity**: Developers need to understand two different types
- ❌ **Mapping**: Constant conversion between structs
- ❌ **Maintenance**: Changes need to be made in multiple places
- ❌ **Type Safety**: Easy to use wrong struct in wrong context

## Better Solution: Enhanced Single Struct

```go
// Enhanced DerivedContent - single struct with URLs
type DerivedContent struct {
    // Persisted fields
    ParentID           uuid.UUID              `json:"parent_id" db:"parent_id"`
    ContentID          uuid.UUID              `json:"content_id" db:"content_id"`
    DerivationType     string                 `json:"derivation_type" db:"derivation_type"`
    DerivationParams   map[string]interface{} `json:"derivation_params" db:"derivation_params"`
    ProcessingMetadata map[string]interface{} `json:"processing_metadata" db:"processing_metadata"`
    CreatedAt          time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
    DocumentType       string                 `json:"document_type" db:"document_type"`
    Status             string                 `json:"status" db:"status"`

    // Computed fields (not persisted)
    DownloadURL        string                 `json:"download_url,omitempty" db:"-"`
    PreviewURL         string                 `json:"preview_url,omitempty" db:"-"`
    ThumbnailURL       string                 `json:"thumbnail_url,omitempty" db:"-"`
    Variant            string                 `json:"variant,omitempty" db:"-"`

    // Optional enhanced data (not persisted)
    Objects            []*Object              `json:"objects,omitempty" db:"-"`
    Metadata           *ContentMetadata       `json:"metadata,omitempty" db:"-"`
    ParentContent      *Content               `json:"parent_content,omitempty" db:"-"`
}
```

### Database Tag Usage

The `db:"-"` tag tells the ORM/database layer to ignore these fields during persistence:

**PostgreSQL Repository:**
```go
func (r *Repository) scanDerivedContent(rows *sql.Rows) (*DerivedContent, error) {
    derived := &DerivedContent{}

    // Only scan persisted fields
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

    return derived, err
}

func (r *Repository) insertDerivedContent(ctx context.Context, derived *DerivedContent) error {
    query := `
        INSERT INTO content_derived (parent_id, content_id, derivation_type, derivation_params, processing_metadata, document_type, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

    // Only insert persisted fields - URLs are ignored
    _, err := r.db.ExecContext(ctx, query,
        derived.ParentID,
        derived.ContentID,
        derived.DerivationType,
        derived.DerivationParams,
        derived.ProcessingMetadata,
        derived.DocumentType,
        derived.Status,
        // URLs are NOT inserted
    )

    return err
}
```

## Service Layer Enhancement

With this approach, the service can populate URLs when needed:

```go
// Enhanced service method that populates URLs
func (s *service) ListDerivedContentWithURLs(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error) {
    // Get base derived content from repository
    derived, err := s.repository.ListDerivedContent(ctx, params)
    if err != nil {
        return nil, err
    }

    // Enhance with URLs if requested
    if params.IncludeURLs {
        for _, d := range derived {
            if err := s.populateURLs(ctx, d); err != nil {
                // Log error but don't fail entire operation
                log.Printf("Failed to populate URLs for content %s: %v", d.ContentID, err)
            }
        }
    }

    return derived, nil
}

func (s *service) populateURLs(ctx context.Context, derived *DerivedContent) error {
    // Get objects for this content
    objects, err := s.repository.GetObjectsByContentID(ctx, derived.ContentID)
    if err != nil || len(objects) == 0 {
        return err
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

    // Extract variant from processing metadata or params
    derived.Variant = extractVariant(derived)

    return nil
}
```

## Usage Examples

### Simple Usage (no URLs)
```go
// Basic listing - no URLs populated (fast)
derived, err := service.ListDerivedByParent(ctx, parentID)
// derived[0].DownloadURL == "" (empty)
// derived[0].PreviewURL == ""   (empty)
```

### Enhanced Usage (with URLs)
```go
// Enhanced listing - URLs populated on demand
params := ListDerivedContentParams{
    ParentID:    &parentID,
    IncludeURLs: true, // New option
}
derived, err := service.ListDerivedContentWithURLs(ctx, params)

// Now URLs are available
for _, d := range derived {
    fmt.Printf("Thumbnail URL: %s\n", d.ThumbnailURL)
    fmt.Printf("Download URL: %s\n", d.DownloadURL)
    fmt.Printf("Preview URL: %s\n", d.PreviewURL)
}
```

### Specific Thumbnail URLs
```go
// Get thumbnails with URLs
params := ListDerivedContentParams{
    ParentID:       &parentID,
    DerivationType: stringPtr("thumbnail"),
    IncludeURLs:    true,
}
thumbnails, err := service.ListDerivedContentWithURLs(ctx, params)

// Create URL map by variant
thumbnailURLs := make(map[string]string)
for _, thumb := range thumbnails {
    thumbnailURLs[thumb.Variant] = thumb.ThumbnailURL
}

// Usage: thumbnailURLs["thumbnail_256"] = "https://..."
```

## Benefits of Single Struct Approach

### ✅ **Simplicity**
```go
// One struct to rule them all
var derived *DerivedContent

// URLs populated when needed
if needURLs {
    derived = getWithURLs(id)
} else {
    derived = getBasic(id)    // URLs empty but same struct
}
```

### ✅ **Type Safety**
```go
// Same type everywhere - no conversion needed
func processDerivations(items []*DerivedContent) {
    for _, item := range items {
        // Can access both basic fields AND URLs
        fmt.Printf("Type: %s, URL: %s\n", item.DerivationType, item.ThumbnailURL)
    }
}
```

### ✅ **Performance Control**
```go
// Fast: No URL generation
basic := service.ListDerivedByParent(ctx, parentID)

// Slower but complete: URLs generated on demand
enhanced := service.ListDerivedContentWithURLs(ctx, ListDerivedContentParams{
    ParentID: &parentID,
    IncludeURLs: true,
})
```

### ✅ **Backward Compatibility**
```go
// Existing code continues working unchanged
derived, err := service.ListDerivedByParent(ctx, parentID)
// derived[0].DownloadURL is just empty string, no breaking change
```

### ✅ **JSON API Consistency**
```json
{
  "parent_id": "uuid",
  "content_id": "uuid",
  "derivation_type": "thumbnail",
  "variant": "thumbnail_256",
  "download_url": "https://...",      // Present when requested
  "preview_url": "https://...",       // Present when requested
  "thumbnail_url": "https://..."      // Present when requested
}
```

## Implementation Strategy

### Phase 1: Add URL fields to DerivedContent
```go
// Add to existing DerivedContent struct in types.go
type DerivedContent struct {
    // ... existing fields

    // New computed fields (not persisted)
    DownloadURL   string `json:"download_url,omitempty" db:"-"`
    PreviewURL    string `json:"preview_url,omitempty" db:"-"`
    ThumbnailURL  string `json:"thumbnail_url,omitempty" db:"-"`
    Variant       string `json:"variant,omitempty" db:"-"`
}
```

### Phase 2: Add enhanced service methods
```go
// Add to Service interface
type Service interface {
    // Existing methods unchanged
    ListDerivedByParent(ctx context.Context, parentID uuid.UUID) ([]*DerivedContent, error)

    // New enhanced methods
    ListDerivedContentWithURLs(ctx context.Context, params ListDerivedContentParams) ([]*DerivedContent, error)
    GetDerivedContentWithURLs(ctx context.Context, contentID uuid.UUID) (*DerivedContent, error)
}
```

### Phase 3: Extend ListDerivedContentParams
```go
type ListDerivedContentParams struct {
    // ... existing fields

    // New options
    IncludeURLs     bool `json:"include_urls"`
    IncludeObjects  bool `json:"include_objects"`
    IncludeMetadata bool `json:"include_metadata"`
}
```

## Performance Considerations

**URL Generation Cost:**
- Memory backend: No cost (URLs not supported)
- Filesystem backend: Minimal cost (URL construction)
- S3 backend: Moderate cost (presigned URL generation)

**Optimization Strategy:**
```go
func (s *service) populateURLsBatch(ctx context.Context, derived []*DerivedContent) error {
    // Batch object queries by storage backend
    backendGroups := make(map[string][]*DerivedContent)

    for _, d := range derived {
        objects, _ := s.getObjectsForContent(ctx, d.ContentID)
        if len(objects) > 0 {
            backend := objects[0].StorageBackendName
            backendGroups[backend] = append(backendGroups[backend], d)
        }
    }

    // Generate URLs per backend to optimize backend-specific operations
    for backendName, group := range backendGroups {
        backend, _ := s.GetBackend(backendName)
        for _, d := range group {
            // Generate URLs using backend
            s.populateURLsForDerived(ctx, d, backend)
        }
    }

    return nil
}
```

## Conclusion

**Single Enhanced Struct >> Two Separate Structs**

The enhanced `DerivedContent` with optional URL fields provides:
- ✅ **Simplicity**: One struct for all use cases
- ✅ **Performance**: URLs generated only when needed
- ✅ **Flexibility**: Same struct works with/without URLs
- ✅ **Maintainability**: Single source of truth
- ✅ **Type Safety**: No struct conversion needed
- ✅ **Backward Compatibility**: Existing code unaffected

This approach follows the principle of **progressive enhancement** - the basic struct works for simple use cases, and URLs are added when explicitly requested.