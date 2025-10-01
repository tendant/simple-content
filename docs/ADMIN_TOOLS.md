# Admin Tools for In-Memory vs PostgreSQL

Comprehensive guide for choosing the right admin tool based on your database type.

## Quick Comparison

| Database Type | Best Admin Tool | Why |
|---------------|----------------|-----|
| **In-Memory** | HTTP Admin API or Admin Shell | Direct access to same memory instance |
| **PostgreSQL** | Standalone CLI (`cmd/admin`) | No storage setup needed, just database |

---

## For In-Memory Database

### Option 1: HTTP Admin API ⭐ **Recommended**

**Best for:** Development, testing, API integration

```bash
# Start server with admin API enabled
ENABLE_ADMIN_API=true DATABASE_TYPE=memory go run ./cmd/server-configured

# In another terminal, use HTTP API
curl http://localhost:8080/api/v1/admin/contents
curl http://localhost:8080/api/v1/admin/contents/count
curl http://localhost:8080/api/v1/admin/contents/stats
```

**Pros:**
- ✅ Accesses same memory instance as server
- ✅ Works while server is running
- ✅ Full REST API for automation
- ✅ Can use any HTTP client (curl, Postman, scripts)

**Cons:**
- ⚠️ Requires server to be running
- ⚠️ Should add authentication in production

**Example with sample data:**

```bash
# Terminal 1: Start server
ENABLE_ADMIN_API=true DATABASE_TYPE=memory PORT=8080 go run ./cmd/server-configured &

# Terminal 2: Create some test data
curl -X POST http://localhost:8080/api/v1/contents \
  -H "Content-Type: application/json" \
  -d '{
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "650e8400-e29b-41d4-a716-446655440001",
    "name": "Test Document",
    "document_type": "application/pdf"
  }'

# Terminal 2: Query via admin API
curl http://localhost:8080/api/v1/admin/contents?limit=10
curl http://localhost:8080/api/v1/admin/contents/count
curl http://localhost:8080/api/v1/admin/contents/stats | jq
```

### Option 2: Interactive Admin Shell

**Best for:** Debugging, manual inspection, development

```bash
# Start server in admin shell mode
ADMIN_SHELL=true ENABLE_ADMIN_API=true DATABASE_TYPE=memory \
  go run ./cmd/server-configured
```

**Interactive commands:**

```
admin> help
admin> list
admin> count
admin> stats
admin> get <content-id>
admin> exit
```

**Pros:**
- ✅ Interactive REPL interface
- ✅ Direct access to in-memory data
- ✅ Easy to explore data manually
- ✅ Great for debugging

**Cons:**
- ⚠️ Not scriptable
- ⚠️ Blocks server (can't handle HTTP requests)

**Example session:**

```bash
# Start admin shell
ADMIN_SHELL=true ENABLE_ADMIN_API=true DATABASE_TYPE=memory \
  go run ./cmd/server-configured

# Inside admin shell:
admin> list
No contents found

admin> count
Total count: 0

admin> stats
Total Count: 0

admin> help
[shows help]

admin> exit
Goodbye!
```

### Option 3: Programmatic Access

**Best for:** Unit tests, integration tests, custom tools

```go
package main

import (
    "context"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func main() {
    // Create shared in-memory repository
    repo := memory.New()

    // Create regular service
    svc, _ := simplecontent.New(
        simplecontent.WithRepository(repo),
        // ... other options
    )

    // Create admin service sharing same repo
    adminSvc := admin.New(repo)

    // Both access the same data!
    ctx := context.Background()

    // Add data via regular service
    svc.CreateContent(ctx, simplecontent.CreateContentRequest{...})

    // Query via admin service
    stats, _ := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{})
    fmt.Printf("Total: %d\n", stats.Statistics.TotalCount)
}
```

---

## For PostgreSQL Database

### Standalone CLI Tool ⭐ **Recommended**

**Best for:** Operations, monitoring, scripting, production

```bash
# Build once
go build -o admin ./cmd/admin

# Configure database
export DATABASE_URL="postgres://user:pass@localhost:5432/dbname"
export DATABASE_TYPE=postgres
export DB_SCHEMA=content

# Use anywhere
./admin list
./admin count
./admin stats
./admin list --tenant-id=<uuid> --json
```

**Pros:**
- ✅ **No storage backends required** - just database
- ✅ Works without running server
- ✅ Lightweight single binary
- ✅ Easy to deploy (copy one file)
- ✅ Perfect for scripting and automation
- ✅ Read-only by design (if using read-only DB user)

**Cons:**
- ❌ Doesn't work with in-memory (different process)

**Complete example:**

```bash
# Setup
export DATABASE_URL="postgres://admin:secret@localhost:5432/content_db"
export DATABASE_TYPE=postgres

# List all contents
./admin list

# Filter by tenant
./admin list --tenant-id=550e8400-e29b-41d4-a716-446655440000

# Count by status
./admin count --status=uploaded

# Get statistics as JSON
./admin stats --json | jq '.statistics.by_tenant'

# Export to file
./admin list --limit=10000 --json > export.json

# Scripting example
for tenant_id in $(cat tenant_ids.txt); do
  count=$(./admin count --tenant-id=$tenant_id --json | jq -r '.count')
  echo "Tenant $tenant_id: $count contents"
done
```

### HTTP Admin API (Also Works)

```bash
# Start server connected to PostgreSQL
export DATABASE_URL="postgres://user:pass@localhost/dbname"
export DATABASE_TYPE=postgres
export ENABLE_ADMIN_API=true

go run ./cmd/server-configured

# Use HTTP API
curl http://localhost:8080/api/v1/admin/contents
curl http://localhost:8080/api/v1/admin/contents/count
curl http://localhost:8080/api/v1/admin/contents/stats
```

**When to use:**
- Server is already running
- Want web UI or remote access
- Need real-time data

---

## Decision Matrix

### Choose HTTP Admin API when:
- ✅ Using **in-memory** database
- ✅ Server is already running
- ✅ Need **real-time** access to live data
- ✅ Building web dashboards or UIs
- ✅ Want REST API for integration

### Choose Standalone CLI when:
- ✅ Using **PostgreSQL** database
- ✅ Want **minimal dependencies**
- ✅ Need to query **without running server**
- ✅ Writing **scripts and automation**
- ✅ Prefer **command-line tools**
- ✅ Running **operations tasks**

### Choose Admin Shell when:
- ✅ Using **in-memory** for development
- ✅ Want **interactive exploration**
- ✅ Debugging during development
- ✅ Don't need HTTP server

---

## Common Scenarios

### Development with In-Memory

```bash
# Run server with both HTTP and admin enabled
ENABLE_ADMIN_API=true DATABASE_TYPE=memory PORT=8080 \
  go run ./cmd/server-configured

# Terminal 2: Use HTTP API to inspect
curl http://localhost:8080/api/v1/admin/contents
```

### Testing with In-Memory

```go
// In test code - share repository
repo := memory.New()
svc, _ := simplecontent.New(simplecontent.WithRepository(repo))
adminSvc := admin.New(repo)

// Test can use both
svc.CreateContent(ctx, req)
count, _ := adminSvc.CountContents(ctx, countReq)
assert.Equal(t, int64(1), count.Count)
```

### Production with PostgreSQL

```bash
# One-time setup
go build -o admin ./cmd/admin
scp admin production-server:/usr/local/bin/

# On production server
export DATABASE_URL="$PROD_DATABASE_URL"
export DATABASE_TYPE=postgres

# Quick health check
admin count

# Generate report
admin stats --json > daily_report.json

# Monitor specific tenant
admin count --tenant-id=$CUSTOMER_TENANT_ID
```

### Operations & Monitoring

```bash
# Standalone CLI for ops
./admin stats --json | telegraf --config monitoring.conf

# Or HTTP API if server running
curl http://metrics-server:8080/api/v1/admin/contents/stats | \
  jq '.statistics' > metrics.json
```

---

## Environment Variables Summary

### For HTTP Admin API

```bash
ENABLE_ADMIN_API=true          # Enable admin endpoints
DATABASE_TYPE=memory|postgres   # Database type
DATABASE_URL=<connection-string> # For PostgreSQL
DB_SCHEMA=content              # PostgreSQL schema
PORT=8080                      # Server port
```

### For Standalone CLI

```bash
DATABASE_TYPE=postgres         # Required
DATABASE_URL=<connection-string> # Required
DB_SCHEMA=content              # Optional, default: content
```

### For Admin Shell

```bash
ADMIN_SHELL=true               # Enable shell mode
ENABLE_ADMIN_API=true          # Required for shell
DATABASE_TYPE=memory           # Usually memory for dev
```

---

## Security Considerations

### In-Memory (Development)
- Usually runs in development environment
- Less sensitive (ephemeral data)
- Still protect HTTP endpoints with auth

### PostgreSQL (Production)
- **Use read-only database user for CLI**
  ```sql
  CREATE USER admin_readonly WITH PASSWORD 'secure';
  GRANT CONNECT ON DATABASE content_db TO admin_readonly;
  GRANT USAGE ON SCHEMA content TO admin_readonly;
  GRANT SELECT ON ALL TABLES IN SCHEMA content TO admin_readonly;
  ```
- **Protect database credentials**
- **Use VPN/bastion for remote access**
- **Audit all admin operations**
- **Add authentication to HTTP admin endpoints**

---

## Quick Reference

| Task | In-Memory | PostgreSQL |
|------|-----------|------------|
| List contents | `curl localhost:8080/api/v1/admin/contents` | `./admin list` |
| Count | `curl localhost:8080/api/v1/admin/contents/count` | `./admin count` |
| Statistics | `curl localhost:8080/api/v1/admin/contents/stats` | `./admin stats` |
| Filter by tenant | Add `?tenant_id=<uuid>` to URL | `./admin list --tenant-id=<uuid>` |
| JSON output | Already JSON | Add `--json` flag |
| Interactive mode | Set `ADMIN_SHELL=true` | Not available |

---

## Examples Repository

See working examples in:
- `examples/admin-operations/` - Programmatic admin API usage
- `cmd/admin/demo.sh` - CLI tool demonstration
- `cmd/server-configured/` - HTTP admin API integration
