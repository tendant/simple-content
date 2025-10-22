# Development Preset Example

This example demonstrates the **Development Preset** - the fastest way to get started with Simple Content.

## Features

- **One-line setup**: `NewDevelopment()` creates a fully configured service
- **In-memory database**: No PostgreSQL or database setup required
- **Filesystem storage**: Data persists at `./dev-data/` across restarts
- **Automatic cleanup**: Cleanup function removes storage directory

## Quick Start

```bash
go run main.go
```

## What This Example Shows

1. **Service Creation** - One line to create a working service
2. **Upload Content** - Upload documents and images
3. **Derived Content** - Create thumbnails from images
4. **Download Content** - Download uploaded files
5. **Content Details** - Get comprehensive information about content

## Code Walkthrough

### Create Service

```go
svc, cleanup, err := simplecontentpresets.NewDevelopment()
if err != nil {
    log.Fatal(err)
}
defer cleanup() // Remove ./dev-data/ when done
```

### Upload Content

```go
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      uuid.New(),
    TenantID:     uuid.New(),
    Name:         "Development Guide",
    DocumentType: "text/plain",
    Reader:       strings.NewReader("This is a guide for local development."),
    FileName:     "dev-guide.txt",
})
```

### Create Derived Content

```go
thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       imageID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         thumbnailReader,
    FileName:       "screenshot_thumb.png",
    MimeType:       "image/png",
})
```

## Customization

The development preset supports customization options:

```go
// Custom storage directory
svc, cleanup, err := simplecontentpresets.NewDevelopment(
    simplecontentpresets.WithDevStorage("./my-custom-dir"),
)

// Custom port (for future server integration)
svc, cleanup, err := simplecontentpresets.NewDevelopment(
    simplecontentpresets.WithDevPort("3000"),
)
```

## When to Use

Use the **Development Preset** when:

- Learning Simple Content for the first time
- Prototyping new features
- Local development without database setup
- Testing integration with your application
- Running demos and presentations

## Next Steps

- See [preset-testing](../preset-testing/) for unit testing patterns
- See [CONFIGURATION_GUIDE.md](../../CONFIGURATION_GUIDE.md) for advanced configuration
- See [QUICKSTART.md](../../QUICKSTART.md) for more examples
