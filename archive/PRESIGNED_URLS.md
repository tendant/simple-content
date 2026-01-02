# Presigned URL Guide

This comprehensive guide covers presigned URL functionality in Simple Content, including client uploads, downloads, security, and library usage.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Client Upload Workflow](#client-upload-workflow)
- [Download URLs](#download-urls)
- [Security & Authentication](#security--authentication)
- [Package Library Usage](#package-library-usage)
- [Examples](#examples)

## Overview

Presigned URLs allow clients to upload or download files directly to/from storage backends (S3, MinIO, etc.) without routing data through your application server.

**Benefits:**
- **Performance**: Direct uploads/downloads reduce server load
- **Scalability**: No bandwidth bottleneck at application layer
- **Cost**: Lower server resource usage
- **Speed**: Parallel uploads and better CDN integration

**Architecture:**
```
┌─────────┐    1. Request URL      ┌─────────────────┐
│ Client  │ ───────────────────────▶│ Simple-Content  │
│         │    2. Presigned URL     │ Service         │
│         │ ◀───────────────────────│                 │
└─────────┘                         └─────────────────┘
     │                                       │
     │ 3. Upload/Download                    │
     ▼                                       ▼
┌─────────────────┐                 ┌─────────────────┐
│ Storage Backend │                 │ Repository      │
│ (S3/MinIO/FS)   │                 │ (Database)      │
└─────────────────┘                 └─────────────────┘
```

## Quick Start

### Generate Presigned Upload URL

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

// Create service with S3 backend (required for presigned URLs)
cfg, _ := config.Load(
    config.WithStorageBackend("s3", map[string]interface{}{
        "bucket": "my-bucket",
        "region": "us-west-2",
    }),
)
svc, _ := cfg.BuildService()

// Create content placeholder
content, _ := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
    OwnerID:      ownerID,
    TenantID:     tenantID,
    Name:         "document.pdf",
    DocumentType: "application/pdf",
})

// Get StorageService interface for presigned operations
storageSvc := svc.(simplecontent.StorageService)

// Create object
object, _ := storageSvc.CreateObject(ctx, simplecontent.CreateObjectRequest{
    ContentID:          content.ID,
    StorageBackendName: "s3",
    FileName:           "document.pdf",
})

// Generate presigned upload URL
uploadURL, _ := storageSvc.GetUploadURL(ctx, object.ID)

// Client uploads directly to this URL
```

### Generate Presigned Download URL

```go
// Get presigned download URL
downloadURL, _ := storageSvc.GetDownloadURL(ctx, object.ID)

// Client downloads directly from this URL
```

## Client Upload Workflow

### Pattern 1: Basic Presigned Upload

**Server-side:**
```go
type PrepareUploadResponse struct {
    ContentID  uuid.UUID `json:"content_id"`
    ObjectID   uuid.UUID `json:"object_id"`
    UploadURL  string    `json:"upload_url"`
    ExpiresAt  time.Time `json:"expires_at"`
}

func (h *Handler) PrepareUpload(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req struct {
        FileName     string `json:"file_name"`
        DocumentType string `json:"document_type"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Create content
    content, _ := h.svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        Name:         req.FileName,
        DocumentType: req.DocumentType,
        Status:       string(simplecontent.ContentStatusCreated),
    })

    // Create object
    storageSvc := h.svc.(simplecontent.StorageService)
    object, _ := storageSvc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          content.ID,
        StorageBackendName: "s3",
        FileName:           req.FileName,
    })

    // Generate presigned URL
    uploadURL, _ := storageSvc.GetUploadURL(ctx, object.ID)

    // Return to client
    json.NewEncoder(w).Encode(PrepareUploadResponse{
        ContentID: content.ID,
        ObjectID:  object.ID,
        UploadURL: uploadURL,
        ExpiresAt: time.Now().Add(1 * time.Hour),
    })
}
```

**Client-side:**
```javascript
// 1. Request presigned URL
const response = await fetch('/api/v1/prepare-upload', {
  method: 'POST',
  body: JSON.stringify({
    file_name: 'document.pdf',
    document_type: 'application/pdf'
  })
});
const { upload_url, content_id } = await response.json();

// 2. Upload file directly to storage
await fetch(upload_url, {
  method: 'PUT',
  body: fileData,
  headers: {
    'Content-Type': 'application/pdf'
  }
});

// 3. (Optional) Confirm upload completion
await fetch(`/api/v1/contents/${content_id}/confirm-upload`, {
  method: 'POST'
});
```

### Pattern 2: With Status Tracking

```go
// After presigned upload completes, update status
func (h *Handler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
    contentID := chi.URLParam(r, "contentID")

    // Update content status to uploaded
    err := h.svc.UpdateContentStatus(ctx,
        uuid.MustParse(contentID),
        simplecontent.ContentStatusUploaded,
    )

    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

## Download URLs

### Content-Based Downloads

```go
// Get unified content details (includes download URL)
details, _ := svc.GetContentDetails(ctx, contentID)
downloadURL := details.Download
```

### Presigned Storage Downloads

```go
// Get presigned download URL from storage backend
storageSvc := svc.(simplecontent.StorageService)
downloadURL, _ := storageSvc.GetDownloadURL(ctx, objectID)
```

### URL Strategy Configuration

Configure download URL generation strategy:

```bash
# Content-based (via app)
URL_STRATEGY=content-based
API_BASE_URL=/api/v1

# CDN (direct to storage)
URL_STRATEGY=cdn
CDN_BASE_URL=https://cdn.example.com

# Storage-delegated (presigned URLs)
URL_STRATEGY=storage-delegated
```

## Security & Authentication

### HMAC Signing (Application-Level)

For custom presigned URL signing at the application level:

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presigned"

// Create signer
signer := presigned.New(
    presigned.WithSecretKey("your-secret-key-min-32-chars"),
    presigned.WithDefaultExpiration(15*time.Minute),
)

// Generate signed URL
url, _ := signer.SignURLWithBase(
    "https://api.example.com",
    "PUT",
    "/upload/myfile.pdf",
    1*time.Hour,
)
// Returns: https://api.example.com/upload/myfile.pdf?signature=abc...&expires=1696789012

// Validate upload request
err := signer.ValidateRequest(r)
if err != nil {
    http.Error(w, "Invalid signature", http.StatusUnauthorized)
    return
}
```

### Storage Backend Signing (S3/MinIO)

S3 and MinIO provide native presigned URL signing:

```go
// Configure S3 with presigned duration
cfg, _ := config.Load(
    config.WithStorageBackend("s3", map[string]interface{}{
        "bucket":           "my-bucket",
        "region":           "us-west-2",
        "presign_duration": 3600, // 1 hour
    }),
)
```

### Security Best Practices

1. **Short expiration times**: 15 minutes for uploads, 1 hour for downloads
2. **Content-Type validation**: Enforce expected MIME types
3. **File size limits**: Configure max upload size
4. **Rate limiting**: Prevent abuse of URL generation
5. **HTTPS only**: Always use TLS for presigned URLs
6. **Signature validation**: Verify HMAC signatures server-side

## Package Library Usage

The `pkg/simplecontent/presigned` package provides reusable signing:

### Server Integration

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presigned"

// Create signer with options
signer := presigned.New(
    presigned.WithSecretKey(os.Getenv("PRESIGN_SECRET")),
    presigned.WithDefaultExpiration(15*time.Minute),
    presigned.WithClockSkew(5*time.Minute),
)

// Use as middleware
http.Handle("/upload/", presigned.Middleware(signer)(uploadHandler))
```

### Client SDK

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presigned/client"

// Create client
client := client.New("https://api.example.com")

// Upload file
err := client.UploadWithPresignedURL(ctx, presignedURL, fileData)
```

## Examples

Complete working examples:

- **[examples/presigned-upload](./examples/presigned-upload/)** - Client presigned upload workflow
- **[examples/presigned-handlers](./examples/presigned-handlers/)** - Server-side handler implementation
- **[examples/photo-gallery](./examples/photo-gallery/)** - Real-world application using presigned URLs

### Running Examples

```bash
# Presigned upload example
cd examples/presigned-upload
go run main.go

# Presigned handlers example
cd examples/presigned-handlers
go run main.go
```

## Advanced Topics

### Multi-Part Uploads

For large files, use multi-part uploads with presigned URLs:

```go
// TODO: Multi-part upload example
// See AWS S3 SDK documentation for multi-part presigned URLs
```

### Cross-Origin Uploads

Configure CORS for browser-based uploads:

```go
// Configure S3 bucket CORS
// Allow: GET, PUT, POST from your domain
```

### Custom Storage Backends

Implement `BlobStore` interface with presigned URL support:

```go
type CustomStorage struct {
    // ...
}

func (s *CustomStorage) GetPresignedUploadURL(ctx context.Context, key string, duration time.Duration) (string, error) {
    // Generate presigned URL for your custom storage
    return url, nil
}
```

## Troubleshooting

### Common Issues

**Expired URLs:**
- Increase `presign_duration` configuration
- Check server/client clock synchronization

**Access Denied:**
- Verify S3 bucket permissions
- Check IAM role/credentials
- Validate CORS configuration

**Invalid Signature:**
- Ensure secret key matches on server/client
- Check for URL encoding issues
- Verify timestamp within clock skew window

## Additional Resources

- [CONFIGURATION_GUIDE.md](./CONFIGURATION_GUIDE.md) - Storage backend configuration
- [API.md](./API.md) - Complete API reference
- [pkg/simplecontent/presigned](./pkg/simplecontent/presigned/) - Package documentation
