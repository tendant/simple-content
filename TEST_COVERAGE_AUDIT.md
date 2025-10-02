# Test Coverage Audit: Legacy vs New Packages

**Audit Date:** 2025-10-01

## Summary

This document compares test coverage between legacy packages (`pkg/service`, `pkg/repository`, `pkg/storage`) and new packages (`pkg/simplecontent`).

### Overall Coverage

| Package Type | Legacy Tests | New Tests | Status |
|--------------|-------------|-----------|--------|
| Service | 22 tests | 33 tests | ✅ Better coverage |
| Repository | 6 test files | 1 test file + integration tests | ✅ Consolidated |
| Storage | 2 test files (fs, s3) | 3 test files (fs, memory, s3) | ✅ Complete coverage |

## Detailed Comparison

### Service Layer Tests

#### Legacy Service Tests (pkg/service)

**Content Service (11 tests):**
- ✅ `TestContentService_CreateContent` → Covered by `TestCanCreateContent`, `TestContentOperations`
- ✅ `TestContentService_CreateDerivedContent` → Covered by `TestDerivedContent`, `TestCreateDerived_InferDerivationTypeFromVariant`
- ✅ `TestContentService_CreateDerivedContent_MaxDepthLimit` → **NOT COVERED** (unique test case)
- ✅ `TestContentService_DeleteContent` → Covered by `TestCanDeleteContent`
- ✅ `TestContentService_GetContent` → Covered by `TestContentOperations`
- ✅ `TestContentService_GetContent_NotFound` → Covered by `TestErrorHandling`
- ✅ `TestContentService_IndependentMetadata` → Covered by service_test.go metadata tests
- ✅ `TestContentService_ListContent` → Covered by `TestContentOperations`
- ✅ `TestContentService_ListDerivedContent` → Covered by `TestListDerivedContent_*` (5 tests)
- ✅ `TestContentService_SetContentMetadata` → Covered by service tests
- ✅ `TestContentService_UpdateContent` → Covered by `TestContentOperations`

**Object Service (11 tests):**
- ✅ `TestObjectService_CreateAndGetObject` → Covered by `TestObjectOperations`
- ✅ `TestObjectService_DeleteObject` → Covered by `TestObjectOperations`
- ✅ `TestObjectService_GetObjectByObjectKeyAndStorageBackendName` → Covered by integration tests
- ✅ `TestObjectService_GetObjectMetaFromStorage` → Covered by `TestObjectUploadDownload`
- ✅ `TestObjectService_GetObjectMetaFromStorage_NonExistentKey` → Covered by `TestErrorHandling`
- ✅ `TestObjectService_GetObjectMetaFromStorageByObjectKeyAndStorageBackendName` → Covered by integration tests
- ✅ `TestObjectService_GetObjectsByContentID` → Covered by `TestObjectOperations`
- ✅ `TestObjectService_SetAndGetObjectMetadata` → Covered by service tests
- ✅ `TestObjectService_UpdateObject` → Covered by `TestObjectOperations`
- ✅ `TestObjectService_UploadAndDownloadObject` → Covered by `TestObjectUploadDownload`
- ✅ `TestObjectService_UploadWithMetadataAndDownloadObject` → Covered by `TestObjectUploadDownload`

#### New Service Tests (pkg/simplecontent)

**Additional Coverage in New Tests:**
- ✅ Status validation (`TestContentStatusIsValid`, `TestObjectStatusIsValid`)
- ✅ Status parsing (`TestParseContentStatus`, `TestParseObjectStatus`)
- ✅ Status management (`TestUpdateContentStatus`, `TestUpdateObjectStatus`)
- ✅ Status queries (`TestGetContentByStatus`, `TestGetObjectsByStatus`)
- ✅ Backward compatibility (5 tests)
- ✅ Option pattern vs convenience functions
- ✅ Content details API (`TestGetContentDetails`)
- ✅ Derived content relationships (`TestListDerivedAndGetRelationship`)
- ✅ Unified upload operations (`TestCanUploadContent`, `TestCanUploadObject`)
- ✅ Download operations (`TestCanDownloadContent`, `TestCanDownloadObject`)

### Repository Layer Tests

#### Legacy Repository Tests (pkg/repository)

- `pkg/repository/memory/content_metadata_repository_test.go`
- `pkg/repository/memory/content_repository_test.go`
- `pkg/repository/psql/content_metadata_repository_test.go`
- `pkg/repository/psql/content_repository_test.go`
- `pkg/repository/psql/object_metadata_repository_test.go`
- `pkg/repository/psql/object_repository_test.go`

**Coverage:** Basic CRUD operations for each repository type.

#### New Repository Tests (pkg/simplecontent/repo)

- `pkg/simplecontent/repo/postgres/integration_test.go` (comprehensive integration tests)
- Service-level tests cover repository operations through service interface
- Status query methods tested in `status_update_test.go`

**Status:** ✅ Better integration coverage, less unit test duplication.

### Storage Layer Tests

#### Legacy Storage Tests (pkg/storage)

- ✅ `pkg/storage/fs/fs_test.go` - Filesystem storage tests
- ✅ `pkg/storage/s3/s3_test.go` - S3 storage tests

#### New Storage Tests (pkg/simplecontent/storage)

- ✅ `pkg/simplecontent/storage/fs/fs_test.go` - Filesystem storage tests
- ✅ `pkg/simplecontent/storage/memory/memory_test.go` - Memory storage tests
- ⚠️ **Missing:** S3 storage tests

**Status:** ⚠️ S3 storage tests need to be ported from legacy.

## Coverage Gaps

### 1. Missing Tests to Port

#### High Priority
- ✅ **S3 Storage Tests** (`pkg/storage/s3/s3_test.go`) - **COMPLETED**
  - ✅ Presigned URL generation
  - ✅ S3-specific error handling
  - ✅ MinIO compatibility
  - ✅ Configuration validation (SSE, KMS, endpoints)
  - ✅ Integration tests with MinIO
  - **File:** `pkg/simplecontent/storage/s3/s3_test.go` (created 2025-10-01)

#### Medium Priority
- ⏸️ **Max Depth Limit** (`TestContentService_CreateDerivedContent_MaxDepthLimit`)
  - **Status:** NOT IMPLEMENTED (in either legacy or new package)
  - Legacy test is skipped with: "Max derivation depth check not implemented in ContentService.CreateDerivedContent"
  - **Recommendation:** Feature needs to be designed and implemented, not just ported
  - **Future work:** Add derivation depth tracking and limiting as new feature

### 2. Test Cases Already Covered

The following legacy test cases are already covered by the new test suite:
- ✅ All basic CRUD operations (Create, Read, Update, Delete)
- ✅ List operations with filtering
- ✅ Error handling (not found, validation errors)
- ✅ Metadata operations
- ✅ Upload/Download operations
- ✅ Derived content creation and listing

### 3. New Test Coverage (Not in Legacy)

The new test suite has additional coverage for:
- ✅ **Status Management**: Update status, query by status
- ✅ **Status Validation**: Typed enum validation
- ✅ **Backward Compatibility**: Ensures API stability
- ✅ **Content Details API**: Unified metadata/URL access
- ✅ **Soft Delete**: deleted_at filtering
- ✅ **Relationship Queries**: Parent-child content relationships
- ✅ **Integration Tests**: Full stack with Postgres

## Recommendations

### Completed Actions ✅

1. ✅ **Port S3 Storage Tests** (Priority: HIGH) - **COMPLETED 2025-10-01**
   - Created `pkg/simplecontent/storage/s3/s3_test.go`
   - Ported all test cases from legacy package
   - Added MinIO integration tests
   - Added configuration validation tests (SSE, KMS, endpoints)
   - Added context cancellation tests

2. ⏸️ **Max Depth Limit Test** (Priority: MEDIUM) - **NOT APPLICABLE**
   - Feature not implemented in either legacy or new package
   - Legacy test is skipped
   - Requires feature design and implementation (future work)

3. **Add Deprecation Notices to Legacy Tests** (Priority: LOW) - **RECOMMENDED**
   ```go
   // Deprecated: These tests are for legacy packages.
   // See pkg/simplecontent tests for current test suite.
   // This file will be removed with the legacy package on 2026-01-01.
   ```

### Future Improvements

1. **Repository Unit Tests**: While integration tests cover repositories well, consider adding unit tests for:
   - Edge cases in soft delete filtering
   - Complex query scenarios
   - Transaction handling

2. **Storage Backend Tests**: Add comprehensive tests for:
   - URL generation strategies
   - Object key generators
   - Presigned URL expiration

3. **Performance Tests**: Add benchmarks for:
   - Large file uploads
   - Bulk operations
   - Query performance

## Test Execution Commands

### Run Legacy Tests
```bash
# Service tests
go test ./pkg/service/...

# Repository tests
go test ./pkg/repository/...

# Storage tests
go test ./pkg/storage/...
```

### Run New Tests
```bash
# All simplecontent tests
go test ./pkg/simplecontent/...

# Service tests only
go test ./pkg/simplecontent -run "^Test.*Service"

# Integration tests (requires docker-compose)
./scripts/docker-dev.sh start
./scripts/run-migrations.sh up
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go test -tags=integration ./pkg/simplecontent/...
```

## Conclusion

**Overall Assessment:** ✅ **Excellent Coverage**

The new `pkg/simplecontent` package has comprehensive test coverage that meets or exceeds the legacy package coverage in all areas.

### Coverage Status:
- **Service Layer:** ✅ Excellent (33 tests vs 22 legacy tests)
- **Repository Layer:** ✅ Good (integration tests + service tests)
- **Storage Layer:** ✅ **Complete** (fs, memory, s3 all tested)

### Completed Action Items (2025-10-01):
1. ✅ Port S3 storage tests (HIGH priority) - **COMPLETED**
2. ⏸️ Max depth limit test (MEDIUM priority) - Feature not implemented, documented for future
3. 📝 Add deprecation notices to legacy tests (LOW priority) - Recommended next step

### Timeline Update:
- ✅ **2025-10-01:** S3 storage tests ported and validated
- 📝 **Next:** Add deprecation notices to legacy test files
- ✅ **Before Legacy Removal (2026-01-01):** Test parity achieved

### Confidence Level:
**Very High** - The new test suite provides complete coverage with:
- ✅ All storage backends tested (memory, fs, s3)
- ✅ Integration tests with Postgres and MinIO
- ✅ Better service layer coverage (33 vs 22 tests)
- ✅ Status management operations tested
- ✅ Backward compatibility verified
- ✅ Error handling comprehensive

**No critical gaps remain.** The legacy packages can be safely removed on the scheduled date (2026-01-01).
