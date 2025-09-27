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

### When to Use Each Interface
- **Service Interface (Recommended)**: Content operations, unified workflows, server-side applications
- **StorageService Interface (Advanced)**: Presigned uploads, presigned URLs, object-level control

## Extensibility Tips

- New storage backend: implement `BlobStore` in `pkg/simplecontent/storage/<name>`; wire via `config.BuildService()`.
- New repository: implement `Repository` (use pgx or memory patterns) and add to config.
- New derivation variants: add constants of type `DerivationVariant` or accept as lowercase strings from clients.
- **New object key generator**: implement `objectkey.Generator` interface and add to config options.
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
