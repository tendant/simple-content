# Simple Content Library

A reusable Go library for content management with a unified API that simplifies content operations while providing pluggable storage backends and repository implementations.

## Overview

The `simplecontent` package provides a clean, pluggable architecture for content management systems with a content-focused API design. It separates concerns between:

- **Domain types**: Content, Object, metadata types
- **Unified Service Interface**: Content-focused operations that hide storage implementation details
- **Advanced StorageService Interface**: Object-level operations for advanced users
- **Repository & Storage**: Memory, PostgreSQL, S3, filesystem backends

## Quick Start

### Simple Unified API (Recommended)

```go
package main

import (
    "context"
    "log"
    "strings"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

func main() {
    // Configure service using config system
    cfg, err := config.Load(
        config.WithDatabaseType("memory"),
        config.WithStorageBackend("memory", map[string]interface{}{}),
    )
    if err != nil {
        log.Fatal(err)
    }

    svc, err := cfg.BuildService()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Upload content with data in one operation (NEW!)
    content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
        OwnerID:      uuid.New(),
        TenantID:     uuid.New(),
        Name:         "My Document",
        Description:  "A sample document",
        DocumentType: "text/plain",
        Reader:       strings.NewReader("Hello, World!"),
        FileName:     "hello.txt",
        Tags:         []string{"sample", "document"},
    })
    if err != nil {
        log.Fatal(err)
    }

    // Download content data directly
    reader, err := svc.DownloadContent(ctx, content.ID)
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()

    // Get all content information in one call
    details, err := svc.GetContentDetails(ctx, content.ID)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Content: %s (%s)\\n", details.FileName, details.MimeType)
    fmt.Printf("Download URL: %s\\n", details.Download)
}
```

### Manual Construction (Advanced)

```go
// For advanced users who need custom configuration
repo := memory.New()
store := memorystorage.New()

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("memory", store),
)

// Cast to StorageService for object operations if needed
storageSvc, ok := svc.(simplecontent.StorageService)
if ok {
    // Use object-level operations
    object, err := storageSvc.CreateObject(ctx, req)
    uploadURL, err := storageSvc.GetUploadURL(ctx, object.ID)
}
```

## Architecture

### Core Interfaces

#### Main Service Interface (Recommended)
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

#### StorageService Interface (Advanced)
```go
type StorageService interface {
    // Object operations (for advanced users who need direct object access)
    CreateObject(ctx, CreateObjectRequest) (*Object, error)
    UploadObject(ctx, UploadObjectRequest) error
    GetUploadURL(ctx, objectID) (string, error)
    // ... other object operations
}
```

#### Backend Interfaces
- **Repository**: Data persistence abstraction for contents, objects, and metadata
- **BlobStore**: Storage backend abstraction for binary data
- **EventSink**: Event handling for lifecycle events
- **Previewer**: Content preview generation

### Available Implementations

#### Repositories
- `repo/memory`: In-memory repository (testing)
- `repo/postgres`: PostgreSQL repository (production)

#### Storage Backends  
- `storage/memory`: In-memory storage (testing)
- `storage/fs`: Filesystem storage
- `storage/s3`: S3-compatible storage

### Configuration Options

The service supports functional options for configuration:

```go
svc, err := simplecontent.New(
    simplecontent.WithRepository(postgresRepo),
    simplecontent.WithBlobStore("s3-primary", s3Store),
    simplecontent.WithBlobStore("s3-backup", s3BackupStore),
    simplecontent.WithBlobStore("local", fsStore),
    simplecontent.WithEventSink(eventSink),
    simplecontent.WithPreviewer(previewer),
)
```

## Features

- **Unified Content Operations**: Single-call upload/download operations replace multi-step workflows
- **Content-Focused API**: Work with content concepts, not storage objects
- **Interface Separation**: Service interface for most users, StorageService for advanced use cases
- **Pluggable architecture**: Swap repositories and storage backends easily
- **Multi-tenant**: Built-in tenant isolation
- **Derived Content**: Built-in support for thumbnails, previews, and transcodes
- **Metadata management**: Rich metadata support with unified details API
- **Event system**: Lifecycle event notifications
- **Preview generation**: Extensible preview system
- **Error handling**: Typed errors for better error handling

## Metadata Strategy

The library uses a hybrid metadata approach:

- First-class fields capture common, structured attributes directly on domain types (e.g., `Content.Name`, `Content.Description`, `Object.ObjectType`, `ContentMetadata.FileName`, `ContentMetadata.MimeType`). These fields are authoritative for their respective values.
- Flexible JSON maps (`ContentMetadata.Metadata`, `ObjectMetadata.Metadata`) accommodate extensible, application-specific attributes. Prefer namespaced keys as needed to avoid collisions.
- Avoid duplicating authoritative values in the JSON map. If mirroring is desired for compatibility, treat first-class fields as the source of truth and ensure the JSON copy is consistent.
- Standard keys when present in metadata JSON: `mime_type`, `file_name`, `file_size`, `etag`, plus additional backend-provided attributes. Applications can add custom keys (e.g., `category`, `priority`).

## Derived Content Typing

- Derivation type (user-facing): stored on derived `Content.DerivationType` (e.g., `thumbnail`, `preview`, `transcode`). Omitted for originals.
- Variant (specific): stored on the `content_derived` relationship (DB column `variant`), e.g., `thumbnail_256`, `thumbnail_720`, `conversion`.
- All keyword values use lowercase to minimize typos and normalization overhead. If only `variant` is provided when creating derived content, the service infers `derivation_type` from the variant prefix.

### Typed constants

For clarity and IDE hints, typed string constants are provided:

- Content statuses: `simplecontent.ContentStatus` with constants like `ContentStatusCreated`.
- Object statuses: `simplecontent.ObjectStatus` with constants like `ObjectStatusUploaded`.
- Derivation:
  - Variant: `simplecontent.DerivationVariant` (e.g., `VariantThumbnail256`).

Struct fields remain `string` for compatibility. You can extend by declaring your own typed constants:

```go
const VariantThumbnail1024 simplecontent.DerivationVariant = "thumbnail_1024"
```




## API Migration Guide

### Before: Multi-Step Object Workflow
```go
// Old way (3 steps):
content := svc.CreateContent(ctx, createReq)
object := svc.CreateObject(ctx, objectReq)  // StorageService required
err := svc.UploadObject(ctx, uploadReq)     // StorageService required
```

### After: Unified Content Workflow
```go
// New way (1 step):
content, err := svc.UploadContent(ctx, uploadReq)

// For derived content:
thumbnail, err := svc.UploadDerivedContent(ctx, derivedReq)
```

### When to Use Each Interface

**Use Service Interface (Recommended) when:**
- Uploading content from server-side applications
- Working with content concepts (documents, images, videos)
- Need simplified workflow with minimal complexity
- Files under 100MB

**Use StorageService Interface (Advanced) when:**
- Need direct object access for presigned URLs
- Implementing direct client uploads to storage
- Large files requiring specialized upload patterns
- Need fine-grained control over storage operations

## Use Cases

- Document management systems
- Media asset management
- File storage services
- Content delivery platforms
- Multi-tenant SaaS applications
- Thumbnail and preview generation systems
- Direct client upload applications

## Testing

The library includes in-memory implementations perfect for testing:

```go
func TestMyFeature(t *testing.T) {
    cfg, err := config.Load(
        config.WithDatabaseType("memory"),
        config.WithStorageBackend("memory", map[string]interface{}{}),
    )
    require.NoError(t, err)

    svc, err := cfg.BuildService()
    require.NoError(t, err)

    // Test unified operations
    content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
        OwnerID:      uuid.New(),
        TenantID:     uuid.New(),
        Name:         "Test Content",
        DocumentType: "text/plain",
        Reader:       strings.NewReader("test data"),
    })
    require.NoError(t, err)

    // Test your code...
}
```
