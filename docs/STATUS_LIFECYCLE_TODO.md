# Status Lifecycle Refactoring - TODO List

This document provides a detailed task breakdown for implementing the status lifecycle improvements outlined in `STATUS_LIFECYCLE_REFACTORING.md`.

## Pre-Refactoring Tasks

- [ ] Review and approve `STATUS_LIFECYCLE_REFACTORING.md` with team
- [ ] Answer open questions in refactoring document
- [ ] Create feature branch: `feature/status-lifecycle-refactoring`
- [ ] Audit existing database for invalid status values
- [ ] Create data cleanup script for pre-migration
- [ ] Set up tracking for backward compatibility issues

---

## Phase 1: Foundation (Sprint 1-2) 🔴

### 1.1 Status Validation

#### Code Changes
- [ ] Add `IsValid()` method to `ContentStatus` type in `pkg/simplecontent/types.go`
- [ ] Add `IsValid()` method to `ObjectStatus` type in `pkg/simplecontent/types.go`
- [ ] Add `ParseContentStatus()` function in `pkg/simplecontent/types.go`
- [ ] Add `ParseObjectStatus()` function in `pkg/simplecontent/types.go`
- [ ] Add validation helper `ValidateStatus()` for general use

#### Tests
- [ ] Create `pkg/simplecontent/types_test.go` if not exists
- [ ] Test `ContentStatus.IsValid()` with all valid values
- [ ] Test `ContentStatus.IsValid()` with invalid values
- [ ] Test `ObjectStatus.IsValid()` with all valid values
- [ ] Test `ObjectStatus.IsValid()` with invalid values
- [ ] Test `ParseContentStatus()` success cases
- [ ] Test `ParseContentStatus()` error cases
- [ ] Test `ParseObjectStatus()` success cases
- [ ] Test `ParseObjectStatus()` error cases

#### Documentation
- [ ] Add godoc comments for validation methods
- [ ] Update `CLAUDE.md` with status validation patterns
- [ ] Add validation examples to README

**Estimated Effort:** 2 days

---

### 1.2 Status Transition State Machine

#### Code Changes
- [ ] Create new package `pkg/simplecontent/status/`
- [ ] Create `pkg/simplecontent/status/transitions.go`
- [ ] Define `ContentTransitions` map with all valid transitions
- [ ] Define `ObjectTransitions` map with all valid transitions
- [ ] Implement `ValidateContentTransition(from, to)` function
- [ ] Implement `ValidateObjectTransition(from, to)` function
- [ ] Add `GetAllowedTransitions(currentStatus)` helper function
- [ ] Add `IsTerminalStatus(status)` helper function

#### Tests
- [ ] Create `pkg/simplecontent/status/transitions_test.go`
- [ ] Test all valid content status transitions
- [ ] Test invalid content status transitions return errors
- [ ] Test terminal status (deleted) has no outgoing transitions
- [ ] Test all valid object status transitions
- [ ] Test invalid object status transitions return errors
- [ ] Test transition validation with invalid statuses
- [ ] Test `GetAllowedTransitions()` returns correct next states
- [ ] Test `IsTerminalStatus()` correctly identifies terminal states

#### Documentation
- [ ] Add transition diagram to documentation
- [ ] Document state machine rules in `STATUS_LIFECYCLE_REFACTORING.md`
- [ ] Add transition examples to `CLAUDE.md`

**Estimated Effort:** 3 days

---

### 1.3 Expand Content Status Enums

#### Code Changes
- [ ] Add `ContentStatusUploading` constant to `pkg/simplecontent/types.go`
- [ ] Add `ContentStatusProcessing` constant to `pkg/simplecontent/types.go`
- [ ] Add `ContentStatusProcessed` constant to `pkg/simplecontent/types.go`
- [ ] Add `ContentStatusFailed` constant to `pkg/simplecontent/types.go`
- [ ] Add `ContentStatusArchived` constant to `pkg/simplecontent/types.go`
- [ ] Update `ContentStatus.IsValid()` to include new statuses
- [ ] Update `ContentTransitions` map with new status transitions
- [ ] Add status description comments for each status

#### Tests
- [ ] Update `types_test.go` to validate all new statuses
- [ ] Update transition tests for new status flows
- [ ] Test backward compatibility with existing 3-status model

#### Documentation
- [ ] Update status lifecycle diagram with new statuses
- [ ] Document when each new status should be used
- [ ] Add migration guide for clients expecting old statuses
- [ ] Update API documentation with new status values

**Estimated Effort:** 1 day

---

### 1.4 Add Status Update Methods

#### Code Changes - Interfaces
- [ ] Add `UpdateContentStatus()` method to `Service` interface in `interfaces.go`
- [ ] Add `UpdateObjectStatus()` method to `Service` interface in `interfaces.go`
- [ ] Add `GetContentByStatus()` method to `Service` interface
- [ ] Add `GetObjectsByStatus()` method to `Service` interface

#### Code Changes - Implementation
- [ ] Implement `UpdateContentStatus()` in `service_impl.go`
  - [ ] Fetch current content
  - [ ] Validate status transition
  - [ ] Update status with timestamp
  - [ ] Fire status change event
  - [ ] Handle errors properly
- [ ] Implement `UpdateObjectStatus()` in `service_impl.go`
  - [ ] Fetch current object
  - [ ] Validate status transition
  - [ ] Update status with timestamp
  - [ ] Fire status change event
  - [ ] Handle errors properly
- [ ] Implement `GetContentByStatus()` in `service_impl.go`
- [ ] Implement `GetObjectsByStatus()` in `service_impl.go`

#### Code Changes - Repository
- [ ] Add `GetContentByStatus()` to `Repository` interface
- [ ] Implement in `pkg/simplecontent/repo/memory/repository.go`
- [ ] Implement in `pkg/simplecontent/repo/postgres/repository.go`
- [ ] Add `GetObjectsByStatus()` to `Repository` interface
- [ ] Implement in memory repository
- [ ] Implement in postgres repository

#### Tests
- [ ] Test `UpdateContentStatus()` with valid transition
- [ ] Test `UpdateContentStatus()` with invalid transition (should fail)
- [ ] Test `UpdateContentStatus()` with non-existent content
- [ ] Test `UpdateContentStatus()` fires status change event
- [ ] Test `UpdateObjectStatus()` with valid transition
- [ ] Test `UpdateObjectStatus()` with invalid transition (should fail)
- [ ] Test `UpdateObjectStatus()` with non-existent object
- [ ] Test `UpdateObjectStatus()` fires status change event
- [ ] Test `GetContentByStatus()` returns correct results
- [ ] Test `GetObjectsByStatus()` returns correct results

#### Documentation
- [ ] Add godoc for new service methods
- [ ] Add usage examples to `CLAUDE.md`
- [ ] Update API reference

**Estimated Effort:** 4 days

---

## Phase 2: Business Logic (Sprint 3-4) 🟡

### 2.1 Status-Based Authorization

#### Code Changes - Download Operations
- [ ] Add status validation to `DownloadContent()`
  - [ ] Allow: uploaded, processed
  - [ ] Deny: deleted, processing, failed
  - [ ] Return descriptive errors
- [ ] Add status validation to `DownloadObject()`
  - [ ] Similar rules as content download

#### Code Changes - Upload Operations
- [ ] Add status validation to `UploadContent()`
  - [ ] Allow: created, failed (retry)
  - [ ] Deny: processing, deleted
- [ ] Add status validation to `UploadObject()`
  - [ ] Similar rules as content upload

#### Code Changes - Delete Operations
- [ ] Add status validation to `DeleteContent()`
  - [ ] Warn/prevent deletion of processing content
  - [ ] Add optional `force` flag for override
- [ ] Update `DeleteContent()` to use status transition validation
- [ ] Add status validation to `DeleteObject()`

#### Code Changes - Derived Content
- [ ] Add parent status validation to `CreateDerivedContent()`
  - [ ] Require parent status: uploaded or processed
  - [ ] Deny if parent is deleted, processing
- [ ] Add parent status validation to `UploadDerivedContent()`

#### Tests
- [ ] Test download denied for deleted content
- [ ] Test download denied for processing content
- [ ] Test download denied for failed content
- [ ] Test download allowed for uploaded content
- [ ] Test download allowed for processed content
- [ ] Test upload denied for deleted content
- [ ] Test upload denied for processing content
- [ ] Test upload allowed for created content
- [ ] Test upload allowed for failed content (retry)
- [ ] Test delete denied for processing content (without force)
- [ ] Test delete allowed for processing content (with force)
- [ ] Test derived content creation denied for non-uploaded parent
- [ ] Test derived content creation allowed for uploaded parent

#### Documentation
- [ ] Document status-based authorization rules
- [ ] Add authorization examples to API docs
- [ ] Update error message documentation

**Estimated Effort:** 3 days

---

### 2.2 Database Constraints

#### Migration Files
- [ ] Create migration: `migrations/postgres/YYYYMMDDHHMMSS_add_status_constraints.sql`
- [ ] Add CHECK constraint for `content.status`
- [ ] Add CHECK constraint for `object.status`
- [ ] Add CHECK constraint for `content_derived.status`
- [ ] Create index `idx_content_status_not_deleted`
- [ ] Create index `idx_object_status_uploaded`
- [ ] Create index `idx_object_status_processing`
- [ ] Add down migration (rollback)

#### Pre-Migration
- [ ] Create data audit script to find invalid statuses
- [ ] Create data cleanup script to fix invalid statuses
- [ ] Run cleanup on development database
- [ ] Verify no invalid statuses remain

#### Migration Execution
- [ ] Test migration on local database
- [ ] Test migration on staging database
- [ ] Test rollback migration
- [ ] Document migration process
- [ ] Execute on production (during maintenance window)

#### Tests
- [ ] Test database rejects invalid content status
- [ ] Test database rejects invalid object status
- [ ] Test database rejects invalid derived content status
- [ ] Test indexes improve query performance
- [ ] Test constraint enforcement doesn't break existing queries

#### Documentation
- [ ] Document valid status values in schema
- [ ] Add migration guide for DBAs
- [ ] Update deployment documentation

**Estimated Effort:** 2 days

---

### 2.3 Status Change Events

#### Code Changes - Interface
- [ ] Add `ContentStatusChanged()` to `EventSink` interface
- [ ] Add `ObjectStatusChanged()` to `EventSink` interface
- [ ] Update `NoOpEventSink` with new methods

#### Code Changes - Event Firing
- [ ] Fire `ContentStatusChanged()` in `UpdateContentStatus()`
- [ ] Fire `ContentStatusChanged()` in `UpdateContent()` (if status changed)
- [ ] Fire `ObjectStatusChanged()` in `UpdateObjectStatus()`
- [ ] Fire `ObjectStatusChanged()` in `UpdateObject()` (if status changed)
- [ ] Include old and new status in event payload

#### Code Changes - Event Consumers (Optional)
- [ ] Create example event logger
- [ ] Create example webhook publisher
- [ ] Create example metrics collector

#### Tests
- [ ] Test `ContentStatusChanged()` event fired on status update
- [ ] Test event includes correct old and new status values
- [ ] Test event includes content/object ID
- [ ] Test `ObjectStatusChanged()` event fired on status update
- [ ] Test event fired even when other operations fail (best-effort)
- [ ] Test event not fired when status doesn't change

#### Documentation
- [ ] Document event payload structure
- [ ] Add event handling examples
- [ ] Document event ordering guarantees (or lack thereof)

**Estimated Effort:** 2 days

---

### 2.4 Fix "active" Status Reference

#### Code Investigation
- [ ] Search codebase for all references to "active" status
- [ ] Identify all locations using undefined "active" status
- [ ] Determine intent of "active" status usage

#### Code Changes - Option 1 (Add Alias)
- [ ] Add `ContentStatusActive` as alias for `uploaded`
- [ ] Update documentation for alias
- [ ] Mark as deprecated in favor of explicit status

#### Code Changes - Option 2 (Fix Logic - Recommended)
- [ ] Replace `"active"` with proper status checks
- [ ] Update `GetContentDetails()` at line 924
- [ ] Use switch statement for status checking
- [ ] Define "ready" statuses: uploaded, processed

#### Tests
- [ ] Test `GetContentDetails()` with uploaded content (ready=true)
- [ ] Test `GetContentDetails()` with processed content (ready=true)
- [ ] Test `GetContentDetails()` with created content (ready=false)
- [ ] Test `GetContentDetails()` with processing content (ready=false)
- [ ] Test `GetContentDetails()` with deleted content (ready=false)

#### Documentation
- [ ] Document "ready" status definition
- [ ] Update API response documentation

**Estimated Effort:** 0.5 days

---

## Phase 3: Advanced Features (Sprint 5+) 🟢

### 3.1 Status History / Audit Trail

#### Database Schema
- [ ] Create migration: `migrations/postgres/YYYYMMDDHHMMSS_add_status_history.sql`
- [ ] Create `content_status_history` table
- [ ] Create `object_status_history` table
- [ ] Add indexes on content_id, object_id
- [ ] Add indexes on changed_at for time-based queries
- [ ] Test migration up and down

#### Code Changes - Types
- [ ] Add `StatusChange` type to `types.go`
- [ ] Add `StatusHistoryEntry` with metadata fields
- [ ] Add `changed_by` field to track user/system
- [ ] Add `reason` field for audit notes

#### Code Changes - Repository
- [ ] Add `RecordContentStatusChange()` to `Repository` interface
- [ ] Add `RecordObjectStatusChange()` to `Repository` interface
- [ ] Add `GetContentStatusHistory()` to `Repository` interface
- [ ] Add `GetObjectStatusHistory()` to `Repository` interface
- [ ] Implement in memory repository
- [ ] Implement in postgres repository

#### Code Changes - Service
- [ ] Update `UpdateContentStatus()` to record history
- [ ] Update `UpdateObjectStatus()` to record history
- [ ] Add `GetContentStatusHistory()` to `Service` interface
- [ ] Add `GetObjectStatusHistory()` to `Service` interface
- [ ] Implement service methods

#### Tests
- [ ] Test status history recorded on status change
- [ ] Test history includes old and new status
- [ ] Test history includes timestamp
- [ ] Test history can be queried by content_id
- [ ] Test history ordered by time (descending)
- [ ] Test multiple status changes create multiple history entries

#### Documentation
- [ ] Document status history schema
- [ ] Add query examples for audit trail
- [ ] Document retention policy (if any)

**Estimated Effort:** 3 days

---

### 3.2 Retry/Recovery Patterns

#### Code Changes - Status Enums
- [ ] Add `ContentStatusPending` constant
- [ ] Add `ContentStatusRetrying` constant
- [ ] Add `ContentStatusSuspended` constant
- [ ] Update `IsValid()` to include new statuses
- [ ] Update transitions map for retry flows

#### Code Changes - Metadata
- [ ] Add `RetryCount` field to `ContentMetadata`
- [ ] Add `MaxRetries` field to `ContentMetadata`
- [ ] Add `LastError` field to `ContentMetadata`
- [ ] Add `NextRetryAt` field to `ContentMetadata`

#### Code Changes - Service Methods
- [ ] Implement `RetryFailedContent()` method
  - [ ] Check current status is failed
  - [ ] Check retry count < max retries
  - [ ] Transition to retrying status
  - [ ] Increment retry count
  - [ ] Schedule next retry
- [ ] Implement `RetryFailedObject()` method
- [ ] Implement `GetRetryableContent()` method (returns failed content eligible for retry)
- [ ] Implement retry scheduler/worker (optional, async)

#### Tests
- [ ] Test retry denied for non-failed content
- [ ] Test retry allowed for failed content
- [ ] Test retry increments retry count
- [ ] Test retry denied when max retries exceeded
- [ ] Test retry updates next_retry_at timestamp
- [ ] Test retry transitions status from failed to retrying

#### Documentation
- [ ] Document retry strategy
- [ ] Add retry configuration examples
- [ ] Document max retry limits
- [ ] Add recovery workflow diagrams

**Estimated Effort:** 4 days

---

### 3.3 Consolidate Soft Delete

#### Decision
- [ ] Review options A, B, C in refactoring doc
- [ ] Team decision on approach
- [ ] Document chosen approach in `CLAUDE.md`

#### Code Changes - Option C (Keep Both - Recommended)
- [ ] Document semantic difference in `CLAUDE.md`:
  - `status = 'deleted'` - prevents business operations
  - `deleted_at` - enables recovery/purge workflows
- [ ] Ensure consistency: when status='deleted', deleted_at must be set
- [ ] Add validation in service layer
- [ ] Add database constraint (trigger) to enforce consistency

#### Tests
- [ ] Test delete sets both status and deleted_at
- [ ] Test query filters use deleted_at IS NULL
- [ ] Test deleted content cannot be downloaded
- [ ] Test deleted content can be listed with special flag
- [ ] Test purge workflow finds old deleted content

#### Documentation
- [ ] Document soft delete semantics
- [ ] Add recovery examples
- [ ] Add purge workflow documentation

**Estimated Effort:** 1 day

---

## Cross-Cutting Tasks

### Testing
- [ ] Run all unit tests
- [ ] Run all integration tests
- [ ] Add performance tests for status queries
- [ ] Test backward compatibility with old clients
- [ ] Load test with new status constraints

### Documentation
- [ ] Update main README with status lifecycle
- [ ] Update `CLAUDE.md` with status patterns
- [ ] Create status lifecycle diagram
- [ ] Update API documentation
- [ ] Add migration guide for API consumers
- [ ] Create developer guide for status handling

### Code Quality
- [ ] Run linter on all changed files
- [ ] Add godoc comments to all public functions
- [ ] Ensure consistent error messages
- [ ] Refactor duplicated status checking logic
- [ ] Add logging for status transitions

### Deployment
- [ ] Create deployment checklist
- [ ] Document rollback procedure
- [ ] Create database migration runbook
- [ ] Schedule maintenance window
- [ ] Prepare monitoring/alerts for new status values

---

## Review & Approval

- [ ] Code review for Phase 1 changes
- [ ] Code review for Phase 2 changes
- [ ] Code review for Phase 3 changes
- [ ] Security review of status-based authorization
- [ ] Performance review of database constraints
- [ ] API review for backward compatibility
- [ ] Documentation review
- [ ] Final approval from tech lead

---

## Post-Implementation

- [ ] Monitor error logs for status-related issues
- [ ] Monitor metrics for status distribution
- [ ] Collect feedback from API consumers
- [ ] Plan deprecation of old patterns (if any)
- [ ] Create follow-up tasks for improvements
- [ ] Update team knowledge base

---

## Estimated Timeline

- **Phase 1 (Foundation):** 2 sprints (10 working days)
- **Phase 2 (Business Logic):** 2 sprints (10 working days)
- **Phase 3 (Advanced Features):** 2-3 sprints (15 working days)
- **Total:** ~35 working days (7 weeks)

**Dependencies:**
- Phase 2 depends on Phase 1 completion
- Phase 3 can run in parallel with Phase 2 (different focus areas)
- Database migrations should be staggered across sprints

**Risk Mitigation:**
- Feature flags for gradual rollout
- Backward compatibility mode for initial release
- Comprehensive test coverage before each phase
- Staged deployment (dev → staging → production)

---

## Success Metrics

- [ ] Zero invalid status values in database
- [ ] 100% of status transitions validated
- [ ] All business operations enforce status-based authorization
- [ ] Status change events published for all transitions
- [ ] API consumers successfully handle new statuses
- [ ] Performance metrics within acceptable range
- [ ] Zero production incidents related to status handling

---

## Notes

- Keep `STATUS_LIFECYCLE_REFACTORING.md` as reference documentation
- Update this TODO as tasks are completed
- Use task checkboxes to track progress
- Add sub-tasks as needed during implementation
- Document any deviations from plan with reasons
