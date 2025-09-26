# Content with Derived Service Example

This example demonstrates how to implement and use a service that fetches content along with its derived contents in a single operation, providing more efficient API access for applications that need complete content hierarchies.

## Problem Statement

The current simple-content library requires multiple API calls to get a complete picture of content and its derived items:

1. `GET /api/v1/contents/{id}` - Get the main content
2. `GET /api/v1/contents/{id}/derived` - Get derived content relationships
3. `GET /api/v1/contents/{derived-id}` - Get each derived content individually
4. `GET /api/v1/contents/{derived-id}/metadata` - Get metadata for each derived content
5. `GET /api/v1/contents/{derived-id}/objects` - Get objects for each derived content

This results in **N+1 query problems** and increased latency.

## Solution

This example implements an enhanced service that provides:
- **Single API call** to get content with all derived contents
- **Flexible options** for including metadata, objects, and controlling depth
- **Filtering capabilities** by derivation type
- **Hierarchical support** for nested derived content

## Features

### 1. **Enhanced Data Structures**
```go
type ContentWithDerived struct {
    *Content
    DerivedContents []*DerivedContentItem `json:"derived_contents,omitempty"`
    ParentContent   *ContentReference     `json:"parent_content,omitempty"`
}
```

### 2. **Flexible Options**
```go
type GetContentWithDerivedOptions struct {
    IncludeObjects   bool     `json:"include_objects"`
    IncludeMetadata  bool     `json:"include_metadata"`
    MaxDepth         int      `json:"max_depth"`
    DerivationFilter []string `json:"derivation_filter"`
}
```

### 3. **Multiple Service Methods**
- `GetContentWithDerived()` - Single content with derived
- `GetMultipleContentWithDerived()` - Batch operation
- `GetContentHierarchy()` - Complete hierarchy with higher depth

## Running the Example

1. **Start the demo server**:
   ```bash
   cd examples/content-with-derived
   go run main.go
   ```

2. **Open your browser** to `http://localhost:8080`

3. **Create demo data** using the web interface

4. **Test the enhanced APIs**

## API Endpoints

### Enhanced Content Endpoints

#### Get Content with Derived
```http
GET /api/v1/contents/{id}/with-derived?include_metadata=true&max_depth=3&derivation_filter=thumbnail,preview
```

**Response Example:**
```json
{
  "id": "content-uuid",
  "name": "Original Photo",
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
        "size": "256px",
        "algorithm": "lanczos3"
      },
      "metadata": {
        "file_size": 15420,
        "mime_type": "image/jpeg",
        "file_name": "thumbnail_256.jpg"
      }
    }
  ]
}
```

#### Get Multiple Contents with Derived
```http
POST /api/v1/contents/batch/with-derived
Content-Type: application/json

{
  "content_ids": ["uuid1", "uuid2", "uuid3"],
  "options": {
    "include_metadata": true,
    "max_depth": 2,
    "derivation_filter": ["thumbnail"]
  }
}
```

#### Get Content Hierarchy
```http
GET /api/v1/contents/{id}/hierarchy?include_metadata=true&max_depth=5
```

### Demo Endpoints

#### Create Demo Data
```http
GET /api/v1/demo/setup
```

Creates:
- 1 original photo content
- 3 thumbnail variants (128px, 256px, 512px)
- 1 preview variant (web format)

## Usage Examples

### 1. Basic Usage

```bash
# Create demo data
curl http://localhost:8080/api/v1/demo/setup

# Get content with all derived content (basic)
curl "http://localhost:8080/api/v1/contents/{id}/with-derived"

# Get content with metadata and objects included
curl "http://localhost:8080/api/v1/contents/{id}/with-derived?include_metadata=true&include_objects=true"
```

### 2. Advanced Filtering

```bash
# Get only thumbnails with metadata
curl "http://localhost:8080/api/v1/contents/{id}/with-derived?derivation_filter=thumbnail&include_metadata=true"

# Get hierarchy up to 5 levels deep
curl "http://localhost:8080/api/v1/contents/{id}/hierarchy?max_depth=5&include_metadata=true"
```

### 3. Batch Operations

```bash
curl -X POST http://localhost:8080/api/v1/contents/batch/with-derived \
  -H "Content-Type: application/json" \
  -d '{
    "content_ids": ["uuid1", "uuid2"],
    "options": {
      "include_metadata": true,
      "derivation_filter": ["thumbnail", "preview"]
    }
  }'
```

## Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `include_objects` | boolean | `false` | Include object details for each content |
| `include_metadata` | boolean | `false` | Include content metadata |
| `max_depth` | integer | `1` | Maximum depth for nested derived content |
| `derivation_filter` | string | - | Comma-separated derivation types to include |

## Programmatic Usage

```go
// Create the enhanced service
service, err := NewExtendedContentService()
if err != nil {
    log.Fatal(err)
}

// Basic usage
result, err := service.GetContentWithDerived(ctx, contentID, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Content: %s has %d derived items\n",
    result.Name, len(result.DerivedContents))

// Advanced usage with options
opts := &GetContentWithDerivedOptions{
    IncludeObjects:   true,
    IncludeMetadata:  true,
    MaxDepth:         3,
    DerivationFilter: []string{"thumbnail", "preview"},
}

result, err = service.GetContentWithDerived(ctx, contentID, opts)
if err != nil {
    log.Fatal(err)
}

// Process results
for _, derived := range result.DerivedContents {
    fmt.Printf("Derived: %s (%s)\n", derived.Name, derived.Variant)

    if derived.Metadata != nil {
        fmt.Printf("  Size: %d bytes\n", derived.Metadata.FileSize)
        fmt.Printf("  Type: %s\n", derived.Metadata.MimeType)
    }

    if len(derived.Objects) > 0 {
        fmt.Printf("  Objects: %d\n", len(derived.Objects))
    }
}

// Batch operation
contentIDs := []uuid.UUID{id1, id2, id3}
results, err := service.GetMultipleContentWithDerived(ctx, contentIDs, opts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Retrieved %d contents with derived data\n", len(results))
```

## Performance Benefits

### Before (Multiple API Calls)
```
1 call:  GET /contents/{id}
1 call:  GET /contents/{id}/derived
N calls: GET /contents/{derived-id} (for each derived)
N calls: GET /contents/{derived-id}/metadata (if needed)
N calls: GET /contents/{derived-id}/objects (if needed)

Total: 2 + 3N API calls
```

### After (Single API Call)
```
1 call: GET /contents/{id}/with-derived?include_metadata=true&include_objects=true

Total: 1 API call
```

**Example**: For content with 5 derived items:
- **Before**: 17 API calls
- **After**: 1 API call
- **Improvement**: 94% reduction in API calls

## Response Structure

### Basic Response
```json
{
  "id": "original-uuid",
  "name": "Original Photo",
  "document_type": "image/jpeg",
  "derivation_type": "",
  "derived_contents": [
    {
      "id": "derived-uuid",
      "name": "Thumbnail 256px",
      "derivation_type": "thumbnail",
      "variant": "thumbnail_256"
    }
  ]
}
```

### With Metadata and Objects
```json
{
  "id": "original-uuid",
  "name": "Original Photo",
  "derived_contents": [
    {
      "id": "derived-uuid",
      "name": "Thumbnail 256px",
      "variant": "thumbnail_256",
      "derivation_params": {
        "size": "256px",
        "algorithm": "lanczos3"
      },
      "metadata": {
        "file_size": 15420,
        "mime_type": "image/jpeg",
        "file_name": "thumbnail_256.jpg",
        "tags": ["thumbnail", "derived"]
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

### Derived Content with Parent Reference
```json
{
  "id": "derived-uuid",
  "name": "Thumbnail 256px",
  "derivation_type": "thumbnail",
  "parent_content": {
    "id": "parent-uuid",
    "name": "Original Photo",
    "document_type": "image/jpeg"
  },
  "derived_contents": []
}
```

## Web Interface Features

The demo includes an interactive web interface:

1. **Demo Data Setup** - Creates sample content hierarchy
2. **Flexible Options** - Checkboxes and inputs for all options
3. **Real-time Testing** - Test API endpoints with different parameters
4. **JSON Display** - Pretty-printed JSON responses
5. **Content Browser** - View all available content items

## Extension Points

### 1. Caching Layer
```go
type CachedContentService struct {
    service *ExtendedContentService
    cache   Cache
    ttl     time.Duration
}

func (c *CachedContentService) GetContentWithDerived(ctx context.Context, contentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error) {
    key := fmt.Sprintf("content_derived:%s:%v", contentID, opts)

    if cached := c.cache.Get(key); cached != nil {
        return cached.(*ContentWithDerived), nil
    }

    result, err := c.service.GetContentWithDerived(ctx, contentID, opts)
    if err != nil {
        return nil, err
    }

    c.cache.Set(key, result, c.ttl)
    return result, nil
}
```

### 2. Database Optimization
```go
// Add database hints for better query performance
func (s *service) getDerivedContentsOptimized(ctx context.Context, parentID uuid.UUID) ([]*DerivedContentItem, error) {
    // Use joins instead of N+1 queries
    query := `
        SELECT c.*, d.variant, d.derivation_params, d.processing_metadata,
               m.file_size, m.mime_type, m.file_name,
               o.id as object_id, o.storage_backend_name, o.status
        FROM content c
        JOIN content_derived d ON c.id = d.content_id
        LEFT JOIN content_metadata m ON c.id = m.content_id
        LEFT JOIN object o ON c.id = o.content_id
        WHERE d.parent_id = $1
        ORDER BY c.created_at
    `
    // Implementation would use single query instead of multiple calls
}
```

### 3. Async Processing
```go
// For very large hierarchies, consider async processing
func (s *service) GetContentWithDerivedAsync(ctx context.Context, contentID uuid.UUID, opts *GetContentWithDerivedOptions) (<-chan *ContentWithDerived, <-chan error) {
    resultChan := make(chan *ContentWithDerived, 1)
    errorChan := make(chan error, 1)

    go func() {
        defer close(resultChan)
        defer close(errorChan)

        result, err := s.GetContentWithDerived(ctx, contentID, opts)
        if err != nil {
            errorChan <- err
            return
        }

        resultChan <- result
    }()

    return resultChan, errorChan
}
```

## Production Considerations

1. **Rate Limiting**: Implement rate limiting for batch operations
2. **Pagination**: Add pagination support for large result sets
3. **Monitoring**: Add metrics for query performance and cache hit rates
4. **Authorization**: Ensure proper access control for derived content
5. **Versioning**: Consider API versioning for backward compatibility

This enhanced service significantly improves efficiency and developer experience when working with content hierarchies while maintaining the flexibility and power of the underlying simple-content library.