# Simple Content - Quickstart Guide

Get started with Simple Content in 5 minutes! This guide shows you how to set up and use the library for common content management scenarios.

## 📦 Installation

```bash
go get github.com/tendant/simple-content
```

## 🎯 Quick Examples

### Example 1: Basic Setup (In-Memory)

Perfect for development, testing, or learning:

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
    // Create service with in-memory storage (zero configuration!)
    svc, err := simplecontent.New(
        simplecontent.WithRepository(memoryrepo.New()),
        simplecontent.WithBlobStore("memory", memorystorage.New()),
    )
    if err != nil {
        panic(err)
    }

    // Upload your first content
    content, err := svc.UploadContent(context.Background(), simplecontent.UploadContentRequest{
        OwnerID:      uuid.New(),
        TenantID:     uuid.New(),
        Name:         "hello.txt",
        DocumentType: "text",
        Reader:       strings.NewReader("Hello, Simple Content!"),
        FileName:     "hello.txt",
        MimeType:     "text/plain",
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("✅ Content uploaded! ID: %s\n", content.ID)

    // Download it back
    reader, err := svc.DownloadContent(context.Background(), content.ID)
    if err != nil {
        panic(err)
    }
    defer reader.Close()

    // Read and print
    buf := new(strings.Builder)
    io.Copy(buf, reader)
    fmt.Printf("📄 Content: %s\n", buf.String())
}
```

**Output:**
```
✅ Content uploaded! ID: 7a8e9f3c-...
📄 Content: Hello, Simple Content!
```

### Example 2: Filesystem Storage

Persist content to local filesystem:

```go
package main

import (
    "context"
    "os"

    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

func main() {
    // Create data directory
    dataDir := "./content-data"
    os.MkdirAll(dataDir, 0755)

    // Setup filesystem storage
    fsBackend, err := fsstorage.New(fsstorage.Config{
        BaseDir: dataDir,
    })
    if err != nil {
        panic(err)
    }

    svc, err := simplecontent.New(
        simplecontent.WithRepository(memoryrepo.New()),
        simplecontent.WithBlobStore("fs", fsBackend),
    )
    if err != nil {
        panic(err)
    }

    // Upload a file from disk
    file, _ := os.Open("document.pdf")
    defer file.Close()

    content, err := svc.UploadContent(context.Background(), simplecontent.UploadContentRequest{
        OwnerID:      uuid.New(),
        TenantID:     uuid.New(),
        Name:         "Important Document",
        DocumentType: "document",
        Reader:       file,
        FileName:     "document.pdf",
        MimeType:     "application/pdf",
    })

    fmt.Printf("✅ Saved to: %s/originals/...\n", dataDir)
}
```

### Example 3: Production Setup (PostgreSQL + S3)

```go
package main

import (
    "context"
    "os"

    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

func main() {
    // Load from environment variables - zero code configuration!
    // Just set: DATABASE_URL, S3_BUCKET, AWS_REGION, etc.
    cfg, err := config.LoadServerConfig()
    if err != nil {
        panic(err)
    }

    svc, err := config.BuildService(context.Background(), cfg)
    if err != nil {
        panic(err)
    }

    // Use the service - same API, production storage!
    // ... your business logic here
}
```

**Environment variables:**
```bash
export DATABASE_TYPE=postgres
export DATABASE_URL=postgres://user:pass@localhost:5432/content
export STORAGE_BACKEND=s3
export S3_BUCKET=my-content-bucket
export AWS_REGION=us-east-1
```

### Example 4: Creating Derived Content (Thumbnails)

```go
package main

import (
    "context"
    "image"
    "image/jpeg"
    "os"

    "github.com/nfnt/resize"
    "github.com/tendant/simple-content/pkg/simplecontent"
)

func main() {
    // Assume svc is already created...

    // Upload original image
    imageFile, _ := os.Open("photo.jpg")
    defer imageFile.Close()

    original, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
        OwnerID:      ownerID,
        TenantID:     tenantID,
        Name:         "Vacation Photo",
        DocumentType: "image",
        Reader:       imageFile,
        FileName:     "photo.jpg",
        MimeType:     "image/jpeg",
    })

    // Generate thumbnail (256x256)
    imageFile.Seek(0, 0) // Reset reader
    img, _ := jpeg.Decode(imageFile)
    thumbnail := resize.Thumbnail(256, 256, img, resize.Lanczos3)

    // Save as derived content
    thumbnailBuf := new(bytes.Buffer)
    jpeg.Encode(thumbnailBuf, thumbnail, nil)

    derived, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
        ParentID:       original.ID,
        DerivationType: "thumbnail",
        Variant:        "thumbnail_256",
        Reader:         thumbnailBuf,
        FileName:       "photo_thumb.jpg",
        MimeType:       "image/jpeg",
    })

    fmt.Printf("✅ Thumbnail created: %s\n", derived.ID)
}
```

### Example 5: Working with Metadata

```go
package main

import (
    "context"

    "github.com/tendant/simple-content/pkg/simplecontent"
)

func main() {
    // Upload content...
    content, _ := svc.UploadContent(ctx, uploadReq)

    // Set rich metadata
    err := svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
        ContentID:   content.ID,
        ContentType: "video/mp4",
        Title:       "Product Demo Video",
        Description: "Demonstration of our new product features",
        Tags:        []string{"product", "demo", "tutorial"},
        FileName:    "demo.mp4",
        FileSize:    52428800, // 50MB
        CreatedBy:   "john@company.com",
        CustomMetadata: map[string]interface{}{
            "duration":    "00:05:30",
            "resolution":  "1920x1080",
            "bitrate":     "5000kbps",
            "codec":       "h264",
            "category":    "marketing",
            "internal_id": "PROD-2024-001",
        },
    })

    // Retrieve metadata
    metadata, err := svc.GetContentMetadata(ctx, content.ID)
    fmt.Printf("Title: %s\n", metadata.Metadata["title"])
    fmt.Printf("Duration: %s\n", metadata.Metadata["duration"])

    // Query by status
    uploadedContent, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusUploaded)
    fmt.Printf("Found %d uploaded items\n", len(uploadedContent))
}
```

## 🎨 Common Use Cases

### Photo Gallery Application

```go
// Upload photo with automatic status management
photo, _ := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      userID,
    TenantID:     tenantID,
    Name:         "Sunset",
    DocumentType: "photo",
    Reader:       photoFile,
    FileName:     "sunset.jpg",
    MimeType:     "image/jpeg",
})

// Create multiple thumbnail sizes
for _, size := range []int{128, 256, 512, 1024} {
    thumb := generateThumbnail(photoFile, size)
    svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
        ParentID:       photo.ID,
        DerivationType: "thumbnail",
        Variant:        fmt.Sprintf("thumbnail_%d", size),
        Reader:         thumb,
        FileName:       fmt.Sprintf("sunset_%d.jpg", size),
    })
}

// Get all details with URLs
details, _ := svc.GetContentDetails(ctx, photo.ID)
fmt.Printf("Download URL: %s\n", details.DownloadURL)
fmt.Printf("Thumbnails: %v\n", details.Thumbnails)
```

### Document Management System

```go
// Upload document
doc, _ := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      orgID,
    TenantID:     tenantID,
    Name:         "Q4 Financial Report",
    DocumentType: "document",
    Reader:       pdfFile,
    FileName:     "q4-2024.pdf",
    MimeType:     "application/pdf",
})

// Set document metadata
svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
    ContentID:   doc.ID,
    ContentType: "application/pdf",
    Tags:        []string{"financial", "q4", "2024", "confidential"},
    CustomMetadata: map[string]interface{}{
        "department":   "finance",
        "fiscal_year":  2024,
        "quarter":      4,
        "pages":        45,
        "approved_by":  "cfo@company.com",
        "approval_date": "2024-10-15",
    },
})

// Generate preview (first page)
preview := generatePDFPreview(pdfFile)
svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       doc.ID,
    DerivationType: "preview",
    Variant:        "preview_first_page",
    Reader:         preview,
    FileName:       "q4-2024-preview.png",
})
```

### Video Platform

```go
// Upload video
video, _ := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      creatorID,
    TenantID:     tenantID,
    Name:         "How to Build a REST API",
    DocumentType: "video",
    Reader:       videoFile,
    FileName:     "rest-api-tutorial.mp4",
})

// Create transcoded versions
for _, format := range []string{"1080p", "720p", "480p", "360p"} {
    transcoded := transcodeVideo(videoFile, format)
    svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
        ParentID:       video.ID,
        DerivationType: "transcode",
        Variant:        fmt.Sprintf("transcode_%s", format),
        Reader:         transcoded,
        FileName:       fmt.Sprintf("tutorial-%s.mp4", format),
    })
}

// Get all versions
derived, _ := svc.ListDerivedContent(ctx,
    simplecontent.WithParentID(video.ID),
    simplecontent.WithDerivationType("transcode"),
)
fmt.Printf("Available in %d formats\n", len(derived))
```

## 🔧 Configuration Presets

### Development (Fast, Simple)

```go
svc, _ := simplecontent.New(
    simplecontent.WithRepository(memoryrepo.New()),
    simplecontent.WithBlobStore("memory", memorystorage.New()),
)
```

### Testing (Isolated, Reproducible)

```go
tempDir, _ := os.MkdirTemp("", "content-test-*")
defer os.RemoveAll(tempDir)

fsBackend, _ := fsstorage.New(fsstorage.Config{BaseDir: tempDir})
svc, _ := simplecontent.New(
    simplecontent.WithRepository(memoryrepo.New()),
    simplecontent.WithBlobStore("fs", fsBackend),
)
```

### Production (PostgreSQL + S3)

```go
// Option 1: Environment-based (recommended)
cfg, _ := config.LoadServerConfig()
svc, _ := config.BuildService(ctx, cfg)

// Option 2: Explicit configuration
pgRepo, _ := postgresrepo.New(ctx, databaseURL)
s3Backend, _ := s3storage.New(s3storage.Config{
    Bucket: "my-bucket",
    Region: "us-east-1",
})

svc, _ := simplecontent.New(
    simplecontent.WithRepository(pgRepo),
    simplecontent.WithBlobStore("s3", s3Backend),
)
```

## 📚 Next Steps

1. **Read the full documentation**: Check out [CLAUDE.md](./CLAUDE.md) for architecture details
2. **Explore examples**: See [examples/](./examples/) for complete working applications
3. **Run the example server**: `go run ./cmd/server-configured` to try the REST API
4. **Check out tests**: [pkg/simplecontent/*_test.go](./pkg/simplecontent/) for more usage patterns

## 🆘 Common Issues

**Q: How do I handle large files?**
```go
// Use streaming uploads - the Reader interface handles any size
file, _ := os.Open("large-video.mp4") // Could be 10GB+
svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    Reader: file, // Streams directly, no memory issues
    // ... other fields
})
```

**Q: How do I implement access control?**
```go
// Check ownership before operations
content, _ := svc.GetContent(ctx, contentID)
if content.OwnerID != currentUserID {
    return errors.New("access denied")
}
```

**Q: How do I handle concurrent uploads?**
```go
// Simple Content is safe for concurrent use
var wg sync.WaitGroup
for _, file := range files {
    wg.Add(1)
    go func(f File) {
        defer wg.Done()
        svc.UploadContent(ctx, ...) // Safe!
    }(file)
}
wg.Wait()
```

## 🤝 Need Help?

- 📖 [Full Documentation](./CLAUDE.md)
- 💬 [GitHub Discussions](https://github.com/tendant/simple-content/discussions)
- 🐛 [Report Issues](https://github.com/tendant/simple-content/issues)
- 📧 Email: support@example.com

---

**Ready to build something awesome? Start with Example 1 above and go from there!** 🚀
