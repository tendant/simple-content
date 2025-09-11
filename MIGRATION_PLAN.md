# Migration Plan: Legacy → pkg/simplecontent

This document outlines how to migrate from the legacy packages (pkg/service, pkg/repository, pkg/storage) to the new library-first API in pkg/simplecontent and the configured HTTP server in cmd/server-configured.

## Overview

- New library: pkg/simplecontent (typed errors, soft delete, derivation_type + variant, pluggable backends)
- New server: cmd/server-configured (chi-based handlers, consistent JSON + error mapping)
- Database: dedicated schema recommended (content), relationship table renamed to content_derived with variant column, soft-delete columns added

## Scope

- Application service migration (Go imports and method mapping)
- HTTP API migration (endpoints and payloads)
- Database shape alignment (manual SQL)
- Rollout, validation, and rollback guidance

## Prerequisites

- Go 1.24+
- Postgres 13+ (recommended); MinIO/AWS if using s3 storage
- Ability to run goose migrations for greenfield DBs; for existing DBs, apply the manual SQL below

## Database Alignment (manual)

For existing databases, apply the following SQL in the target schema (default: content). Adjust schema qualifiers as needed.

1) Rename relationship table + column

```
-- In your target schema (e.g., SET search_path TO content;)
ALTER TABLE content.derived_content RENAME TO content_derived;
ALTER TABLE content.content_derived RENAME COLUMN derivation_type TO variant;
```

2) Add soft-delete columns (nullable timestamptz)

```
ALTER TABLE content.content
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE content.object
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE content.content_derived
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE content.object_preview
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;
```

3) Indexes (optional but recommended)

```
CREATE INDEX IF NOT EXISTS idx_content_derived_parent
  ON content.content_derived(parent_id);

CREATE INDEX IF NOT EXISTS idx_content_derived_variant
  ON content.content_derived(variant);
```

Note: New greenfield DBs created via migrations/postgres already use content_derived(variant) and include deleted_at columns.

## Application Migration (Go)

Replace legacy usages with pkg/simplecontent. The new service consolidates content, object, and storage operations.

- Build service

```
repo := memoryrepo.New() // or Postgres repo
store := memorystorage.New()
svc, _ := simplecontent.New(
  simplecontent.WithRepository(repo),
  simplecontent.WithBlobStore("default", store),
)
```

- Content

```
// Create/Get/Update/Delete (Delete is soft-delete)
svc.CreateContent(ctx, CreateContentRequest{ OwnerID, TenantID, Name })
svc.GetContent(ctx, id)
svc.UpdateContent(ctx, UpdateContentRequest{ Content: c })
svc.DeleteContent(ctx, id) // sets status=deleted, deleted_at
svc.ListContent(ctx, ListContentRequest{ OwnerID, TenantID }) // excludes soft-deleted
```

- Derived Content

```
// Create (either pass both or just variant; derivation_type inferred from variant prefix)
svc.CreateDerivedContent(ctx, CreateDerivedContentRequest{
  ParentID, OwnerID, TenantID,
  DerivationType: "thumbnail",
  Variant:        "thumbnail_256",
  Metadata:       map[string]any{"width":256},
})

// List relationships for a parent; each has .DerivationType = variant value
svc.ListDerivedByParent(ctx, parentID)

// Get relationship by derived content ID
svc.GetDerivedRelationshipByContentID(ctx, derivedID)
```

- Objects (blobs)

```
svc.CreateObject(ctx, CreateObjectRequest{ ContentID, StorageBackendName:"default", Version:1 })
svc.UploadObject(ctx, objectID, reader)
svc.DownloadObject(ctx, objectID)
svc.DeleteObject(ctx, objectID) // soft-delete
svc.GetObjectByObjectKeyAndStorageBackendName(ctx, key, backend)
svc.UpdateObjectMetaFromStorage(ctx, objectID)
```

- Storage backends

Use config to construct backends from environment for server deployments, or wire BlobStore implementations directly in code.

## HTTP API Migration

Use cmd/server-configured (base path /api/v1):

- Contents
  - POST /contents — create
  - GET /contents/{contentID} — get (includes derivation_type for derived, and variant when available)
  - PUT /contents/{contentID} — update
  - DELETE /contents/{contentID} — soft delete
  - GET /contents?owner_id=&tenant_id= — list (excludes soft-deleted)
  - POST /contents/{parentID}/derived — create derived (body: owner_id, tenant_id, derivation_type, variant, metadata)
  - GET /contents/{contentID}/derived — list all derived contents for a parent

- Content metadata
  - POST /contents/{contentID}/metadata — set
  - GET /contents/{contentID}/metadata — get

- Objects
  - POST /contents/{contentID}/objects — create
  - GET /objects/{objectID} — get
  - DELETE /objects/{objectID} — soft delete
  - GET /contents/{contentID}/objects — list objects by content

- Upload/Download
  - POST /objects/{objectID}/upload — direct upload (uses Content-Type)
  - GET /objects/{objectID}/download — stream
  - GET /objects/{objectID}/upload-url — presigned upload
  - GET /objects/{objectID}/download-url — presigned download
  - GET /objects/{objectID}/preview-url — preview URL

Error mapping is consistent and typed (not_found, invalid_status, storage_error, etc.).

## Rollout Plan

1) Database
- Apply manual SQL in staging; validate table/column names and data integrity

2) Staging Deploy
- Deploy cmd/server-configured with DATABASE_TYPE=postgres and CONTENT_DB_SCHEMA set
- Run unit tests and -tags=integration tests where possible
- Verify core flows: content CRUD, object upload/download, derived create/list

3) Application Switchover
- Replace legacy service imports with pkg/simplecontent in app code
- Update any direct HTTP consumers to the new endpoints/payloads
- Validate soft-delete expectations (Get after Delete returns not found)

4) Production Deploy (phased)
- Deploy new server and app behind a flag or route subset
- Monitor errors, storage ops, and DB changes
- Increase traffic until full cutover

5) Decommission Legacy
- Archive/remove pkg/service, pkg/repository, pkg/storage (legacy) after a stability window

## Validation Checklist

- DB
  - content_derived exists and has variant column
  - No unexpected NULL derivation values; deleted_at columns present
- API
  - Derived create works with variant-only input (derivation_type inferred)
  - Soft-deleted content/objects not listed or retrievable
  - Presigned URLs function for s3 if configured
- App
  - No compile-time references to legacy packages
  - All flows use pkg/simplecontent

## Rollback Strategy (minimal)

- Application
  - Re-point to legacy services and routes
- Database
  - If necessary, rename content_derived back to derived_content and variant back to derivation_type
  - Note: Only required if new server cannot run; avoid rolling back soft deletes unless strictly needed

## Notes

- Tenancy
  - Current DTOs keep TenantID; recommended long-term: enforce tenancy in repos via context (RLS/search_path)
- Uniqueness on (parent_id, variant)
  - Not enforced by design; if a single canonical record is desired, select by status/time in service
- OpenAPI
  - Consider adding an OpenAPI spec for the configured server to aid clients

