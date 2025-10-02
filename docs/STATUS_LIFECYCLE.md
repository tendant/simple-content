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
| `created` | Content record exists in database, but no binary data uploaded yet | `uploaded` |
| `uploaded` | Binary data successfully uploaded and stored in at least one storage backend | _(no status transitions)_ |
| ~~`deleted`~~ | **DEPRECATED:** Use `deleted_at` timestamp instead. Kept for backward compatibility only. | _(do not use)_ |

> **⚠️ Soft Delete:** Deletion is tracked via the `deleted_at` timestamp field, NOT the status field.
> When content is deleted, `deleted_at` is set to the deletion timestamp and status remains unchanged.
> See [CLAUDE.md § Soft Delete Pattern](../CLAUDE.md#soft-delete-pattern) for details.

**Use Cases:**
- Determining if content has data available
- Filtering out deleted content
- Basic availability checking

**Limitations (Current Implementation):**
- Doesn't track processing state
- Can't distinguish between "uploaded" and "processed"
- Too coarse-grained for complex workflows

> **⚠️ Implementation Gap:** These limitations are acknowledged and addressed in the refactoring plan.
> Future versions will include `processing`, `processed`, `failed`, and `archived` statuses for Content.
> See [STATUS_LIFECYCLE_REFACTORING.md § 1.3](STATUS_LIFECYCLE_REFACTORING.md#13-expand-content-status-enums)

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

### Content Derived Status (Processing Tracking)

Content derived status tracks the processing state of derived content relationships (e.g., thumbnails, previews).

| Status | Description | Next States |
|--------|-------------|-------------|
| `created` | Relationship created, processing not started | `processing`, `processed`, `failed` |
| `processing` | Derived content generation in progress | `processed`, `failed` |
| `processed` | Derived content successfully generated and verified | `deleted` |
| `failed` | Derived content generation failed | `processing` (retry), `deleted` |
| `uploaded` | _(Deprecated)_ Binary uploaded but not verified - use `processed` instead | `processed` |

**Use Cases:**
- Tracking which thumbnails are ready
- Retry failed thumbnail generation
- Monitoring processing backlog
- Verification that derived content exists

**Important:** Content derived status should mirror **object status** semantics (not content status) because:
- Derived content represents the result of processing work
- Need to distinguish "processing" from "completed"
- Need to handle processing failures explicitly

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

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Job published for thumbnail generation                   │
│    → content_derived row created                            │
│    → content_derived.status = "created"                     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Worker picks up job from queue                           │
│    → content_derived.status = "processing"                  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Worker downloads source image                            │
│    → Reads original content binary                          │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Worker generates thumbnail                               │
│    → Resizes image to target dimensions                     │
│    → Creates derived content record                         │
│    → derived_content.status = "created"                     │
│    → derived_object.status = "created"                      │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. Worker uploads thumbnail via UploadDerivedContent()      │
│    → derived_content.status = "uploaded"                    │
│    → derived_object.status = "uploaded"                     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 6. Worker marks job complete                                │
│    → content_derived.status = "processed"  ← FINAL STATE    │
│    → Publishes completion event                             │
└─────────────────────────────────────────────────────────────┘
```

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
│    → Check expected variants exist (thumbnail, preview, full)│
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Verify thumbnail objects exist and are uploaded          │
│    → Check derived_content.status = "uploaded"              │
│    → Check derived_object.status = "uploaded"               │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Update relationship status (if verification passes)      │
│    → content_derived.status = "processed"                   │
│    → Log verification success                               │
└─────────────────────────────────────────────────────────────┘
                           ↓ (if missing)
┌─────────────────────────────────────────────────────────────┐
│ 5. Publish job for missing thumbnails                       │
│    → Creates new job in NATS queue                          │
│    → Returns to step 2 in thumbnail generation flow         │
└─────────────────────────────────────────────────────────────┘
```

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

### Content Derived Status State Machine

```
    ┌─────────┐
    │ created │
    └────┬────┘
         │
         │ Worker picks up job
         ↓
    ┌────────────┐
    │ processing │───────┐
    └──────┬─────┘       │
           │             │ Generation fails
           │ Generation  │
           │ completes   ↓
           │         ┌────────┐
           ↓         │ failed │
    ┌───────────┐   └────┬───┘
    │ processed │        │
    └───────────┘        │ Retry
                         │
                         └───────────┐
                                     │
                                     ↓
                              (back to processing)
```

## Database Schema

### Content Table Status

```sql
CREATE TABLE content (
    id UUID PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    -- Valid values: 'created', 'uploaded', 'deleted'
    -- (Future: 'uploading', 'processing', 'processed', 'failed', 'archived')
    ...
);
```

> **⚠️ Implementation Gap:** No CHECK constraint enforces valid status values.
> See [STATUS_LIFECYCLE_REFACTORING.md § 2.2](STATUS_LIFECYCLE_REFACTORING.md#22-database-constraints)

### Object Table Status

```sql
CREATE TABLE object (
    id UUID PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    -- Valid values: 'created', 'uploading', 'uploaded',
    --               'processing', 'processed', 'failed', 'deleted'
    ...
);
```

> **⚠️ Implementation Gap:** No CHECK constraint enforces valid status values.
> Recommended constraint:
> ```sql
> ALTER TABLE object ADD CONSTRAINT object_status_check
> CHECK (status IN ('created', 'uploading', 'uploaded', 'processing', 'processed', 'failed', 'deleted'));
> ```

### Content Derived Table Status

```sql
CREATE TABLE content_derived (
    parent_id UUID NOT NULL,
    content_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    -- Valid values: 'created', 'processing', 'processed', 'failed'
    -- Note: Uses object-like status semantics
    ...
);
```

> **⚠️ Implementation Gap:** No CHECK constraint enforces valid status values.

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
SELECT status, COUNT(*)
FROM content_derived
WHERE derivation_type = 'thumbnail'
GROUP BY status;
```

### Find stuck processing jobs
```sql
SELECT cd.parent_id, cd.content_id, cd.status, cd.updated_at,
       EXTRACT(EPOCH FROM (NOW() - cd.updated_at)) as seconds_in_state
FROM content_derived cd
WHERE cd.status = 'processing'
  AND cd.updated_at < NOW() - INTERVAL '10 minutes'
ORDER BY cd.updated_at ASC;
```

### Find status inconsistencies
```sql
-- Derived content marked 'processed' but content not uploaded
SELECT cd.parent_id, cd.content_id, c.status as content_status, cd.status as derived_status
FROM content_derived cd
JOIN content c ON cd.content_id = c.id
WHERE cd.status = 'processed'
  AND c.status != 'uploaded';
```

## Migration Notes

### From 'uploaded' to 'processed' in content_derived

If you have existing data using 'uploaded' status in content_derived:

```sql
-- One-time migration
UPDATE content_derived
SET status = 'processed', updated_at = NOW()
WHERE status = 'uploaded'
  AND derivation_type = 'thumbnail';
```

### Verification Query

```sql
-- Verify all processed thumbnails have uploaded content
SELECT
    COUNT(*) as total_processed,
    COUNT(CASE WHEN c.status = 'uploaded' THEN 1 END) as valid_count,
    COUNT(CASE WHEN c.status != 'uploaded' THEN 1 END) as invalid_count
FROM content_derived cd
JOIN content c ON cd.content_id = c.id
WHERE cd.status = 'processed'
  AND cd.derivation_type = 'thumbnail';
```

## Troubleshooting

### Content stuck in 'created'
**Symptom:** Content has status='created' but upload completed
**Cause:** UploadContent() didn't complete status update
**Fix:** Manually update status if object exists and is uploaded

### Derived content stuck in 'processing'
**Symptom:** content_derived.status='processing' for extended period
**Cause:** Worker crashed or job failed without updating status
**Fix:** Check worker logs, retry job, or mark as failed

### Status mismatch between tables
**Symptom:** content.status='uploaded' but content_derived.status='created'
**Cause:** Status update logic incomplete or backfill not run
**Fix:** Run backfill with `-fix-status` flag

## Implementation Status Summary

### ✅ Currently Implemented
- Three-tier status tracking (Content, Object, Content Derived)
- Basic status enums defined in code
- Soft delete with `deleted` status
- Status fields in database tables

### ⚠️ Implementation Gaps
- **No status validation** - Any string value accepted
- **No transition enforcement** - Invalid transitions not prevented
- **Incomplete content lifecycle** - Missing processing/failed/archived statuses
- **No database constraints** - Status values not enforced at DB level
- **No status-based authorization** - Operations don't check status
- **No audit trail** - Status changes not logged
- **"active" status bug** - Referenced in code but undefined

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
