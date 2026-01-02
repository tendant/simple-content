# Presigned URLs Guide

Comprehensive guide to presigned URL functionality in Simple Content for client uploads, downloads, and secure direct-to-storage access.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Client Upload Workflow](#client-upload-workflow)
- [Download URLs](#download-urls)
- [Security](#security)
- [Examples](#examples)

## Overview

Presigned URLs allow clients to upload or download files directly to/from storage backends (S3, MinIO, filesystem) without routing data through your application server.

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

### Server-Side Handler

```go
func (h *Handler) PrepareUpload(w http.ResponseWriter, r *http.Request) {
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
    json.NewEncoder(w).Encode(map[string]interface{}{
        "content_id": content.ID,
        "object_id":  object.ID,
        "upload_url": uploadURL,
        "expires_at": time.Now().Add(1 * time.Hour),
    })
}
```

### Client-Side Upload

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

### Status Tracking

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

## Security

### S3/MinIO Presigned URLs

S3 and MinIO provide native presigned URL signing with AWS Signature V4:

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

## Examples

Complete working examples:

- **[examples/presigned-upload](./examples/presigned-upload/)** - Client presigned upload workflow
- **[examples/photo-gallery](./examples/photo-gallery/)** - Real-world application using presigned URLs

### Running Examples

```bash
# Presigned upload example
cd examples/presigned-upload
go run main.go

# Photo gallery with presigned URLs
cd examples/photo-gallery
go run main.go
```

## Advanced Topics

### Storage Backend Support

Presigned URLs are supported by:
- ✅ **S3** - AWS S3 native presigned URLs
- ✅ **MinIO** - S3-compatible presigned URLs
- ⚠️ **Filesystem** - Application-generated signed URLs

### Configuration

```bash
# S3 storage configuration
STORAGE_BACKEND=s3
AWS_S3_BUCKET=my-bucket
AWS_S3_REGION=us-west-2
AWS_S3_PRESIGN_DURATION=3600

# Filesystem storage (requires custom signing)
STORAGE_BACKEND=fs
FS_BASE_PATH=./data
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
- Ensure credentials are correct
- Check for URL encoding issues
- Verify timestamp within allowed window

## Additional Resources

- [CONFIGURATION_GUIDE.md](./CONFIGURATION_GUIDE.md) - Storage backend configuration
- [API.md](./API.md) - Complete API reference
- [examples/presigned-upload](./examples/presigned-upload/) - Complete working example

## See Also

For detailed implementation guides, see the `archive/` directory:
- `archive/PRESIGNED_CLIENT_UPLOAD.md` - Detailed client upload patterns
- `archive/PRESIGNED_DOWNLOAD_IMPLEMENTATION.md` - Download implementation details
- `archive/PRESIGNED_PACKAGE.md` - Package library usage
- `archive/PRESIGNED_UPLOAD_AUTH.md` - Authentication and security details
