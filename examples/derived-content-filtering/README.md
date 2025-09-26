# Enhanced Derived Content Filtering Demo

This example demonstrates advanced filtering capabilities for derived content, allowing you to filter by derivation type, variant, or combinations of both.

## Problem Statement

The current simple-content library has limited filtering for derived content:
- Only supports single derivation type filtering
- No variant-specific filtering
- No complex combination filtering

This creates challenges when you need to:
- Get specific thumbnail sizes (e.g., only 256px thumbnails)
- Filter by multiple variants (e.g., all small thumbnails: 128px, 256px)
- Get specific type+variant combinations (e.g., thumbnail:256px + preview:web)

## Solution

This example implements comprehensive filtering with:

### 1. **Enhanced Filter Parameters**
```go
type EnhancedListDerivedContentParams struct {
    // Basic filtering
    ParentID         *uuid.UUID  // Single parent
    ParentIDs        []uuid.UUID // Multiple parents

    // Derivation type filtering
    DerivationType   *string     // Single type (e.g., "thumbnail")
    DerivationTypes  []string    // Multiple types (e.g., ["thumbnail", "preview"])

    // Variant filtering - NEW!
    Variant          *string     // Single variant (e.g., "thumbnail_256")
    Variants         []string    // Multiple variants (e.g., ["thumbnail_128", "thumbnail_256"])

    // Advanced combinations
    TypeVariantPairs []TypeVariantPair // Precise type+variant pairs

    // Additional filters
    ContentStatus    *string     // Filter by status
    CreatedAfter     *time.Time  // Temporal filtering
    SortBy           *string     // Sorting options
    Limit            *int        // Pagination
}
```

### 2. **Flexible Filtering Strategies**
- **Single filters**: Get specific types or variants
- **Multiple filters**: Get several types/variants at once
- **Combination filters**: Precise type+variant pair matching
- **Temporal filters**: Recent content, date ranges
- **Status filters**: Only uploaded, processing, etc.

### 3. **Performance Optimizations**
- Smart variant extraction from metadata/params
- Efficient in-memory filtering for demo
- Database query optimization suggestions
- Pagination and sorting support

## Running the Example

1. **Start the demo server**:
   ```bash
   cd examples/derived-content-filtering
   go run main.go
   ```

2. **Open your browser** to `http://localhost:8080`

3. **Create demo data** to populate test content

4. **Try different filtering combinations**

## Filtering Capabilities

### 1. **Single Derivation Type**
Get all thumbnails:
```bash
curl "http://localhost:8080/api/v1/derived-content/filter?parent_id=uuid&derivation_type=thumbnail"
```

### 2. **Multiple Derivation Types**
Get thumbnails and previews:
```bash
curl "http://localhost:8080/api/v1/derived-content/filter?parent_id=uuid&derivation_types=thumbnail,preview"
```

### 3. **Single Variant**
Get only 256px thumbnails:
```bash
curl "http://localhost:8080/api/v1/derived-content/filter?parent_id=uuid&variant=thumbnail_256"
```

### 4. **Multiple Variants**
Get small and medium thumbnails:
```bash
curl "http://localhost:8080/api/v1/derived-content/filter?parent_id=uuid&variants=thumbnail_128,thumbnail_256"
```

### 5. **Type + Variant Combinations**
Get specific type+variant pairs:
```bash
curl "http://localhost:8080/api/v1/derived-content/filter?parent_id=uuid&type_variant_pairs=thumbnail:thumbnail_256,preview:preview_web"
```

### 6. **Complex Combinations**
Combine multiple filter types:
```bash
curl "http://localhost:8080/api/v1/derived-content/filter?parent_id=uuid&derivation_types=thumbnail,preview&variants=thumbnail_256,preview_web&limit=10"
```

## API Endpoints

### Enhanced Filtering
```http
GET /api/v1/derived-content/filter
```

**Query Parameters:**
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `parent_id` | UUID | Required parent content ID | `550e8400-e29b-41d4-a716-446655440000` |
| `derivation_type` | string | Single derivation type | `thumbnail` |
| `derivation_types` | string | Comma-separated types | `thumbnail,preview` |
| `variant` | string | Single variant | `thumbnail_256` |
| `variants` | string | Comma-separated variants | `thumbnail_128,thumbnail_256` |
| `type_variant_pairs` | string | Type:variant pairs | `thumbnail:thumbnail_256,preview:preview_web` |
| `content_status` | string | Filter by status | `uploaded` |
| `limit` | integer | Max results | `10` |
| `offset` | integer | Skip results | `0` |

### Convenience Endpoints

#### Get Thumbnails by Size
```http
GET /api/v1/derived-content/thumbnails?parent_id=uuid&sizes=128,256,512
```

#### Get Recent Derived Content
```http
GET /api/v1/derived-content/recent?parent_id=uuid&since=2024-01-01T00:00:00Z
```

## Response Format

### Basic Response
```json
{
  "results": [
    {
      "parent_id": "parent-uuid",
      "content_id": "derived-uuid",
      "derivation_type": "thumbnail",
      "actual_variant": "thumbnail_256",
      "matched_by": "variant:thumbnail_256",
      "derivation_params": {
        "variant": "thumbnail_256",
        "size": "256px"
      },
      "processing_metadata": {
        "algorithm": "lanczos3"
      },
      "status": "uploaded",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "filters": {
    "parent_id": "parent-uuid",
    "variant": "thumbnail_256"
  }
}
```

### Enhanced Response Features
- **`actual_variant`**: Extracted variant from content metadata
- **`matched_by`**: Debug info showing which filter matched
- **`count`**: Total number of results
- **`filters`**: Applied filter parameters for debugging

## Filtering Logic

### Variant Extraction Strategies
The service uses multiple strategies to extract variants:

1. **ProcessingMetadata**: Look for `variant` key
2. **DerivationParams**: Look for `variant` key
3. **DerivationType parsing**: Extract from patterns like `thumbnail_256`
4. **Fallback**: Use derivation type as variant

```go
func extractVariant(derived *DerivedContent) string {
    // Strategy 1: ProcessingMetadata
    if variant, exists := derived.ProcessingMetadata["variant"]; exists {
        return variant.(string)
    }

    // Strategy 2: DerivationParams
    if variant, exists := derived.DerivationParams["variant"]; exists {
        return variant.(string)
    }

    // Strategy 3: Parse DerivationType
    parts := strings.Split(derived.DerivationType, "_")
    if len(parts) > 1 {
        return derived.DerivationType
    }

    // Strategy 4: Fallback
    return derived.DerivationType
}
```

### Filter Matching Logic
```go
func matchesFilters(derived *DerivedContent, params FilterParams) bool {
    // Type filtering
    if params.DerivationType != nil &&
       derived.DerivationType != *params.DerivationType {
        return false
    }

    // Variant filtering - NEW!
    actualVariant := extractVariant(derived)
    if params.Variant != nil && actualVariant != *params.Variant {
        return false
    }

    // Type+Variant pair filtering
    for _, pair := range params.TypeVariantPairs {
        if derived.DerivationType == pair.DerivationType &&
           actualVariant == pair.Variant {
            return true
        }
    }

    return true
}
```

## Demo Data Structure

The demo creates the following test data:

### Original Content
- **ID**: Generated UUID
- **Name**: "Demo Photo"
- **Type**: "image/jpeg"

### Derived Content (6 items)
1. **thumbnail_128**: 128px thumbnail
2. **thumbnail_256**: 256px thumbnail
3. **thumbnail_512**: 512px thumbnail
4. **preview_web**: Web preview (WebP)
5. **preview_mobile**: Mobile preview
6. **video_720p**: Video transcode (720p)

## Usage Examples

### Programmatic Usage
```go
// Create service
service, err := NewEnhancedDerivedContentService()

// Filter by single variant
params := EnhancedListDerivedContentParams{
    ParentID: &parentID,
    Variant:  stringPtr("thumbnail_256"),
}
results, err := service.ListDerivedContentWithFilters(ctx, params)

// Filter by multiple variants
params = EnhancedListDerivedContentParams{
    ParentID: &parentID,
    Variants: []string{"thumbnail_128", "thumbnail_256"},
}
results, err = service.ListDerivedContentWithFilters(ctx, params)

// Filter by type+variant combinations
params = EnhancedListDerivedContentParams{
    ParentID: &parentID,
    TypeVariantPairs: []TypeVariantPair{
        {DerivationType: "thumbnail", Variant: "thumbnail_256"},
        {DerivationType: "preview", Variant: "preview_web"},
    },
}
results, err = service.ListDerivedContentWithFilters(ctx, params)

// Convenience methods
thumbnails, err := service.GetThumbnailsBySize(ctx, parentID, []string{"128", "256"})
recent, err := service.GetRecentDerived(ctx, parentID, time.Now().Add(-24*time.Hour))
```

### Web Interface Features

The demo includes an interactive web interface with:

1. **Filter Controls**: Form inputs for all filter parameters
2. **Quick Examples**: Preset filter combinations
3. **Convenience Endpoints**: One-click access to common patterns
4. **Real-time Results**: JSON display of filtered results
5. **Debug Information**: Shows which filters matched

## Performance Considerations

### Database Optimization
For production use, add these indexes:
```sql
-- Core filtering indexes
CREATE INDEX idx_content_derived_parent_type ON content_derived(parent_id, derivation_type);
CREATE INDEX idx_content_derived_parent_variant ON content_derived(parent_id, variant);
CREATE INDEX idx_content_derived_type_variant ON content_derived(derivation_type, variant);

-- Temporal filtering
CREATE INDEX idx_content_derived_created_at ON content_derived(created_at);
CREATE INDEX idx_content_derived_updated_at ON content_derived(updated_at);

-- Composite index for common patterns
CREATE INDEX idx_content_derived_composite ON content_derived(parent_id, derivation_type, variant, created_at);
```

### Query Optimization
```go
// Use prepared statements for frequent queries
stmt := `
    SELECT cd.parent_id, cd.content_id, cd.variant, cd.derivation_params
    FROM content_derived cd
    WHERE cd.parent_id = $1
      AND cd.variant = ANY($2)
      AND cd.deleted_at IS NULL
    ORDER BY cd.created_at DESC
    LIMIT $3
`
```

### Caching Strategy
```go
// Cache frequent filter combinations
key := fmt.Sprintf("derived:%s:%s", parentID, hashFilters(params))
if cached := cache.Get(key); cached != nil {
    return cached.([]*DerivedContent), nil
}
```

## Production Integration

### 1. Repository Layer
Implement these methods in your repository:
```go
type Repository interface {
    ListDerivedContentWithFilters(ctx context.Context, params EnhancedListDerivedContentParams) ([]*DerivedContent, error)
    CountDerivedContent(ctx context.Context, params EnhancedListDerivedContentParams) (int64, error)
}
```

### 2. Service Layer
Add convenience methods:
```go
type Service interface {
    ListDerivedByTypeAndVariant(ctx context.Context, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error)
    ListDerivedByVariants(ctx context.Context, parentID uuid.UUID, variants []string) ([]*DerivedContent, error)
    GetThumbnailsBySize(ctx context.Context, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error)
}
```

### 3. HTTP Layer
Implement advanced query parameter parsing and validation.

This enhanced filtering system provides precise control over derived content queries while maintaining excellent performance and usability.