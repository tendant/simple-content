# Content Status Lifecycle

This document describes the **intended design** for the complete status lifecycle for content, objects, and derived content relationships in the simple-content system.

> **📋 Documentation Set:**
> - **This Document** - Operational guide for status lifecycle (intended design)
> - [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) - Gap analysis and refactoring plan
> - [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) - Implementation task checklist
> - [Documentation Index](README.md) - Overview of all documentation

> **⚠️ Implementation Note:**
> This document describes the **target state** of the status system. For current implementation gaps and planned improvements, see [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md).

## Overview

The system uses a three-tier status tracking approach:

1. **Content Status** - High-level lifecycle tracking
2. **Object Status** - Detailed processing state tracking
3. **Content Derived Status** - Processing completion tracking for derived content

## Status Types

### Content Status (High-Level Lifecycle)

Content status represents the high-level lifecycle state of a content entity.

| Status | Description | Next States |
|--------|-------------|-------------|
| `created` | Content record exists in database, but no binary data uploaded yet | `uploading`, `uploaded`, `failed` |
| `uploading` | Upload in progress (optional intermediate state) | `uploaded`, `failed` |
| `uploaded` | Binary data successfully uploaded and stored in at least one storage backend | `processing`, `archived` |
| `processing` | Post-upload processing in progress (e.g., validation, indexing) | `processed`, `failed` |
| `processed` | Processing completed successfully, content ready for use | `archived` |
| `failed` | Upload or processing failed, manual intervention or retry may be required | `uploading`, `processing` (retry) |
| `archived` | Content archived for long-term storage (future use) | _(terminal state)_ |
| ~~`deleted`~~ | **DEPRECATED:** Use `deleted_at` timestamp instead. Kept for backward compatibility only. | _(do not use)_ |

> **⚠️ Soft Delete:** Deletion is tracked via the `deleted_at` timestamp field, NOT the status field.
> When content is deleted, `deleted_at` is set to the deletion timestamp and status remains unchanged.
> See [CLAUDE.md § Soft Delete Pattern](../CLAUDE.md#soft-delete-pattern) for details.

> **✅ Status Expansion Implemented:** As of Phase 1.3, Content now has full granular status tracking to match Object capabilities.
> This enables tracking upload progress, processing states, and failure handling at the Content level.

**Use Cases:**
- Tracking upload progress (`uploading` state)
- Monitoring post-upload processing (`processing` → `processed`)
- Handling failures with retry logic (`failed` → retry)
- Long-term archival workflows (`archived`)
- Distinguishing "uploaded" from "ready to serve" (`uploaded` vs `processed`)

### Object Status (Detailed Processing State)

Object status provides granular tracking of binary data and processing states.

| Status | Description | Next States |
|--------|-------------|-------------|
| `created` | Object placeholder reserved in database, no binary data yet | `uploading`, `uploaded`, `failed`, `deleted` |
| `uploading` | Upload in progress (optional intermediate state) | `uploaded`, `failed`, `deleted` |
| `uploaded` | Binary successfully stored in blob storage | `processing`, `deleted` |
| `processing` | Post-upload processing in progress (e.g., thumbnail generation, transcoding) | `processed`, `failed` |
| `processed` | Processing completed successfully, ready for use | `deleted` |
| `failed` | Processing failed, manual intervention may be required | `processing` (retry), `deleted` |
| `deleted` | Soft delete, object marked for deletion | _(terminal)_ |

**Use Cases:**
- Tracking long-running uploads
- Monitoring post-upload processing (thumbnails, transcodes)
- Retry logic for failed processing
- Distinguishing between "uploaded" and "ready to serve"

### Derived Content Status (Processing Output)

**Current Implementation (as of 2025-10-02):**

Derived content status is tracked in the **`content.status`** column (same table as original content). The `content_derived` table does NOT have a status column (removed to avoid duplication).

**Status Semantics by Content Type:**

| Content Type | Status | Description |
|-------------|--------|-------------|
| **Original Content** | `uploaded` | Binary uploaded to storage (terminal state for originals) |
| **Derived Content** | `processed` | Generated output ready to serve (terminal state for derivatives) |

**Lifecycle:**
- **Original content**: `created` → `uploaded` (source material)
- **Derived content**: `created` → `processed` (processing output)

**Key Points:**
- Derived content IS the output of processing (thumbnails, previews, transcodes)
- Once uploaded via `UploadDerivedContent()`, status is immediately set to `"processed"`
- No intermediate "uploaded" state for derived content
- Status field makes content type semantically clear: "uploaded" = original, "processed" = derived

**Use Cases:**
- Tracking which thumbnails are ready: query `derivation_type='thumbnail' AND status='processed'`
- Monitoring processing backlog: query `derivation_type != '' AND status='created'`
- Distinguishing originals from derivatives: filter by status

## Complete Lifecycle Flows

### Original Content Upload

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Client calls UploadContent()                             │
│    → content.status = "created"                             │
│    → object.status = "created"                              │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Service uploads binary to blob storage                   │
│    → object.status = "uploading" (optional)                 │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Upload completes successfully                            │
│    → content.status = "uploaded"                            │
│    → object.status = "uploaded"                             │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Post-processing (optional)                               │
│    → object.status = "processing"                           │
│    → object.status = "processed" or "failed"                │
└─────────────────────────────────────────────────────────────┘
```

### Derived Content (Thumbnail) Generation

**Current Implementation:**

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Worker downloads source image from parent content        │
│    → Reads original content binary                          │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Worker generates thumbnail                               │
│    → Resizes image to target dimensions                     │
│    → Prepares thumbnail data for upload                     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Worker calls UploadDerivedContent()                      │
│    → Creates derived content record (content.status="created")│
│    → Creates object record (object.status="created")        │
│    → Creates content_derived relationship row               │
│    → Uploads binary to blob storage                         │
│    → Updates content.status = "processed"  ← FINAL STATE    │
│    → Updates object.status = "uploaded"                     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Content is ready to serve                                │
│    → content.status = "processed"                           │
│    → content.derivation_type = "thumbnail"                  │
│    → content_derived.parent_id = original content ID        │
│    → Derived content immediately available for download     │
└─────────────────────────────────────────────────────────────┘
```

**Key Changes from Legacy Design:**
- ✅ Single-step upload: `UploadDerivedContent()` handles everything atomically
- ✅ No separate content_derived.status column (removed for simplicity)
- ✅ Status tracked in content.status (same as original content)
- ✅ Derived content goes directly to "processed" (not "uploaded")
- ✅ Semantics: "processed" status indicates generated output ready to serve

### Async Derived Content Generation (Worker-Based)

**New async workflow (as of 2025-10-02):**

For scenarios where processing happens asynchronously (e.g., queue-based workers, long-running transcoding):

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Client/Worker calls CreateDerivedContent()              │
│    → content.status = "created" (default, job queued)      │
│    → Creates content_derived relationship row              │
│    → NO object created yet (placeholder only)              │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Worker picks up job and downloads source from parent    │
│    → Reads original content binary                         │
│    → Download may be slow or fail                          │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Worker calls UpdateContentStatus("processing")          │
│    → content.status = "processing" (download succeeded)    │
│    → Tracks that download phase completed                  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Worker generates thumbnail (expensive operation)        │
│    → Resizes image to target dimensions                    │
│    → Prepares thumbnail data in memory                     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. Worker calls UploadObjectForContent()  ← NEW METHOD     │
│    → Creates object record (object.status="created")       │
│    → Uploads binary to blob storage                        │
│    → Updates object.status = "uploaded"                    │
│    → Content status still "processing" (worker controls)   │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 6. Worker calls UpdateContentStatus("processed")           │
│    → content.status = "processed"  ← FINAL STATE           │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 7. Content is ready to serve                               │
│    → content.status = "processed"                          │
│    → Derived content available for download                │
└─────────────────────────────────────────────────────────────┘
```

**Async Workflow Benefits:**
- **Early visibility**: Placeholder created before processing (UI can show "queued" state)
- **Download tracking**: Status change after download shows download phase completed
- **Worker flexibility**: Processing happens separately from content creation
- **Status queries**: Workers can query `GetContentByStatus(ContentStatusCreated)` to find work
- **Error isolation**: Distinguish download failures from processing failures

**Error Handling in Async Workflow:**
```
┌─────────────────────────────────────────────────────────────┐
│ Download fails (network error, parent not ready, etc.)     │
│    → content.status remains "created"                      │
│    → Worker stores error in content metadata               │
│    → Worker retries later by querying created content      │
│    → Full retry including download                         │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ Processing fails after download (invalid image, etc.)      │
│    → content.status remains "processing"                   │
│    → Worker stores error in content metadata               │
│    → Worker can retry generation without re-downloading    │
│    → More efficient retry (skip download phase)            │
└─────────────────────────────────────────────────────────────┘
```

**Status Lifecycle for Async:**
```
created (queued) → [download] → processing (generating) → processed ✓
   ↓ (download fails)               ↓ (generation fails)
created (full retry)              processing (retry generation only)
```

**Worker Query Patterns:**
```sql
-- Find jobs waiting to start
SELECT * FROM content
WHERE status = 'created'
  AND derivation_type != ''
  AND deleted_at IS NULL;

-- Find jobs actively processing
SELECT * FROM content
WHERE status = 'processing'
  AND derivation_type != ''
  AND deleted_at IS NULL;

-- Find stalled downloads (created > 5 minutes ago)
SELECT * FROM content
WHERE status = 'created'
  AND derivation_type != ''
  AND updated_at < NOW() - INTERVAL '5 minutes'
  AND deleted_at IS NULL;

-- Find stalled processing (processing > 30 minutes ago)
SELECT * FROM content
WHERE status = 'processing'
  AND derivation_type != ''
  AND updated_at < NOW() - INTERVAL '30 minutes'
  AND deleted_at IS NULL;
```

**Key Differences from Synchronous:**
- Synchronous: `UploadDerivedContent()` - single atomic operation
- Async: `CreateDerivedContent()` → download → `UpdateContentStatus("processing")` → generate → `UploadObjectForContent()` → `UpdateContentStatus("processed")` - multi-step
- Async allows content placeholder to exist before data is ready
- Async enables worker-based architectures with job queues
- Async provides better observability with status tracking at each phase

### Status Verification (Backfill)

The backfill tool verifies and fixes status inconsistencies:

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Scan for content with status = "uploaded"                │
│    → Filter by derivation_type = "" (originals only)        │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Check if derived content exists                          │
│    → Query content_derived for thumbnail variants           │
│    → Check expected variants exist (thumbnail_256, etc.)    │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Verify derived content status                            │
│    → Query content table for derived content IDs            │
│    → Check content.status = "processed" for each derivative │
│    → Check objects exist and are uploaded                   │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Log verification success or schedule missing work        │
│    → If all derivatives are "processed": mark parent ready  │
│    → If missing: publish job for thumbnail generation       │
└─────────────────────────────────────────────────────────────┘
```

**Note:** Since content_derived table no longer has a status column, status is checked in the content table using the derived content's ID.

## Status State Machine Diagrams

### Content Status State Machine

> **⚠️ Note on Soft Delete:** The "deleted" status shown below is **deprecated**.
> Deletion is indicated by the `deleted_at` timestamp, not by status.
> Status remains at its last operational state (e.g., "uploaded") when deleted.

```
    ┌─────────┐
    │ created │
    └────┬────┘
         │
         │ UploadContent()
         │ completes
         ↓
    ┌──────────┐
    │ uploaded │──────┐
    └──────────┘      │
                      │ DELETE (sets deleted_at)
                      │ Status stays "uploaded"
                      ↓
               (soft deleted via
                deleted_at timestamp)
```

### Object Status State Machine

> **⚠️ Note on Soft Delete:** The "deleted" status shown below is **deprecated**.
> Deletion is indicated by the `deleted_at` timestamp, not by status.
> Status remains at its last operational state when deleted.

```
    ┌─────────┐
    │ created │
    └────┬────┘
         │
         │ Upload starts
         ↓
    ┌───────────┐
    │ uploading │───────┐
    └─────┬─────┘       │
          │             │ Upload fails
          │ Upload      │
          │ completes   ↓
          │         ┌────────┐
          ↓         │ failed │
    ┌──────────┐   └────┬───┘
    │ uploaded │        │
    └─────┬────┘        │ Retry
          │             │
          │ Processing  │
          │ starts      │
          ↓             │
    ┌────────────┐      │
    │ processing │──────┘
    └──────┬─────┘
           │
           │ Processing completes
           ↓
    ┌───────────┐
    │ processed │──────┐
    └───────────┘      │
                       │ DELETE (sets deleted_at)
                       │ Status stays "processed"
                       ↓
                (soft deleted via
                 deleted_at timestamp)
```

### Derived Content Status State Machine

**Current Implementation (Simplified Model):**

```
    ┌─────────┐
    │ created │
    └────┬────┘
         │
         │ UploadDerivedContent()
         │ completes
         ↓
    ┌───────────┐
    │ processed │ ← TERMINAL STATE
    └───────────┘

```

**Key Points:**
- Derived content goes directly from `created` to `processed` in a single operation
- No intermediate states (no "processing", "uploading", etc.)
- Status is tracked in content.status column (not content_derived table)
- "processed" status indicates the derivative is generated and ready to serve
- Compare to original content which uses "uploaded" as terminal state

**Legacy Model (Deprecated):**
The previous multi-step model with "processing" and "failed" states is no longer used. Worker-based async processing should create the content record only after successful generation, then immediately upload it to "processed" status.

## Database Schema

### Content Table Status

**Current Implementation (as of 2025-10-02):**

```sql
CREATE TABLE content (
    id UUID PRIMARY KEY,
    derivation_type VARCHAR(32),  -- Empty/NULL for originals, set for derived
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    -- Valid values:
    --   For originals: 'created', 'uploaded'
    --   For derived:   'created', 'processed'
    --   Reserved:      'uploading', 'processing', 'failed', 'archived'
    deleted_at TIMESTAMP NULL,  -- Soft delete (NOT using status='deleted')
    ...
);
```

**Status Semantics:**
- Original content (`derivation_type = '' OR NULL`): Uses `'uploaded'` as terminal state
- Derived content (`derivation_type != ''`): Uses `'processed'` as terminal state
- Status field makes content type semantically clear

> **⚠️ Note:** No CHECK constraint enforces valid status values (application-level validation only).

### Object Table Status

```sql
CREATE TABLE object (
    id UUID PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    -- Valid values: 'created', 'uploading', 'uploaded',
    --               'processing', 'processed', 'failed'
    deleted_at TIMESTAMP NULL,  -- Soft delete (NOT using status='deleted')
    ...
);
```

> **⚠️ Note:** No CHECK constraint enforces valid status values (application-level validation only).

### Content Derived Table (Relationship Only)

**Current Implementation:**

```sql
CREATE TABLE content_derived (
    parent_id UUID NOT NULL,
    content_id UUID NOT NULL,
    variant VARCHAR(32) NOT NULL,
    derivation_type VARCHAR(32) NOT NULL,
    -- NO STATUS COLUMN - status is tracked in content.status
    ...
);
```

**Key Points:**
- This table only stores the parent-child relationship
- Status is tracked in the `content.status` column for the derived content
- Removed `status` column to eliminate duplication and sync issues
- Query derived content status by joining with content table

## Best Practices

### Status Updates

1. **Always update status atomically** - Use transactions when updating multiple status fields
2. **Update timestamps** - Always update `updated_at` when changing status
3. **Log status transitions** - Log before and after status for debugging
4. **Handle failures gracefully** - Set `failed` status rather than leaving in limbo

> **⚠️ Implementation Gap:** Status transition validation is not enforced. The system currently allows
> any status transition, including invalid ones (e.g., `deleted` → `uploaded`).
> See [STATUS_LIFECYCLE_REFACTORING.md § 1.2](STATUS_LIFECYCLE_REFACTORING.md#12-status-transition-state-machine)

### Status Queries

1. **Use indexed status fields** - Ensure status columns are indexed for performance
2. **Filter by status combinations** - e.g., `status IN ('uploaded', 'processed')`
3. **Join tables carefully** - Be aware of status field conflicts when joining

### Error Handling

1. **Distinguish temporary vs permanent failures**
   - Temporary: Network issues, resource limits → retry with `failed` status
   - Permanent: Invalid data, missing source → mark `failed` and alert

2. **Implement retry logic**
   - Check `failed` status and retry count
   - Use exponential backoff
   - Alert after N failures

3. **Monitor stuck processing**
   - Track items in `processing` state longer than threshold
   - Alert on stale processing jobs

## Monitoring Queries

### Count content by status
```sql
SELECT status, COUNT(*)
FROM content
GROUP BY status;
```

### Count objects by status
```sql
SELECT status, COUNT(*)
FROM object
GROUP BY status;
```

### Count derived content by status
```sql
-- Status is in content table, not content_derived
SELECT c.status, COUNT(*)
FROM content c
WHERE c.derivation_type = 'thumbnail'
  AND c.deleted_at IS NULL
GROUP BY c.status;
```

### Count content by type and status
```sql
-- Distinguish original vs derived by status
SELECT
    CASE
        WHEN derivation_type = '' OR derivation_type IS NULL THEN 'original'
        ELSE 'derived'
    END as content_type,
    status,
    COUNT(*) as count
FROM content
WHERE deleted_at IS NULL
GROUP BY content_type, status
ORDER BY content_type, status;
```

### Find derived content not yet processed
```sql
-- Find derivatives that haven't been generated yet
SELECT cd.parent_id, cd.content_id, cd.variant, c.status, c.updated_at
FROM content_derived cd
JOIN content c ON cd.content_id = c.id
WHERE c.status != 'processed'
  AND c.deleted_at IS NULL
ORDER BY c.updated_at ASC;
```

### Find status inconsistencies
```sql
-- Derived content with wrong status (should be 'processed')
SELECT cd.parent_id, cd.content_id, cd.derivation_type, cd.variant, c.status
FROM content_derived cd
JOIN content c ON cd.content_id = c.id
WHERE c.derivation_type != ''
  AND c.status NOT IN ('created', 'processed')
  AND c.deleted_at IS NULL;
```

## Migration Notes

### Update Derived Content Status from 'uploaded' to 'processed'

**Current Implementation (as of 2025-10-02):**

Derived content now uses `status='processed'` instead of `'uploaded'`. If you have existing derived content with the old status, update them:

```sql
-- Update all derived content from 'uploaded' to 'processed'
UPDATE content
SET status = 'processed', updated_at = NOW()
WHERE derivation_type != ''
  AND derivation_type IS NOT NULL
  AND status = 'uploaded'
  AND deleted_at IS NULL;
```

### Verification Query

```sql
-- Verify migration: count by content type and status
SELECT
    CASE WHEN derivation_type = '' OR derivation_type IS NULL THEN 'original' ELSE 'derived' END as type,
    status,
    COUNT(*) as count
FROM content
WHERE deleted_at IS NULL
GROUP BY type, status
ORDER BY type, status;

-- Expected results:
-- original  | uploaded   | N  (originals stay as 'uploaded')
-- derived   | processed  | M  (derivatives should be 'processed')
```

## Troubleshooting

### Content stuck in 'created'
**Symptom:** Content has status='created' but upload completed
**Cause:** UploadContent() didn't complete status update
**Fix:** Manually update status based on content type:
```sql
-- For original content
UPDATE content SET status = 'uploaded', updated_at = NOW()
WHERE id = '<content-id>' AND (derivation_type = '' OR derivation_type IS NULL);

-- For derived content
UPDATE content SET status = 'processed', updated_at = NOW()
WHERE id = '<content-id>' AND derivation_type != '';
```

### Derived content with wrong status
**Symptom:** Derived content has status='uploaded' instead of 'processed'
**Cause:** Created before 2025-10-02 status semantics change
**Fix:** Run migration query:
```sql
UPDATE content
SET status = 'processed', updated_at = NOW()
WHERE derivation_type != '' AND status = 'uploaded' AND deleted_at IS NULL;
```

### Status mismatch - derived content not ready
**Symptom:** Parent is uploaded but GetContentDetails() shows not ready
**Cause:** Derived content status is not 'processed'
**Fix:** Check derived content status and update if objects exist:
```sql
-- Find the issue
SELECT cd.parent_id, cd.content_id, c.status, cd.derivation_type, cd.variant
FROM content_derived cd
JOIN content c ON cd.content_id = c.id
WHERE cd.parent_id = '<parent-id>' AND c.status != 'processed';
```

## Implementation Status Summary

### ✅ Currently Implemented (as of 2025-10-02)
- **Simplified status model**: Two terminal states (uploaded/processed) based on content type
- **Status tracked in content.status**: Single source of truth (no content_derived.status column)
- **Type-based semantics**:
  - Original content: `created` → `uploaded`
  - Derived content: `created` → `processed`
- **Typed enums**: ContentStatus and ObjectStatus with validation
- **Status-based authorization**: Operations validate status before execution
- **Soft delete**: Uses `deleted_at` timestamp (not status field)
- **Ready logic**: Content-type-aware ready checking in GetContentDetails()

### ✅ Recent Improvements
- Removed content_derived.status column (eliminated duplication)
- Made "processed" status exclusive to derived content
- Updated UploadDerivedContent() to set status="processed" directly
- Added comprehensive status validation in status_validation.go
- Clarified semantic distinction between original and derived content

### ⚠️ Known Limitations
- **No database constraints**: Status values not enforced at DB level (application-level validation only)
- **No audit trail**: Status changes not logged (consider adding in future)
- **Reserved statuses**: `processing`, `failed`, `archived` defined but not actively used
- **No transition enforcement**: State machine transitions not strictly enforced

### 🔄 Refactoring Plan
See [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) for:
- Detailed gap analysis
- Implementation plan (3 phases)
- Code examples and migrations
- Testing strategy

See [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) for:
- Sprint-by-sprint task breakdown
- ~35 working days estimated timeline
- Testing requirements

## Related Documentation

- [STATUS_LIFECYCLE_REFACTORING.md](STATUS_LIFECYCLE_REFACTORING.md) - Gap analysis and refactoring plan
- [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) - Implementation checklist
- [Documentation Index](README.md) - Overview of all documentation
- [CLAUDE.md](../CLAUDE.md) - Project conventions and AI coding guide
