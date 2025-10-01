# Admin Shell

Interactive admin shell for Simple Content service. **Best for in-memory databases** where you need to inspect data in the same process.

## Overview

The admin shell provides an interactive REPL (Read-Eval-Print Loop) for administrative operations. It shares the same in-memory data as the service, making it perfect for development and debugging.

## When to Use

✅ **Use admin shell when:**
- Working with **in-memory database** during development
- Need **interactive exploration** of data
- Debugging and manual inspection
- Learning the data structure

❌ **Don't use admin shell when:**
- Using PostgreSQL (use `cmd/admin` CLI instead)
- Need automation/scripting (use HTTP Admin API)
- Server needs to handle requests (shell blocks the process)

## Installation

```bash
# Build
go build -o admin-shell ./cmd/admin-shell

# Or run directly
go run ./cmd/admin-shell
```

## Usage

### Basic Usage

```bash
# Start with in-memory database
DATABASE_TYPE=memory ./admin-shell
```

### Using .env File (Recommended)

```bash
# Copy example configuration
cp .env.example .env

# Edit .env with your settings (usually DATABASE_TYPE=memory)
nano .env

# Run shell (config loaded from .env automatically)
./admin-shell
```

### With Environment Variables

```bash
# Configure via environment
export DATABASE_TYPE=memory
export ENABLE_ADMIN_API=true

./admin-shell
```

**Note:** Environment variables override .env file settings.

### Interactive Session

```
=== Simple Content Admin Shell ===
Type 'help' for available commands, 'exit' to quit

admin> help
admin> list
admin> count
admin> stats
admin> exit
```

## Commands

### `help` or `h`
Show available commands

```
admin> help
```

### `list` or `ls`
List all contents (limit: 20)

```
admin> list
admin> ls

# List for specific tenant
admin> list 550e8400-e29b-41d4-a716-446655440000
```

**Output:**
```
ID                                    Name                  Status      Type
──────────────────────────────────────────────────────────────────────────────
550e8400-e29b-41d4-a716-446655440000  Document 1            uploaded    application/pdf
650e8400-e29b-41d4-a716-446655440001  Image 2               created     image/jpeg

Total: 2
```

### `count`
Count all contents

```
admin> count

# Count for specific tenant
admin> count 550e8400-e29b-41d4-a716-446655440000
```

**Output:**
```
Total count: 42
```

### `stats`
Show statistics with breakdowns

```
admin> stats

# Stats for specific tenant
admin> stats 550e8400-e29b-41d4-a716-446655440000
```

**Output:**
```
Total Count: 42

By Status:
  created        : 10
  uploaded       : 30
  deleted        : 2

By Tenant:
  550e8400-e29b-41d4-a716-446655440000: 25
  650e8400-e29b-41d4-a716-446655440001: 17
```

### `get <content-id>`
Get detailed information for specific content

```
admin> get 550e8400-e29b-41d4-a716-446655440000
```

**Output:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "650e8400-e29b-41d4-a716-446655440001",
  "owner_id": "750e8400-e29b-41d4-a716-446655440002",
  "name": "Document 1",
  "status": "uploaded",
  "document_type": "application/pdf",
  "created_at": "2024-01-15T10:30:45Z",
  "updated_at": "2024-01-15T10:30:45Z"
}
```

### `exit`, `quit`, or `q`
Exit the shell

```
admin> exit
Goodbye!
```

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_TYPE` | Database type (`memory` or `postgres`) | `memory` | No |
| `DATABASE_URL` | PostgreSQL connection string | - | Yes (for postgres) |
| `DB_SCHEMA` | PostgreSQL schema name | `content` | No |

## Example Sessions

### Development Session

```bash
# Terminal 1: Start admin shell
DATABASE_TYPE=memory ./admin-shell

# Inside shell:
admin> count
Total count: 0

admin> # (In another terminal, create some data via API)

admin> count
Total count: 5

admin> list
[Shows 5 contents]

admin> stats
[Shows breakdown]

admin> exit
```

### Debugging Session

```bash
# Start shell
./admin-shell

admin> # Check if tenant has content
admin> count 550e8400-e29b-41d4-a716-446655440000
Total count: 10

admin> # List them
admin> list 550e8400-e29b-41d4-a716-446655440000
[Shows 10 contents for that tenant]

admin> # Get specific content details
admin> get abcd1234-5678-90ef-ghij-klmnopqrstuv
[Shows JSON details]

admin> exit
```

## Comparison with Other Tools

| Feature | Admin Shell | Admin CLI | HTTP Admin API |
|---------|-------------|-----------|----------------|
| **Database** | In-memory (best) | PostgreSQL (best) | Both |
| **Interface** | Interactive REPL | Command line | HTTP/REST |
| **Use Case** | Development/debug | Operations/scripts | Automation/integration |
| **Output** | Table/JSON | Table/JSON | JSON only |
| **Blocks Server** | Yes | No | No |
| **Scripting** | No | Yes | Yes |

## Integration with Development Workflow

### Option 1: Separate Terminals

```bash
# Terminal 1: Run tests that populate data
go test ./pkg/simplecontent/...

# Terminal 2: Inspect data
DATABASE_TYPE=memory ./admin-shell
admin> stats
admin> list
```

### Option 2: Embedded in Service

```go
// In your development code
package main

import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func main() {
    repo := memory.New()

    // Build service
    svc, _ := simplecontent.New(
        simplecontent.WithRepository(repo),
        // ... other options
    )

    // Populate some data
    svc.CreateContent(ctx, ...)

    // Now inspect with admin shell
    adminSvc := admin.New(repo)
    shell := NewAdminShell(svc, adminSvc)
    shell.Run() // Interactive session
}
```

## Limitations

1. **Not for production** - This is a development/debugging tool
2. **Blocks the process** - Can't handle other requests while shell is active
3. **Limited to 20 results** - Use HTTP API for larger datasets
4. **No authentication** - Should only run in development environment
5. **In-memory focus** - For PostgreSQL, use `cmd/admin` CLI instead

## Tips

1. **Quick data check**: Use `count` to see if data exists
2. **Explore structure**: Use `list` to see what's in the database
3. **Debug specific items**: Use `get <id>` for detailed inspection
4. **Monitor activity**: Run `stats` periodically to track changes
5. **Tenant debugging**: Always specify tenant ID when troubleshooting

## Troubleshooting

**Shell shows no data:**
```
admin> count
Total count: 0
```
→ Data is in a different process. For PostgreSQL, use `cmd/admin` instead.

**Can't start shell:**
```
Failed to build service: ...
```
→ Check DATABASE_TYPE and DATABASE_URL environment variables.

**Need pagination:**
→ Admin shell is limited to 20 results. Use HTTP Admin API for more:
```bash
curl http://localhost:8080/api/v1/admin/contents?limit=100
```

## See Also

- [cmd/admin](../admin/README.md) - Standalone CLI for PostgreSQL
- [docs/ADMIN_TOOLS.md](../../docs/ADMIN_TOOLS.md) - Complete admin tools guide
- [pkg/simplecontent/admin](../../pkg/simplecontent/admin/README.md) - Admin package documentation
