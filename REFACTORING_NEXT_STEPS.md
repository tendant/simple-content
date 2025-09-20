# Simple Content Refactor — Next Steps Plan

Last updated: 2025-09-08

## Scope

Track completion of the refactor centered on `pkg/simplecontent`, finishing the configured HTTP server, wiring Postgres, consolidating tests, and deprecating legacy packages. Complements REFACTORING_STATUS.md.

## Current Status (summary)

- Core library in `pkg/simplecontent` implemented (Service, interfaces, DTOs, typed errors).
- Repositories: memory implemented and used in tests; Postgres repository implemented with `schema.sql` but not yet wired by config.
- Storage backends: memory, fs, and s3 implemented under `pkg/simplecontent/storage`.
- Event/Preview: Noop and simple logging/image previewers available.
- Example servers:
  - `cmd/server-simple`: demo-only endpoint working pattern.
  - `cmd/server-configured`: wiring via env-driven `config`, but REST handlers are placeholders.
- Docs inconsistency: `REFACTORING_COMPLETE.md` claims completion; `REFACTORING_STATUS.md` shows pending work.

## Gaps / Issues

- DTO cleanup: Unified on `DerivationType` (user-facing) and `Variant` (specific). Removed `Category` field.
- `pkg/simplecontent/config` does not wire Postgres (returns error); migrations are not integrated.
- `cmd/server-configured` handlers are stubbed (`Not implemented yet`).
- Legacy packages (`pkg/service`, `pkg/repository`, `pkg/storage`) still present alongside new code.
- Tests: good coverage for memory; fs/s3 and server paths need tests; integration path not set up.
- Docs: status files conflict; README lacks clear server configuration examples and env matrix.

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
- [ ] Update `docker-compose.yml` to include Postgres (and optional MinIO) for local integration tests

4) Testing

- [ ] Consolidate on `pkg/simplecontent` tests; port or remove legacy tests to avoid duplication
- [x] Add fs backend unit tests (temp dir) under `pkg/simplecontent/storage/fs`
- [x] Add service-level tests (derived creation inference; relationship listing)
- [x] Add integration tests (tagged) for Postgres and MinIO via docker-compose
- [x] Add basic httptest coverage for configured server (content create/list; object create/upload/download)

5) Deprecate legacy packages

- [ ] Add deprecation notices to `pkg/service`, `pkg/repository`, `pkg/storage` (comments); stop referencing them from any new code
- [ ] Plan removal once `cmd/server-configured` reaches parity and passes tests

6) Docs and CI

- [ ] Reconcile `REFACTORING_STATUS.md` and `REFACTORING_COMPLETE.md` (single source of truth)
- [ ] Update README: library usage, configured server setup, environment variables, backend matrix
- [ ] Add CI: `go vet`, lint, unit tests, (optional) integration matrix; enforce `go mod tidy`
- [x] Add `claude.md` with conventions, API outline, and migration docs

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


## FIXME
    
• Edited pkg/simplecontent/repo/postgres/repository.go (+9 -7)
    368         query := `
    369   -        INSERT INTO content_derived (
    370   -            parent_id, content_id, variant, derivation_params,
    371   -            processing_metadata, created_at, updated_at, status
    372   -        ) VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), 'created')
    373   -        RETURNING parent_id, content_id, variant as derivation_type, derivation_params,
    374   -                  processing_metadata, created_at, updated_at, status`
    369   +             INSERT INTO content_derived (
    370   +                 parent_id, content_id, variant, derivation_type, derivation_params,
    371   +                 processing_metadata, created_at, updated_at, status
    372   +             ) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), 'created')
    373   +             RETURNING parent_id, content_id, variant as derivation_type, derivation_params,
    374   +                       processing_metadata, created_at, updated_at, status`
    375
    376         var derived simplecontent.DerivedContent
    377   +     derivationType := simplecontent.DerivationTypeFromVariant(params.DerivationType)
    378   +
    379         err := r.db.QueryRow(ctx, query,
    378   -             params.ParentID, params.DerivedContentID, params.DerivationType,
    380   +             params.ParentID, params.DerivedContentID, params.DerivationType, derivationType,
    381                 params.DerivationParams, params.ProcessingMetadata).Scan(