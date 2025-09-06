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