# Presigned Client Upload Guide

This guide demonstrates how to implement presigned client uploads to storage services (S3, MinIO, etc.) using presigned URLs, bypassing the service for the actual file transfer while maintaining content tracking and metadata management.

## Overview

Presigned client uploads offer several advantages:
- **Performance**: Files upload to storage, reducing server load
- **Scalability**: No bandwidth limitations from your application server
- **Cost**: Lower server resource usage and network costs
- **Speed**: Parallel uploads and better CDN integration

The workflow involves:
1. Client requests upload permission from your service
2. Service creates content/object metadata and returns presigned URL
3. Client uploads to storage using the presigned URL
4. Optional: Service updates object status after successful upload

## Architecture

```
┌─────────┐    1. Request Upload    ┌─────────────────┐
│ Client  │ ───────────────────────▶│ Simple-Content  │
│         │                         │ Service         │
│         │    2. Presigned URL     │                 │
│         │ ◀───────────────────────│                 │
└─────────┘                         └─────────────────┘
     │                                       │
     │ 3. Upload                             │ 4. Update Status
     │                                       │    (Optional)
     ▼                                       ▼
┌─────────────────┐                 ┌─────────────────┐
│ Storage Backend │                 │ Repository      │
│ (S3/MinIO)      │                 │ (Database)      │
└─────────────────┘                 └─────────────────┘
```

## Implementation Patterns

### Pattern 1: Basic Direct Upload (Programmatic)

This pattern creates the content metadata first, then provides a presigned URL for presigned upload. Note that presigned uploads require the advanced StorageService interface for object operations.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/google/uuid"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

// DirectUploadService wraps the simple-content service for presigned upload workflows
type DirectUploadService struct {
    svc        simplecontent.Service
    storageSvc simplecontent.StorageService
}

// NewDirectUploadService creates a service configured for presigned uploads
func NewDirectUploadService() (*DirectUploadService, error) {
    // Configure for S3 or S3-compatible storage (required for presigned URLs)
    cfg, err := config.Load(
        config.WithStorageBackend("s3", map[string]interface{}{
            "region":           "us-west-2",
            "bucket":           "my-content-bucket",
            "access_key_id":    os.Getenv("AWS_ACCESS_KEY_ID"),
            "secret_access_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
            "presign_duration": 3600, // 1 hour
        }),
    )
    if err != nil {
        return nil, err
    }

    svc, err := cfg.BuildService()
    if err != nil {
        return nil, err
    }

    // Cast to StorageService for object operations (required for presigned uploads)
    storageSvc, ok := svc.(simplecontent.StorageService)
    if !ok {
        return nil, fmt.Errorf("service doesn't support storage operations")
    }

    return &DirectUploadService{
        svc:        svc,
        storageSvc: storageSvc,
    }, nil
}

// PrepareUploadRequest contains parameters for preparing a presigned upload
type PrepareUploadRequest struct {
    OwnerID      uuid.UUID
    TenantID     uuid.UUID
    FileName     string
    ContentType  string
    FileSize     int64
    Name         string
    Description  string
    Tags         []string
}

// PrepareUploadResponse contains the prepared upload information
type PrepareUploadResponse struct {
    ContentID   uuid.UUID `json:"content_id"`
    ObjectID    uuid.UUID `json:"object_id"`
    UploadURL   string    `json:"upload_url"`
    ExpiresIn   int       `json:"expires_in"`
    // Instructions for the client
    UploadMethod string            `json:"upload_method"` // "PUT"
    Headers      map[string]string `json:"headers,omitempty"`
}

// PrepareDirectUpload prepares everything needed for a presigned client upload
func (dus *DirectUploadService) PrepareDirectUpload(ctx context.Context, req PrepareUploadRequest) (*PrepareUploadResponse, error) {
    // 1. Create the content entity
    content, err := dus.svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID:      req.OwnerID,
        TenantID:     req.TenantID,
        Name:         req.Name,
        Description:  req.Description,
        DocumentType: req.ContentType,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create content: %w", err)
    }

    // 2. Set content metadata
    err = dus.svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
        ContentID:   content.ID,
        ContentType: req.ContentType,
        FileName:    req.FileName,
        FileSize:    req.FileSize,
        Tags:        req.Tags,
        CustomMetadata: map[string]interface{}{
            "upload_method": "direct_client_upload",
            "prepared_at":   ctx.Value("request_time"),
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to set content metadata: %w", err)
    }

    // 3. Create object for storage (uses StorageService interface)
    object, err := dus.storageSvc.CreateObject(ctx, simplecontent.CreateObjectRequest{
        ContentID:          content.ID,
        StorageBackendName: "s3", // Use your configured S3 backend
        Version:            1,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create object: %w", err)
    }

    // 4. Get presigned upload URL (uses StorageService interface)
    uploadURL, err := dus.storageSvc.GetUploadURL(ctx, object.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to get upload URL: %w", err)
    }

    return &PrepareUploadResponse{
        ContentID:    content.ID,
        ObjectID:     object.ID,
        UploadURL:    uploadURL,
        ExpiresIn:    3600, // 1 hour
        UploadMethod: "PUT",
        Headers: map[string]string{
            "Content-Type": req.ContentType,
        },
    }, nil
}

// ConfirmUpload marks an upload as completed and updates object status
func (dus *DirectUploadService) ConfirmUpload(ctx context.Context, objectID uuid.UUID) error {
    // Get the object to update its status (uses StorageService interface)
    object, err := dus.storageSvc.GetObject(ctx, objectID)
    if err != nil {
        return fmt.Errorf("failed to get object: %w", err)
    }

    // Update object status to indicate upload completion
    object.Status = string(simplecontent.ObjectStatusUploaded)
    err = dus.storageSvc.UpdateObject(ctx, object)
    if err != nil {
        return fmt.Errorf("failed to update object status: %w", err)
    }

    // Optionally, sync metadata from storage backend
    _, err = dus.storageSvc.UpdateObjectMetaFromStorage(ctx, objectID)
    if err != nil {
        log.Printf("Warning: failed to sync object metadata from storage: %v", err)
        // Don't fail the confirmation for metadata sync issues
    }

    return nil
}
```

### Pattern 2: HTTP API for Direct Upload

This shows how to expose presigned upload capabilities via HTTP endpoints.

```go
// HTTP handlers for presigned upload workflow
func (dus *DirectUploadService) SetupRoutes() http.Handler {
    mux := http.NewServeMux()

    // Prepare presigned upload
    mux.HandleFunc("/api/v1/uploads/prepare", dus.handlePrepareUpload)

    // Confirm upload completion
    mux.HandleFunc("/api/v1/uploads/confirm", dus.handleConfirmUpload)

    // Check upload status
    mux.HandleFunc("/api/v1/uploads/status", dus.handleUploadStatus)

    return mux
}

func (dus *DirectUploadService) handlePrepareUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        OwnerID     string   `json:"owner_id"`
        TenantID    string   `json:"tenant_id"`
        FileName    string   `json:"file_name"`
        ContentType string   `json:"content_type"`
        FileSize    int64    `json:"file_size"`
        Name        string   `json:"name"`
        Description string   `json:"description,omitempty"`
        Tags        []string `json:"tags,omitempty"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Validate required fields
    if req.OwnerID == "" || req.TenantID == "" || req.FileName == "" {
        http.Error(w, "Missing required fields", http.StatusBadRequest)
        return
    }

    ownerID, err := uuid.Parse(req.OwnerID)
    if err != nil {
        http.Error(w, "Invalid owner_id", http.StatusBadRequest)
        return
    }

    tenantID, err := uuid.Parse(req.TenantID)
    if err != nil {
        http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
        return
    }

    // Prepare the upload
    response, err := dus.PrepareDirectUpload(r.Context(), PrepareUploadRequest{
        OwnerID:     ownerID,
        TenantID:    tenantID,
        FileName:    req.FileName,
        ContentType: req.ContentType,
        FileSize:    req.FileSize,
        Name:        req.Name,
        Description: req.Description,
        Tags:        req.Tags,
    })
    if err != nil {
        log.Printf("Failed to prepare upload: %v", err)
        http.Error(w, "Failed to prepare upload", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (dus *DirectUploadService) handleConfirmUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        ObjectID string `json:"object_id"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    objectID, err := uuid.Parse(req.ObjectID)
    if err != nil {
        http.Error(w, "Invalid object_id", http.StatusBadRequest)
        return
    }

    err = dus.ConfirmUpload(r.Context(), objectID)
    if err != nil {
        log.Printf("Failed to confirm upload: %v", err)
        http.Error(w, "Failed to confirm upload", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "confirmed",
        "object_id": req.ObjectID,
    })
}
```

## Client-Side Implementation Examples

### JavaScript/Browser Client

```javascript
class DirectUploadClient {
    constructor(baseURL) {
        this.baseURL = baseURL;
    }

    async uploadFile(file, metadata = {}) {
        try {
            // Step 1: Prepare the upload
            const prepareResponse = await this.prepareUpload(file, metadata);

            // Step 2: Upload directly to storage
            await this.performDirectUpload(file, prepareResponse);

            // Step 3: Confirm upload completion
            await this.confirmUpload(prepareResponse.object_id);

            return prepareResponse;
        } catch (error) {
            console.error('Upload failed:', error);
            throw error;
        }
    }

    async prepareUpload(file, metadata) {
        const response = await fetch(`${this.baseURL}/api/v1/uploads/prepare`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                owner_id: metadata.owner_id,
                tenant_id: metadata.tenant_id,
                file_name: file.name,
                content_type: file.type,
                file_size: file.size,
                name: metadata.name || file.name,
                description: metadata.description,
                tags: metadata.tags || [],
            }),
        });

        if (!response.ok) {
            throw new Error(`Failed to prepare upload: ${response.statusText}`);
        }

        return await response.json();
    }

    async performDirectUpload(file, prepareResponse) {
        const response = await fetch(prepareResponse.upload_url, {
            method: prepareResponse.upload_method,
            headers: prepareResponse.headers,
            body: file,
        });

        if (!response.ok) {
            throw new Error(`Presigned upload failed: ${response.statusText}`);
        }
    }

    async confirmUpload(objectId) {
        const response = await fetch(`${this.baseURL}/api/v1/uploads/confirm`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                object_id: objectId,
            }),
        });

        if (!response.ok) {
            throw new Error(`Failed to confirm upload: ${response.statusText}`);
        }

        return await response.json();
    }
}

// Usage example
const uploader = new DirectUploadClient('http://localhost:8080');
const fileInput = document.getElementById('file-input');

fileInput.addEventListener('change', async (event) => {
    const file = event.target.files[0];
    if (!file) return;

    try {
        const result = await uploader.uploadFile(file, {
            owner_id: 'user-uuid',
            tenant_id: 'tenant-uuid',
            name: 'My uploaded file',
            description: 'File uploaded via presigned upload',
            tags: ['user-upload', 'direct'],
        });

        console.log('Upload successful:', result);
    } catch (error) {
        console.error('Upload failed:', error);
    }
});
```

### Go Client Example

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
)

type DirectUploadClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewDirectUploadClient(baseURL string) *DirectUploadClient {
    return &DirectUploadClient{
        baseURL:    baseURL,
        httpClient: &http.Client{},
    }
}

func (c *DirectUploadClient) UploadFile(ctx context.Context, filePath string, metadata map[string]interface{}) error {
    // Open the file
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    // Get file info
    fileInfo, err := file.Stat()
    if err != nil {
        return fmt.Errorf("failed to get file info: %w", err)
    }

    // Step 1: Prepare upload
    prepareReq := map[string]interface{}{
        "owner_id":     metadata["owner_id"],
        "tenant_id":    metadata["tenant_id"],
        "file_name":    filepath.Base(filePath),
        "content_type": "application/octet-stream", // You might want to detect this
        "file_size":    fileInfo.Size(),
        "name":         metadata["name"],
        "description":  metadata["description"],
        "tags":         metadata["tags"],
    }

    prepareResponse, err := c.makeRequest(ctx, "POST", "/api/v1/uploads/prepare", prepareReq)
    if err != nil {
        return fmt.Errorf("failed to prepare upload: %w", err)
    }

    // Step 2: Presigned upload to storage
    err = c.performDirectUpload(ctx, file, prepareResponse)
    if err != nil {
        return fmt.Errorf("presigned upload failed: %w", err)
    }

    // Step 3: Confirm upload
    confirmReq := map[string]interface{}{
        "object_id": prepareResponse["object_id"],
    }

    _, err = c.makeRequest(ctx, "POST", "/api/v1/uploads/confirm", confirmReq)
    if err != nil {
        return fmt.Errorf("failed to confirm upload: %w", err)
    }

    fmt.Printf("Successfully uploaded file. Content ID: %s, Object ID: %s\n",
        prepareResponse["content_id"], prepareResponse["object_id"])

    return nil
}

func (c *DirectUploadClient) performDirectUpload(ctx context.Context, file *os.File, prepareResponse map[string]interface{}) error {
    uploadURL := prepareResponse["upload_url"].(string)

    // Reset file pointer
    file.Seek(0, 0)

    req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, file)
    if err != nil {
        return err
    }

    // Set headers if provided
    if headers, ok := prepareResponse["headers"].(map[string]interface{}); ok {
        for key, value := range headers {
            req.Header.Set(key, value.(string))
        }
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("upload failed with status %d", resp.StatusCode)
    }

    return nil
}

func (c *DirectUploadClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (map[string]interface{}, error) {
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, err
        }
        reqBody = bytes.NewReader(jsonBody)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reqBody)
    if err != nil {
        return nil, err
    }

    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
    }

    var result map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&result)
    if err != nil {
        return nil, err
    }

    return result, nil
}

// Usage example
func main() {
    client := NewDirectUploadClient("http://localhost:8080")

    err := client.UploadFile(context.Background(), "./myfile.jpg", map[string]interface{}{
        "owner_id":    "550e8400-e29b-41d4-a716-446655440000",
        "tenant_id":   "550e8400-e29b-41d4-a716-446655440001",
        "name":        "My Photo",
        "description": "Presigned uploaded photo",
        "tags":        []string{"photo", "direct-upload"},
    })

    if err != nil {
        log.Fatal(err)
    }
}
```

## Storage Backend Configuration

### S3 Configuration

```go
// For presigned uploads, you need S3 or S3-compatible storage
cfg, err := config.Load(
    config.WithStorageBackend("s3", map[string]interface{}{
        "region":           "us-west-2",
        "bucket":           "my-content-bucket",
        "access_key_id":    os.Getenv("AWS_ACCESS_KEY_ID"),
        "secret_access_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
        "presign_duration": 3600,          // 1 hour expiry
        "use_ssl":         true,
        "use_path_style":  false,          // Use virtual-hosted-style URLs
    }),
)
```

### MinIO Configuration

```go
cfg, err := config.Load(
    config.WithStorageBackend("minio", map[string]interface{}{
        "region":           "us-east-1",   // MinIO default
        "bucket":           "content-bucket",
        "access_key_id":    "minioadmin",
        "secret_access_key": "minioadmin",
        "endpoint":         "http://localhost:9000",
        "use_ssl":         false,
        "use_path_style":  true,           // Required for MinIO
        "presign_duration": 1800,         // 30 minutes
    }),
)
```

## Security Considerations

### 1. URL Expiration
Set appropriate expiration times for presigned URLs:
```go
// Short expiry for sensitive content
"presign_duration": 900  // 15 minutes

// Longer expiry for bulk uploads
"presign_duration": 7200 // 2 hours
```

### 2. Content Validation
Validate file types and sizes:
```go
func validateUploadRequest(req PrepareUploadRequest) error {
    // File size limits
    if req.FileSize > 100*1024*1024 { // 100MB
        return errors.New("file too large")
    }

    // Content type restrictions
    allowedTypes := []string{
        "image/jpeg", "image/png", "image/gif",
        "application/pdf", "text/plain",
    }

    if !contains(allowedTypes, req.ContentType) {
        return errors.New("unsupported file type")
    }

    return nil
}
```

### 3. Access Control
Implement proper authorization:
```go
func (dus *DirectUploadService) PrepareDirectUpload(ctx context.Context, req PrepareUploadRequest) (*PrepareUploadResponse, error) {
    // Check user permissions
    user := getUserFromContext(ctx)
    if !user.CanUploadTo(req.TenantID) {
        return nil, errors.New("insufficient permissions")
    }

    // Continue with upload preparation...
}
```

## Best Practices

### 1. Progress Tracking
For large files, implement progress tracking:
```javascript
async performDirectUpload(file, prepareResponse) {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();

        // Track upload progress
        xhr.upload.onprogress = (event) => {
            if (event.lengthComputable) {
                const percent = (event.loaded / event.total) * 100;
                this.onProgress?.(percent);
            }
        };

        xhr.onload = () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                resolve();
            } else {
                reject(new Error(`Upload failed: ${xhr.statusText}`));
            }
        };

        xhr.onerror = () => reject(new Error('Upload failed'));

        xhr.open(prepareResponse.upload_method, prepareResponse.upload_url);

        // Set headers
        Object.entries(prepareResponse.headers || {}).forEach(([key, value]) => {
            xhr.setRequestHeader(key, value);
        });

        xhr.send(file);
    });
}
```

### 2. Error Handling and Retry
```go
func (c *DirectUploadClient) performDirectUploadWithRetry(ctx context.Context, file *os.File, prepareResponse map[string]interface{}) error {
    maxRetries := 3
    var lastErr error

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := c.performDirectUpload(ctx, file, prepareResponse)
        if err == nil {
            return nil
        }

        lastErr = err
        if attempt < maxRetries {
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }

    return fmt.Errorf("upload failed after %d attempts: %w", maxRetries, lastErr)
}
```

### 3. Multipart Uploads for Large Files
For files larger than 100MB, consider implementing multipart uploads:
```go
// This would require extending the service to support multipart upload preparation
type MultipartUploadRequest struct {
    PrepareUploadRequest
    PartSize int64 // Size of each part (minimum 5MB for S3)
}

func (dus *DirectUploadService) PrepareMultipartUpload(ctx context.Context, req MultipartUploadRequest) (*MultipartUploadResponse, error) {
    // Implementation would create multipart upload in S3
    // and return URLs for each part
}
```

## Comparison: Direct Upload vs Service Upload

| Aspect | Service Upload | Direct Upload |
|--------|----------------|---------------|
| **Performance** | Limited by server bandwidth | Direct to storage, faster |
| **Scalability** | Server becomes bottleneck | Highly scalable |
| **Complexity** | Simple implementation | More complex workflow |
| **Security** | Centralized control | Requires URL expiration |
| **Monitoring** | Easy to monitor | Requires additional tracking |
| **Cost** | Higher bandwidth costs | Lower operational costs |
| **Client Support** | Any HTTP client | Requires presigned URL support |

## Troubleshooting

### Common Issues

1. **CORS Errors in Browser**
   - Configure your S3 bucket CORS policy
   - Example CORS configuration:
   ```json
   [
     {
       "AllowedHeaders": ["*"],
       "AllowedMethods": ["PUT", "POST"],
       "AllowedOrigins": ["http://localhost:3000"],
       "ExposeHeaders": ["ETag"]
     }
   ]
   ```

2. **Presigned URL Expired**
   - Check your system clock synchronization
   - Increase `presign_duration` if needed
   - Implement URL refresh logic in client

3. **Upload Permissions**
   - Verify S3 IAM permissions include `s3:PutObject`
   - Check bucket policies don't restrict uploads

4. **Content-Type Mismatch**
   - Ensure client sets correct Content-Type header
   - S3 validates Content-Type against presigned URL

## New Unified API Alternative

While the above patterns show presigned upload workflows using the StorageService interface for advanced users, for most use cases you should prefer the new unified content operations:

### Simple Content Upload (Recommended)

```go
// Instead of presigned upload complexity, use the unified API:
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:            req.OwnerID,
    TenantID:           req.TenantID,
    Name:               req.Name,
    Description:        req.Description,
    DocumentType:       req.ContentType,
    StorageBackendName: "s3",
    Reader:             fileReader,
    FileName:           req.FileName,
    FileSize:           req.FileSize,
    Tags:               req.Tags,
})
// Single call replaces the entire prepare->upload->confirm workflow
```

### When to Use Direct Upload vs Unified API

**Use Direct Upload (StorageService) when:**
- Large files (>100MB) that benefit from client-side upload
- Need to minimize server bandwidth usage
- Implementing file upload from browser/mobile clients
- Need presigned URL functionality

**Use Unified API (Service) when:**
- Files uploaded from server-side applications
- Simpler implementation requirements
- Files under 100MB
- Don't need presigned URL complexity

### Getting Upload URLs with Unified API

The unified API also supports getting upload URLs without the complexity:

```go
// Get content details with upload access
details, err := svc.GetContentDetails(ctx, contentID,
    simplecontent.WithUploadAccess(),
)

if details.Upload != "" {
    // Client can upload directly to details.Upload URL
    fmt.Printf("Upload URL: %s\n", details.Upload)
    fmt.Printf("Expires at: %v\n", details.ExpiresAt)
}
```

Presigned client uploads provide excellent performance and scalability benefits while maintaining the content management capabilities of the simple-content library. However, consider whether the unified API meets your needs before implementing the more complex presigned upload workflow. The key for presigned uploads is proper implementation of the three-step workflow: prepare, upload, confirm.