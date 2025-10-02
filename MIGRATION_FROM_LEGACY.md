# Migration Guide: Legacy Packages → pkg/simplecontent

**Deprecation Date:** 2025-10-01
**Removal Date:** 2026-01-01 (3 months)

This guide helps you migrate from the legacy packages (`pkg/service`, `pkg/repository`, `pkg/storage`) to the new unified `pkg/simplecontent` package.

## Quick Migration Checklist

- [ ] Update imports from `pkg/service` → `pkg/simplecontent`
- [ ] Update imports from `pkg/repository` → `pkg/simplecontent/repo`
- [ ] Update imports from `pkg/storage` → `pkg/simplecontent/storage`
- [ ] Replace multi-step workflows with unified operations
- [ ] Update error handling to use typed errors
- [ ] Test with new API
- [ ] Remove legacy package references

## Why Migrate?

The new `pkg/simplecontent` package provides:

✅ **Better API Design**
- Unified operations (single-call upload/download)
- Content-focused instead of object-focused
- Cleaner interfaces with fewer steps

✅ **Improved Error Handling**
- Typed sentinel errors for specific cases
- Better error wrapping and context
- Easier error checking with `errors.Is()`

✅ **More Features**
- Status management operations
- Soft delete support
- Query by status
- Event firing for observability
- URL strategy system (content-based, CDN, storage-delegated)
- Object key generators (git-like, tenant-aware, etc.)

✅ **Better Testing**
- Comprehensive test coverage
- Integration test support
- Docker compose test environment

## Package Mapping

| Legacy Package | New Package | Notes |
|----------------|-------------|-------|
| `pkg/service` | `pkg/simplecontent` | Main service interface |
| `pkg/repository/memory` | `pkg/simplecontent/repo/memory` | Memory repository |
| `pkg/repository/psql` | `pkg/simplecontent/repo/postgres` | Postgres with dedicated schema |
| `pkg/storage/memory` | `pkg/simplecontent/storage/memory` | Memory blob store |
| `pkg/storage/fs` | `pkg/simplecontent/storage/fs` | Filesystem blob store |
| `pkg/storage/s3` | `pkg/simplecontent/storage/s3` | S3 blob store |

## Import Changes

### Before (Legacy)
```go
import (
    "github.com/tendant/simple-content/pkg/service"
    "github.com/tendant/simple-content/pkg/repository/memory"
    "github.com/tendant/simple-content/pkg/repository/psql"
    "github.com/tendant/simple-content/pkg/storage/fs"
    "github.com/tendant/simple-content/pkg/storage/s3"
)
```

### After (New)
```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    postgresrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
    fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
    s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)
```

## API Migration Examples

### Example 1: Service Initialization

#### Before (Legacy)
```go
// Multiple separate services
contentRepo := memory.NewContentRepository()
metadataRepo := memory.NewContentMetadataRepository()
objectRepo := memory.NewObjectRepository()

contentSvc := service.NewContentService(contentRepo, metadataRepo)
objectSvc := service.NewObjectService(objectRepo, storageBackend)
```

#### After (New)
```go
// Single unified service
repo := memoryrepo.New()
memBackend := memorystorage.New()

svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("memory", memBackend),
)
if err != nil {
    log.Fatal(err)
}
```

### Example 2: Content Upload

#### Before (Legacy - 3 steps)
```go
// Step 1: Create content
content := &domain.Content{
    ID:       uuid.New(),
    OwnerID:  ownerID,
    TenantID: tenantID,
    Name:     "Document",
}
err := contentSvc.CreateContent(ctx, content)

// Step 2: Create object
object := &domain.Object{
    ID:          uuid.New(),
    ContentID:   content.ID,
    StorageKey:  "path/to/file",
}
err = objectSvc.CreateObject(ctx, object)

// Step 3: Upload data
err = storageBackend.Upload(ctx, object.StorageKey, dataReader)
```

#### After (New - 1 step)
```go
// Single unified operation
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      ownerID,
    TenantID:     tenantID,
    Name:         "Document",
    DocumentType: "text/plain",
    Reader:       dataReader,
    FileName:     "doc.txt",
    Tags:         []string{"sample"},
})
```

### Example 3: Content Download

#### Before (Legacy)
```go
// Get content
content, err := contentSvc.GetContent(ctx, contentID)

// Get object
objects, err := objectSvc.ListObjectsByContent(ctx, contentID)
object := objects[0]

// Download data
reader, err := storageBackend.Download(ctx, object.StorageKey)
```

#### After (New)
```go
// Single operation
reader, err := svc.DownloadContent(ctx, contentID)
defer reader.Close()
```

### Example 4: Error Handling

#### Before (Legacy)
```go
content, err := contentSvc.GetContent(ctx, id)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Handle not found
    }
    // Other errors
}
```

#### After (New - Typed Errors)
```go
content, err := svc.GetContent(ctx, id)
if err != nil {
    if errors.Is(err, simplecontent.ErrContentNotFound) {
        // Handle not found
    } else if errors.Is(err, simplecontent.ErrInvalidContentStatus) {
        // Handle invalid status
    }
    // Other errors
}
```

### Example 5: Repository Initialization

#### Before (Legacy - Postgres)
```go
db, err := sql.Open("postgres", connString)
if err != nil {
    log.Fatal(err)
}

contentRepo := repository.NewContentRepository(db)
objectRepo := repository.NewObjectRepository(db)
```

#### After (New - Postgres with pgx)
```go
import "github.com/jackc/pgx/v5/pgxpool"

pool, err := pgxpool.New(ctx, connString)
if err != nil {
    log.Fatal(err)
}

repo, err := postgresrepo.New(pool)
if err != nil {
    log.Fatal(err)
}
```

### Example 6: Storage Backend Setup

#### Before (Legacy - S3)
```go
s3Client := s3storage.NewS3Storage(
    endpoint,
    accessKey,
    secretKey,
    bucket,
    region,
)
```

#### After (New - S3 with Options)
```go
s3Backend, err := s3storage.New(
    s3storage.WithEndpoint(endpoint),
    s3storage.WithCredentials(accessKey, secretKey),
    s3storage.WithBucket(bucket),
    s3storage.WithRegion(region),
)
if err != nil {
    log.Fatal(err)
}
```

## New Features You Get for Free

### 1. Status Management

```go
// Update status with validation and events
err := svc.UpdateContentStatus(ctx, contentID, simplecontent.ContentStatusProcessed)

// Query by status
processing, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusProcessing)
uploaded, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatusUploaded)
```

### 2. Soft Delete

```go
// Soft delete (sets deleted_at timestamp)
err := svc.DeleteContent(ctx, contentID)

// Query operations automatically exclude soft-deleted records
content, err := svc.GetContent(ctx, contentID) // Returns ErrContentNotFound if deleted
```

### 3. Derived Content

```go
// Upload thumbnail in one step
thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       originalContentID,
    OwnerID:        ownerID,
    TenantID:       tenantID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         thumbReader,
    FileName:       "thumb.jpg",
})

// List all derived content
derived, err := svc.ListDerivedContent(ctx,
    simplecontent.WithParentID(originalContentID),
    simplecontent.WithDerivationType("thumbnail"),
)
```

### 4. Unified Content Details

```go
// Get everything in one call
details, err := svc.GetContentDetails(ctx, contentID)

// Includes:
// - Download URLs
// - Thumbnail URLs (organized by size)
// - Preview URLs
// - File metadata
// - Status and timestamps
```

### 5. URL Strategies

```go
// Content-based URLs (default - through application)
svc, _ := simplecontent.New(
    simplecontent.WithURLStrategy(
        urlstrategy.NewContentBasedStrategy("/api/v1"),
    ),
)

// CDN URLs (hybrid - downloads via CDN, uploads via app)
svc, _ := simplecontent.New(
    simplecontent.WithURLStrategy(
        urlstrategy.NewCDNStrategyWithUpload(
            "https://cdn.example.com",
            "https://api.example.com",
        ),
    ),
)
```

### 6. Object Key Generators

```go
// Git-like sharding (default - better filesystem performance)
svc, _ := simplecontent.New(
    simplecontent.WithObjectKeyGenerator(
        objectkey.NewGitLikeGenerator(),
    ),
)

// Tenant-aware organization
svc, _ := simplecontent.New(
    simplecontent.WithObjectKeyGenerator(
        objectkey.NewTenantAwareGitLikeGenerator(),
    ),
)
```

## Common Migration Patterns

### Pattern 1: Replace Repository Factories

#### Before
```go
factory := repository.NewRepositoryFactory(db)
contentRepo := factory.ContentRepository()
objectRepo := factory.ObjectRepository()
```

#### After
```go
repo, err := postgresrepo.New(pool)
// repo implements the complete Repository interface
```

### Pattern 2: Replace Storage Backend Registration

#### Before
```go
backendSvc := service.NewStorageBackendService(backendRepo)
err := backendSvc.RegisterBackend(ctx, "s3", s3Backend)
```

#### After
```go
// Register backends during service creation
svc, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Backend),
    simplecontent.WithBlobStore("fs", fsBackend),
)
```

### Pattern 3: Replace Metadata Operations

#### Before
```go
metadata := &domain.ContentMetadata{
    ContentID: contentID,
    Tags:      []string{"tag1", "tag2"},
    FileSize:  1024,
}
err := metadataRepo.SetContentMetadata(ctx, metadata)
```

#### After
```go
// Metadata included in content creation
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    // ... other fields ...
    Tags:     []string{"tag1", "tag2"},
    FileSize: 1024,
})

// Or update metadata separately
err := svc.SetContentMetadata(ctx, &simplecontent.ContentMetadata{
    ContentID: contentID,
    Tags:      []string{"tag1", "tag2"},
    FileSize:  1024,
})
```

## Database Schema Changes

The new package uses a different schema structure:

### Legacy Schema
- Multiple tables without dedicated schema
- Less optimized indexes
- No soft delete support

### New Schema
- Dedicated `content` schema (default, configurable)
- Optimized indexes for status queries
- Soft delete with `deleted_at` timestamp
- `content_derived` relationship table with both `variant` and `derivation_type`

### Migration Steps

1. **Create new schema:**
```sql
CREATE SCHEMA content;
```

2. **Run migrations:**
```bash
goose -dir ./migrations/postgres postgres "postgresql://user:pass@localhost/db?search_path=content" up
```

3. **Migrate data** (if needed):
   - Export from old tables
   - Transform to new structure
   - Import to new schema

4. **Update connection strings:**
```bash
# Add search_path parameter
DATABASE_URL="postgresql://user:pass@localhost/db?sslmode=disable&search_path=content"
```

## Testing Your Migration

### 1. Unit Tests

```go
func TestMigration(t *testing.T) {
    // Setup new service
    repo := memoryrepo.New()
    memBackend := memorystorage.New()

    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("memory", memBackend),
    )
    require.NoError(t, err)

    // Test your operations
    content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
        OwnerID:      ownerID,
        TenantID:     tenantID,
        Name:         "Test",
        DocumentType: "text/plain",
        Reader:       strings.NewReader("Hello"),
    })
    require.NoError(t, err)
    assert.NotNil(t, content)
}
```

### 2. Integration Tests

```bash
# Start test services
./scripts/docker-dev.sh start

# Run migrations
./scripts/run-migrations.sh up

# Run integration tests
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go test -tags=integration ./...

# Clean up
./scripts/docker-dev.sh stop
```

## Troubleshooting

### Issue: "package not found"

**Solution:** Update go.mod and run `go mod tidy`

### Issue: "interface not implemented"

**Solution:** Check that you're using the new interface definitions from `pkg/simplecontent/interfaces.go`

### Issue: "database schema not found"

**Solution:**
1. Create the schema: `CREATE SCHEMA content;`
2. Run migrations: `./scripts/run-migrations.sh up`
3. Add `search_path=content` to connection string

### Issue: "method not found on service"

**Solution:** Some methods may have been renamed or consolidated:
- `CreateContent` + `CreateObject` + `UploadData` → `UploadContent`
- `GetObject` + `DownloadData` → `DownloadContent`
- Consult the API docs in `pkg/simplecontent/service.go`

## Timeline

| Date | Milestone |
|------|-----------|
| 2025-10-01 | Deprecation announced, legacy packages marked deprecated |
| 2025-11-01 | Migration guide published, support available |
| 2025-12-01 | Legacy packages stop receiving updates |
| 2026-01-01 | Legacy packages removed from codebase |

## Getting Help

- **Documentation:** See [CLAUDE.md](./CLAUDE.md) for complete API documentation
- **Examples:** Check `examples/` directory for working code samples
- **Issues:** Report migration issues on GitHub
- **Questions:** Open a discussion on GitHub Discussions

## Migration Checklist Template

Use this checklist for your migration:

```markdown
## Migration Checklist for [Your Project]

- [ ] Review MIGRATION_FROM_LEGACY.md
- [ ] Update dependencies in go.mod
- [ ] Replace imports (service → simplecontent)
- [ ] Replace imports (repository → repo)
- [ ] Replace imports (storage → simplecontent/storage)
- [ ] Update service initialization
- [ ] Replace multi-step workflows
- [ ] Update error handling
- [ ] Test with memory backend
- [ ] Setup docker-compose for integration tests
- [ ] Run integration tests
- [ ] Update CI/CD pipelines
- [ ] Deploy to staging
- [ ] Monitor for errors
- [ ] Deploy to production
- [ ] Remove legacy package references
- [ ] Celebrate! 🎉
```

## Example: Complete Migration

Here's a complete before/after example of a typical application:

### Before (Legacy)

```go
package main

import (
    "context"
    "log"
    "github.com/tendant/simple-content/pkg/service"
    "github.com/tendant/simple-content/pkg/repository/memory"
    "github.com/tendant/simple-content/pkg/storage/fs"
)

func main() {
    // Multiple separate components
    contentRepo := memory.NewContentRepository()
    metadataRepo := memory.NewContentMetadataRepository()
    objectRepo := memory.NewObjectRepository()

    fsBackend := fs.NewFileSystemStorage("/data")

    contentSvc := service.NewContentService(contentRepo, metadataRepo)
    objectSvc := service.NewObjectService(objectRepo, fsBackend)

    // Multi-step upload
    ctx := context.Background()
    content := &domain.Content{
        ID:       uuid.New(),
        OwnerID:  ownerID,
        Name:     "Document",
    }

    if err := contentSvc.CreateContent(ctx, content); err != nil {
        log.Fatal(err)
    }

    object := &domain.Object{
        ID:         uuid.New(),
        ContentID:  content.ID,
        StorageKey: "doc.txt",
    }

    if err := objectSvc.CreateObject(ctx, object); err != nil {
        log.Fatal(err)
    }

    if err := fsBackend.Upload(ctx, object.StorageKey, dataReader); err != nil {
        log.Fatal(err)
    }
}
```

### After (New)

```go
package main

import (
    "context"
    "log"
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

func main() {
    // Unified service initialization
    repo := memoryrepo.New()
    fsBackend, err := fsstorage.New(fsstorage.WithBasePath("/data"))
    if err != nil {
        log.Fatal(err)
    }

    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("fs", fsBackend),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Single-step upload
    ctx := context.Background()
    content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
        OwnerID:      ownerID,
        TenantID:     tenantID,
        Name:         "Document",
        DocumentType: "text/plain",
        Reader:       dataReader,
        FileName:     "doc.txt",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Uploaded content: %s", content.ID)
}
```

## Summary

The migration to `pkg/simplecontent` provides:
- **Simpler API** with fewer steps
- **Better error handling** with typed errors
- **More features** (status management, soft delete, events)
- **Better testing** with docker-compose integration
- **Future-proof** architecture

Start migrating today! The 3-month timeline gives you plenty of time to migrate at your own pace.
