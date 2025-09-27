# Simple Content Management System

A flexible content management system with simplified APIs that focus on content operations while abstracting storage implementation details.

## Features

- **Unified Content Operations**: Single-call upload/download operations
- **Content-Focused API**: Work with content concepts, not storage objects
- **Multi-Backend Storage**: Support for memory, filesystem, and S3-compatible storage
- **Derived Content**: Automatic thumbnail, preview, and transcode generation
- **Flexible Metadata**: Rich metadata support with content details API
- **Clean Architecture**: Library-first design with optional HTTP server

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

1. Clone the repository:

```bash
git clone https://github.com/tendant/simple-content.git
cd simple-content
```

2. Build the application:

```bash
go build -o simple-content ./cmd/server-configured
```

3. Run the server:

```bash
./simple-content
```

The server will start on port 8080 by default. You can change the port by setting the `PORT` environment variable.

### Database Setup

- **Postgres**: Uses dedicated `content` schema by default
- **Migrations**: Use goose with migration files in `migrations/postgres/`
- **Schema**: Create schema first with `migrations/manual/000_create_schema.sql`

```bash
# Run migrations
goose -dir ./migrations/postgres postgres "$DATABASE_URL" up
```

## API Overview

### New Simplified Content API

The simple-content library provides two interfaces:

#### Service Interface (Main API)
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

#### StorageService Interface (Advanced API)
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

### HTTP API Endpoints

The configured server exposes a clean REST API under `/api/v1`:

#### Content Operations
- `POST /api/v1/contents` — create content
- `GET /api/v1/contents/{contentID}` — get content
- `PUT /api/v1/contents/{contentID}` — update content
- `DELETE /api/v1/contents/{contentID}` — delete content
- `GET /api/v1/contents?owner_id=&tenant_id=` — list contents

#### Derived Content
- `POST /api/v1/contents/{parentID}/derived` — create derived content

#### Unified Content Details (New!)
- `GET /api/v1/contents/{contentID}/details` — get all content information (URLs + metadata)

#### Legacy Object Operations (Advanced)
- Available for users who need direct object access
- Recommended to use content-focused APIs instead

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

Response:
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

## Docker Deployment

For a complete development environment:

```bash
# Create network
docker network create simple-content-network

# Start all services
docker compose up --build
```

This starts:
- **PostgreSQL** database on port 5432
- **MinIO** object storage on ports 9000 (API) and 9001 (Console)
- **Content API** server on port 8080

Access:
- Content API: http://localhost:8080
- MinIO Console: http://localhost:9001 (admin/minioadmin)
- Health check: http://localhost:8080/health

## Key Concepts

### Content vs Objects
- **Content**: Logical entity that users work with (document, image, video)
- **Objects**: Storage implementation detail (hidden from main API)
- **Derived Content**: Generated content (thumbnails, previews, transcodes)

### Derivation Types and Variants
- **DerivationType**: User-facing category (`thumbnail`, `preview`, `transcode`)
- **Variant**: Specific variant (`thumbnail_256`, `preview_720p`, `mp4_1080p`)

### Storage Backends
- **Memory**: In-memory storage for testing
- **Filesystem**: Local file system storage
- **S3**: Amazon S3 or S3-compatible storage (MinIO)

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

## Architecture

**Clean Architecture Layers:**
- **Domain**: Core entities (Content, Object, DerivedContent)
- **Service**: Business logic with simplified interfaces
- **Repository**: Data persistence abstraction
- **Storage**: Pluggable backend implementations (Memory, FS, S3)
- **API**: HTTP handlers with consistent error handling

**Interface Separation:**
- **Service**: Content-focused operations for most users
- **StorageService**: Object-level operations for advanced use cases

## Examples

See the `examples/` directory for complete working examples:

- **`examples/basic/`**: Simple content upload and download
- **`examples/thumbnail-generation/`**: Image thumbnails with derived content
- **`examples/presigned-upload/`**: Client presigned upload to storage
- **`examples/content-with-derived/`**: Working with derived content

## Documentation

- **`PROGRAMMATIC_USAGE.md`**: Library usage patterns
- **`PRESIGNED_CLIENT_UPLOAD.md`**: Presigned upload workflows
- **`examples/*/README.md`**: Example-specific documentation

## License

This project is licensed under the MIT License - see the LICENSE file for details.