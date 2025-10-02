# Status Lifecycle Refactoring Plan

> **📋 Documentation Set:**
> - [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) - Operational guide (intended design)
> - **This Document** - Gap analysis and refactoring plan
> - [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) - Implementation checklist
> - [Documentation Index](README.md) - Overview of all documentation

## Executive Summary

This document outlines the **current gaps** between the intended status lifecycle design (documented in [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md)) and the actual implementation. It provides a comprehensive refactoring plan to improve status management, validation, and transition logic.

**Target State:** See [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) for the complete operational guide to how the status system should work.

## Current Status Design

### Content Status Enums
**Location:** `pkg/simplecontent/types.go:14-17`

```go
type ContentStatus string

const (
    ContentStatusCreated  ContentStatus = "created"
    ContentStatusUploaded ContentStatus = "uploaded"
    ContentStatusDeleted  ContentStatus = "deleted"
)
```

**Lifecycle:** `created` → `uploaded` → `deleted`

### Object Status Enums
**Location:** `pkg/simplecontent/types.go:42-49`

```go
type ObjectStatus string

const (
    ObjectStatusCreated    ObjectStatus = "created"
    ObjectStatusUploading  ObjectStatus = "uploading"
    ObjectStatusUploaded   ObjectStatus = "uploaded"
    ObjectStatusProcessing ObjectStatus = "processing"
    ObjectStatusProcessed  ObjectStatus = "processed"
    ObjectStatusFailed     ObjectStatus = "failed"
    ObjectStatusDeleted    ObjectStatus = "deleted"
)
```

**Lifecycle:** `created` → `uploading` → `uploaded` → `processing` → `processed` → `failed` → `deleted`

---

## Identified Issues

### 1. No Status Validation ⚠️ **CRITICAL**

**Problem:**
- No validation logic ensures valid status values
- Status field accepts any string value (struct field is `string`, not typed enum)
- No compile-time or runtime checks prevent invalid statuses

**Impact:**
- Database can contain invalid status values like "activ", "deletd" (typos)
- No error when setting invalid statuses
- Query logic breaks when unexpected statuses exist

**Evidence:**
- `Content.Status` is `string` type (types.go:64)
- `Object.Status` is `string` type (types.go:122)
- No validation in service layer before database writes

---

### 2. No Status Transition Logic ⚠️ **CRITICAL**

**Problem:**
- Missing state machine to enforce valid transitions
- No validation prevents invalid state changes

**Impact:**
- Problematic transitions not prevented:
  - Object: `deleted` → `uploaded`
  - Content: `uploaded` → `created` (backwards)
  - Object: `uploaded` → `creating` (typo)
  - Content: `deleted` → `uploaded` (resurrection)

**Evidence:**
- `UpdateContent()` accepts any status change (service_impl.go:241-260)
- `UpdateObject()` accepts any status change (service_impl.go:742-754)
- No transition validation in repository layer

---

### 3. Incomplete Content Status Lifecycle ⚠️ **HIGH**

**Problem:**
- Content has only 3 states vs Object's 7 states
- Missing intermediate and error states

**Missing Statuses:**
- `uploading` - during active upload operations
- `processing` - for async operations (thumbnail generation, virus scanning, AI processing)
- `failed` - for permanent failures requiring intervention
- `archived` - for retention/compliance workflows
- `published`/`draft` - for content approval workflows
- `pending` - for queued operations

**Impact:**
- Cannot track upload progress for content
- No way to represent failed content operations
- Cannot implement async processing workflows
- No support for content moderation/approval workflows

---

### 4. No Status-Based Business Logic ⚠️ **HIGH**

**Problem:**
- Operations don't properly validate status before execution
- Missing authorization based on status

**Evidence:**
```go
// service_impl.go:572 - Only checks uploaded status
if obj.Status == string(ObjectStatusUploaded) {
    targetObject = obj
    break
}

// service_impl.go:924 - References undefined "active" status
if content.Status != "created" && content.Status != "active" {
    result.Ready = false
}
```

**Missing Business Rules:**
- Cannot prevent download from `deleted` content
- Cannot prevent upload to `processing` content
- Cannot prevent deletion of `processing` content
- No restrictions on concurrent status changes

---

### 5. Status Inconsistency ⚠️ **MEDIUM**

**Problem:**
- Multiple representations of status across codebase
- Typed enums defined but not enforced

**Evidence:**
1. **DerivedContent Status:**
   - `content_derived` table has `status` field (schema:75)
   - `DerivedContent` struct field is `string` not typed enum (types.go:85)

2. **Database Schema:**
   - Status stored as `VARCHAR(32)` (schema:16, :47, :75)
   - No CHECK constraints enforce valid values

3. **Status Checking:**
   - Mix of typed constants: `string(ObjectStatusUploaded)` (service_impl.go:369)
   - String literals: `"created"`, `"active"` (service_impl.go:924)

4. **Undefined Status Reference:**
   - "active" status referenced but not defined in `ContentStatus` enum

---

### 6. Missing Status Operations ⚠️ **MEDIUM**

**Problem:**
- No dedicated methods for status updates
- Status changes bundled with full entity updates

**Missing Operations:**
- `UpdateContentStatus(ctx, contentID, newStatus)` - dedicated status update
- `UpdateObjectStatus(ctx, objectID, newStatus)` - dedicated status update
- `GetContentByStatus(ctx, status)` - query by status
- `BulkUpdateStatus(ctx, ids, newStatus)` - batch operations

**Impact:**
- Cannot update status without full entity
- No audit trail for status changes
- No events specifically for status transitions
- Inefficient database operations (full UPDATE vs status-only)

**Evidence:**
- Status updates require full `UpdateContent()` call (service_impl.go:241)
- `EventSink` has no status-specific events (interfaces.go:76-94)

---

### 7. No Error Recovery Patterns ⚠️ **MEDIUM**

**Problem:**
- No patterns for handling transient failures
- No retry mechanisms built into status model

**Missing Patterns:**
- `retrying` status - for operations being retried
- `pending` status - for queued async operations
- `suspended` status - for temporary holds
- Retry count metadata
- Error message/reason tracking

**Impact:**
- Cannot implement robust retry logic
- No visibility into failure reasons
- Manual intervention required for all failures
- No automatic recovery workflows

---

### 8. Soft Delete Issues ⚠️ **LOW**

**Problem:**
- Inconsistent soft delete implementation
- Both status field AND timestamp used for deletion

**Evidence:**
```go
// repo/memory/repository.go:94
c.Status = string(simplecontent.ContentStatusDeleted)
c.DeletedAt = &now

// Query filters use deleted_at, not status
// postgres/repository.go:97
WHERE id = $1 AND deleted_at IS NULL
```

**Impact:**
- Redundant deletion markers
- Confusion about source of truth
- Potential inconsistency (status=deleted but deleted_at=null)
- Unnecessary complexity

**Options:**
1. Use status='deleted' only (remove deleted_at)
2. Use deleted_at only (remove status='deleted')
3. Keep both but document clear semantics

---

## Refactoring Plan

### Phase 1: Foundation (High Priority) 🔴

#### 1.1 Add Status Validation

**File:** `pkg/simplecontent/types.go`

```go
// IsValid validates if the content status is a known valid status
func (s ContentStatus) IsValid() bool {
    switch s {
    case ContentStatusCreated, ContentStatusUploaded, ContentStatusDeleted:
        return true
    }
    return false
}

// IsValid validates if the object status is a known valid status
func (s ObjectStatus) IsValid() bool {
    switch s {
    case ObjectStatusCreated, ObjectStatusUploading, ObjectStatusUploaded,
         ObjectStatusProcessing, ObjectStatusProcessed,
         ObjectStatusFailed, ObjectStatusDeleted:
        return true
    }
    return false
}

// ParseContentStatus parses a string into a ContentStatus with validation
func ParseContentStatus(s string) (ContentStatus, error) {
    status := ContentStatus(s)
    if !status.IsValid() {
        return "", fmt.Errorf("%w: %s", ErrInvalidContentStatus, s)
    }
    return status, nil
}

// ParseObjectStatus parses a string into an ObjectStatus with validation
func ParseObjectStatus(s string) (ObjectStatus, error) {
    status := ObjectStatus(s)
    if !status.IsValid() {
        return "", fmt.Errorf("%w: %s", ErrInvalidObjectStatus, s)
    }
    return status, nil
}
```

---

#### 1.2 Status Transition State Machine

**File:** `pkg/simplecontent/status/transitions.go` (new file)

```go
package status

import (
    "fmt"
    "github.com/tendant/simple-content/pkg/simplecontent"
)

// ContentTransitions defines valid content status transitions
var ContentTransitions = map[simplecontent.ContentStatus][]simplecontent.ContentStatus{
    simplecontent.ContentStatusCreated: {
        simplecontent.ContentStatusUploading,
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusUploading: {
        simplecontent.ContentStatusUploaded,
        simplecontent.ContentStatusFailed,
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusUploaded: {
        simplecontent.ContentStatusProcessing,
        simplecontent.ContentStatusArchived,
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusProcessing: {
        simplecontent.ContentStatusProcessed,
        simplecontent.ContentStatusFailed,
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusProcessed: {
        simplecontent.ContentStatusArchived,
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusFailed: {
        simplecontent.ContentStatusUploading, // retry
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusArchived: {
        simplecontent.ContentStatusDeleted,
    },
    simplecontent.ContentStatusDeleted: {
        // Terminal state - no transitions
    },
}

// ValidateContentTransition checks if a status transition is valid
func ValidateContentTransition(from, to simplecontent.ContentStatus) error {
    // Validate both statuses are valid
    if !from.IsValid() {
        return fmt.Errorf("%w: invalid from status: %s",
            simplecontent.ErrInvalidContentStatus, from)
    }
    if !to.IsValid() {
        return fmt.Errorf("%w: invalid to status: %s",
            simplecontent.ErrInvalidContentStatus, to)
    }

    // Check if transition is allowed
    allowedTransitions, exists := ContentTransitions[from]
    if !exists {
        return fmt.Errorf("no transitions defined for status: %s", from)
    }

    for _, allowed := range allowedTransitions {
        if allowed == to {
            return nil
        }
    }

    return fmt.Errorf("invalid transition from %s to %s", from, to)
}

// Similar for ObjectTransitions...
```

---

#### 1.3 Expand Content Status Enums

**File:** `pkg/simplecontent/types.go`

```go
const (
    ContentStatusCreated    ContentStatus = "created"
    ContentStatusUploading  ContentStatus = "uploading"  // NEW
    ContentStatusUploaded   ContentStatus = "uploaded"
    ContentStatusProcessing ContentStatus = "processing" // NEW
    ContentStatusProcessed  ContentStatus = "processed"  // NEW
    ContentStatusFailed     ContentStatus = "failed"     // NEW
    ContentStatusArchived   ContentStatus = "archived"   // NEW
    ContentStatusDeleted    ContentStatus = "deleted"
)
```

**Migration Required:** Yes - update validation logic and documentation

---

#### 1.4 Add Status Update Methods

**File:** `pkg/simplecontent/interfaces.go`

```go
type Service interface {
    // ... existing methods ...

    // Status operations
    UpdateContentStatus(ctx context.Context, contentID uuid.UUID, newStatus ContentStatus) error
    UpdateObjectStatus(ctx context.Context, objectID uuid.UUID, newStatus ObjectStatus) error
}
```

**File:** `pkg/simplecontent/service_impl.go`

```go
func (s *service) UpdateContentStatus(ctx context.Context, contentID uuid.UUID, newStatus ContentStatus) error {
    // Get current content
    content, err := s.repository.GetContent(ctx, contentID)
    if err != nil {
        return &ContentError{ContentID: contentID, Op: "update_status", Err: err}
    }

    // Validate transition
    currentStatus := ContentStatus(content.Status)
    if err := status.ValidateContentTransition(currentStatus, newStatus); err != nil {
        return &ContentError{ContentID: contentID, Op: "update_status", Err: err}
    }

    // Update status
    oldStatus := content.Status
    content.Status = string(newStatus)
    content.UpdatedAt = time.Now().UTC()

    if err := s.repository.UpdateContent(ctx, content); err != nil {
        return &ContentError{ContentID: contentID, Op: "update_status", Err: err}
    }

    // Fire status change event
    if s.eventSink != nil {
        if err := s.eventSink.ContentStatusChanged(ctx, contentID, oldStatus, string(newStatus)); err != nil {
            // Log error but don't fail the operation
        }
    }

    return nil
}

// Similar for UpdateObjectStatus...
```

---

### Phase 2: Business Logic (Medium Priority) 🟡

#### 2.1 Status-Based Authorization

**File:** `pkg/simplecontent/service_impl.go`

```go
func (s *service) DownloadContent(ctx context.Context, contentID uuid.UUID) (io.ReadCloser, error) {
    // Get content to check status
    content, err := s.repository.GetContent(ctx, contentID)
    if err != nil {
        return nil, &ContentError{ContentID: contentID, Op: "download", Err: err}
    }

    // Validate content status allows download
    status := ContentStatus(content.Status)
    switch status {
    case ContentStatusUploaded, ContentStatusProcessed:
        // OK to download
    case ContentStatusDeleted:
        return nil, &ContentError{
            ContentID: contentID,
            Op: "download",
            Err: fmt.Errorf("cannot download deleted content"),
        }
    case ContentStatusProcessing:
        return nil, &ContentError{
            ContentID: contentID,
            Op: "download",
            Err: fmt.Errorf("content is being processed"),
        }
    default:
        return nil, &ContentError{
            ContentID: contentID,
            Op: "download",
            Err: fmt.Errorf("content not ready for download (status: %s)", status),
        }
    }

    // ... existing download logic ...
}

// Add similar checks to:
// - UploadContent (prevent upload to processing content)
// - DeleteContent (prevent deletion of processing content, or require force flag)
// - CreateDerivedContent (require parent is uploaded/processed)
```

---

#### 2.2 Database Constraints

**File:** `migrations/postgres/YYYYMMDDHHMMSS_add_status_constraints.sql`

```sql
-- +goose Up

-- Add CHECK constraints for status columns
ALTER TABLE content
ADD CONSTRAINT content_status_check
CHECK (status IN (
    'created', 'uploading', 'uploaded', 'processing',
    'processed', 'failed', 'archived', 'deleted'
));

ALTER TABLE object
ADD CONSTRAINT object_status_check
CHECK (status IN (
    'created', 'uploading', 'uploaded', 'processing',
    'processed', 'failed', 'deleted'
));

ALTER TABLE content_derived
ADD CONSTRAINT content_derived_status_check
CHECK (status IN (
    'created', 'uploading', 'uploaded', 'processing',
    'processed', 'failed', 'deleted'
));

-- Create indexes for status-based queries
CREATE INDEX IF NOT EXISTS idx_content_status_not_deleted
ON content(status) WHERE status != 'deleted';

CREATE INDEX IF NOT EXISTS idx_object_status_uploaded
ON object(status) WHERE status = 'uploaded';

-- +goose Down

DROP INDEX IF EXISTS idx_object_status_uploaded;
DROP INDEX IF EXISTS idx_content_status_not_deleted;

ALTER TABLE content_derived DROP CONSTRAINT IF EXISTS content_derived_status_check;
ALTER TABLE object DROP CONSTRAINT IF EXISTS object_status_check;
ALTER TABLE content DROP CONSTRAINT IF EXISTS content_status_check;
```

---

#### 2.3 Status Change Events

**File:** `pkg/simplecontent/interfaces.go`

```go
type EventSink interface {
    // ... existing methods ...

    // Status change events
    ContentStatusChanged(ctx context.Context, contentID uuid.UUID, oldStatus, newStatus string) error
    ObjectStatusChanged(ctx context.Context, objectID uuid.UUID, oldStatus, newStatus string) error
}
```

**File:** `pkg/simplecontent/noop.go`

```go
func (n *NoOpEventSink) ContentStatusChanged(ctx context.Context, contentID uuid.UUID, oldStatus, newStatus string) error {
    return nil
}

func (n *NoOpEventSink) ObjectStatusChanged(ctx context.Context, objectID uuid.UUID, oldStatus, newStatus string) error {
    return nil
}
```

---

#### 2.4 Fix "active" Status Reference

**File:** `pkg/simplecontent/service_impl.go:924`

**Current Code:**
```go
if content.Status != "created" && content.Status != "active" {
    result.Ready = false
}
```

**Option 1:** Add "active" as alias for "uploaded"
```go
const ContentStatusActive ContentStatus = "uploaded" // Alias for backward compatibility
```

**Option 2:** Fix the logic to use proper statuses
```go
// Check if content is ready
status := ContentStatus(content.Status)
switch status {
case ContentStatusUploaded, ContentStatusProcessed:
    result.Ready = true
default:
    result.Ready = false
}
```

---

### Phase 3: Advanced Features (Low Priority) 🟢

#### 3.1 Status History / Audit Trail

**File:** `migrations/postgres/YYYYMMDDHHMMSS_add_status_history.sql`

```sql
-- +goose Up

CREATE TABLE IF NOT EXISTS content_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    old_status VARCHAR(32) NOT NULL,
    new_status VARCHAR(32) NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    changed_by UUID,
    reason TEXT,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_content_status_history_content
ON content_status_history(content_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS object_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    object_id UUID NOT NULL REFERENCES object(id) ON DELETE CASCADE,
    old_status VARCHAR(32) NOT NULL,
    new_status VARCHAR(32) NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    changed_by UUID,
    reason TEXT,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_object_status_history_object
ON object_status_history(object_id, changed_at DESC);

-- +goose Down

DROP TABLE IF EXISTS object_status_history;
DROP TABLE IF EXISTS content_status_history;
```

**File:** `pkg/simplecontent/types.go`

```go
type StatusChange struct {
    ID         uuid.UUID              `json:"id"`
    ContentID  *uuid.UUID             `json:"content_id,omitempty"`
    ObjectID   *uuid.UUID             `json:"object_id,omitempty"`
    OldStatus  string                 `json:"old_status"`
    NewStatus  string                 `json:"new_status"`
    ChangedAt  time.Time              `json:"changed_at"`
    ChangedBy  *uuid.UUID             `json:"changed_by,omitempty"`
    Reason     string                 `json:"reason,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

---

#### 3.2 Retry/Recovery Patterns

**File:** `pkg/simplecontent/types.go`

```go
// Add retry-related statuses
const (
    ContentStatusPending   ContentStatus = "pending"   // Queued for processing
    ContentStatusRetrying  ContentStatus = "retrying"  // Being retried after failure
    ContentStatusSuspended ContentStatus = "suspended" // Temporarily paused
)

// Add to ContentMetadata
type ContentMetadata struct {
    // ... existing fields ...

    // Retry metadata
    RetryCount    int                    `json:"retry_count,omitempty"`
    MaxRetries    int                    `json:"max_retries,omitempty"`
    LastError     string                 `json:"last_error,omitempty"`
    NextRetryAt   *time.Time             `json:"next_retry_at,omitempty"`
}
```

**File:** `pkg/simplecontent/service_impl.go`

```go
func (s *service) RetryFailedContent(ctx context.Context, contentID uuid.UUID) error {
    content, err := s.repository.GetContent(ctx, contentID)
    if err != nil {
        return err
    }

    // Only retry failed content
    if content.Status != string(ContentStatusFailed) {
        return fmt.Errorf("content is not in failed status")
    }

    // Get metadata to check retry count
    metadata, err := s.repository.GetContentMetadata(ctx, contentID)
    if err == nil && metadata.RetryCount >= metadata.MaxRetries {
        return fmt.Errorf("max retries exceeded")
    }

    // Transition to retrying status
    if err := s.UpdateContentStatus(ctx, contentID, ContentStatusRetrying); err != nil {
        return err
    }

    // Increment retry count
    if metadata != nil {
        metadata.RetryCount++
        metadata.UpdatedAt = time.Now().UTC()
        s.repository.SetContentMetadata(ctx, metadata)
    }

    return nil
}
```

---

#### 3.3 Consolidate Soft Delete

**Decision Required:** Choose one approach

**Option A: Use status='deleted' only**

```go
// Remove deleted_at from schema
ALTER TABLE content DROP COLUMN deleted_at;
ALTER TABLE object DROP COLUMN deleted_at;

// Update queries to filter by status
WHERE status != 'deleted'
```

**Option B: Use deleted_at only**

```go
// Remove status='deleted' from enums
const (
    ContentStatusCreated    ContentStatus = "created"
    ContentStatusUploaded   ContentStatus = "uploaded"
    // ContentStatusDeleted removed
)

// Soft delete implementation
func (r *Repository) DeleteContent(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE content SET deleted_at = NOW() WHERE id = $1`
    _, err := r.db.Exec(ctx, query, id)
    return err
}
```

**Option C: Keep both with clear semantics** (Recommended)

```go
// Semantic:
// - deleted_at = soft delete timestamp (for recovery)
// - status = 'deleted' for business logic

// Document in CLAUDE.md:
// Deletion uses both markers:
// 1. status = 'deleted' - prevents business operations
// 2. deleted_at = timestamp - enables recovery/purge workflows
// Queries should filter on deleted_at IS NULL for active records
```

---

## Testing Strategy

### Unit Tests Required

1. **Status Validation Tests** (`pkg/simplecontent/types_test.go`)
   - Test `IsValid()` for all valid statuses
   - Test `IsValid()` returns false for invalid statuses
   - Test `ParseContentStatus()` with valid/invalid inputs

2. **Transition Validation Tests** (`pkg/simplecontent/status/transitions_test.go`)
   - Test all valid transitions
   - Test all invalid transitions return errors
   - Test terminal state (deleted) has no transitions

3. **Status Update Tests** (`pkg/simplecontent/service_test.go`)
   - Test `UpdateContentStatus()` with valid transition
   - Test `UpdateContentStatus()` rejects invalid transition
   - Test status change events are fired

4. **Business Logic Tests**
   - Test download prevented for deleted content
   - Test upload prevented for processing content
   - Test delete prevented for processing content (or requires force)

### Integration Tests Required

1. **Database Constraint Tests** (`tests/integration/status_test.go`)
   - Test database rejects invalid status values
   - Test unique constraints with soft delete

2. **Repository Tests** (`pkg/simplecontent/repo/postgres/repository_test.go`)
   - Test status filtering in queries
   - Test soft delete with status + deleted_at

---

## Migration Path

### Step 1: Add New Code (Non-Breaking)
- Add validation methods to types.go
- Add transition validation package
- Add new status enums (backward compatible)
- Add new service methods

### Step 2: Update Existing Code
- Update service methods to use validation
- Add status checks to business logic
- Update repository methods

### Step 3: Database Migration
- Add CHECK constraints (will fail if invalid data exists)
- **Pre-migration:** Clean up any invalid statuses in database
- Add indexes for status-based queries

### Step 4: Deprecation
- Mark old patterns as deprecated
- Update documentation
- Add migration guide for API consumers

### Step 5: Cleanup
- Remove deprecated code after grace period
- Remove backward compatibility shims

---

## Backward Compatibility Considerations

### API Compatibility
- New status values are additive (no breaking change)
- Existing statuses remain valid
- Status validation is opt-in initially (log warnings)

### Database Compatibility
- CHECK constraints must match existing data
- Run data cleanup script before migration
- Provide rollback script

### Client Compatibility
- Clients expecting only 3 content statuses will see new values
- Document new statuses in API changelog
- Consider API versioning if needed

---

## Implementation Checklist

See `STATUS_LIFECYCLE_TODO.md` for detailed implementation tasks.

---

## References

### Documentation
- [STATUS_LIFECYCLE.md](STATUS_LIFECYCLE.md) - Operational guide (target state)
- [STATUS_LIFECYCLE_TODO.md](STATUS_LIFECYCLE_TODO.md) - Implementation checklist
- [CLAUDE.md](../CLAUDE.md) - Project conventions and coding guidelines

### Code References
- Types definition: `pkg/simplecontent/types.go`
- Service implementation: `pkg/simplecontent/service_impl.go`
- Repository interface: `pkg/simplecontent/interfaces.go`
- Database schema: `migrations/postgres/202509090002_core_tables.sql`
- Error definitions: `pkg/simplecontent/errors.go`

---

## Open Questions

1. **Should we enforce status validation immediately or gradually?**
   - Option A: Hard enforcement (breaking change)
   - Option B: Log warnings first, enforce later
   - **Recommendation:** Option B with 2-sprint grace period

2. **How to handle existing invalid statuses in database?**
   - Option A: Migration script auto-corrects to nearest valid status
   - Option B: Migration fails, requires manual cleanup
   - **Recommendation:** Option A with detailed logging

3. **Should we version the API for status changes?**
   - Option A: Keep v1, add new statuses (non-breaking)
   - Option B: Create v2 API with full status model
   - **Recommendation:** Option A, reserve v2 for other breaking changes

4. **Soft delete strategy?**
   - See section 3.3 for options
   - **Recommendation:** Option C (keep both with clear semantics)

5. **Event publishing for status changes?**
   - Should status changes always publish events?
   - Should we batch status change events?
   - **Recommendation:** Always publish, add batching later if needed
