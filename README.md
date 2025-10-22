# Simple Content Management System

[![CI](https://github.com/tendant/simple-content/workflows/CI/badge.svg)](https://github.com/tendant/simple-content/actions)
[![codecov](https://codecov.io/gh/tendant/simple-content/branch/main/graph/badge.svg)](https://codecov.io/gh/tendant/simple-content)
[![Go Report Card](https://goreportcard.com/badge/github.com/tendant/simple-content)](https://goreportcard.com/report/github.com/tendant/simple-content)
[![Go Reference](https://pkg.go.dev/badge/github.com/tendant/simple-content.svg)](https://pkg.go.dev/github.com/tendant/simple-content)

A flexible content management system with simplified APIs that focus on content operations while abstracting storage implementation details.

> **⚠️ DEPRECATION NOTICE**
>
> The legacy packages (`pkg/service`, `pkg/repository`, `pkg/storage`) are **deprecated as of 2025-10-01** and will be removed on **2026-01-01**.
>
> **Please migrate to `pkg/simplecontent`** which provides a better API, improved error handling, and more features.
>
> **Migration Guide:** See [MIGRATION_FROM_LEGACY.md](./MIGRATION_FROM_LEGACY.md) for complete migration instructions.

## Quick Links

- **[5-Minute Quickstart](./QUICKSTART.md)** - Get started with working examples
- **[Configuration Guide](./CONFIGURATION_GUIDE.md)** - Three ways to configure (presets, builder, environment)
- **[Photo Gallery Example](./examples/photo-gallery/)** - Complete application demonstrating real-world usage
- **[Hooks & Extensibility Guide](./HOOKS_GUIDE.md)** - Extend functionality with plugins (service-level)
- **[Middleware Guide](./MIDDLEWARE_GUIDE.md)** - HTTP request/response processing (HTTP-level)
- **[Developer Adoption](./DEVELOPER_ADOPTION.md)** - Implementation summary and roadmap

## Features

### Core Capabilities
- **Unified Content Operations**: Single-call upload/download operations
- **Content-Focused API**: Work with content concepts, not storage objects
- **Multi-Backend Storage**: Support for memory, filesystem, and S3-compatible storage
- **Pluggable URL Strategies**: Flexible URL generation for different deployment patterns
- **Derived Content**: Automatic thumbnail, preview, and transcode generation
- **Flexible Metadata**: Rich metadata support with content details API
- **Clean Architecture**: Library-first design with optional HTTP server

### Developer Experience ✨
- **One-Line Setup**: `NewDevelopment()` and `NewTesting()` presets for instant setup
- **5-Minute Quickstart**: Get started immediately with copy-paste examples
- **Complete Examples**: Real-world photo gallery and middleware applications included
- **Three Configuration Approaches**: Presets, builder pattern, or environment variables
- **Hook System**: 14 lifecycle hooks for service-level extensibility
- **Middleware System**: 14 built-in middleware for HTTP-level extensibility
- **Plugin Architecture**: Build and compose plugins for custom behavior
- **Good Defaults**: Works out-of-the-box with in-memory storage, customizable for production

## Getting Started

### Quick Start (Recommended)

**New to Simple Content?** Start with our [5-Minute Quickstart](./QUICKSTART.md) for immediate hands-on experience!

The quickstart includes:
1. **Basic Setup** (in-memory, < 20 lines of code)
2. **Filesystem Storage** (persistent local storage)
3. **Production Setup** (PostgreSQL + S3)
4. **Derived Content** (automatic thumbnail generation)
5. **Metadata Management** (rich structured data)

### Complete Example Application

See a real-world application in action with the [Photo Gallery Example](./examples/photo-gallery/):

```bash
cd examples/photo-gallery
go run main.go
```

Features demonstrated:
- Photo upload with automatic storage
- Multiple thumbnail sizes (128px, 256px, 512px)
- Rich EXIF-like metadata
- Derived content tracking
- Query and list operations

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

## Configuration

Simple Content provides three ways to configure your service, from simplest to most flexible:

### 1. Configuration Presets (Recommended)

**The fastest way to get started.** One-line setup for common scenarios.

#### Development Preset

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presets"

// One line - creates fully configured service
svc, cleanup, err := presets.NewDevelopment()
if err != nil {
    log.Fatal(err)
}
defer cleanup() // Removes ./dev-data/ when done

// Use service immediately
content, err := svc.UploadContent(ctx, request)
```

Features:
- ✅ In-memory database (no PostgreSQL required)
- ✅ Filesystem storage at `./dev-data/`
- ✅ Automatic cleanup
- ✅ Zero configuration

**See:** [examples/preset-development/](./examples/preset-development/)

#### Testing Preset

```go
func TestMyFeature(t *testing.T) {
    // One line - creates isolated service
    svc := presets.NewTesting(t)

    // Use service in tests
    content, err := svc.UploadContent(ctx, request)
    require.NoError(t, err)

    // Cleanup automatic via t.Cleanup()
}
```

Features:
- ✅ In-memory database (isolated per test)
- ✅ In-memory storage (blazingly fast)
- ✅ Automatic cleanup
- ✅ Parallel test execution support
- ✅ No mocking required

**See:** [examples/preset-testing/](./examples/preset-testing/)

### 2. Configuration Builder

**For custom configurations.** Build services programmatically with functional options.

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

// Create backends
repo := memoryrepo.New()
fsBackend, _ := fsstorage.New(fsstorage.Config{
    BaseDir: "./content-data",
})

// Build service with options
svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", fsBackend),
)
```

**See:** [examples/config-options/](./examples/config-options/)

### 3. Environment Variables

**For production deployments.** Configure via environment variables.

```go
import "github.com/tendant/simple-content/pkg/simplecontent/config"

// Load configuration from environment
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Build service from configuration
svc, err := config.BuildService(cfg)
if err != nil {
    log.Fatal(err)
}
```

Required environment variables:
- `DATABASE_TYPE=postgres` (postgres, mysql, sqlite, memory)
- `DATABASE_URL=postgresql://...`
- `STORAGE_BACKEND=s3` (s3, fs, memory)
- `AWS_S3_BUCKET=my-bucket`
- `AWS_S3_REGION=us-east-1`

**See:** [CONFIGURATION_GUIDE.md](./CONFIGURATION_GUIDE.md) for complete documentation.

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

The system supports multiple pluggable storage backends:

| Backend | Use Case | Presigned URLs | Streaming | Performance | Best For |
|---------|----------|----------------|-----------|-------------|----------|
| **Memory** | Testing, development | ❌ | ✅ | Fastest | Unit tests, local dev |
| **Filesystem** | Simple deployments | ✅ | ✅ | Fast | Single server, development |
| **S3/MinIO** | Production | ✅ | ✅ | Scalable | Production, multi-server, CDN |

**Configuration:**
- **Memory**: No configuration needed, data lost on restart
- **Filesystem**: Set `FS_BASE_PATH` (default: `./data`)
- **S3/MinIO**: Set `AWS_S3_ENDPOINT`, credentials, bucket, region

### Repository Backends

The system supports multiple database backends for metadata storage:

| Backend | Use Case | Transactions | Concurrency | Schema | Best For |
|---------|----------|--------------|-------------|--------|----------|
| **Memory** | Testing | ✅ | Thread-safe (mutex) | N/A | Unit tests, demos |
| **PostgreSQL** | Production | ✅ | Full ACID | Dedicated `content` schema | Production, multi-user |

**PostgreSQL Features:**
- Dedicated schema support (default: `content`, configurable)
- Soft delete with `deleted_at` timestamp
- Optimized indexes for status queries
- Goose-compatible migrations
- Search path configuration

**Configuration:**
- **Memory**: No configuration needed
- **PostgreSQL**: Set `DATABASE_URL` or individual `CONTENT_PG_*` vars

### URL Strategies

The system supports multiple URL generation strategies:

| Strategy | Download | Upload | Best For |
|----------|----------|--------|----------|
| **Content-Based** | Via app `/api/v1/contents/{id}/download` | Via app | Development, debugging, security |
| **CDN (Hybrid)** | Direct CDN URLs | Via app | Production, performance |
| **Storage-Delegated** | Via storage backend | Via storage backend | Legacy compatibility |

**Quick Configuration:**

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

### Featured Examples ⭐
- **[`examples/photo-gallery/`](./examples/photo-gallery/)**: Complete photo management application
  - Demonstrates: Upload, thumbnails, metadata, derived content, queries
  - Run: `cd examples/photo-gallery && go run main.go`
  - Time to run: < 2 minutes

- **[`examples/middleware/`](./examples/middleware/)**: HTTP middleware system demonstration
  - Demonstrates: Request ID, logging, auth, rate limiting, CORS, metrics
  - Run: `cd examples/middleware && go run main.go`
  - Time to run: < 1 minute

### Basic Examples
- **`examples/basic/`**: Simple content upload and download
- **`examples/thumbnail-generation/`**: Image thumbnails with derived content
- **`examples/presigned-upload/`**: Client presigned upload to storage
- **`examples/content-with-derived/`**: Working with derived content
- **`examples/objectkey/`**: Custom object key generation

## Extensibility

### Hook System

Simple Content provides a powerful hook system that lets you extend functionality without modifying core code. Hooks allow you to inject custom logic at 14 different lifecycle points.

**Quick Example:**
```go
hooks := &simplecontent.Hooks{
    AfterContentUpload: []simplecontent.AfterContentUploadHook{
        func(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
            log.Printf("✅ Uploaded %d bytes to content %s", bytes, contentID)
            return nil
        },
    },
}

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", backend),
    simplecontent.WithHooks(hooks),
)
```

**Available Hooks:**
- Content lifecycle: BeforeContentCreate, AfterContentCreate, BeforeContentUpload, AfterContentUpload, etc.
- Derived content: BeforeDerivedCreate, AfterDerivedCreate
- Metadata: BeforeMetadataSet, AfterMetadataSet
- Events: OnStatusChange, OnError

**Common Use Cases:**
- Audit logging
- Metrics & analytics (Prometheus)
- Webhook notifications
- Virus scanning
- Access control
- Custom validation

**Learn More:** See [HOOKS_GUIDE.md](./HOOKS_GUIDE.md) for comprehensive documentation and examples.

### Plugin System

Build composable plugins that provide hooks for specific functionality:

```go
type Plugin interface {
    Name() string
    Version() string
    Hooks() *simplecontent.Hooks
    Initialize(config map[string]interface{}) error
}

// Register multiple plugins
registry := NewPluginRegistry()
registry.Register(&ImageProcessingPlugin{})
registry.Register(&VirusScannerPlugin{})
registry.Register(&AuditLogPlugin{})

svc, _ := simplecontent.New(
    simplecontent.WithHooks(registry.Hooks()),
)
```

## Documentation

### Getting Started
- **[Quickstart Guide](./QUICKSTART.md)**: 5-minute getting started with examples
- **[Photo Gallery Example](./examples/photo-gallery/)**: Complete application walkthrough
- **[Middleware Example](./examples/middleware/)**: HTTP middleware demonstration

### Extensibility & Customization
- **[Hooks & Plugins Guide](./HOOKS_GUIDE.md)**: Service-level extensibility and plugin development
- **[Middleware Guide](./MIDDLEWARE_GUIDE.md)**: HTTP-level request/response processing
- **[Developer Adoption](./DEVELOPER_ADOPTION.md)**: Implementation summary and roadmap

### Advanced Topics
- **[Programmatic Usage](./PROGRAMMATIC_USAGE.md)**: Library usage patterns
- **[Presigned Upload](./PRESIGNED_CLIENT_UPLOAD.md)**: Presigned upload workflows
- **Example READMEs**: Each example has detailed documentation

## License

This project is licensed under the MIT License - see the LICENSE file for details.