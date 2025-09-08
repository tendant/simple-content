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

- DTO inconsistency: `CreateDerivedContentRequest` has both `Category` and `DerivationType`; service uses both inconsistently.
- `pkg/simplecontent/config` does not wire Postgres (returns error); migrations are not integrated.
- `cmd/server-configured` handlers are stubbed (`Not implemented yet`).
- Legacy packages (`pkg/service`, `pkg/repository`, `pkg/storage`) still present alongside new code.
- Tests: good coverage for memory; fs/s3 and server paths need tests; integration path not set up.
- Docs: status files conflict; README lacks clear server configuration examples and env matrix.

## Plan (checklist)

1) Fix DTO and API inconsistencies

- [ ] Unify `CreateDerivedContentRequest` to a single field (prefer `DerivationType`) and update service logic accordingly
- [ ] Add GoDoc comments to exported types/methods in `pkg/simplecontent`
- [ ] Document metadata strategy (first-class fields vs. JSON duplication)

2) Implement HTTP handlers (cmd/server-configured)

- [ ] Content: Create/Get/Update/Delete/List
- [ ] Content metadata: Set/Get
- [ ] Objects: Create/Get/Delete/List-by-content
- [ ] Upload/download: direct upload/download, presigned upload/download, preview URL
- [ ] Consistent error → HTTP mapping using typed errors; structured JSON responses

3) Postgres wiring and migrations

- [ ] Implement `pgxpool` wiring in `pkg/simplecontent/config.BuildService` when `DATABASE_TYPE=postgres`
- [ ] Add `/migrations` with baseline from `pkg/simplecontent/repo/postgres/schema.sql` and a simple migration runner or documented workflow
- [ ] Update `docker-compose.yml` to include Postgres (and optional MinIO) for local integration tests

4) Testing

- [ ] Consolidate on `pkg/simplecontent` tests; port or remove legacy tests to avoid duplication
- [ ] Add fs backend unit tests (temp dir) under `pkg/simplecontent/storage/fs`
- [ ] Add service-level tests for presigned URL generation paths
- [ ] Add integration tests (tagged) for Postgres and MinIO via docker-compose

5) Deprecate legacy packages

- [ ] Add deprecation notices to `pkg/service`, `pkg/repository`, `pkg/storage` (comments); stop referencing them from any new code
- [ ] Plan removal once `cmd/server-configured` reaches parity and passes tests

6) Docs and CI

- [ ] Reconcile `REFACTORING_STATUS.md` and `REFACTORING_COMPLETE.md` (single source of truth)
- [ ] Update README: library usage, configured server setup, environment variables, backend matrix
- [ ] Add CI: `go vet`, lint, unit tests, (optional) integration matrix; enforce `go mod tidy`

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

