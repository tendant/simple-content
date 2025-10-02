# Test Coverage Audit: Legacy vs New Packages

**Audit Date:** 2025-10-01

## Summary

This document compares test coverage between legacy packages (`pkg/service`, `pkg/repository`, `pkg/storage`) and new packages (`pkg/simplecontent`).

### Overall Coverage

| Package Type | Legacy Tests | New Tests | Status |
|--------------|-------------|-----------|--------|
| Service | 22 tests | 33 tests | ✅ Better coverage |
| Repository | 6 test files | 1 test file + integration tests | ✅ Consolidated |
| Storage | 2 test files (fs, s3) | 2 test files (fs, memory) | ⚠️ S3 needs porting |

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
- ⚠️ **S3 Storage Tests** (`pkg/storage/s3/s3_test.go`)
  - Presigned URL generation
  - S3-specific error handling
  - MinIO compatibility
  - Multipart upload support (if applicable)

#### Medium Priority
- ⚠️ **Max Depth Limit** (`TestContentService_CreateDerivedContent_MaxDepthLimit`)
  - Tests recursive derivation depth limits
  - Prevents infinite derivation chains
  - Should be ported to `pkg/simplecontent`

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

### Immediate Actions

1. **Port S3 Storage Tests** (Priority: HIGH)
   ```bash
   # Create pkg/simplecontent/storage/s3/s3_test.go
   # Port tests from pkg/storage/s3/s3_test.go
   # Add MinIO integration tests
   ```

2. **Port Max Depth Limit Test** (Priority: MEDIUM)
   ```bash
   # Add to pkg/simplecontent/service_test.go or derived_service_test.go
   # Test recursive derivation depth limiting
   ```

3. **Add Deprecation Notices to Legacy Tests** (Priority: LOW)
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

**Overall Assessment:** ✅ **Good Coverage**

The new `pkg/simplecontent` package has comprehensive test coverage that meets or exceeds the legacy package coverage in most areas.

### Coverage Status:
- **Service Layer:** ✅ Excellent (33 tests vs 22 legacy tests)
- **Repository Layer:** ✅ Good (integration tests + service tests)
- **Storage Layer:** ⚠️ Needs S3 tests ported

### Action Items:
1. Port S3 storage tests (HIGH priority)
2. Add max depth limit test (MEDIUM priority)
3. Add deprecation notices to legacy tests (LOW priority)

### Timeline:
- **Week 1:** Port S3 storage tests
- **Week 2:** Add max depth limit test
- **Week 3:** Review and add any additional edge case tests
- **Before Legacy Removal (2026-01-01):** Ensure 100% feature parity in tests

### Confidence Level:
**High** - The new test suite provides better coverage overall, with only minor gaps that can be filled quickly.
