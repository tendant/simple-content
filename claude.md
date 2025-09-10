# Project Brief for AI Assistants

This document gives AI coding assistants (Claude, ChatGPT, etc.) the context and conventions to work safely and effectively in this repository.

## Overview

- Language: Go
- Library-first design under `pkg/simplecontent` with a thin HTTP server in `cmd/server-configured`.
- Goals: clean architecture, pluggable storage/repository backends, strong typing, clear errors, easy testing.

## Core Concepts

- Content: abstraction for a logical piece of content (e.g., a document, image, video). It represents the item as users think about it, with its own metadata and lifecycle status. A content can have multiple associated objects (versions, formats).
- Object: an individual blob stored in a storage backend (memory/fs/s3). Objects belong to a content, have an `object_key`, a `version`, and storage-specific metadata.
- Derived Content: generated content produced from an original (parent) content (e.g., thumbnails, previews, transcodes). It is stored as its own Content row and linked to the parent via the `content_derived` relationship.

## Key Packages

- `pkg/simplecontent` (core library)
  - Service interface and implementation (`service.go`, `service_impl.go`)
  - Domain types and typed enums (`types.go`)
    - ContentStatus, ObjectStatus (typed string enums)
    - DerivationVariant (specific)
  - Requests/DTOs (`requests.go`)
  - Interfaces (`interfaces.go`): Repository, BlobStore, EventSink, Previewer
  - Errors (`errors.go`): typed sentinel errors for mapping
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

- Content
  - `POST /contents` create
  - `POST /contents/{parentID}/derived` create derived (body: owner_id, tenant_id, derivation_type, variant, metadata)
  - `GET /contents/{contentID}` get
  - `PUT /contents/{contentID}` update (partial)
  - `DELETE /contents/{contentID}` delete
  - `GET /contents?owner_id=&tenant_id=` list

- Content metadata
  - `POST /contents/{contentID}/metadata` set
  - `GET /contents/{contentID}/metadata` get

- Objects
  - `POST /contents/{contentID}/objects` create (also accepts `content_id` in JSON)
  - `GET /objects/{objectID}` get
  - `DELETE /objects/{objectID}` delete
  - `GET /contents/{contentID}/objects` list by content

- Upload/Download
  - `POST /objects/{objectID}/upload` (direct upload; uses Content-Type if provided)
  - `GET /objects/{objectID}/download` (streams; sets Content-Type/Disposition when available)
  - `GET /objects/{objectID}/upload-url` presigned upload
  - `GET /objects/{objectID}/download-url` presigned download
  - `GET /objects/{objectID}/preview-url` preview URL

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
- Example: `go run ./examples/basic`
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

- Keep changes minimal and scoped; respect existing structure and naming.
- Prefer typed enums from `pkg/simplecontent/types.go` for statuses/variants.
- Normalize user-provided categories/variants to lowercase.
- Use and propagate typed errors; don’t string-match error messages.
- For new handlers, follow existing JSON helpers and error mapping.
- Avoid adding new external deps unless necessary; use stdlib and existing libs.

## Extensibility Tips

- New storage backend: implement `BlobStore` in `pkg/simplecontent/storage/<name>`; wire via `config.BuildService()`.
- New repository: implement `Repository` (use pgx or memory patterns) and add to config.
- New derivation variants: add constants of type `DerivationVariant` or accept as lowercase strings from clients.
- Events/Previews: implement `EventSink`/`Previewer` and add via functional options.

## Refactor Roadmap

- See `REFACTORING_NEXT_STEPS.md` for the current plan, milestones, and definition of done.

## Safe Ops for AI

- Do not remove legacy packages until the configured server is fully validated.
- Keep API responses stable and documented before broad changes.
- When in doubt, open a small PR with clear rationale and tests.
