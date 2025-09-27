# Simple Content Library - Programmatic Usage Guide

This guide demonstrates how to use the simple-content library as a service library in your Go applications, focusing on content upload and thumbnail generation workflows.

## Overview

The simple-content library provides a clean, type-safe API for managing content and objects through its `Service` interface. Instead of using the REST API, you can directly integrate the library into your Go applications for better performance and type safety.

## Key Concepts

- **Content**: A logical content entity representing a document, image, video, etc.
- **Object**: A physical blob stored in a storage backend, associated with content
- **Derived Content**: Generated content (thumbnails, previews) linked to original content
- **Storage Backends**: Pluggable storage systems (memory, filesystem, S3)
- **Repository**: Database persistence layer (memory, PostgreSQL)

## Basic Setup

### 1. Import Required Packages

```go
import (
    "context"
    "io"
    "log"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
    memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)
```

### 2. Service Creation Methods

#### Method A: Using Config (Recommended)
```go
// Load configuration from environment or defaults
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithDatabaseType("memory"), // or "postgres"
    config.WithStorageBackend("memory", map[string]interface{}{}),
)
if err != nil {
    log.Fatal(err)
}

// Build service from config
svc, err := cfg.BuildService()
if err != nil {
    log.Fatal(err)
}
```

#### Method B: Manual Construction
```go
// Create components manually
repo := memoryrepo.New()
store := memorystorage.New()

// Create service with functional options
svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("memory", store),
)
if err != nil {
    log.Fatal(err)
}
```

## Content Upload Workflow

### Complete Upload Example

```go
func uploadContent(svc simplecontent.Service, data io.Reader, filename string, mimeType string) (*simplecontent.Content, *simplecontent.Object, error) {
    ctx := context.Background()

    // 1. Create content entity
    content, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID:      uuid.New(), // Your user/owner ID
        TenantID:     uuid.New(), // Your tenant/organization ID
        Name:         filename,
        Description:  "Uploaded content",
        DocumentType: mimeType,
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create content: %w", err)
    }

    // 2. Set content metadata
    err = svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
        ContentID:   content.ID,
        ContentType: mimeType,
        FileName:    filename,
        Tags:        []string{"uploaded", "original"},
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to set metadata: %w", err)
    }

    // 3. Create object for storage
    object, err := svc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          content.ID,
        StorageBackendName: "memory", // or your configured backend
        Version:            1,
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create object: %w", err)
    }

    // 4. Upload data
    err = svc.UploadObject(ctx, simplecontent.UploadObjectRequest{
        ObjectID: object.ID,
        Reader:   data,
        MimeType: mimeType, // Optional - for metadata
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to upload data: %w", err)
    }

    return content, object, nil
}
```

### Usage Example

```go
// Upload an image file
file, err := os.Open("image.jpg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

content, object, err := uploadContent(svc, file, "image.jpg", "image/jpeg")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Uploaded content: %s, object: %s\n", content.ID, object.ID)
```

## Thumbnail Generation Workflow

### Creating Derived Content (Thumbnails)

```go
func generateThumbnail(svc simplecontent.Service, parentContentID uuid.UUID, thumbnailSize string) (*simplecontent.Content, error) {
    ctx := context.Background()

    // Get parent content for metadata
    parentContent, err := svc.GetContent(ctx, parentContentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get parent content: %w", err)
    }

    // Create derived content for thumbnail
    thumbnailContent, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
        ParentID:       parentContentID,
        OwnerID:        parentContent.OwnerID,
        TenantID:       parentContent.TenantID,
        DerivationType: "thumbnail",        // User-facing type
        Variant:        thumbnailSize,      // Specific variant (e.g., "thumbnail_256")
        Metadata: map[string]interface{}{
            "source_type": "image_resize",
            "dimensions":  thumbnailSize,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create derived content: %w", err)
    }

    return thumbnailContent, nil
}
```

### Complete Thumbnail Pipeline

```go
func createThumbnailPipeline(svc simplecontent.Service, originalContentID uuid.UUID, thumbnailData io.Reader) (*simplecontent.Content, *simplecontent.Object, error) {
    ctx := context.Background()

    // 1. Create derived content
    thumbnailContent, err := generateThumbnail(svc, originalContentID, "thumbnail_256")
    if err != nil {
        return nil, nil, err
    }

    // 2. Create object for thumbnail
    thumbnailObject, err := svc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          thumbnailContent.ID,
        StorageBackendName: "memory",
        Version:            1,
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create thumbnail object: %w", err)
    }

    // 3. Upload thumbnail data
    err = svc.UploadObject(ctx, simplecontent.UploadObjectRequest{
        ObjectID: thumbnailObject.ID,
        Reader:   thumbnailData,
        MimeType: "image/jpeg",
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to upload thumbnail: %w", err)
    }

    // 4. Set thumbnail metadata
    err = svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
        ContentID:   thumbnailContent.ID,
        ContentType: "image/jpeg",
        FileName:    "thumbnail_256.jpg",
        Tags:        []string{"thumbnail", "derived", "256px"},
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to set thumbnail metadata: %w", err)
    }

    return thumbnailContent, thumbnailObject, nil
}
```

## Convenience Functions

The library provides package-level convenience functions for common operations:

### Thumbnail Operations

```go
// Get thumbnails of specific sizes
thumbnails, err := simplecontent.GetThumbnailsBySize(ctx, svc, parentContentID, []string{"128", "256", "512"})
if err != nil {
    log.Fatal(err)
}

// List derived content by specific type and variant
derived, err := simplecontent.ListDerivedByTypeAndVariant(ctx, svc, parentContentID, "thumbnail", "thumbnail_256")
if err != nil {
    log.Fatal(err)
}

// List by multiple variants
variants := []string{"thumbnail_128", "thumbnail_256", "preview_720"}
derived, err := simplecontent.ListDerivedByVariants(ctx, svc, parentContentID, variants)
if err != nil {
    log.Fatal(err)
}
```

### Enhanced Listing with URLs

```go
// Get derived content with URLs populated
params := simplecontent.ListDerivedContentParams{
    ParentID:    &parentContentID,
    IncludeURLs: true, // This will populate DownloadURL, PreviewURL, ThumbnailURL
}
derived, err := simplecontent.ListDerivedContentWithURLs(ctx, svc, params)
if err != nil {
    log.Fatal(err)
}

// Get single derived content with URLs
derivedWithURLs, err := simplecontent.GetDerivedContentWithURLs(ctx, svc, derivedContentID)
if err != nil {
    log.Fatal(err)
}
```

### Upload Convenience Functions

```go
// Simple upload without metadata
err := simplecontent.UploadObjectSimple(ctx, svc, objectID, dataReader)
if err != nil {
    log.Fatal(err)
}

// Upload with MIME type
err := simplecontent.UploadObjectWithMimeType(ctx, svc, objectID, dataReader, "image/jpeg")
if err != nil {
    log.Fatal(err)
}
```

## Advanced Usage Patterns

### Multi-Size Thumbnail Generation

```go
func generateMultipleThumbnails(svc simplecontent.Service, originalContentID uuid.UUID, sizes []string) error {
    ctx := context.Background()

    // Get original content and its objects
    objects, err := svc.GetObjectsByContentID(ctx, originalContentID)
    if err != nil {
        return fmt.Errorf("failed to get original objects: %w", err)
    }

    if len(objects) == 0 {
        return fmt.Errorf("no objects found for content %s", originalContentID)
    }

    // Download original data
    originalData, err := svc.DownloadObject(ctx, objects[0].ID)
    if err != nil {
        return fmt.Errorf("failed to download original: %w", err)
    }
    defer originalData.Close()

    // Generate thumbnails for each size
    for _, size := range sizes {
        // Here you would typically use an image processing library
        // to resize the image. For this example, we'll simulate it.

        thumbnailData := processImageResize(originalData, size) // Your image processing logic

        _, _, err := createThumbnailPipeline(svc, originalContentID, thumbnailData)
        if err != nil {
            return fmt.Errorf("failed to create %s thumbnail: %w", size, err)
        }

        log.Printf("Generated %s thumbnail for content %s", size, originalContentID)
    }

    return nil
}

// Usage
sizes := []string{"thumbnail_128", "thumbnail_256", "thumbnail_512"}
err := generateMultipleThumbnails(svc, contentID, sizes)
if err != nil {
    log.Fatal(err)
}
```

### Querying Derived Content

```go
func getDerivedContent(svc simplecontent.Service, parentContentID uuid.UUID) ([]*simplecontent.DerivedContent, error) {
    ctx := context.Background()

    // Method 1: Get all derived content for a parent (legacy)
    derived, err := svc.ListDerivedByParent(ctx, parentContentID)
    if err != nil {
        return nil, fmt.Errorf("failed to list derived content: %w", err)
    }

    return derived, nil
}

func getFilteredDerivedContent(svc simplecontent.Service, parentContentID uuid.UUID) ([]*simplecontent.DerivedContent, error) {
    ctx := context.Background()

    // Method 2: Enhanced filtering with advanced parameters
    params := simplecontent.ListDerivedContentParams{
        ParentID:       &parentContentID,
        DerivationType: stringPtr("thumbnail"), // Only thumbnails
        Variants:       []string{"thumbnail_256", "thumbnail_512"}, // Specific sizes
        IncludeURLs:    true, // Include download/preview URLs
        SortBy:         stringPtr("created_at_desc"),
        Limit:          intPtr(10),
    }

    derived, err := svc.ListDerivedContent(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("failed to list filtered derived content: %w", err)
    }

    return derived, nil
}

// Helper functions
func stringPtr(s string) *string { return &s }
func intPtr(i int) *int { return &i }

// Usage
derived, err := getDerivedContent(svc, originalContentID)
if err != nil {
    log.Fatal(err)
}

for _, d := range derived {
    fmt.Printf("Derived content: %s, type: %s\n", d.ContentID, d.DerivationType)

    // Get the actual content
    content, err := svc.GetContent(ctx, d.ContentID)
    if err != nil {
        continue
    }

    // Get objects for this derived content
    objects, err := svc.GetObjectsByContentID(ctx, content.ID)
    if err != nil {
        continue
    }

    fmt.Printf("  Content: %s, Objects: %d\n", content.Name, len(objects))
}
```

## Storage Backend Configuration

### Filesystem Storage

```go
import (
    fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

// Configure filesystem storage
fsStore, err := fsstorage.New(fsstorage.Config{
    BaseDir:   "./uploads",
    URLPrefix: "https://mysite.com/files/",
})
if err != nil {
    log.Fatal(err)
}

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("filesystem", fsStore),
)
```

### S3 Storage

```go
import (
    s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)

// Configure S3 storage
s3Store, err := s3storage.New(s3storage.Config{
    Region:          "us-west-2",
    Bucket:          "my-content-bucket",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
})
if err != nil {
    log.Fatal(err)
}

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Store),
)
```

### PostgreSQL Repository

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
)

// Configure PostgreSQL
pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost/dbname")
if err != nil {
    log.Fatal(err)
}

repo := repopg.NewWithPool(pool)

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Store),
)
```

## Error Handling

The library uses typed errors for consistent error handling:

```go
import (
    "errors"
    "github.com/tendant/simple-content/pkg/simplecontent"
)

content, err := svc.GetContent(ctx, contentID)
if err != nil {
    if errors.Is(err, simplecontent.ErrContentNotFound) {
        // Handle content not found
        return nil, fmt.Errorf("content does not exist")
    }

    if errors.Is(err, simplecontent.ErrInvalidContentStatus) {
        // Handle invalid status
        return nil, fmt.Errorf("invalid content status")
    }

    // Handle other errors
    return nil, fmt.Errorf("unexpected error: %w", err)
}
```

## Best Practices

### 1. Context Management
Always pass appropriate contexts with timeouts for long-running operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := svc.UploadObject(ctx, simplecontent.UploadObjectRequest{
    ObjectID: objectID,
    Reader:   largeFile,
})
```

### 2. Resource Cleanup
Always close readers and handle cleanup:

```go
reader, err := svc.DownloadObject(ctx, objectID)
if err != nil {
    return err
}
defer reader.Close() // Important!

// Process the reader...
```

### 3. Batch Operations
For multiple operations, consider batching where possible:

```go
// Create multiple objects at once
var objects []*simplecontent.Object
for _, version := range versions {
    obj, err := svc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          contentID,
        StorageBackendName: "s3",
        Version:            version,
    })
    if err != nil {
        return err
    }
    objects = append(objects, obj)
}
```

### 4. Metadata Management
Use structured metadata for better querying and organization:

```go
err = svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
    ContentID:   contentID,
    ContentType: "image/jpeg",
    FileName:    "photo.jpg",
    Tags:        []string{"photo", "landscape", "2024"},
    CustomMetadata: map[string]interface{}{
        "camera":     "Canon EOS R5",
        "location":   "Yosemite",
        "iso":        100,
        "f_stop":     "f/8",
        "created_by": userID,
    },
})
```

## Testing

For testing, use the in-memory implementations:

```go
import (
    "testing"
    memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func setupTestService(t *testing.T) simplecontent.Service {
    repo := memoryrepo.New()
    store := memorystorage.New()

    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("test", store),
    )
    if err != nil {
        t.Fatal(err)
    }

    return svc
}

func TestContentUpload(t *testing.T) {
    svc := setupTestService(t)

    // Your test code here...
}
```

## Performance Considerations

1. **Connection Pooling**: Use connection pools for database repositories
2. **Streaming**: Use streaming for large file uploads/downloads
3. **Caching**: Consider adding caching layers for frequently accessed metadata
4. **Async Processing**: For thumbnail generation, consider async processing patterns

This guide provides a comprehensive overview of using the simple-content library programmatically. The library's clean interface design makes it easy to integrate into existing Go applications while providing flexibility for various storage and processing workflows.