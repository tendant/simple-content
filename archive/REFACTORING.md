# Refactoring Plan: simple-content

Goal: Restructure `simple-content` into a reusable Go library with pluggable storage/repository backends and an optional HTTP server.

---

## Current Issues
- Core logic coupled to HTTP handlers and environment config.
- Storage/backend implementations not cleanly pluggable (only memory exists).
- Service layer knows too much about persistence and transport.
- Initialization path is server-first (hard to embed in another app).

---

## Target Shape

- **`pkg/simplecontent`**: Core library (public API).
- **Interfaces**: `Repository`, `BlobStore`, `EventSink`, `Previewer`.
- **Adapters**: 
  - Repositories: `repo/memory`, `repo/postgres`
  - Storage: `storage/memory`, `storage/fs`, `storage/s3`
- **Optional Server**: Thin HTTP layer in `/cmd/server` (imports library).
- **Config**: Library built with functional options (no env reads).

---

## Step-by-Step Refactor

### Phase 1: Foundations
- [ ] Create `/pkg/simplecontent` package
- [ ] Move domain types (Content, Object) here
- [ ] Define `Service` interface, DTOs, typed errors
- [ ] Introduce functional options (`WithRepo`, `WithStorage`, etc.)

### Phase 2: Interfaces
- [ ] Define `BlobStore` interface
- [ ] Move memory storage → `/pkg/storage/memory`
- [ ] Define `Repository` interface
- [ ] Move DB code into `/pkg/repo/postgres`
- [ ] Create `/pkg/repo/memory` for testing

### Phase 3: Service Layer
- [ ] Implement orchestration in `usecase_content.go` and `usecase_object.go`
- [ ] Add idempotency handling, status transitions, checksum/size capture

### Phase 4: HTTP Server
- [ ] Move handlers to `/cmd/server` or `/pkg/httpserver`
- [ ] Refactor to call library only (no direct DB/S3)
- [ ] Add OpenAPI doc generation

### Phase 5: Config & Options
- [ ] Server: read env/flags, construct repo/storage backends
- [ ] Pass into `simplecontent.New(...)`

### Phase 6: Backends
- [ ] Implement `storage/fs`
- [ ] Implement `storage/s3` (MinIO/AWS)
- [ ] Support presigned URLs (optional)

### Phase 7: Repository Improvements
- [ ] Add SQL migrations in `/migrations`
- [ ] Adopt `sqlc` for Postgres queries

### Phase 8: Extensibility
- [ ] Define `EventSink` + `Previewer` interfaces
- [ ] Provide `noop` implementations
- [ ] Fire lifecycle events (`ContentCreated`, `ObjectUploaded`, etc.)

### Phase 9: Testing
- [ ] Unit tests with memory repo + storage
- [ ] Integration tests with Postgres + MinIO (docker compose)
- [ ] Golden tests for metadata serialization
- [ ] Race detector + fuzz tests

### Phase 10: Docs & Examples
- [ ] Update `README` with "Embed as library" examples
- [ ] Add `/examples/` with small programs
- [ ] Add CHANGELOG

### Phase 11: CI/CD
- [ ] GitHub Actions: lint, unit tests, integration tests, `go vet`, race detector
- [ ] Build & push Docker image for server

### Phase 12: Release
- [ ] Tag `v0.1.0` once API stabilizes
- [ ] Maintain semantic versioning

---

## Example Usage

```go
repo := repomemory.New()
store := storagememory.New()

svc, _ := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore(store),
)

c, _ := svc.CreateContent(ctx, "owner-1", "tenant-1", map[string]any{"filename":"demo.txt"})
obj, _ := svc.CreateObject(ctx, c.ID, "memory", 1)

f, _ := os.Open("demo.txt")
defer f.Close()
_, _ = svc.PutObject(ctx, obj.ID, f)