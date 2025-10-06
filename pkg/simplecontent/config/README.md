# Configuration Package

The `config` package provides flexible configuration options for the simple-content server, supporting both **programmatic** (functional options) and **environment variable** configuration patterns.

## Quick Start

### Environment Variables (12-Factor App Style)

```bash
# Basic configuration
export PORT="8080"
export ENVIRONMENT="production"
export DATABASE_TYPE="postgres"
export DATABASE_URL="postgresql://user:pass@localhost/db"

# Filesystem storage
export FS_BASE_DIR="./data/storage"
export FS_URL_PREFIX="http://localhost:8080/api/v1"
export FS_SIGNATURE_SECRET_KEY="$(openssl rand -hex 32)"
export DEFAULT_STORAGE_BACKEND="fs"

# URL strategy
export URL_STRATEGY="storage-delegated"

# Run server
./server-configured
```

### Programmatic Configuration

```go
package main

import (
    "log"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

func main() {
    cfg, err := config.Load(
        config.WithPort("8080"),
        config.WithEnvironment("production"),
        config.WithDatabase("postgres", "postgresql://user:pass@localhost/db"),
        config.WithFilesystemStorage("fs", "./data/storage", "/api/v1", "secret-key"),
        config.WithDefaultStorage("fs"),
        config.WithStorageDelegatedURLs(),
    )
    if err != nil {
        log.Fatal(err)
    }

    svc, err := cfg.BuildService()
    if err != nil {
        log.Fatal(err)
    }

    // Use service...
}
```

## Configuration Patterns

### Pattern 1: Pure Environment Variables

Best for containerized deployments (Docker, Kubernetes):

```bash
# .env file
PORT=8080
ENVIRONMENT=production
DATABASE_TYPE=postgres
DATABASE_URL=postgresql://localhost/mydb
FS_BASE_DIR=/var/data/storage
DEFAULT_STORAGE_BACKEND=fs
```

```go
cfg, err := config.LoadServerConfig() // Uses WithEnv("") internally
```

### Pattern 2: Pure Programmatic

Best for libraries, testing, and embedded usage:

```go
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithDatabase("memory", ""),
    config.WithMemoryStorage(""),
    config.WithDefaultStorage("memory"),
    config.WithContentBasedURLs("/api/v1"),
)
```

### Pattern 3: Mixed (Programmatic + Environment Overrides)

Best for development with production parity:

```go
cfg, err := config.Load(
    // Set sensible defaults programmatically
    config.WithPort("8080"),
    config.WithDatabase("postgres", "postgresql://localhost/dev"),
    config.WithFilesystemStorage("fs", "./data", "/api/v1", "dev-secret"),
    config.WithDefaultStorage("fs"),

    // Allow environment variables to override
    config.WithEnv(""),
)
```

## Available Options

### Server Options

#### WithPort
```go
config.WithPort("8080")
```
Sets the HTTP server port.

#### WithEnvironment
```go
config.WithEnvironment("production")  // or "development", "testing"
```
Sets the runtime environment.

### Database Options

#### WithDatabase
```go
// Memory (for testing)
config.WithDatabase("memory", "")

// PostgreSQL
config.WithDatabase("postgres", "postgresql://user:pass@host/db")
```
Configures the database backend.

#### WithDatabaseSchema
```go
config.WithDatabaseSchema("content")  // Default schema for Postgres
```
Sets the PostgreSQL schema name.

### Storage Backend Options

#### WithMemoryStorage
```go
config.WithMemoryStorage("")  // Name defaults to "memory"
config.WithMemoryStorage("test-storage")  // Custom name
```
Adds an in-memory storage backend (useful for testing).

#### WithFilesystemStorage
```go
config.WithFilesystemStorage(
    "fs",                    // Backend name
    "./data/storage",        // Base directory
    "/api/v1",              // URL prefix for presigned URLs
    "your-secret-key",      // HMAC secret key (empty = unsigned URLs)
)
```
Adds a filesystem storage backend.

**Full configuration:**
```go
config.WithFilesystemStorageFull(
    "fs",                    // Backend name
    "./data/storage",        // Base directory
    "/api/v1",              // URL prefix
    "secret-key",           // HMAC secret
    1800,                   // Presigned URL expiry (seconds)
)
```

#### WithS3Storage
```go
config.WithS3Storage(
    "s3",           // Backend name
    "my-bucket",    // S3 bucket
    "us-west-2",    // AWS region
)
```
Adds an S3 storage backend.

**With credentials:**
```go
config.WithS3Storage("s3", "my-bucket", "us-west-2"),
config.WithS3Credentials("s3", "AKIAIOSFODNN7EXAMPLE", "wJalrXUt..."),
```

**With custom endpoint (MinIO, LocalStack):**
```go
config.WithS3Storage("s3", "my-bucket", "us-east-1"),
config.WithS3Endpoint(
    "s3",
    "http://localhost:9000",  // MinIO endpoint
    false,                    // useSSL
    true,                     // usePathStyle (required for MinIO)
),
```

**Full configuration:**
```go
config.WithS3StorageFull(
    "s3",                            // Backend name
    "my-bucket",                     // Bucket
    "us-west-2",                     // Region
    "AKIAIOSFODNN7EXAMPLE",         // Access key
    "wJalrXUt...",                  // Secret key
    "http://localhost:9000",         // Endpoint (MinIO)
    false,                           // useSSL
    true,                            // usePathStyle
)
```

#### WithDefaultStorage
```go
config.WithDefaultStorage("fs")  // Use filesystem as default
```
Sets which storage backend to use by default.

### URL Strategy Options

#### WithContentBasedURLs
```go
config.WithContentBasedURLs("/api/v1")
```
Routes all upload/download requests through the application server:
- Upload: `/api/v1/contents/{id}/upload`
- Download: `/api/v1/contents/{id}/download`

**Use when:** Simple development setup, full control over access.

#### WithStorageDelegatedURLs
```go
config.WithStorageDelegatedURLs()
```
Delegates URL generation to storage backends (presigned URLs):
- Upload: `/api/v1/upload/{objectKey}?signature=...&expires=...`
- Download: `/api/v1/download/{objectKey}?signature=...&expires=...`

**Use when:** Using filesystem or S3 presigned URLs with HMAC authentication.

#### WithCDNURLs
```go
config.WithCDNURLs(
    "https://cdn.example.com",      // CDN base URL (for downloads)
    "https://api.example.com",      // Upload base URL
)
```
Hybrid strategy: CDN for downloads, application for uploads.

**Use when:** Production with CDN in front of storage.

### Advanced Options

#### WithObjectKeyGenerator
```go
config.WithObjectKeyGenerator("git-like")  // Default, recommended
config.WithObjectKeyGenerator("tenant-aware")
config.WithObjectKeyGenerator("high-performance")
config.WithObjectKeyGenerator("legacy")
```
Sets the object key generation strategy.

#### WithEventLogging
```go
config.WithEventLogging(true)   // Enable
config.WithEventLogging(false)  // Disable
```
Enables or disables event logging.

#### WithPreviews
```go
config.WithPreviews(true)
```
Enables or disables preview generation.

#### WithAdminAPI
```go
config.WithAdminAPI(true)
```
Enables or disables admin API endpoints.

## Complete Examples

### Development Setup

```go
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithEnvironment("development"),
    config.WithDatabase("memory", ""),
    config.WithFilesystemStorage("fs", "./data/storage", "/api/v1", "dev-secret"),
    config.WithDefaultStorage("fs"),
    config.WithContentBasedURLs("/api/v1"),
    config.WithObjectKeyGenerator("git-like"),
    config.WithEventLogging(true),
    config.WithPreviews(true),
    config.WithAdminAPI(true),
)
```

### Production with PostgreSQL and Filesystem

```go
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithEnvironment("production"),
    config.WithDatabase("postgres", os.Getenv("DATABASE_URL")),
    config.WithDatabaseSchema("content"),
    config.WithFilesystemStorageFull(
        "fs",
        "/var/data/storage",
        "https://api.example.com",
        os.Getenv("FS_SECRET_KEY"),
        1800,  // 30 minutes
    ),
    config.WithDefaultStorage("fs"),
    config.WithStorageDelegatedURLs(),
    config.WithObjectKeyGenerator("git-like"),
    config.WithEventLogging(true),
    config.WithPreviews(true),
    config.WithAdminAPI(false),
)
```

### Production with S3

```go
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithEnvironment("production"),
    config.WithDatabase("postgres", os.Getenv("DATABASE_URL")),
    config.WithS3StorageFull(
        "s3",
        os.Getenv("S3_BUCKET"),
        os.Getenv("AWS_REGION"),
        os.Getenv("AWS_ACCESS_KEY_ID"),
        os.Getenv("AWS_SECRET_ACCESS_KEY"),
        "",     // No custom endpoint
        true,   // useSSL
        false,  // usePathStyle
    ),
    config.WithDefaultStorage("s3"),
    config.WithCDNURLs(
        "https://cdn.example.com",
        "https://api.example.com",
    ),
    config.WithObjectKeyGenerator("git-like"),
)
```

### Testing Configuration

```go
cfg, err := config.Load(
    config.WithPort("0"),  // Random port
    config.WithEnvironment("testing"),
    config.WithDatabase("memory", ""),
    config.WithMemoryStorage(""),
    config.WithDefaultStorage("memory"),
    config.WithContentBasedURLs("/api/v1"),
    config.WithEventLogging(false),
    config.WithPreviews(false),
)
```

## Environment Variable Reference

### Server
- `PORT` - Server port (default: "8080")
- `ENVIRONMENT` - Runtime environment (default: "development")

### Database
- `DATABASE_TYPE` - Database backend: "memory", "postgres" (default: "memory")
- `DATABASE_URL` - Database connection URL (required for postgres)
- `DATABASE_SCHEMA` - PostgreSQL schema name (default: "content")

### Filesystem Storage
- `FS_BASE_DIR` - Base directory for files (required to enable FS backend)
- `FS_URL_PREFIX` - URL prefix for presigned URLs (optional)
- `FS_SIGNATURE_SECRET_KEY` - HMAC secret key for signed URLs (optional)
- `FS_PRESIGN_EXPIRES_SECONDS` - Presigned URL expiry in seconds (default: 3600)

### S3 Storage
- `S3_BUCKET` - S3 bucket name (required to enable S3 backend)
- `S3_REGION` - AWS region (default: "us-east-1")
- `S3_ACCESS_KEY_ID` - AWS access key ID
- `S3_SECRET_ACCESS_KEY` - AWS secret access key
- `S3_ENDPOINT` - Custom S3 endpoint (for MinIO, LocalStack)
- `S3_USE_SSL` - Use SSL for S3 connections (default: true)
- `S3_USE_PATH_STYLE` - Use path-style URLs (required for MinIO)
- `S3_PRESIGN_DURATION` - Presigned URL duration in seconds (default: 3600)

### URL Strategy
- `URL_STRATEGY` - URL generation strategy: "content-based", "cdn", "storage-delegated" (default: "content-based")
- `CDN_BASE_URL` - CDN base URL (required for CDN strategy)
- `UPLOAD_BASE_URL` - Upload base URL (for CDN hybrid mode)
- `API_BASE_URL` - API base URL (for content-based strategy, default: "/api/v1")

### Advanced
- `DEFAULT_STORAGE_BACKEND` - Default storage backend name (default: "memory")
- `OBJECT_KEY_GENERATOR` - Object key generator: "git-like", "tenant-aware", "legacy" (default: "git-like")
- `ENABLE_EVENT_LOGGING` - Enable event logging (default: true)
- `ENABLE_PREVIEWS` - Enable preview generation (default: true)
- `ENABLE_ADMIN_API` - Enable admin API endpoints (default: false)

## Best Practices

### Use Environment Variables for Secrets

```go
// ✅ Good - secrets from environment
cfg, err := config.Load(
    config.WithDatabase("postgres", os.Getenv("DATABASE_URL")),
    config.WithS3Credentials("s3",
        os.Getenv("AWS_ACCESS_KEY_ID"),
        os.Getenv("AWS_SECRET_ACCESS_KEY"),
    ),
)

// ❌ Bad - secrets in code
cfg, err := config.Load(
    config.WithDatabase("postgres", "postgresql://user:pass@localhost/db"),
)
```

### Layer Configuration Sources

```go
// 1. Start with sensible defaults
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithEnvironment("development"),

    // 2. Apply application-specific config
    config.WithDatabase("postgres", "postgresql://localhost/dev"),
    config.WithFilesystemStorage("fs", "./data", "/api/v1", ""),

    // 3. Allow environment overrides
    config.WithEnv(""),
)
```

### Validate Configuration Early

```go
cfg, err := config.Load(/* options */)
if err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}

// Validate database connection
if cfg.DatabaseType == "postgres" {
    if err := config.PingPostgres(cfg.DatabaseURL, cfg.DBSchema); err != nil {
        log.Fatalf("Database connection failed: %v", err)
    }
}
```

## Migration from Environment-Only Configuration

If you're currently using only environment variables:

```go
// Before (environment-only)
cfg, err := config.LoadServerConfig()

// After (same behavior, more explicit)
cfg, err := config.Load(config.WithEnv(""))

// Or with programmatic defaults + env overrides
cfg, err := config.Load(
    config.WithPort("8080"),
    config.WithDatabase("postgres", "postgresql://localhost/dev"),
    config.WithEnv(""),  // Env vars override programmatic config
)
```

No breaking changes - existing environment variable configuration continues to work!
