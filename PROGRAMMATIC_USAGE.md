# Simple Content Library - Programmatic Usage Guide

This guide demonstrates how to use the simple-content library's new simplified API for content management in your Go applications.

## Overview

The simple-content library provides a **content-focused API** that abstracts storage implementation details. Instead of managing objects manually, you work with content concepts directly through unified operations.

## Key Concepts

- **Content**: Logical entity representing documents, images, videos, etc.
- **Derived Content**: Generated content (thumbnails, previews) linked to originals
- **Service Interface**: Main API that hides storage implementation details
- **StorageService**: Advanced interface for users who need direct object access
- **Storage Backends**: Pluggable systems (memory, filesystem, S3)

## API Architecture

### Service Interface (Recommended)
```go
type Service interface {
    // Unified upload operations (NEW!)
    UploadContent(ctx, UploadContentRequest) (*Content, error)
    UploadDerivedContent(ctx, UploadDerivedContentRequest) (*Content, error)

    // Content data access
    DownloadContent(ctx, contentID) (io.ReadCloser, error)

    // Unified details API (NEW!)
    GetContentDetails(ctx, contentID, ...ContentDetailsOption) (*ContentDetails, error)

    // Standard content operations
    CreateContent(ctx, CreateContentRequest) (*Content, error)
    GetContent(ctx, uuid.UUID) (*Content, error)
    ListContent(ctx, ListContentRequest) ([]*Content, error)

    // Derived content operations
    ListDerivedContent(ctx, ...ListDerivedContentOption) ([]*DerivedContent, error)
}
```

### StorageService Interface (Advanced)
```go
type StorageService interface {
    // Direct object operations (for advanced users)
    CreateObject(ctx, CreateObjectRequest) (*Object, error)
    UploadObject(ctx, UploadObjectRequest) error
    GetUploadURL(ctx, objectID) (string, error)
    // ... other object operations
}
```

## Basic Setup

### 1. Import Required Packages

```go
import (
    "context"
    "io"
    "strings"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)
```

### 2. Service Creation

#### Using Config (Recommended)
```go
cfg, err := config.Load(
    config.WithDatabaseType("memory"), // or "postgres"
    config.WithStorageBackend("memory", map[string]interface{}{}),
)
if err != nil {
    log.Fatal(err)
}

svc, err := cfg.BuildService()
if err != nil {
    log.Fatal(err)
}
```

#### Manual Construction
```go
svc, err := simplecontent.New(
    simplecontent.WithRepository(memoryRepo),
    simplecontent.WithBlobStore("memory", memoryStore),
)
if err != nil {
    log.Fatal(err)
}
```

## New Unified Content Operations

### Simple Content Upload (1-Step Process)

```go
// OLD WAY (3 steps):
// content := svc.CreateContent(...)
// object := svc.CreateObject(...)
// svc.UploadObject(...)

// NEW WAY (1 step):
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:            uuid.New(),
    TenantID:           uuid.New(),
    Name:               "My Document",
    Description:        "Sample document",
    DocumentType:       "text/plain",
    StorageBackendName: "memory", // Optional - uses default
    Reader:             strings.NewReader("Hello, World!"),
    FileName:           "hello.txt",
    FileSize:           13,
    Tags:               []string{"sample", "text"},
    CustomMetadata: map[string]interface{}{
        "author":  "John Doe",
        "project": "demo",
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Content uploaded: %s\n", content.ID)
```

### Thumbnail Generation (Unified Derived Content)

```go
// Generate thumbnail image data (your image processing logic)
thumbnailData := generateThumbnail(originalImageData, 256)

// Upload thumbnail as derived content (1 step)
thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:           originalContentID,
    OwnerID:            ownerID,
    TenantID:           tenantID,
    DerivationType:     "thumbnail",
    Variant:            "thumbnail_256",
    StorageBackendName: "memory",
    Reader:             bytes.NewReader(thumbnailData),
    FileName:           "thumb_256.jpg",
    FileSize:           int64(len(thumbnailData)),
    Tags:               []string{"thumbnail", "256px"},
    Metadata: map[string]interface{}{
        "thumbnail_size": 256,
        "algorithm":      "lanczos3",
        "generated_by":   "image_processor",
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Thumbnail created: %s\n", thumbnail.ID)
```

### Download Content

```go
// Download content data directly using content ID
reader, err := svc.DownloadContent(ctx, contentID)
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

data, err := io.ReadAll(reader)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Downloaded %d bytes\n", len(data))
```

### Get All Content Information

```go
// Get everything in one call
details, err := svc.GetContentDetails(ctx, contentID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Content Details:\n")
fmt.Printf("  Download URL: %s\n", details.Download)
fmt.Printf("  File Name: %s\n", details.FileName)
fmt.Printf("  File Size: %d bytes\n", details.FileSize)
fmt.Printf("  MIME Type: %s\n", details.MimeType)
fmt.Printf("  Tags: %v\n", details.Tags)
fmt.Printf("  Ready: %t\n", details.Ready)

// Access organized thumbnails
for size, url := range details.Thumbnails {
    fmt.Printf("  Thumbnail %s: %s\n", size, url)
}
```

### Get Upload URLs (For Direct Client Upload)

```go
// Get content details with upload access
details, err := svc.GetContentDetails(ctx, contentID,
    simplecontent.WithUploadAccess(),
)
if err != nil {
    log.Fatal(err)
}

if details.Upload != "" {
    fmt.Printf("Upload URL: %s\n", details.Upload)
    fmt.Printf("Expires at: %v\n", details.ExpiresAt)
}

// With custom expiry time
details, err := svc.GetContentDetails(ctx, contentID,
    simplecontent.WithUploadAccessExpiry(3600), // 1 hour
)
```

## Complete Examples

### Multi-Size Thumbnail Generation

```go
func generateThumbnails(svc simplecontent.Service, originalContentID uuid.UUID, sizes []int) error {
    ctx := context.Background()

    // Get original content info
    originalContent, err := svc.GetContent(ctx, originalContentID)
    if err != nil {
        return fmt.Errorf("failed to get original content: %w", err)
    }

    // Download original for processing
    reader, err := svc.DownloadContent(ctx, originalContentID)
    if err != nil {
        return fmt.Errorf("failed to download original: %w", err)
    }
    defer reader.Close()

    originalData, err := io.ReadAll(reader)
    if err != nil {
        return fmt.Errorf("failed to read original data: %w", err)
    }

    // Generate thumbnails for each size
    for _, size := range sizes {
        // Process image (your image processing logic)
        thumbnailData := resizeImage(originalData, size)

        // Upload as derived content
        _, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
            ParentID:       originalContentID,
            OwnerID:        originalContent.OwnerID,
            TenantID:       originalContent.TenantID,
            DerivationType: "thumbnail",
            Variant:        fmt.Sprintf("thumbnail_%d", size),
            Reader:         bytes.NewReader(thumbnailData),
            FileName:       fmt.Sprintf("thumb_%dpx.jpg", size),
            FileSize:       int64(len(thumbnailData)),
            Tags:           []string{"thumbnail", fmt.Sprintf("%dpx", size)},
            Metadata: map[string]interface{}{
                "size":        size,
                "source_type": "image_resize",
            },
        })
        if err != nil {
            return fmt.Errorf("failed to create %dpx thumbnail: %w", size, err)
        }

        log.Printf("Generated %dpx thumbnail\n", size)
    }

    return nil
}

// Usage
sizes := []int{128, 256, 512}
err := generateThumbnails(svc, contentID, sizes)
if err != nil {
    log.Fatal(err)
}
```

### Content Gallery with Thumbnails

```go
func getContentGallery(svc simplecontent.Service, ownerID, tenantID uuid.UUID) ([]*simplecontent.ContentDetails, error) {
    ctx := context.Background()

    // List all content for user
    contents, err := svc.ListContent(ctx, simplecontent.ListContentRequest{
        OwnerID:  ownerID,
        TenantID: tenantID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to list content: %w", err)
    }

    // Get details for each content (includes thumbnails)
    var gallery []*simplecontent.ContentDetails
    for _, content := range contents {
        // Skip derived content, only show originals
        if content.DerivationType != "" {
            continue
        }

        details, err := svc.GetContentDetails(ctx, content.ID)
        if err != nil {
            log.Printf("Failed to get details for %s: %v", content.ID, err)
            continue
        }

        gallery = append(gallery, details)
    }

    return gallery, nil
}

// Usage
gallery, err := getContentGallery(svc, userID, tenantID)
if err != nil {
    log.Fatal(err)
}

for _, item := range gallery {
    fmt.Printf("Content: %s\n", item.FileName)
    fmt.Printf("  Download: %s\n", item.Download)
    fmt.Printf("  Thumbnail: %s\n", item.Thumbnail)

    // Show all available thumbnail sizes
    for size, url := range item.Thumbnails {
        fmt.Printf("  Thumb %s: %s\n", size, url)
    }
}
```

## Advanced Patterns

### Working with StorageService (Advanced Users)

```go
// For advanced users who need direct object access
func setupDirectUpload(svc simplecontent.Service) error {
    // Cast to StorageService for object operations
    storageSvc, ok := svc.(simplecontent.StorageService)
    if !ok {
        return fmt.Errorf("service doesn't support storage operations")
    }

    // Create content first
    content, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID:  ownerID,
        TenantID: tenantID,
        Name:     "Direct Upload",
    })
    if err != nil {
        return err
    }

    // Create object for direct upload
    object, err := storageSvc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          content.ID,
        StorageBackendName: "s3",
        Version:            1,
    })
    if err != nil {
        return err
    }

    // Get presigned upload URL
    uploadURL, err := storageSvc.GetUploadURL(ctx, object.ID)
    if err != nil {
        return err
    }

    fmt.Printf("Upload your file to: %s\n", uploadURL)
    return nil
}
```

### Batch Content Processing

```go
func processContentBatch(svc simplecontent.Service, contentIDs []uuid.UUID) error {
    ctx := context.Background()

    for _, contentID := range contentIDs {
        // Get content details
        details, err := svc.GetContentDetails(ctx, contentID)
        if err != nil {
            log.Printf("Failed to get details for %s: %v", contentID, err)
            continue
        }

        // Process based on content type
        switch {
        case strings.HasPrefix(details.MimeType, "image/"):
            err = processImage(svc, contentID, details)
        case strings.HasPrefix(details.MimeType, "video/"):
            err = processVideo(svc, contentID, details)
        case strings.HasPrefix(details.MimeType, "audio/"):
            err = processAudio(svc, contentID, details)
        default:
            log.Printf("Unsupported content type: %s", details.MimeType)
            continue
        }

        if err != nil {
            log.Printf("Failed to process %s: %v", contentID, err)
        }
    }

    return nil
}

func processImage(svc simplecontent.Service, contentID uuid.UUID, details *simplecontent.ContentDetails) error {
    // Generate thumbnails if not already present
    if len(details.Thumbnails) == 0 {
        return generateThumbnails(svc, contentID, []int{128, 256, 512})
    }
    return nil
}
```

## Storage Backend Configuration

### Filesystem Storage
```go
cfg, err := config.Load(
    config.WithStorageBackend("filesystem", map[string]interface{}{
        "base_dir":   "./uploads",
        "url_prefix": "https://example.com/files/",
    }),
)
```

### S3 Storage
```go
cfg, err := config.Load(
    config.WithStorageBackend("s3", map[string]interface{}{
        "region":            "us-west-2",
        "bucket":            "my-bucket",
        "access_key_id":     "your-key",
        "secret_access_key": "your-secret",
    }),
)
```

## Error Handling

```go
import "errors"

content, err := svc.UploadContent(ctx, req)
if err != nil {
    if errors.Is(err, simplecontent.ErrContentNotFound) {
        return fmt.Errorf("content not found")
    }
    if errors.Is(err, simplecontent.ErrStorageBackendNotFound) {
        return fmt.Errorf("storage backend not configured")
    }
    return fmt.Errorf("upload failed: %w", err)
}
```

## Best Practices

### 1. Use Unified Operations
```go
// ✅ Good: Single operation
content, err := svc.UploadContent(ctx, req)

// ❌ Avoid: Multi-step process (use StorageService only if needed)
content := svc.CreateContent(...)
object := storageSvc.CreateObject(...)
storageSvc.UploadObject(...)
```

### 2. Leverage ContentDetails
```go
// ✅ Good: Get everything in one call
details, err := svc.GetContentDetails(ctx, contentID)

// ❌ Avoid: Multiple API calls
content := svc.GetContent(...)
metadata := svc.GetContentMetadata(...)
urls := svc.GetContentURLs(...)
```

### 3. Handle Resources Properly
```go
reader, err := svc.DownloadContent(ctx, contentID)
if err != nil {
    return err
}
defer reader.Close() // Always close readers

// Process reader...
```

### 4. Use Context with Timeouts
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

content, err := svc.UploadContent(ctx, req)
```

## Testing

```go
func setupTestService(t *testing.T) simplecontent.Service {
    cfg, err := config.Load(
        config.WithDatabaseType("memory"),
        config.WithStorageBackend("memory", map[string]interface{}{}),
    )
    if err != nil {
        t.Fatal(err)
    }

    svc, err := cfg.BuildService()
    if err != nil {
        t.Fatal(err)
    }

    return svc
}

func TestContentUpload(t *testing.T) {
    svc := setupTestService(t)

    content, err := svc.UploadContent(context.Background(), simplecontent.UploadContentRequest{
        OwnerID:      uuid.New(),
        TenantID:     uuid.New(),
        Name:         "Test",
        DocumentType: "text/plain",
        Reader:       strings.NewReader("test data"),
    })

    assert.NoError(t, err)
    assert.NotNil(t, content)
}
```

## Migration from Old API

### Before: Object-based Workflow (3 steps)
```go
content := svc.CreateContent(ctx, createReq)
object := svc.CreateObject(ctx, objectReq)
err := svc.UploadObject(ctx, uploadReq)
```

### After: Content-focused Workflow (1 step)
```go
content, err := svc.UploadContent(ctx, uploadReq)
```

### Deprecated Operations:
- `CreateObject()`, `UploadObject()` → Use `UploadContent()`
- `GetContentMetadata()`, `GetContentURLs()` → Use `GetContentDetails()`
- `DownloadObject()` → Use `DownloadContent()`

The new unified API significantly reduces complexity while providing the same functionality through content-focused operations.