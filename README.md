# Simple Content Management System

A flexible content management system with simplified APIs that focus on content operations while abstracting storage implementation details.

## Features

- **Unified Content Operations**: Single-call upload/download operations
- **Content-Focused API**: Work with content concepts, not storage objects
- **Multi-Backend Storage**: Support for memory, filesystem, and S3-compatible storage
- **Pluggable URL Strategies**: Flexible URL generation for different deployment patterns
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

### Local Development with Docker Compose

The easiest way to get started is using Docker Compose for local development:

```bash
# Start Postgres and MinIO services
./scripts/docker-dev.sh start

# Run database migrations
./scripts/run-migrations.sh up

# Create MinIO bucket (optional, for S3 storage)
aws --endpoint-url http://localhost:9000 s3 mb s3://content-bucket

# Run the application
ENVIRONMENT=development \
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
STORAGE_BACKEND=memory \
go run ./cmd/server-configured
```

**Development Services:**
- **Postgres**: `localhost:5433` (user: `content`, password: `contentpass`, db: `simple_content`)
- **MinIO**: `localhost:9000` (console: `localhost:9001`, credentials: `minioadmin/minioadmin`)

**Helper Scripts:**
- `./scripts/docker-dev.sh start|stop|restart|logs|clean|status` - Manage Docker services
- `./scripts/run-migrations.sh up|down|status` - Run database migrations

### Manual Database Setup

If you prefer to manage your own database:

**Postgres Setup:**
- Uses dedicated `content` schema by default (configurable via `CONTENT_DB_SCHEMA`)
- Migrations located in `migrations/postgres/`
- Requires Go migration tool [goose](https://github.com/pressly/goose)

```bash
# Install goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Set your database URL
export DATABASE_URL="postgresql://user:password@localhost:5432/dbname?sslmode=disable&search_path=content"

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

## Environment Variables

### Core Configuration
- `ENVIRONMENT` - Environment name (`development`, `production`) (default: `development`)
- `PORT` - HTTP server port (default: `8080`)
- `HOST` - HTTP server host (default: `0.0.0.0`)

### Database Configuration
- `DATABASE_TYPE` - Database type: `memory` or `postgres` (default: `memory`)
- `DATABASE_URL` - Postgres connection string (format: `postgresql://user:pass@host:port/db?sslmode=disable&search_path=content`)
- `CONTENT_DB_SCHEMA` - Postgres schema name (default: `content`)

**Individual Postgres Settings** (alternative to DATABASE_URL):
- `CONTENT_PG_HOST` - Postgres host
- `CONTENT_PG_PORT` - Postgres port
- `CONTENT_PG_NAME` - Database name
- `CONTENT_PG_USER` - Database user
- `CONTENT_PG_PASSWORD` - Database password

### Storage Configuration
- `STORAGE_BACKEND` - Storage backend: `memory`, `fs`, or `s3` (default: `memory`)

**Filesystem Storage:**
- `FS_BASE_PATH` - Base path for file storage (default: `./data`)

**S3 Storage:**
- `AWS_S3_ENDPOINT` - S3 endpoint URL (for MinIO/compatible services)
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_S3_BUCKET` - S3 bucket name
- `AWS_S3_REGION` - AWS region (default: `us-east-1`)
- `AWS_S3_USE_SSL` - Use SSL for S3 (default: `true`)

### URL Strategy Configuration
- `URL_STRATEGY` - URL generation strategy: `content-based`, `cdn`, or `storage-delegated` (default: `content-based`)
- `API_BASE_URL` - Base URL for content-based strategy (default: `/api/v1`)
- `CDN_BASE_URL` - CDN base URL for cdn strategy
- `UPLOAD_BASE_URL` - Upload base URL for hybrid cdn strategy

### Object Key Generation
- `OBJECT_KEY_GENERATOR` - Key generator: `git-like`, `tenant-aware`, `high-performance`, or `legacy` (default: `git-like`)

## Docker Deployment

### Development Environment

Use the provided helper scripts for local development:

```bash
# Quick start - starts Postgres and MinIO
./scripts/docker-dev.sh start

# View logs
./scripts/docker-dev.sh logs

# Stop services
./scripts/docker-dev.sh stop

# Clean up (removes data volumes)
./scripts/docker-dev.sh clean
```

### Full Stack with API Server

To run the complete stack including the API server:

```bash
# Start all services (Postgres + MinIO + API)
docker-compose up --build

# Or start in detached mode
docker-compose up -d --build
```

This starts:
- **PostgreSQL** on `localhost:5433` (mapped from container port 5432)
- **MinIO** on `localhost:9000` (API) and `localhost:9001` (Console)
- **Content API** server on `localhost:4000`

Access:
- Content API: http://localhost:4000
- MinIO Console: http://localhost:9001 (credentials: `minioadmin/minioadmin`)

**Note:** The docker-compose setup uses a local Postgres instance by default. To use an external database, override the environment variables in docker-compose.yml

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

### URL Strategies
- **Content-Based**: Application-routed URLs for maximum control (default)
- **CDN**: Direct CDN URLs with hybrid upload support for maximum performance
- **Storage-Delegated**: Backward compatibility with storage backend URL generation

#### Quick Configuration

**Development (Default):**
```bash
URL_STRATEGY=content-based
API_BASE_URL=/api/v1
```

**Production with CDN:**
```bash
URL_STRATEGY=cdn
CDN_BASE_URL=https://cdn.example.com
UPLOAD_BASE_URL=https://api.example.com
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

## Testing

### Unit Tests

Run unit tests with the memory backend:

```bash
go test ./pkg/simplecontent/...
```

### Integration Tests

Integration tests require Postgres and MinIO. Use docker-compose for easy setup:

```bash
# Start test services
./scripts/docker-dev.sh start

# Run migrations
./scripts/run-migrations.sh up

# Run integration tests
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go test -tags=integration ./pkg/simplecontent/...

# Clean up
./scripts/docker-dev.sh stop
```

### Running All Tests

```bash
# Start services
./scripts/docker-dev.sh start
./scripts/run-migrations.sh up

# Run all tests (unit + integration)
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go test -tags=integration ./...

# Stop services
./scripts/docker-dev.sh stop
```

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