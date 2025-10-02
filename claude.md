# Project Brief for AI Assistants

This document gives AI coding assistants (Claude, ChatGPT, etc.) the context and conventions to work safely and effectively in this repository.

## Overview

- Language: Go
- Library-first design under `pkg/simplecontent` with a thin HTTP server in `cmd/server-configured`.
- **New Unified API Design**: Content-focused operations that hide storage implementation details
- Goals: clean architecture, pluggable storage/repository backends, strong typing, clear errors, easy testing.

## Core Concepts

- Content: abstraction for a logical piece of content (e.g., a document, image, video). It represents the item as users think about it, with its own metadata and lifecycle status. A content can have multiple associated objects (versions, formats).
- Object: an individual blob stored in a storage backend (memory/fs/s3). Objects belong to a content, have an `object_key`, a `version`, and storage-specific metadata.
- Derived Content: generated content produced from an original (parent) content (e.g., thumbnails, previews, transcodes). It is stored as its own Content row and linked to the parent via the `content_derived` relationship.

## Key Packages

- `pkg/simplecontent` (core library)
  - **Unified Service Interface**: Content-focused operations (`UploadContent`, `UploadDerivedContent`, `GetContentDetails`)
  - **StorageService Interface**: Advanced object operations for presigned uploads and presigned URLs
  - Service implementation (`service.go`, `service_impl.go`)
  - Domain types and typed enums (`types.go`)
    - ContentStatus, ObjectStatus (typed string enums)
    - DerivationVariant (specific)
    - ContentDetails (unified metadata structure)
  - Requests/DTOs (`requests.go`): UploadContentRequest, UploadDerivedContentRequest
  - Interfaces (`interfaces.go`): Service, StorageService, Repository, BlobStore, EventSink, Previewer
  - Errors (`errors.go`): typed sentinel errors for mapping
  - **Object Key Generation** (`objectkey/`): Pluggable key generators for optimal storage performance
  - **URL Strategy System** (`urlstrategy/`): Pluggable URL generation strategies for flexible deployment patterns
  - Storage backends: `storage/memory`, `storage/fs`, `storage/s3`
  - Repositories: `repo/memory`, `repo/postgres` (+ `schema.sql`)
  - Config: `pkg/simplecontent/config` builds a Service from env

- `cmd/server-configured` (HTTP server)
  - Uses `config.LoadServerConfig()` + `BuildService()`
  - Handlers implemented with `chi` and JSON helpers, consistent error mapping

## Important Conventions

- Lowercase keywords: all derivation values are normalized to lowercase.
- Derivation terms:
  - `derivation_type` (user-facing) lives on derived Content (e.g., `thumbnail`, `preview`, `transcode`). It is omitted for originals.
  - `variant` (specific) lives on the `content_derived` relationship. Column is named `variant`. No uniqueness is enforced on `(parent_id, variant)`; choose a canonical record by status/time if needed.
- If only `variant` is provided when creating derived content, the service infers `derivation_type` from the variant prefix.
- Typed enums are used for statuses/variants; struct fields remain strings for wire compatibility.
- Error mapping (server): typed errors → HTTP status codes with structured JSON body `{ "error": { code, message } }`.

### Status Enums

**Content Status** (high-level lifecycle):
- `created` - Content record exists, no data uploaded yet
- `uploading` - Upload in progress (optional intermediate state)
- `uploaded` - Binary data successfully uploaded to storage
- `processing` - Post-upload processing in progress (e.g., validation, indexing)
- `processed` - Processing completed, content ready for use
- `failed` - Upload or processing failed, may need retry
- `archived` - Content archived for long-term storage (future use)
- ~~`deleted`~~ - DEPRECATED: Use `deleted_at` timestamp instead

**Object Status** (detailed processing state):
- `created` - Object placeholder reserved, no binary data yet
- `uploading` - Upload in progress
- `uploaded` - Binary successfully stored in blob storage
- `processing` - Post-upload processing in progress
- `processed` - Processing completed successfully
- `failed` - Processing failed, manual intervention may be required
- ~~`deleted`~~ - DEPRECATED: Use `deleted_at` timestamp instead

**Note:** Derived content uses the same ContentStatus enum as original content (tracked in `content.status`).

### Status-Based Authorization

The system enforces status-based authorization rules to ensure operations are only performed when content is in an appropriate state. These validations are implemented in `status_validation.go` and automatically applied by service methods.

**Download Operations** (Content & Object):
- ✅ **Allowed**: `uploaded`, `processed`, `archived` (content) / `uploaded`, `processed` (object)
- ❌ **Denied**: `created`, `uploading`, `processing`, `failed`
- Error: `ErrContentNotReady` / `ErrObjectNotReady`

**Upload Operations** (Content & Object):
- ✅ **Allowed**: `created`, `failed` (allow retry after failure)
- ❌ **Denied**: `uploading`, `uploaded`, `processing`, `processed`, `archived`
- Error: `ErrInvalidUploadState`

**Create Derived Content** (Parent Status):
- ✅ **Allowed**: `uploaded`, `processed` (parent must have data before creating derivatives)
- ❌ **Denied**: `created`, `uploading`, `processing`, `failed`, `archived`
- Error: `ErrParentNotReady`

**Delete Operations**:
- ✅ **Allowed**: All statuses except `processing`
- ❌ **Denied**: `processing` (prevents data loss during active processing)
- Error: `ErrContentBeingProcessed`
- Note: The validation function supports a `force` parameter for emergency deletions, but the current `DeleteContent()` interface does not expose this

**Implementation Example:**
```go
// Download validation (service_impl.go)
func (s *service) DownloadContent(ctx context.Context, contentID uuid.UUID) (io.ReadCloser, error) {
    content, err := s.repository.GetContent(ctx, contentID)
    if err != nil {
        return nil, &ContentError{ContentID: contentID, Op: "download_get_content", Err: err}
    }

    // Validate content status for download
    contentStatus := ContentStatus(content.Status)
    if ok, statusErr := canDownloadContent(contentStatus); !ok {
        return nil, &ContentError{ContentID: contentID, Op: "download", Err: statusErr}
    }

    // ... proceed with download
}
```

**Error Handling:**
All status validation errors wrap the appropriate sentinel error (`ErrContentNotReady`, `ErrObjectNotReady`, `ErrInvalidUploadState`, `ErrParentNotReady`, `ErrContentBeingProcessed`) and can be checked using `errors.Is()`:

```go
_, err := svc.DownloadContent(ctx, contentID)
if errors.Is(err, simplecontent.ErrContentNotReady) {
    // Handle not-ready case
}
```

**Benefits:**
- Prevents invalid operations (e.g., downloading content that hasn't been uploaded)
- Clear error messages with specific status information
- Consistent behavior across all service methods
- Supports retry workflows (upload retry after failure)

### Status Management Operations

The system provides dedicated methods for updating and querying content/object status. These methods are the **recommended** approach for status management.

**Update Status Methods:**
```go
// Update content status
err := svc.UpdateContentStatus(ctx, contentID, simplecontent.ContentStatusProcessed)
if err != nil {
    // Handle error
}

// Update object status
err := svc.UpdateObjectStatus(ctx, objectID, simplecontent.ObjectStatusUploaded)
```

**Query by Status Methods:**
```go
// Find all content in "processing" state
processing, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusProcessing)
for _, content := range processing {
    // Process each content...
}

// Find all objects ready for download
uploaded, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatusUploaded)

// Get all objects for a specific content
objects, err := svc.GetObjectsByContentID(ctx, contentID)
for _, obj := range objects {
    // Process each object (e.g., backfill, migration, multi-version handling)
}
```

**Features:**
- **Automatic validation**: Status value is validated before update
- **Timestamp updates**: `updated_at` automatically set on status change
- **Event firing**: Status change events fired for observability
- **Type safety**: Uses typed `ContentStatus` and `ObjectStatus` enums
- **Soft delete filtering**: Query methods exclude deleted records

**Comparison with Manual Updates:**

✅ **Recommended** (Dedicated Methods):
```go
// Clean, validated, fires events
err := svc.UpdateContentStatus(ctx, id, simplecontent.ContentStatusUploaded)
```

⚠️ **Alternative** (Manual Update):
```go
// Works but bypasses validation and events
content.Status = string(simplecontent.ContentStatusUploaded)
err := svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{Content: content})
```

**Use Cases:**
- **Background workers**: Query processing content and update status when done
- **Retry workflows**: Find failed content/objects and retry operations
- **Status dashboards**: Count content by status for monitoring
- **Cleanup jobs**: Find archived content for deletion

**Event Integration:**
Status change events are automatically fired through the `EventSink` interface:
```go
// EventSink interface methods
ContentStatusChanged(ctx, contentID, oldStatus, newStatus) error
ObjectStatusChanged(ctx, objectID, oldStatus, newStatus) error
```

Use these events for:
- Logging status transitions
- Triggering webhooks on status changes
- Collecting metrics on processing times
- Building audit trails

### Soft Delete Pattern

**Primary Mechanism:** `deleted_at` timestamp

The system uses the `deleted_at` timestamp field as the **single source of truth** for soft deletion:
- `deleted_at IS NULL` → Record is active
- `deleted_at IS NOT NULL` → Record is soft deleted (timestamp indicates when)

**Status Field Behavior:**
- The `status` field remains at its **last operational state** when a record is deleted
- Example: Content with `status="uploaded"` that is deleted will have `status="uploaded"` and `deleted_at=<timestamp>`
- This preserves information about what state the content was in before deletion
- **DO NOT** set `status="deleted"` - this is deprecated

**Implementation:**
```go
// ✅ CORRECT: Soft delete implementation
func DeleteContent(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE content SET deleted_at = NOW() WHERE id = $1`
    // Status field is NOT changed - it keeps its last value
}

// ❌ INCORRECT: Old pattern (deprecated)
func DeleteContent(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE content SET status = 'deleted', deleted_at = NOW() WHERE id = $1`
    // Don't do this - status='deleted' is deprecated
}
```

**Querying Active Records:**
```go
// Always filter by deleted_at for active records
query := `SELECT * FROM content WHERE deleted_at IS NULL`

// NOT by status
// query := `SELECT * FROM content WHERE status != 'deleted'` // ❌ Wrong
```

**Deprecated Constants:**
- `ContentStatusDeleted` and `ObjectStatusDeleted` are **deprecated** (will be removed in v2.0)
- These constants remain valid for backward compatibility with existing data
- New code should NOT use these constants
- Use `deleted_at` timestamp for all soft delete operations

**Recovery/Undelete:**
```go
// To restore a soft-deleted record
query := `UPDATE content SET deleted_at = NULL WHERE id = $1`
// Status field already contains the correct operational status
```

### Content Ready Status

The `ContentDetails.Ready` field indicates when content and its derived content are ready for use.

**Status Lifecycle by Content Type:**
- **Original Content**: `created` → `uploaded` (terminal state)
  - Original content is source material - once uploaded, it's complete
  - Status "uploaded" means the original binary is stored and accessible
- **Derived Content**: `created` → `processed` (terminal state)
  - Derived content (thumbnails, previews, transcodes) is the OUTPUT of processing
  - Status "processed" means the derivative was generated and is ready to serve
  - Once uploaded, immediately marked "processed" (no intermediate "uploaded" state)

**Ready Semantics:**
- `Ready = true` when:
  - **Original content**: `status = "uploaded"`
  - **Derived content**: `status = "processed"`
- `Ready = false` when:
  - Content `status = "created"` (not yet ready), OR
  - Derived content has any status other than `"processed"`

**Semantic Distinction:**
- **"uploaded"** status = original source material
- **"processed"** status = derived/generated content
- This makes content type clear from status alone

**Examples:**
```go
// Original content - ready when uploaded
original, _ := svc.UploadContent(ctx, uploadReq)
// original.Status = "uploaded"
// original.DerivationType = "" (empty for originals)
details, _ := svc.GetContentDetails(ctx, original.ID)
// details.Ready = true

// Derived content - ready when processed
thumbnail, _ := svc.UploadDerivedContent(ctx, UploadDerivedContentRequest{
    ParentID:       parentID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         thumbReader,
})
// thumbnail.Status = "processed" (NOT "uploaded")
// thumbnail.DerivationType = "thumbnail"
details, _ := svc.GetContentDetails(ctx, thumbnail.ID)
// details.Ready = true

// Parent with derived content - ready when all derivatives are processed
parent, _ := svc.UploadContent(ctx, parentReq)                  // parent.Status = "uploaded"
derived, _ := svc.UploadDerivedContent(ctx, derivedReq)         // derived.Status = "processed"
details, _ := svc.GetContentDetails(ctx, parent.ID)
// details.Ready = true (parent uploaded AND all derivatives processed)
```

**Implementation Notes:**
- **Derived content status is tracked in `content.status`** (same as original content)
- Uses ContentStatus enum for both original and derived content
- **Status semantics differ by content type:**
  - Original content: `"uploaded"` = terminal state (source material ready)
  - Derived content: `"processed"` = terminal state (generated output ready)
- When `UploadDerivedContent()` completes:
  - `content.status` is set to `"processed"` (not `"uploaded"`)
  - This reflects that derived content IS the output of processing
- No duplication: `content_derived` table does NOT have a status column (avoids sync issues)

**Migration for Existing Data:**

If you have existing derived content with `status='uploaded'`, update them to `'processed'`:

```sql
-- Update all derived content from 'uploaded' to 'processed'
UPDATE content
SET status = 'processed', updated_at = NOW()
WHERE derivation_type != ''
  AND derivation_type IS NOT NULL
  AND status = 'uploaded'
  AND deleted_at IS NULL;
```

To verify the migration:
```sql
-- Count by content type and status
SELECT
    CASE WHEN derivation_type = '' OR derivation_type IS NULL THEN 'original' ELSE 'derived' END as type,
    status,
    COUNT(*) as count
FROM content
WHERE deleted_at IS NULL
GROUP BY type, status
ORDER BY type, status;
```

## HTTP API (cmd/server-configured)

Base path: `/api/v1`

### Unified Content API (Recommended)
- Content Operations
  - `POST /contents` create content (can include upload data)
  - `POST /contents/{parentID}/derived` create derived content
  - `GET /contents/{contentID}` get content
  - `PUT /contents/{contentID}` update content (partial)
  - `DELETE /contents/{contentID}` delete content
  - `GET /contents?owner_id=&tenant_id=` list contents

- **Unified Content Details (NEW!)**
  - `GET /contents/{contentID}/details` get all content information (URLs + metadata)
  - `GET /contents/{contentID}/details?upload_access=true` include upload URLs

- Content Data Access
  - `GET /contents/{contentID}/download` download content data directly
  - `POST /contents/{contentID}/upload` upload content data directly

### Legacy Object API (Advanced Users)
- Objects (for StorageService interface users)
  - `POST /contents/{contentID}/objects` create object
  - `GET /objects/{objectID}` get object
  - `DELETE /objects/{objectID}` delete object
  - `GET /contents/{contentID}/objects` list objects by content

- Upload/Download (object-level)
  - `POST /objects/{objectID}/upload` presigned upload to object
  - `GET /objects/{objectID}/download` download from object
  - `GET /objects/{objectID}/upload-url` presigned upload
  - `GET /objects/{objectID}/download-url` presigned download
  - `GET /objects/{objectID}/preview-url` preview URL

## Object Key Generation

The system uses pluggable object key generators for optimal storage performance and organization. Object keys determine where and how files are stored in the underlying storage backends.

### Available Generators

- **GitLikeGenerator** (default): Git-style sharded storage for optimal filesystem performance
  - Original: `originals/objects/{shard}/{objectId}_{filename}`
  - Derived: `derived/{type}/{variant}/objects/{shard}/{objectId}_{filename}`
  - Benefits: Limits directory size, clear content hierarchy, better I/O performance

- **TenantAwareGitLikeGenerator**: Multi-tenant organization with Git-like sharding
  - Structure: `tenants/{tenant}/originals/objects/{shard}/{objectId}_{filename}`
  - Use case: Multi-tenant SaaS applications requiring data isolation

- **LegacyGenerator**: Backwards compatibility with existing flat structure
  - Structure: `C/{contentId}/{objectId}/{filename}`
  - Use case: Migration scenarios or legacy compatibility

- **CustomFuncGenerator**: User-defined key generation logic
  - Allows complete control over key generation strategy
  - Use case: Specialized requirements or complex organizational needs

### Configuration

Set via environment variable or config:

```bash
# Git-like sharding (recommended, default)
OBJECT_KEY_GENERATOR=git-like

# Multi-tenant aware
OBJECT_KEY_GENERATOR=tenant-aware

# High-performance (3-char sharding)
OBJECT_KEY_GENERATOR=high-performance

# Legacy compatibility
OBJECT_KEY_GENERATOR=legacy
```

Or programmatically:

```go
// Configure service with custom key generator
service, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", fsBackend),
    simplecontent.WithObjectKeyGenerator(objectkey.NewGitLikeGenerator()),
)
```

### Key Structure Examples

**Git-like Generator (Recommended):**
- Original: `originals/objects/ab/cd1234ef5678_document.pdf`
- Thumbnail: `derived/thumbnail/256x256/objects/ab/cd1234ef5678_thumb.jpg`
- Preview: `derived/preview/1080p/objects/ab/cd1234ef5678_preview.mp4`

**Tenant-aware Generator:**
- Original: `tenants/acme-corp/originals/objects/ab/cd1234ef5678_contract.pdf`
- Derived: `tenants/acme-corp/derived/thumbnail/small/objects/ab/cd1234ef5678_thumb.jpg`

**Performance Benefits:**
- **Sharding**: Limits directory size to ~256 entries for optimal filesystem performance
- **Organization**: Clear separation between originals and derived content
- **Scalability**: Handles millions of objects efficiently
- **Flexibility**: Easy to customize for specific deployment needs

### Migration from Legacy Keys

The system supports gradual migration:
1. New objects use the configured generator
2. Existing objects retain their current keys
3. No disruption to existing functionality
4. Optional bulk migration tools can be implemented

## URL Strategy System

The system uses pluggable URL strategies to generate download, preview, and upload URLs for content. This allows flexible deployment patterns from simple development setups to high-performance CDN configurations.

### Available Strategies

- **ContentBasedStrategy** (default): Application-routed URLs for maximum control
  - Downloads: `/api/v1/contents/{contentID}/download`
  - Previews: `/api/v1/contents/{contentID}/preview`
  - Uploads: `/api/v1/contents/{contentID}/upload`
  - Benefits: Full control, security, metadata handling, easy debugging

- **CDNStrategy**: Direct CDN URLs for maximum performance with hybrid upload support
  - Downloads: `https://cdn.example.com/{objectKey}` (direct CDN access)
  - Previews: `https://cdn.example.com/{objectKey}` (direct CDN access)
  - Uploads: `https://api.example.com/contents/{contentID}/upload` (application endpoint)
  - Benefits: Maximum download performance, CDN caching, reduced server load

- **StorageDelegatedStrategy**: Backward compatibility with storage backend URL generation
  - Delegates URL generation to the underlying storage backends
  - Use case: Migration scenarios or legacy compatibility
  - Maintains existing storage backend URL patterns

### Configuration

Set via environment variables:

```bash
# Content-based strategy (default, recommended for development)
URL_STRATEGY=content-based
API_BASE_URL=/api/v1

# CDN strategy for production (hybrid approach)
URL_STRATEGY=cdn
CDN_BASE_URL=https://cdn.example.com
UPLOAD_BASE_URL=https://api.example.com

# Storage-delegated (legacy compatibility)
URL_STRATEGY=storage-delegated
```

Or programmatically:

```go
// Content-based strategy
strategy := urlstrategy.NewContentBasedStrategy("/api/v1")

// CDN strategy with hybrid upload support
strategy := urlstrategy.NewCDNStrategyWithUpload(
    "https://cdn.example.com", // Downloads via CDN
    "https://api.example.com", // Uploads via API
)

// Factory method
config := urlstrategy.Config{
    Type:          urlstrategy.StrategyTypeCDN,
    CDNBaseURL:    "https://cdn.example.com",
    UploadBaseURL: "https://api.example.com",
}
strategy, err := urlstrategy.NewURLStrategy(config)

// Configure service with URL strategy
service, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Backend),
    simplecontent.WithURLStrategy(strategy),
)
```

### URL Examples by Strategy

**Content-Based Strategy:**
- Download: `/api/v1/contents/123e4567-e89b-12d3-a456-426614174000/download`
- Preview: `/api/v1/contents/123e4567-e89b-12d3-a456-426614174000/preview`
- Upload: `/api/v1/contents/123e4567-e89b-12d3-a456-426614174000/upload`

**CDN Strategy (Hybrid):**
- Download: `https://cdn.example.com/originals/objects/ab/cd1234ef5678_document.pdf`
- Preview: `https://cdn.example.com/originals/objects/ab/cd1234ef5678_document.pdf`
- Upload: `https://api.example.com/contents/123e4567-e89b-12d3-a456-426614174000/upload`

**With Metadata Enhancement:**
- Download: `https://cdn.example.com/originals/objects/ab/cd1234ef5678_document.pdf?filename=contract.pdf`
- Preview: `https://cdn.example.com/originals/objects/ab/cd1234ef5678_document.pdf?type=application/pdf`

### Deployment Patterns

**Development/Testing:**
```bash
URL_STRATEGY=content-based
API_BASE_URL=/api/v1
```
- Easy debugging through application
- Full request/response control
- Security and access control built-in

**Production with CDN:**
```bash
URL_STRATEGY=cdn
CDN_BASE_URL=https://cdn.example.com
UPLOAD_BASE_URL=https://api.example.com
```
- Maximum download performance via direct CDN access
- Reduced server load for content delivery
- Uploads still controlled through application for security
- Hybrid approach: performance + control

**Enterprise Multi-Region:**
```bash
URL_STRATEGY=cdn
CDN_BASE_URL=https://global-cdn.enterprise.com
UPLOAD_BASE_URL=https://upload-api.enterprise.com
```
- Global CDN distribution for downloads
- Dedicated upload infrastructure
- Geographic load distribution

### Performance Characteristics

| Strategy | Download Performance | Upload Control | Debugging | Security |
|----------|---------------------|----------------|-----------|----------|
| Content-Based | Medium (via app) | Full | Easy | High |
| CDN (Hybrid) | High (direct CDN) | Full | Medium | High |
| Storage-Delegated | Variable | Limited | Hard | Variable |

### Migration and Compatibility

The URL strategy system is designed for zero-downtime deployment:

1. **Existing deployments**: Continue using storage-delegated strategy
2. **New deployments**: Start with content-based for development, CDN for production
3. **Gradual migration**: Switch strategies via configuration without code changes
4. **A/B testing**: Different strategies can be used for different content types

### Advanced Usage

**Custom Strategy Implementation:**
```go
type CustomStrategy struct {
    baseURL string
}

func (s *CustomStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
    // Custom logic here
    return fmt.Sprintf("%s/custom/%s", s.baseURL, contentID), nil
}

// Implement other URLStrategy interface methods...
```

**Environment-Based Factory:**
```go
strategy := urlstrategy.NewRecommendedStrategy(
    os.Getenv("ENVIRONMENT"), // "development", "production"
    os.Getenv("CDN_BASE_URL"),
    os.Getenv("API_BASE_URL"),
)
```

## Error Mapping (server-configured)

- `ErrContentNotFound`, `ErrObjectNotFound` → 404 `not_found`
- `ErrInvalidContentStatus`, `ErrInvalidObjectStatus` → 400 `invalid_status`
- `ErrStorageBackendNotFound` → 400 `storage_backend_not_found`
- `ErrUploadFailed`, `ErrDownloadFailed` → 502 `storage_error`
- Default → 500 `internal_error`

## Local Development

- Build server: `go build ./cmd/server-configured`
- Run server: `ENVIRONMENT=development PORT=8080 go run ./cmd/server-configured`
- Unit tests: `go test ./pkg/simplecontent/...`
- Examples:
  - Basic usage: `go run ./examples/basic`
  - Object key generation: `go run ./examples/objectkey`
- Docker compose (Postgres/MinIO) may be extended; see `REFACTORING_NEXT_STEPS.md`.

### Database migrations (Goose)

- Multi‑DB layout using timestamped filenames:
  - `migrations/postgres/202509090001_schema.sql`
  - `migrations/postgres/202509090002_core_tables.sql`
  - `migrations/mysql/…` (placeholder)
  - `migrations/sqlite/…` (placeholder)
- Postgres uses a dedicated schema named `content` by default (customizable via `search_path`).

Run with goose (examples):

```
# Postgres
goose -dir ./migrations/postgres postgres "$DATABASE_URL" up

# Custom schema: create your schema and set search_path in your session/connection
# or edit the migration to set search_path.
```

Notes:

- The legacy `migrations/*.sql` files are superseded by `migrations/postgres/*` and can be ignored.
- MySQL/SQLite directories are placeholders for future support.

Server config:

- `DATABASE_TYPE=postgres` and `DATABASE_URL` (standard Postgres URI) selects Postgres repository.
- `CONTENT_DB_SCHEMA` (default `content`) controls the schema used; the server sets `search_path` for each connection.

## Coding Guidelines

### API Design Principles
- **Prefer Unified Operations**: Use `UploadContent()` and `UploadDerivedContent()` over multi-step object workflows
- **Content-Focused Design**: Work with content concepts, not storage objects in main APIs
- **Interface Separation**: Use Service interface for most cases, StorageService only for advanced object access
- **Single-Call Operations**: Replace multi-step workflows with unified operations

### Implementation Guidelines
- Keep changes minimal and scoped; respect existing structure and naming.
- Prefer typed enums from `pkg/simplecontent/types.go` for statuses/variants.
- Normalize user-provided categories/variants to lowercase.
- Use and propagate typed errors; don't string-match error messages.
- For new handlers, follow existing JSON helpers and error mapping.
- Avoid adding new external deps unless necessary; use stdlib and existing libs.
- **Object Key Generation**: Use the configured generator; avoid hardcoding key patterns.

### Object Key Best Practices
- **Use Git-like sharding** for new deployments (better filesystem performance)
- **Separate originals from derived content** in key structure for clear organization
- **Configure generators per environment**: legacy for compatibility, git-like for performance
- **Custom generators** for specialized requirements (e.g., compliance, auditing)
- **Avoid hardcoding keys** in business logic; use the pluggable generator system

### URL Strategy Best Practices
- **Use content-based strategy** for development and testing (easier debugging)
- **Use CDN strategy** for production (maximum performance with hybrid upload support)
- **Configure strategies per environment**: content-based for dev, CDN for production
- **Leverage hybrid approach**: CDN for downloads, application for uploads
- **Avoid hardcoding URLs** in clients; use GetContentDetails API for URL retrieval

### When to Use Each Interface
- **Service Interface (Recommended)**: Content operations, unified workflows, server-side applications
- **StorageService Interface (Advanced)**: Presigned uploads, presigned URLs, object-level control

## Extensibility Tips

- New storage backend: implement `BlobStore` in `pkg/simplecontent/storage/<name>`; wire via `config.BuildService()`.
- New repository: implement `Repository` (use pgx or memory patterns) and add to config.
- New derivation variants: add constants of type `DerivationVariant` or accept as lowercase strings from clients.
- **New object key generator**: implement `objectkey.Generator` interface and add to config options.
- **New URL strategy**: implement `urlstrategy.URLStrategy` interface and add to config options.
- Events/Previews: implement `EventSink`/`Previewer` and add via functional options.

### Custom Object Key Generator Example
```go
// Custom generator for compliance/auditing requirements
type ComplianceGenerator struct {
    AuditPrefix string
    Classifier  func(metadata *objectkey.KeyMetadata) string
}

func (g *ComplianceGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *objectkey.KeyMetadata) string {
    classification := g.Classifier(metadata)
    timestamp := time.Now().Format("2006/01/02")
    return fmt.Sprintf("%s/%s/%s/%s/%s",
        g.AuditPrefix, classification, timestamp,
        contentID.String()[:8], objectID.String())
}

// Usage
service, err := simplecontent.New(
    simplecontent.WithObjectKeyGenerator(&ComplianceGenerator{
        AuditPrefix: "audit",
        Classifier: func(m *objectkey.KeyMetadata) string {
            if m != nil && m.DerivationType != "" {
                return "derived"
            }
            return "original"
        },
    }),
)
```

### Custom URL Strategy Example
```go
// Custom strategy for multi-region deployment
type MultiRegionStrategy struct {
    regions map[string]string // region -> base URL
    defaultRegion string
}

func (s *MultiRegionStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
    // Extract region from context or use default
    region := s.extractRegion(ctx)
    baseURL, exists := s.regions[region]
    if !exists {
        baseURL = s.regions[s.defaultRegion]
    }
    return fmt.Sprintf("%s/%s", baseURL, objectKey), nil
}

// Implement other URLStrategy interface methods...

// Usage
strategy := &MultiRegionStrategy{
    regions: map[string]string{
        "us-east": "https://us-east-cdn.example.com",
        "eu-west": "https://eu-west-cdn.example.com",
        "ap-south": "https://ap-south-cdn.example.com",
    },
    defaultRegion: "us-east",
}

service, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("s3", s3Backend),
    simplecontent.WithURLStrategy(strategy),
)
```

## Refactor Roadmap

- See `REFACTORING_NEXT_STEPS.md` for the current plan, milestones, and definition of done.

## API Migration Notes

### Recommended Patterns (Current)
```go
// Unified content upload (1 step)
content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      ownerID,
    TenantID:     tenantID,
    Name:         "Document",
    DocumentType: "text/plain",
    Reader:       dataReader,
    FileName:     "doc.txt",
})

// Derived content creation (1 step)
thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       contentID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         thumbReader,
})

// Get all content info (1 call)
details, err := svc.GetContentDetails(ctx, contentID)
```

### Async Workflow Patterns (Worker-Based Processing)

For scenarios where processing happens asynchronously (e.g., thumbnail generation workers, video transcoding):

```go
// Step 1: Create derived content placeholder (before processing)
derived, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
    ParentID:       parentID,
    OwnerID:        ownerID,
    TenantID:       tenantID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    InitialStatus:  simplecontent.ContentStatusProcessing, // NEW: Set initial status
    Metadata: map[string]interface{}{
        "target_size": "256x256",
        "format":      "jpeg",
    },
})

// Step 2: Worker downloads source and generates thumbnail
sourceData, _ := svc.DownloadContent(ctx, parentID)
thumbnailData := generateThumbnail(sourceData) // expensive operation

// Step 3: Upload object for existing content (NEW)
object, err := svc.UploadObjectForContent(ctx, simplecontent.UploadObjectForContentRequest{
    ContentID: derived.ID,
    Reader:    thumbnailData,
    FileName:  "thumb_256.jpg",
    MimeType:  "image/jpeg",
})

// Step 4: Mark processing complete
err = svc.UpdateContentStatus(ctx, derived.ID, simplecontent.ContentStatusProcessed)
```

**Async Workflow Benefits:**
- **Early visibility**: Placeholder created before processing (UI can show "processing" state)
- **Worker flexibility**: Processing happens separately from content creation
- **Status tracking**: Query content by status to find work or monitor progress
- **Error handling**: Status remains "processing" on failure, worker can retry

**Status Lifecycle for Async:**
```
created → processing → processed ✓
         ↓
        (transient error: retry entire job)
```

**Worker Query Patterns:**
```go
// Find content waiting for processing
pending, _ := svc.GetContentByStatus(ctx, simplecontent.ContentStatusProcessing)

// Error handling: Store error metadata for debugging
svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
    ContentID: derived.ID,
    CustomMetadata: map[string]interface{}{
        "last_error":   err.Error(),
        "error_count":  retryCount,
        "last_attempt": time.Now(),
    },
})
```

### Legacy Patterns (Still Supported)
```go
// Multi-step object workflow (3 steps) - use StorageService interface
content := svc.CreateContent(ctx, req)
storageSvc := svc.(simplecontent.StorageService)
object := storageSvc.CreateObject(ctx, objReq)
storageSvc.UploadObject(ctx, uploadReq)
```

## Safe Ops for AI

- **Prefer unified operations** over legacy object workflows when implementing new features
- Use StorageService interface casting only when advanced object operations are truly needed
- Do not remove legacy packages until the configured server is fully validated.
- Keep API responses stable and documented before broad changes.
- When in doubt, open a small PR with clear rationale and tests.
