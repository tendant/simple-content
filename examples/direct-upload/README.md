# Direct Upload Example

This example demonstrates how to implement direct client uploads to storage backends using presigned URLs. Instead of uploading through the service, clients upload directly to the storage backend (S3/MinIO) for better performance and scalability.

## Features

- **Direct Upload Workflow**: Prepare → Upload → Confirm pattern
- **Presigned URL Generation**: Secure, time-limited URLs for direct storage access
- **Web Interface**: Interactive demo with progress tracking
- **Content Management**: Full integration with simple-content library
- **Real-time Progress**: Upload progress tracking with JavaScript
- **Error Handling**: Comprehensive error handling and retry logic

## Prerequisites

1. **Go 1.24+** installed
2. **MinIO server** running locally (or AWS S3 access)
3. **Dependencies** installed (run `go mod tidy` from project root)

### Starting MinIO

```bash
# Using Docker (recommended for demo)
docker run -p 9000:9000 -p 9001:9001 \
  minio/minio server /data --console-address ":9001"

# Or download and run MinIO binary
# https://docs.min.io/docs/minio-quickstart-guide.html
```

Default credentials: `minioadmin` / `minioadmin`

## Running the Example

1. **Start MinIO** (see above)

2. **Start the demo server**:
   ```bash
   cd examples/direct-upload
   go run main.go
   ```

3. **Open your browser** to `http://localhost:8080`

4. **Upload files** using the web interface

## How It Works

### 1. Three-Phase Upload Process

```
Client                    Service                 Storage (S3/MinIO)
  |                         |                           |
  |-- 1. Prepare Upload --->|                           |
  |    (metadata + file info)|                           |
  |                         |-- Create Content -------->| (database)
  |                         |-- Create Object --------->| (database)
  |                         |-- Get Presigned URL ----->|
  |<-- Upload Details ------|                           |
  |    (presigned URL)      |                           |
  |                         |                           |
  |-- 2. Direct Upload -------------------->|
  |    (binary data)                        |
  |<-- Upload Confirmation -----------------|
  |                         |                           |
  |-- 3. Confirm Upload --->|                           |
  |                         |-- Update Status -------->| (database)
  |<-- Confirmation --------|                           |
```

### 2. API Endpoints

#### Prepare Upload
```http
POST /api/v1/uploads/prepare
Content-Type: application/json

{
  "owner_id": "uuid",
  "tenant_id": "uuid",
  "file_name": "document.pdf",
  "content_type": "application/pdf",
  "file_size": 1024000,
  "name": "My Document",
  "description": "Optional description",
  "tags": ["demo", "upload"]
}
```

**Response:**
```json
{
  "content_id": "uuid",
  "object_id": "uuid",
  "upload_url": "https://minio:9000/bucket/object-key?X-Amz-Algorithm=...",
  "expires_in": 1800,
  "upload_method": "PUT",
  "headers": {
    "Content-Type": "application/pdf"
  }
}
```

#### Direct Upload to Storage
```http
PUT [upload_url from prepare response]
Content-Type: application/pdf

[binary file data]
```

#### Confirm Upload
```http
POST /api/v1/uploads/confirm
Content-Type: application/json

{
  "object_id": "uuid"
}
```

### 3. Client Implementation

The example includes both a web client (JavaScript) and shows how to implement a Go client:

```go
// Go client example
client := NewDirectUploadClient("http://localhost:8080")
err := client.UploadFile(context.Background(), "./myfile.pdf", map[string]interface{}{
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "My Document",
    "description": "Uploaded via direct client",
    "tags": []string{"direct", "upload"},
})
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | `minioadmin` | MinIO secret key |
| `MINIO_ENDPOINT` | `http://localhost:9000` | MinIO endpoint URL |

### Storage Backend Configuration

The example configures MinIO as an S3-compatible backend:

```go
config.WithStorageBackend("s3", map[string]interface{}{
    "region":                     "us-east-1",
    "bucket":                     "direct-upload-demo",
    "access_key_id":              "minioadmin",
    "secret_access_key":          "minioadmin",
    "endpoint":                   "http://localhost:9000",
    "use_ssl":                    false,
    "use_path_style":             true, // Required for MinIO
    "presign_duration":           1800, // 30 minutes
    "create_bucket_if_not_exist": true,
})
```

## File Structure

```
examples/direct-upload/
├── main.go              # Main server implementation
├── README.md           # This file
└── client/             # Additional client examples
    ├── go-client.go    # Standalone Go client
    └── curl-examples.sh # cURL command examples
```

## Testing with cURL

### 1. Prepare Upload
```bash
curl -X POST http://localhost:8080/api/v1/uploads/prepare \
  -H "Content-Type: application/json" \
  -d '{
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440001",
    "file_name": "test.txt",
    "content_type": "text/plain",
    "file_size": 13,
    "name": "Test Document"
  }'
```

### 2. Upload to Presigned URL
```bash
# Use the upload_url from the prepare response
curl -X PUT "[PRESIGNED_URL]" \
  -H "Content-Type: text/plain" \
  -d "Hello, World!"
```

### 3. Confirm Upload
```bash
curl -X POST http://localhost:8080/api/v1/uploads/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "object_id": "[OBJECT_ID_FROM_PREPARE]"
  }'
```

## Benefits of Direct Upload

### Performance
- **Reduced Server Load**: Files don't pass through your application server
- **Parallel Uploads**: Multiple files can upload simultaneously
- **Better Throughput**: Direct connection to storage backend

### Scalability
- **No Bandwidth Limits**: Your server bandwidth doesn't limit file uploads
- **Geographic Distribution**: Can use CDN/edge locations for uploads
- **Horizontal Scaling**: Upload performance scales with storage backend

### Cost Efficiency
- **Lower Compute Costs**: Less CPU and memory usage on application servers
- **Reduced Network Costs**: Data doesn't flow through your infrastructure twice
- **Storage Optimization**: Direct integration with cloud storage pricing

## Security Considerations

### 1. Presigned URL Expiration
```go
"presign_duration": 1800, // 30 minutes - adjust based on your needs
```

### 2. File Size Limits
```go
if req.FileSize > 100*1024*1024 { // 100MB limit
    return nil, fmt.Errorf("file too large")
}
```

### 3. Content Type Validation
```go
allowedTypes := []string{
    "image/jpeg", "image/png", "application/pdf",
    // Add your allowed types
}
```

### 4. Access Control
```go
// Validate user has permission to upload to tenant
if !user.CanUploadTo(req.TenantID) {
    return nil, errors.New("insufficient permissions")
}
```

## Advanced Features

### Progress Tracking
The web client includes real-time upload progress:

```javascript
xhr.upload.onprogress = (event) => {
    if (event.lengthComputable) {
        const percent = Math.round((event.loaded / event.total) * 100);
        updateProgressBar(percent);
    }
};
```

### Error Recovery
```go
func (c *Client) uploadWithRetry(file *os.File, prepareResponse map[string]interface{}) error {
    maxRetries := 3
    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := c.performDirectUpload(file, prepareResponse)
        if err == nil {
            return nil
        }

        if attempt < maxRetries {
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }
    return fmt.Errorf("upload failed after %d attempts", maxRetries)
}
```

### Metadata Synchronization
After upload confirmation, the service syncs metadata from storage:

```go
// Sync actual file metadata from storage backend
_, err = dus.svc.UpdateObjectMetaFromStorage(ctx, objectID)
```

## Production Considerations

### 1. CORS Configuration
For browser clients, configure CORS on your storage backend:

```json
{
  "CORSRules": [{
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["PUT", "POST"],
    "AllowedOrigins": ["https://yourdomain.com"],
    "ExposeHeaders": ["ETag"]
  }]
}
```

### 2. Monitoring
- Track upload success/failure rates
- Monitor presigned URL usage
- Alert on unusual upload patterns

### 3. Cleanup
- Implement cleanup for abandoned uploads
- Remove expired presigned URL records
- Handle partial upload scenarios

### 4. Integration
- Integrate with your authentication system
- Add webhook notifications for upload events
- Implement upload quotas and rate limiting

This example provides a solid foundation for implementing direct client uploads in production environments while maintaining all the benefits of the simple-content library for metadata and content management.