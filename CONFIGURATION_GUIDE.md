# Configuration Guide

This guide covers the three ways to configure Simple Content, from simplest to most flexible.

## Quick Navigation

- **[Presets](#configuration-presets)** - One-line setup for development, testing, and production (recommended)
- **[Builder](#configuration-builder)** - Programmatic configuration with functional options
- **[Environment](#environment-variables)** - Environment-based configuration for deployments

---

## Configuration Presets

**The fastest way to get started.** Presets provide sensible defaults for common scenarios.

### Development Preset

Perfect for local development and prototyping.

```go
import "github.com/tendant/simple-content/pkg/simplecontentpresets"

// One line - creates fully configured service
svc, cleanup, err := simplecontentpresets.NewDevelopment()
if err != nil {
    log.Fatal(err)
}
defer cleanup() // Removes ./dev-data/ when done

// Use service immediately
content, err := svc.UploadContent(ctx, request)
```

**Features:**
- ✅ In-memory database (no PostgreSQL required)
- ✅ Filesystem storage at `./dev-data/`
- ✅ Automatic cleanup via `defer cleanup()`
- ✅ Zero configuration required

**Customization:**
```go
svc, cleanup, err := simplecontentpresets.NewDevelopment(
    simplecontentpresets.WithDevStorage("./custom-dir"),
    simplecontentpresets.WithDevPort("3000"),
)
```

**See also:** [examples/preset-development/](./examples/preset-development/)

---

### Testing Preset

Perfect for unit and integration tests.

```go
func TestMyFeature(t *testing.T) {
    // One line - creates isolated service
    svc := simplecontentpresets.NewTesting(t)

    // Use service in tests
    content, err := svc.UploadContent(ctx, request)
    require.NoError(t, err)

    // Cleanup automatic via t.Cleanup()
}
```

**Features:**
- ✅ In-memory database (isolated per test)
- ✅ In-memory storage (blazingly fast)
- ✅ Automatic cleanup via `t.Cleanup()`
- ✅ Parallel test execution support
- ✅ No mocking required

**Parallel Tests:**
```go
func TestParallel(t *testing.T) {
    t.Run("test1", func(t *testing.T) {
        t.Parallel()
        svc := simplecontentpresets.NewTesting(t)
        // Isolated instance
    })

    t.Run("test2", func(t *testing.T) {
        t.Parallel()
        svc := simplecontentpresets.NewTesting(t)
        // Completely separate instance
    })
}
```

**Customization:**
```go
svc := simplecontentpresets.NewTesting(t,
    simplecontentpresets.WithTestFixtures(), // Load sample data
)
```

**See also:** [examples/preset-testing/](./examples/preset-testing/)

---

### Production Preset

*Coming soon* - Environment-based configuration with validation.

```go
// Future API (not yet implemented)
svc, err := simplecontentpresets.NewProduction()
if err != nil {
    log.Fatal(err)
}
```

**Planned Features:**
- PostgreSQL database (from `DATABASE_URL`)
- S3/persistent storage (from `STORAGE_BACKEND`)
- URL strategy configuration (from `URL_STRATEGY`)
- Security best practices
- Validation of required configuration

**Until then:** Use the [Configuration Builder](#configuration-builder) with environment variables.

---

## Configuration Builder

**For when you need full control.** Build services programmatically using functional options.

### Basic Example

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

// Create repository
repo := memoryrepo.New()

// Create storage backend
fsBackend, err := fsstorage.New(fsstorage.Config{
    BaseDir: "./content-data",
})
if err != nil {
    log.Fatal(err)
}

// Build service with options
svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", fsBackend),
)
if err != nil {
    log.Fatal(err)
}
```

### Available Options

#### Repository

```go
// In-memory repository (development/testing)
import memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
repo := memoryrepo.New()

// PostgreSQL repository (production)
import psqlrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
repo, err := psqlrepo.New(psqlrepo.Config{
    DatabaseURL: "postgresql://localhost/simplecontent",
})
```

#### Storage Backends

```go
// In-memory storage (testing)
import memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
storage := memorystorage.New()

// Filesystem storage (development)
import fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
fsBackend, err := fsstorage.New(fsstorage.Config{
    BaseDir: "./content-data",
})

// S3 storage (production)
import s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
s3Backend, err := s3storage.New(s3storage.Config{
    Bucket: "my-content-bucket",
    Region: "us-east-1",
})
```

#### URL Strategy

```go
import "github.com/tendant/simple-content/pkg/simplecontent/urlstrategy"

// Content-based (default) - through application API
strategy := urlstrategy.NewContentBasedStrategy("/api/v1")

// CDN strategy - direct CDN access for downloads
strategy := urlstrategy.NewCDNStrategy("https://cdn.example.com")

// CDN with hybrid upload - CDN for downloads, API for uploads
strategy := urlstrategy.NewCDNStrategyWithUpload(
    "https://cdn.example.com", // Downloads
    "https://api.example.com",  // Uploads
)
```

#### Object Key Generator

```go
import "github.com/tendant/simple-content/pkg/simplecontent/objectkey"

// Git-like sharding (recommended, default)
generator := objectkey.NewGitLikeGenerator()

// Tenant-aware sharding (multi-tenant SaaS)
generator := objectkey.NewTenantAwareGitLikeGenerator()

// High-performance (3-char sharding)
generator := objectkey.NewHighPerformanceGenerator()

// Legacy (backwards compatibility)
generator := objectkey.NewLegacyGenerator()
```

#### Event Sink

```go
// Custom event sink for logging/metrics
type MyEventSink struct {}

func (s *MyEventSink) ContentCreated(ctx context.Context, id uuid.UUID) error {
    log.Printf("Content created: %s", id)
    return nil
}

// ... implement other EventSink interface methods

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", fsBackend),
    simplecontent.WithEventSink(&MyEventSink{}),
)
```

### Complete Example

```go
// Production-like configuration
repo, err := psqlrepo.New(psqlrepo.Config{
    DatabaseURL: os.Getenv("DATABASE_URL"),
})
if err != nil {
    log.Fatal(err)
}

s3Backend, err := s3storage.New(s3storage.Config{
    Bucket: os.Getenv("AWS_S3_BUCKET"),
    Region: os.Getenv("AWS_S3_REGION"),
})
if err != nil {
    log.Fatal(err)
}

strategy := urlstrategy.NewCDNStrategyWithUpload(
    os.Getenv("CDN_BASE_URL"),
    os.Getenv("API_BASE_URL"),
)

keyGenerator := objectkey.NewGitLikeGenerator()

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Backend),
    simplecontent.WithURLStrategy(strategy),
    simplecontent.WithObjectKeyGenerator(keyGenerator),
)
if err != nil {
    log.Fatal(err)
}
```

**See also:** [examples/config-options/](./examples/config-options/)

---

## Environment Variables

**For deployment environments.** Configure via environment variables.

### Using the Config Package

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

### Required Variables

**Database:**
```bash
DATABASE_TYPE=postgres  # postgres, mysql, sqlite, memory
DATABASE_URL=postgresql://user:pass@localhost/simplecontent
```

**Storage:**
```bash
STORAGE_BACKEND=s3  # s3, fs, memory
AWS_S3_BUCKET=my-content-bucket
AWS_S3_REGION=us-east-1
```

### Optional Variables

**URL Strategy:**
```bash
URL_STRATEGY=cdn  # cdn, content-based, storage-delegated
CDN_BASE_URL=https://cdn.example.com
API_BASE_URL=https://api.example.com
UPLOAD_BASE_URL=https://upload.example.com  # Optional separate upload endpoint
```

**Object Key Generator:**
```bash
OBJECT_KEY_GENERATOR=git-like  # git-like, tenant-aware, high-performance, legacy
```

**Server:**
```bash
PORT=8080
API_BASE_PATH=/api/v1
```

### Environment Profiles

**Development:**
```bash
DATABASE_TYPE=memory
STORAGE_BACKEND=fs
FS_BASE_DIR=./dev-data
URL_STRATEGY=content-based
API_BASE_PATH=/api/v1
PORT=8080
```

**Staging:**
```bash
DATABASE_TYPE=postgres
DATABASE_URL=postgresql://staging-db/simplecontent
STORAGE_BACKEND=s3
AWS_S3_BUCKET=staging-content
AWS_S3_REGION=us-east-1
URL_STRATEGY=cdn
CDN_BASE_URL=https://staging-cdn.example.com
API_BASE_URL=https://staging-api.example.com
OBJECT_KEY_GENERATOR=git-like
PORT=8080
```

**Production:**
```bash
DATABASE_TYPE=postgres
DATABASE_URL=postgresql://prod-db/simplecontent
STORAGE_BACKEND=s3
AWS_S3_BUCKET=prod-content
AWS_S3_REGION=us-east-1
URL_STRATEGY=cdn
CDN_BASE_URL=https://cdn.example.com
API_BASE_URL=https://api.example.com
UPLOAD_BASE_URL=https://upload.example.com
OBJECT_KEY_GENERATOR=tenant-aware
PORT=8080
```

---

## Configuration Patterns

### Development Workflow

**Phase 1: Quick Start** (minutes)
```go
// Use development preset
svc, cleanup, err := simplecontentpresets.NewDevelopment()
defer cleanup()
```

**Phase 2: Customization** (hours)
```go
// Add specific backends
repo := memoryrepo.New()
s3Backend, _ := s3storage.New(s3Config)
svc, _ := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Backend),
)
```

**Phase 3: Production** (days)
```go
// Environment-based config
cfg, _ := config.Load()
svc, _ := config.BuildService(cfg)
```

### Testing Strategies

**Unit Tests:**
```go
func TestUpload(t *testing.T) {
    svc := simplecontentpresets.NewTesting(t)
    // Test with real service, in-memory backends
}
```

**Integration Tests:**
```go
func TestWithPostgres(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    repo, _ := psqlrepo.New(psqlrepo.Config{
        DatabaseURL: os.Getenv("TEST_DATABASE_URL"),
    })
    storage := memorystorage.New()

    svc, _ := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("memory", storage),
    )

    // Test with real Postgres, in-memory storage
}
```

**End-to-End Tests:**
```go
func TestE2E(t *testing.T) {
    // Use actual configuration
    cfg, _ := config.Load()
    svc, _ := config.BuildService(cfg)

    // Test full stack
}
```

---

## Migration Paths

### From Manual Configuration to Presets

**Before:**
```go
repo := memoryrepo.New()
storage := memorystorage.New()
svc, _ := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("memory", storage),
)
// Manual cleanup required
```

**After:**
```go
svc, cleanup, _ := simplecontentpresets.NewDevelopment()
defer cleanup() // Automatic cleanup
```

### From Development to Production

**Development:**
```go
svc, cleanup, _ := simplecontentpresets.NewDevelopment()
defer cleanup()
```

**Production:**
```go
cfg, _ := config.Load()
svc, _ := config.BuildService(cfg)
// Configured via environment variables
```

---

## Best Practices

### Choose the Right Method

**Use Presets when:**
- ✅ Getting started with Simple Content
- ✅ Writing tests
- ✅ Local development
- ✅ You want sensible defaults

**Use Builder when:**
- ✅ You need custom backends
- ✅ Integrating with existing infrastructure
- ✅ Testing specific configurations
- ✅ Building libraries/frameworks

**Use Environment when:**
- ✅ Deploying to cloud platforms
- ✅ Docker/Kubernetes deployments
- ✅ Multiple environments (dev/staging/prod)
- ✅ 12-factor app pattern

### Security

**Development:**
- ⚠️ In-memory database (data lost on restart)
- ⚠️ Local filesystem storage (not scalable)
- ⚠️ No authentication (for learning only)

**Production:**
- ✅ PostgreSQL with encrypted connection
- ✅ S3 with IAM roles
- ✅ CDN with signed URLs
- ✅ Secrets in environment variables or secret manager

### Performance

**Development:**
- Fast startup (in-memory database)
- Local I/O (filesystem storage)
- Single instance

**Production:**
- Optimized database queries
- CDN for content delivery
- Object key sharding for filesystem performance
- Horizontal scaling support

---

## Troubleshooting

### Common Issues

**"repository is required" error:**
```go
// Missing repository configuration
svc, err := simplecontent.New()  // ❌ Error

// Fix: Add repository
svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),  // ✅ Required
)
```

**"failed to create filesystem storage" error:**
```bash
# Permission denied on ./dev-data/
chmod +w ./dev-data

# Or use custom directory
svc, _, _ := simplecontentpresets.NewDevelopment(
    simplecontentpresets.WithDevStorage("/tmp/content-data"),
)
```

**"DATABASE_URL is required" error:**
```bash
# Missing environment variable
export DATABASE_URL="postgresql://localhost/simplecontent"

# Or set programmatically
cfg := config.Config{
    DatabaseURL: "postgresql://localhost/simplecontent",
}
svc, _ := config.BuildService(cfg)
```

---

## Next Steps

- **[QUICKSTART.md](./QUICKSTART.md)** - Progressive examples from simple to advanced
- **[Examples Directory](./examples/)** - Working code examples
  - [preset-development](./examples/preset-development/) - Development preset
  - [preset-testing](./examples/preset-testing/) - Testing preset
  - [config-options](./examples/config-options/) - Builder pattern
- **[HOOKS_GUIDE.md](./HOOKS_GUIDE.md)** - Extend functionality with hooks
- **[MIDDLEWARE_GUIDE.md](./MIDDLEWARE_GUIDE.md)** - HTTP request/response handling

---

## Summary

| Method | Setup Time | Flexibility | Best For |
|--------|------------|-------------|----------|
| **Presets** | 1 line | Low | Learning, testing, dev |
| **Builder** | ~10 lines | High | Custom integrations |
| **Environment** | Config file | Medium | Deployments |

**Recommendation:** Start with presets, graduate to builder for custom needs, use environment for production deployments.
