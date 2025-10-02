# Simple Content Refactor — Remaining Work

**Last updated:** 2025-10-01

## Current Status Summary

**🎉 Core Refactoring: COMPLETE**

The simple-content project has been successfully refactored into a clean, reusable Go library with comprehensive test coverage and docker-based development environment.

### ✅ Completed (as of 2025-10-01)
- ✅ Core library `pkg/simplecontent` with unified Service interface
- ✅ All storage backends implemented and tested (memory, fs, s3)
- ✅ Repository implementations (memory, postgres) with migrations
- ✅ Docker compose environment (Postgres + MinIO) with helper scripts
- ✅ Complete test coverage (33 service tests, all storage backends, integration tests)
- ✅ Legacy packages deprecated with 3-month migration window
- ✅ Comprehensive documentation (CLAUDE.md, migration guide, docker setup, test audit)
- ✅ Status management operations (update status, query by status)
- ✅ Soft delete support throughout
- ✅ URL strategy system and object key generators

### ⏳ Remaining Work
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Test coverage reporting
- [ ] Backend comparison tables in README
- [ ] Legacy package removal (after 2026-01-01)

See detailed task lists below for implementation plans.

## Resolved Issues (2025-10-01)

- ✅ DTO cleanup: Unified on `DerivationType` and `Variant`
- ✅ Postgres wiring: Complete via config with schema support
- ✅ Server handlers: All REST endpoints implemented
- ✅ Legacy packages: Deprecated with migration guide
- ✅ Tests: Complete coverage (memory, fs, s3, integration)
- ✅ Docs: Status files reconciled, comprehensive documentation

## Plan (checklist)

1) Fix DTO and API inconsistencies

- [x] Remove `Category` from code; keep `DerivationType` (user-facing) on Content and `Variant` via relationship
- [x] Add GoDoc comments to exported types/methods in `pkg/simplecontent` (package doc, DTOs, Content/DerivedContent notes)
- [x] Document metadata strategy (first-class fields vs. JSON duplication) in README and package docs

2) Implement HTTP handlers (cmd/server-configured)

- [x] Content: Create/Get/Update/Delete/List
- [x] Content metadata: Set/Get
- [x] Objects: Create/Get/Delete/List-by-content
- [x] Upload/download: direct upload/download, presigned upload/download, preview URL
- [x] Consistent error → HTTP mapping using typed errors; structured JSON responses
- [x] Augment content responses with `category` (mirrors `Content.DerivationType`); plan to include `variant` via relationship lookup (see below)

3) Postgres wiring and migrations

- [x] Implement `pgxpool` wiring in `pkg/simplecontent/config.BuildService` with optional `CONTENT_DB_SCHEMA` (search_path)
- [x] Add `migrations/postgres/*` (timestamped) compatible with goose; dedicated schema `content` by default
- [x] Makefile targets for goose (up/down/status)
- [x] Update `docker-compose.yml` to include Postgres (and optional MinIO) for local integration tests
- [x] Add helper scripts (`scripts/docker-dev.sh`, `scripts/run-migrations.sh`, `scripts/init-db.sh`)
- [x] Document docker-compose setup in README and DOCKER_SETUP.md

4) Testing

- [x] Consolidate on `pkg/simplecontent` tests; port or remove legacy tests to avoid duplication
- [x] Add fs backend unit tests (temp dir) under `pkg/simplecontent/storage/fs`
- [x] Add service-level tests (derived creation inference; relationship listing)
- [x] Add integration tests (tagged) for Postgres and MinIO via docker-compose
- [x] Add basic httptest coverage for configured server (content create/list; object create/upload/download)
- [x] Port S3 storage tests from legacy package (pkg/simplecontent/storage/s3/s3_test.go)
- [x] Complete test coverage audit (TEST_COVERAGE_AUDIT.md)

5) Deprecate legacy packages

- [x] Add deprecation notices to `pkg/service`, `pkg/repository`, `pkg/storage` (comments); stop referencing them from any new code
- [x] Create comprehensive migration guide (MIGRATION_FROM_LEGACY.md)
- [x] Add deprecation notice to README
- [x] Set removal timeline: deprecated 2025-10-01, removal 2026-01-01 (3 months)
- [ ] Plan final removal once `cmd/server-configured` reaches parity and passes tests

6) Docs and CI

- [x] Reconcile `REFACTORING_STATUS.md` and `REFACTORING_COMPLETE.md` (single source of truth)
- [x] Update README: library usage, configured server setup, environment variables, backend matrix
- [x] Add CI: `go vet`, lint, unit tests, (optional) integration matrix; enforce `go mod tidy`
- [x] Add `claude.md` with conventions, API outline, and migration docs
- [x] GitHub Actions workflow with unit tests, integration tests, linting
- [x] Backend comparison tables in README
- [x] CI status badges in README

7) Derivation/Variant model

- [x] Normalize `derivation_type` (user-facing type on Content) and `variant` to lowercase in service
- [x] Infer `derivation_type` from `variant` (prefix before `_`) when missing
- [x] Include `variant` via relationship lookup in GET/list responses for derived items; omit both for originals
- [x] Postgres migrations now create `content_derived` with `variant` column (manual migration for existing DBs)
- [x] No unique index on `(parent_id, variant)` by design; multiple rows per variant may exist (e.g., history/backends/locales)

## Milestones

- M1: DTO/API consistency + first 4 handlers (content create/get, object create, upload)
- M2: Full handler set + Postgres wiring (no migrations runner yet)
- M3: Migrations + docker-compose integration + fs tests
- M4: Integration tests (Postgres/MinIO) + docs/CI
- M5: Deprecate legacy packages and remove after stability window

## Definition of Done

- Configured server provides the full REST surface and uses only `pkg/simplecontent`
- Postgres backend wired via config; migrations available and documented
- Unit tests cover memory/fs/s3 paths; integration tests pass locally via compose
- README and refactoring docs updated; CI enforces quality gates
- Legacy packages clearly deprecated or removed


## Notes

### Recently Completed

**Docker Compose Integration (2025-10-01)**
- Added Postgres service to docker-compose.yml (port 5433)
- MinIO service already configured (ports 9000/9001)
- Created helper scripts for development workflow (docker-dev.sh, run-migrations.sh, init-db.sh)
- Added comprehensive docker setup documentation (DOCKER_SETUP.md)
- Fixed: CreateDerivedContent query already includes both `variant` and `derivation_type` columns

**Legacy Package Deprecation (2025-10-01)**
- Added detailed deprecation notices to all legacy packages:
  - `pkg/service` → migrate to `pkg/simplecontent`
  - `pkg/repository` → migrate to `pkg/simplecontent/repo`
  - `pkg/storage` → migrate to `pkg/simplecontent/storage`
- Created comprehensive migration guide (MIGRATION_FROM_LEGACY.md)
- Added deprecation notice to README with timeline
- Deprecation date: 2025-10-01, Removal date: 2026-01-01 (3 months)

**Test Coverage Completion (2025-10-01)**
- Ported S3 storage tests to new package (pkg/simplecontent/storage/s3/s3_test.go)
- Added comprehensive S3/MinIO integration tests
- Completed test coverage audit (TEST_COVERAGE_AUDIT.md)
- **Result**: 100% test parity achieved across all storage backends
- Confidence level: Very High - no critical gaps remain

**Documentation & CI Complete (2025-10-01)**
- Reconciled refactoring status documents (archived outdated files)
- Rewrote REFACTORING_COMPLETE.md with current state
- Created comprehensive GitHub Actions CI workflow:
  - Multi-version Go matrix (1.21, 1.22, 1.23)
  - Unit tests with coverage reporting
  - Integration tests with Postgres + MinIO services
  - Linting (golangci-lint)
  - go mod tidy check
- Added backend comparison tables to README (storage + repository)
- Added CI status badges to README
- **Result**: Automated quality gates in place, comprehensive documentation