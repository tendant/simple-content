# Simple Content Management System

[![CI](https://github.com/tendant/simple-content/workflows/CI/badge.svg)](https://github.com/tendant/simple-content/actions)
[![codecov](https://codecov.io/gh/tendant/simple-content/branch/main/graph/badge.svg)](https://codecov.io/gh/tendant/simple-content)
[![Go Report Card](https://goreportcard.com/badge/github.com/tendant/simple-content)](https://goreportcard.com/report/github.com/tendant/simple-content)
[![Go Reference](https://pkg.go.dev/badge/github.com/tendant/simple-content.svg)](https://pkg.go.dev/github.com/tendant/simple-content)

A flexible content management system with simplified APIs that focus on content operations while abstracting storage implementation details.

> **Note:** The current API uses `pkg/simplecontent` which provides a clean, content-focused interface. For migration from legacy packages, see [MIGRATION_FROM_LEGACY.md](./MIGRATION_FROM_LEGACY.md).

## Quick Links

- **[5-Minute Quickstart](./QUICKSTART.md)** - Get started immediately
- **[API Reference](./API.md)** - Complete API documentation
- **[Configuration Guide](./CONFIGURATION_GUIDE.md)** - Presets, builder, environment variables
- **[Presigned URLs Guide](./PRESIGNED_URLS.md)** - Client uploads, downloads, security
- **[Photo Gallery Example](./examples/photo-gallery/)** - Real-world application demo
- **[Hooks Guide](./HOOKS_GUIDE.md)** - Service-level extensibility
- **[Middleware Guide](./MIDDLEWARE_GUIDE.md)** - HTTP-level extensibility
- **[Migration Guide](./MIGRATION_FROM_LEGACY.md)** - Migrate from legacy packages

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

### Prerequisites

- Go 1.21 or higher

### Quick Start

**New to Simple Content?** Start with our [5-Minute Quickstart](./QUICKSTART.md) for immediate hands-on experience with copy-paste examples.

**Want to see it in action?** Check out the [Photo Gallery Example](./examples/photo-gallery/):
```bash
cd examples/photo-gallery && go run main.go
```

### Installation

```bash
# Clone and build
git clone https://github.com/tendant/simple-content.git
cd simple-content
go build -o simple-content ./cmd/server-configured

# Run (starts on port 8080)
./simple-content
```

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

Simple Content provides a clean, content-focused API:

- **Service Interface**: Main API with unified upload operations (`UploadContent`, `UploadDerivedContent`, `GetContentDetails`)
- **StorageService Interface**: Advanced API for direct object operations and presigned URLs
- **HTTP API**: RESTful endpoints under `/api/v1` for content management

**Quick Example:**
```go
// One-step content upload
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    Name:         "document.pdf",
    DocumentType: "application/pdf",
    Reader:       fileReader,
})

// Get all content info (URLs + metadata)
details, err := svc.GetContentDetails(ctx, content.ID)
```

For complete API documentation, see [API.md](./API.md).


## Environment Variables

**Common variables:**
- `DATABASE_TYPE` - `memory`, `postgres` (default: `memory`)
- `DATABASE_URL` - Postgres connection string
- `STORAGE_BACKEND` - `memory`, `fs`, `s3` (default: `memory`)
- `AWS_S3_BUCKET` / `AWS_S3_REGION` - S3 configuration
- `URL_STRATEGY` - `content-based`, `cdn`, `storage-delegated` (default: `content-based`)
- `PORT` - HTTP server port (default: `8080`)

For complete configuration options, see [CONFIGURATION_GUIDE.md](./CONFIGURATION_GUIDE.md).

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

## Migration from Legacy API

The modern API replaces 3-step object workflows with single-call operations:

```go
// Before: 3-step process
content := svc.CreateContent(...)
object := svc.CreateObject(...)
svc.UploadObject(...)

// After: 1-step process
content := svc.UploadContent(...)
```

For complete migration instructions, see [MIGRATION_FROM_LEGACY.md](./MIGRATION_FROM_LEGACY.md).

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

```bash
# Unit tests (memory backend)
go test ./pkg/simplecontent/...

# Integration tests (requires Postgres + MinIO)
./scripts/docker-dev.sh start
./scripts/run-migrations.sh up
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go test -tags=integration ./...
```

## Examples

Complete working examples in `examples/`:

**Featured:**
- **[photo-gallery](./examples/photo-gallery/)** - Complete photo app with thumbnails and metadata
- **[middleware](./examples/middleware/)** - HTTP middleware system demo

**Basic:**
- **basic** - Simple upload/download
- **thumbnail-generation** - Image thumbnails
- **presigned-upload** - Client-side presigned uploads
- **preset-development** / **preset-testing** - Configuration presets

Run any example: `cd examples/<name> && go run main.go`

## Extensibility

### Hook System

Extend functionality at 14 lifecycle points without modifying core code:

```go
hooks := &simplecontent.Hooks{
    AfterContentUpload: []simplecontent.AfterContentUploadHook{
        func(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
            log.Printf("Uploaded %d bytes", bytes)
            return nil
        },
    },
}

svc, _ := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", backend),
    simplecontent.WithHooks(hooks),
)
```

**Use cases:** Audit logging, metrics, webhooks, virus scanning, access control

See [HOOKS_GUIDE.md](./HOOKS_GUIDE.md) and [MIDDLEWARE_GUIDE.md](./MIDDLEWARE_GUIDE.md) for details.

## Documentation

### Getting Started
- **[Quickstart Guide](./QUICKSTART.md)** - 5-minute getting started with examples
- **[Configuration Guide](./CONFIGURATION_GUIDE.md)** - Setup and configuration options
- **[API Reference](./API.md)** - Complete API documentation
- **[Docker Setup](./DOCKER_SETUP.md)** - Local development with Docker

### Core Guides
- **[Presigned URLs](./PRESIGNED_URLS.md)** - Client uploads, downloads, and security
- **[Programmatic Usage](./PROGRAMMATIC_USAGE.md)** - Library usage patterns
- **[Hooks Guide](./HOOKS_GUIDE.md)** - Service-level extensibility
- **[Middleware Guide](./MIDDLEWARE_GUIDE.md)** - HTTP-level extensibility

### Migration & Deployment
- **[Migration from Legacy](./MIGRATION_FROM_LEGACY.md)** - Code/API migration guide
- **[Migration Plan](./MIGRATION_PLAN.md)** - Database/deployment migration

### Examples
- **[Photo Gallery](./examples/photo-gallery/)** - Complete application walkthrough
- **[Middleware Demo](./examples/middleware/)** - HTTP middleware demonstration
- **[All Examples](./examples/)** - Complete examples directory

## License

This project is licensed under the MIT License - see the LICENSE file for details.