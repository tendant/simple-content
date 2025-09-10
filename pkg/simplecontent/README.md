# Simple Content Library

A reusable Go library for content management with pluggable storage backends and repository implementations.

## Overview

The `simplecontent` package provides a clean, pluggable architecture for content management systems. It separates concerns between:

- **Domain types**: Content, Object, metadata types
- **Interfaces**: Service, Repository, BlobStore, EventSink, Previewer  
- **Implementations**: Memory, PostgreSQL, S3, filesystem storage backends

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
    // Create repository and storage backends
    repo := memory.New()
    store := memorystorage.New()
    
    // Create service with functional options
    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("memory", store),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Create content
    content, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID:     uuid.New(),
        TenantID:    uuid.New(), 
        Name:        "My Document",
        Description: "A sample document",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Create object for storage
    object, err := svc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          content.ID,
        StorageBackendName: "memory",
        Version:            1,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Upload data
    data := strings.NewReader("Hello, World!")
    err = svc.UploadObject(ctx, object.ID, data)
    if err != nil {
        log.Fatal(err)
    }
    
    // Download data  
    reader, err := svc.DownloadObject(ctx, object.ID)
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()
    
    // Read downloaded content
    content, err := io.ReadAll(reader)
    fmt.Printf("Downloaded: %s\\n", content)
}
```

## Architecture

### Core Interfaces

- **Service**: Main interface providing all content management operations
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

- **Pluggable architecture**: Swap repositories and storage backends easily
- **Multi-tenant**: Built-in tenant isolation  
- **Versioning**: Support for content versions
- **Metadata management**: Rich metadata support for content and objects
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




## Use Cases

- Document management systems
- Media asset management
- File storage services
- Content delivery platforms
- Multi-tenant SaaS applications

## Testing

The library includes in-memory implementations perfect for testing:

```go
func TestMyFeature(t *testing.T) {
    repo := memory.New()
    store := memorystorage.New()
    
    svc, _ := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("test", store),
    )
    
    // Test your code...
}
```
