# API Reference

This document provides detailed API documentation for Simple Content Management System.

## Service Interface (Main API)

Content-focused operations that hide storage implementation details:

```go
type Service interface {
    // Unified upload operations
    UploadContent(ctx, UploadContentRequest) (*Content, error)
    UploadDerivedContent(ctx, UploadDerivedContentRequest) (*Content, error)

    // Content management
    CreateContent(ctx, CreateContentRequest) (*Content, error)
    GetContent(ctx, uuid.UUID) (*Content, error)
    UpdateContent(ctx, UpdateContentRequest) error
    DeleteContent(ctx, uuid.UUID) error
    ListContent(ctx, ListContentRequest) ([]*Content, error)

    // Content data access
    DownloadContent(ctx, contentID) (io.ReadCloser, error)

    // Derived content operations
    CreateDerivedContent(ctx, CreateDerivedContentRequest) (*Content, error)
    ListDerivedContent(ctx, ...ListDerivedContentOption) ([]*DerivedContent, error)

    // Unified details API (replaces separate metadata/URLs)
    GetContentDetails(ctx, contentID, ...ContentDetailsOption) (*ContentDetails, error)
}
```

## StorageService Interface (Advanced API)

For advanced users who need direct object operations:

```go
type StorageService interface {
    // Object operations (internal use)
    CreateObject(ctx, CreateObjectRequest) (*Object, error)
    GetObject(ctx, uuid.UUID) (*Object, error)
    UploadObject(ctx, UploadObjectRequest) error
    DownloadObject(ctx, objectID) (io.ReadCloser, error)
    GetUploadURL(ctx, objectID) (string, error)
    GetDownloadURL(ctx, objectID) (string, error)
    // ... other object operations
}
```

## HTTP API Endpoints

Base path: `/api/v1`

### Content Operations

#### Create Content
```
POST /api/v1/contents
```

Request body:
```json
{
  "owner_id": "00000000-0000-0000-0000-000000000001",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "name": "My Document",
  "description": "Sample document",
  "document_type": "text/plain",
  "tags": ["sample", "document"]
}
```

#### Get Content
```
GET /api/v1/contents/{contentID}
```

#### Update Content
```
PUT /api/v1/contents/{contentID}
```

#### Delete Content
```
DELETE /api/v1/contents/{contentID}
```

#### List Contents
```
GET /api/v1/contents?owner_id=&tenant_id=
```

### Derived Content

#### Create Derived Content
```
POST /api/v1/contents/{parentID}/derived
```

### Unified Content Details

#### Get Content Details
```
GET /api/v1/contents/{contentID}/details
```

Returns all content information (URLs + metadata):

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "download": "https://storage.example.com/download/...",
  "thumbnail": "https://storage.example.com/thumb/256/...",
  "thumbnails": {
    "128": "https://storage.example.com/thumb/128/...",
    "256": "https://storage.example.com/thumb/256/...",
    "512": "https://storage.example.com/thumb/512/..."
  },
  "file_name": "document.pdf",
  "file_size": 1024576,
  "mime_type": "application/pdf",
  "tags": ["document", "sample"],
  "ready": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

#### Get Content Details with Upload Access
```
GET /api/v1/contents/{contentID}/details?upload_access=true
```

### Content Data Access

#### Download Content
```
GET /api/v1/contents/{contentID}/download
```

#### Upload Content Data
```
POST /api/v1/contents/{contentID}/upload
```

### Legacy Object Operations (Advanced)

Available for users who need direct object access. Recommended to use content-focused APIs instead.

#### Create Object
```
POST /api/v1/contents/{contentID}/objects
```

#### Get Object
```
GET /api/v1/objects/{objectID}
```

#### Delete Object
```
DELETE /api/v1/objects/{objectID}
```

#### List Objects by Content
```
GET /api/v1/contents/{contentID}/objects
```

#### Upload to Object
```
POST /api/v1/objects/{objectID}/upload
```

#### Download from Object
```
GET /api/v1/objects/{objectID}/download
```

#### Get Presigned Upload URL
```
GET /api/v1/objects/{objectID}/upload-url
```

#### Get Presigned Download URL
```
GET /api/v1/objects/{objectID}/download-url
```

#### Get Preview URL
```
GET /api/v1/objects/{objectID}/preview-url
```

## Usage Examples

### Programmatic Usage (Library)

#### Simple Content Upload

```go
// Old way (3 steps):
// content := svc.CreateContent(...)
// object := svc.CreateObject(...)
// svc.UploadObject(...)

// New way (1 step):
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      ownerID,
    TenantID:     tenantID,
    Name:         "My Document",
    DocumentType: "text/plain",
    Reader:       strings.NewReader("Hello, World!"),
    FileName:     "hello.txt",
    Tags:         []string{"sample", "text"},
})
```

#### Thumbnail Generation

```go
// Upload derived content (thumbnail)
thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       originalContentID,
    OwnerID:        ownerID,
    TenantID:       tenantID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         bytes.NewReader(thumbnailData),
    FileName:       "thumb_256.jpg",
    Tags:           []string{"thumbnail", "256px"},
})
```

#### Get All Content Information

```go
// Get everything in one call
details, err := svc.GetContentDetails(ctx, contentID)

// Includes:
// - Download URLs
// - Thumbnail URLs (organized by size)
// - Preview URLs
// - File metadata (name, size, type, tags)
// - Status and timestamps
```

#### Download Content

```go
// Download content data directly
reader, err := svc.DownloadContent(ctx, contentID)
defer reader.Close()

data, err := io.ReadAll(reader)
```

### HTTP API Usage

#### Upload Content with Metadata

```bash
# Create and upload content in one call
curl -X POST http://localhost:8080/api/v1/contents \
  -H "Content-Type: application/json" \
  -d '{
    "owner_id": "00000000-0000-0000-0000-000000000001",
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "name": "My Document",
    "description": "Sample document",
    "document_type": "text/plain",
    "tags": ["sample", "document"]
  }'
```

#### Get Content Details

```bash
# Get all content information (URLs + metadata)
curl -X GET http://localhost:8080/api/v1/contents/{contentID}/details
```

## Migration from Old API

### Before (Object-based workflow):
```go
// 3-step process
content := svc.CreateContent(...)
object := svc.CreateObject(...)
svc.UploadObject(...)
```

### After (Content-focused workflow):
```go
// 1-step process
content := svc.UploadContent(...)
```

### Deprecated Operations:
- Direct object manipulation
- Separate metadata/URL endpoints
- Multi-step upload workflows

### New Recommended Operations:
- `UploadContent()` for content with data
- `UploadDerivedContent()` for thumbnails/previews
- `GetContentDetails()` for all content information
- `DownloadContent()` for data access
