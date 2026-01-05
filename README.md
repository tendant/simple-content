# Simple Content Management System

[![CI](https://github.com/tendant/simple-content/workflows/CI/badge.svg)](https://github.com/tendant/simple-content/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/tendant/simple-content.svg)](https://pkg.go.dev/github.com/tendant/simple-content)

A lightweight content management library for Go with unified APIs that focus on content operations while abstracting storage implementation details.

## Features

- **Unified API**: Single-call upload operations for content and derived content
- **Multi-Backend Storage**: Support for memory, filesystem, and S3-compatible storage
- **Derived Content**: Built-in support for thumbnails, previews, and transcodes
- **Flexible Configuration**: Presets for quick start, builder pattern, or environment variables
- **Clean Architecture**: Library-first design with optional HTTP server

## Quick Start

### Installation

```bash
go get github.com/tendant/simple-content
```

### Basic Usage

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent/presets"
)

// One-line setup for development
svc, cleanup, err := presets.NewDevelopment()
if err != nil {
    log.Fatal(err)
}
defer cleanup()

// Upload content
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      ownerID,
    TenantID:     tenantID,
    Name:         "document.pdf",
    DocumentType: "application/pdf",
    Reader:       fileReader,
    FileName:     "document.pdf",
})

// Download content
reader, err := svc.DownloadContent(ctx, content.ID)
defer reader.Close()
```

## Core APIs

### Content Operations

```go
// Upload original content (one step)
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      ownerID,
    TenantID:     tenantID,
    Name:         "photo.jpg",
    DocumentType: "image/jpeg",
    Reader:       photoReader,
    FileName:     "photo.jpg",
})

// Upload derived content (e.g., thumbnail)
thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       content.ID,
    OwnerID:        ownerID,
    TenantID:       tenantID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         thumbnailReader,
    FileName:       "thumb.jpg",
})

// Get content details with metadata
details, err := svc.GetContentDetails(ctx, content.ID)
```

### Async Workflow (for workers)

For background processing workflows:

```go
// Step 1: Create derived content placeholder
derived, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
    ParentID:       parentID,
    OwnerID:        ownerID,
    TenantID:       tenantID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    InitialStatus:  simplecontent.ContentStatusCreated,
})

// Step 2: Worker processes and generates thumbnail
// ... thumbnail generation logic ...

// Step 3: Upload object for existing content
object, err := svc.UploadObjectForContent(ctx, simplecontent.UploadObjectForContentRequest{
    ContentID: derived.ID,
    Reader:    thumbnailData,
    FileName:  "thumb.jpg",
})

// Step 4: Mark as processed
err = svc.UpdateContentStatus(ctx, derived.ID, simplecontent.ContentStatusProcessed)
```

## Configuration

### 1. Development Preset (Recommended)

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presets"

// One-line setup with in-memory storage
svc, cleanup, err := presets.NewDevelopment()
defer cleanup()
```

### 2. Production with PostgreSQL + S3

```bash
# Environment variables
export DATABASE_TYPE=postgres
export DATABASE_URL="postgresql://user:pass@localhost:5432/db?search_path=content"
export STORAGE_BACKEND=s3
export AWS_S3_BUCKET=my-bucket
export AWS_S3_REGION=us-east-1
```

```go
import "github.com/tendant/simple-content/pkg/simplecontent/config"

// Load from environment
cfg, err := config.Load()
svc, err := config.BuildService(cfg)
```

### 3. Custom Builder

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
    s3backend "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)

// Build custom service
repo := repopg.New(pool)
s3Store, _ := s3backend.New(s3backend.Config{
    Bucket: "my-bucket",
    Region: "us-east-1",
})

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Store),
)
```

## Database Setup

### PostgreSQL with Goose

```bash
# Install goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
export DATABASE_URL="postgresql://user:pass@localhost:5432/db?search_path=content"
goose -dir ./migrations/postgres postgres "$DATABASE_URL" up
```

### Docker Development

```bash
# Start Postgres + MinIO
./scripts/docker-dev.sh start

# Run migrations
./scripts/run-migrations.sh up

# Run application
ENVIRONMENT=development \
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go run ./cmd/server-configured
```

## Key Concepts

### Content vs Objects

- **Content**: Logical entity (document, image, video) with metadata and status
- **Objects**: Physical storage blobs (internal implementation detail)
- **Derived Content**: Generated variants (thumbnails, previews, transcodes)

### Derivation Types

- **DerivationType**: User-facing category (`thumbnail`, `preview`, `transcode`)
- **Variant**: Specific version (`thumbnail_256`, `preview_1080p`, `mp4_720p`)

### Content Status

- `created` - Content record exists, no data uploaded yet
- `uploaded` - Binary data stored (terminal state for originals)
- `processing` - Post-upload processing in progress
- `processed` - Processing completed (terminal state for derived content)
- `failed` - Upload or processing failed

### Storage Backends

| Backend | Use Case | Best For |
|---------|----------|----------|
| **Memory** | Testing, development | Unit tests, demos |
| **Filesystem** | Simple deployments | Single server, local dev |
| **S3/MinIO** | Production | Multi-server, CDN, scalability |

## HTTP API

The optional HTTP server provides RESTful endpoints:

```bash
# Upload content
POST /api/v1/contents
Content-Type: multipart/form-data

# Get content details
GET /api/v1/contents/{id}/details

# Download content
GET /api/v1/contents/{id}/download

# Create derived content
POST /api/v1/contents/{id}/derived
Content-Type: multipart/form-data

# List derived content
GET /api/v1/contents/{id}/derived?derivation_type=thumbnail
```

## Testing

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presets"

func TestUpload(t *testing.T) {
    // One-line test setup
    svc := presets.NewTesting(t)

    content, err := svc.UploadContent(ctx, request)
    require.NoError(t, err)
    assert.NotEmpty(t, content.ID)
}
```

## Documentation

- **[API Reference](./API.md)** - Complete API documentation
- **[Configuration Guide](./CONFIGURATION_GUIDE.md)** - Detailed setup options
- **[CLAUDE.md](./CLAUDE.md)** - AI assistant development guide
- **Examples**: See `examples/` directory

## Environment Variables

**Database:**
- `DATABASE_TYPE` - `memory`, `postgres` (default: `memory`)
- `DATABASE_URL` - PostgreSQL connection string
- `CONTENT_DB_SCHEMA` - Schema name (default: `content`)

**Storage:**
- `STORAGE_BACKEND` - `memory`, `fs`, `s3` (default: `memory`)
- `FS_BASE_PATH` - Filesystem storage path (default: `./data`)
- `AWS_S3_BUCKET` - S3 bucket name
- `AWS_S3_REGION` - S3 region
- `AWS_S3_ENDPOINT` - S3 endpoint (for MinIO)

**Server:**
- `PORT` - HTTP server port (default: `8080`)
- `ENVIRONMENT` - `development`, `production`

## License

MIT License - see LICENSE file for details.
