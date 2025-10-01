# Admin CLI Tool

A lightweight command-line tool for administrative content operations. **Requires only database access - no storage backends needed.**

## Features

- **List contents** with flexible filtering and pagination
- **Count contents** for monitoring and reporting
- **Get statistics** with aggregated breakdowns
- **JSON output** for scripting and automation
- **Table output** for human-readable viewing
- **Memory mode** for testing without a database

## Installation

```bash
# Build the admin tool
go build -o admin ./cmd/admin

# Or run directly
go run ./cmd/admin <command>
```

## Quick Start

```bash
# Using memory database (for testing)
DATABASE_TYPE=memory ./admin list

# Using PostgreSQL
export DATABASE_URL="postgres://user:pass@localhost/dbname"
export DATABASE_TYPE=postgres
export DB_SCHEMA=content

./admin list
./admin count
./admin stats
```

## Commands

### `list` - List Contents

List contents with optional filtering and pagination.

**Examples:**

```bash
# List all contents (default limit: 100)
./admin list

# List with custom pagination
./admin list --limit=50 --offset=100

# Filter by tenant
./admin list --tenant-id=550e8400-e29b-41d4-a716-446655440000

# Filter by owner
./admin list --owner-id=650e8400-e29b-41d4-a716-446655440000

# Filter by status
./admin list --status=uploaded

# Filter by derivation type
./admin list --derivation-type=thumbnail

# Filter by document type
./admin list --document-type="application/pdf"

# Include deleted content
./admin list --include-deleted

# Multiple filters
./admin list --tenant-id=550e8400-e29b-41d4-a716-446655440000 --status=uploaded --limit=20

# JSON output for scripting
./admin list --json | jq '.contents[] | {id, name, status}'
```

**Table Output:**

```
ID                  NAME             TENANT             OWNER              STATUS    TYPE             CREATED
────────────────    ───────────────  ──────────────     ──────────────     ──────    ──────────────   ─────────────────────
abcd1234...         Document 1       11111111...        aaaaaaaa...        uploaded  application/pdf  2024-01-15 10:30:45
efgh5678...         Image 2          22222222...        bbbbbbbb...        created   image/jpeg       2024-01-15 11:22:33

Total: 2
```

### `count` - Count Contents

Count contents matching filter criteria.

**Examples:**

```bash
# Count all contents
./admin count

# Count by tenant
./admin count --tenant-id=550e8400-e29b-41d4-a716-446655440000

# Count by status
./admin count --status=uploaded

# Count with multiple filters
./admin count --tenant-id=550e8400-e29b-41d4-a716-446655440000 --status=uploaded

# JSON output
./admin count --json
```

**Output:**

```
Total count: 12345
```

### `stats` - Get Statistics

Get aggregated statistics with breakdowns by status, tenant, type, etc.

**Examples:**

```bash
# Get overall statistics
./admin stats

# Get statistics for a specific tenant
./admin stats --tenant-id=550e8400-e29b-41d4-a716-446655440000

# JSON output for dashboards
./admin stats --json
```

**Output:**

```
=== Content Statistics ===

Total Count: 12345

By Status:
  created        : 1000
  uploaded       : 10000
  deleted        : 1345

By Tenant:
  11111111...: 8000
  22222222...: 4345

By Derivation Type:
  original       : 10000
  thumbnail      : 1500
  preview        : 845

By Document Type:
  application/pdf            : 5000
  image/jpeg                 : 4000
  video/mp4                  : 3345

Time Range:
  Oldest: 2024-01-01T00:00:00Z
  Newest: 2024-12-31T23:59:59Z

Computed at: 2024-12-31T23:59:59Z
```

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_TYPE` | Database type (`postgres` or `memory`) | `memory` | No |
| `DATABASE_URL` | PostgreSQL connection string | - | Yes (for postgres) |
| `DB_SCHEMA` | PostgreSQL schema name | `content` | No |

## Filter Options

All commands support the following filters:

| Option | Type | Description | Example |
|--------|------|-------------|---------|
| `--tenant-id` | UUID | Filter by tenant ID | `--tenant-id=550e8400-...` |
| `--owner-id` | UUID | Filter by owner ID | `--owner-id=650e8400-...` |
| `--status` | String | Filter by status | `--status=uploaded` |
| `--derivation-type` | String | Filter by derivation type | `--derivation-type=thumbnail` |
| `--document-type` | String | Filter by document MIME type | `--document-type="application/pdf"` |
| `--include-deleted` | Flag | Include soft-deleted content | `--include-deleted` |
| `--json` | Flag | Output as JSON | `--json` |

**List-only options:**

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `--limit` | Int | Maximum results | `100` |
| `--offset` | Int | Pagination offset | `0` |

## Use Cases

### 1. Monitoring & Reporting

```bash
# Daily content report
./admin count --status=uploaded > daily_report.txt

# Export statistics for dashboard
./admin stats --json | jq '.statistics' > dashboard_data.json

# Check specific tenant usage
./admin count --tenant-id=<uuid>
```

### 2. Data Analysis

```bash
# Find all PDFs
./admin list --document-type="application/pdf" --json | \
  jq '.contents | length'

# Count content by status for all tenants
for status in created uploaded deleted; do
  echo "$status: $(./admin count --status=$status --json | jq '.count')"
done

# Export all content metadata
./admin list --limit=10000 --json > content_export.json
```

### 3. Bulk Operations

```bash
# Get list of old deleted content (for cleanup)
./admin list --status=deleted --include-deleted --limit=1000 --json | \
  jq '.contents[] | select(.updated_at < "2024-01-01") | .id'

# Find orphaned content
./admin list --json | \
  jq '.contents[] | select(.status == "created") | {id, created_at}'
```

### 4. Debugging & Troubleshooting

```bash
# Check if content exists for a tenant
./admin count --tenant-id=<uuid>

# List recent uploads
./admin list --status=uploaded --limit=10

# Verify data migration
./admin count --json > before.json
# ... perform migration ...
./admin count --json > after.json
diff before.json after.json
```

## Database Setup

### PostgreSQL

```bash
# Set connection string
export DATABASE_URL="postgres://username:password@localhost:5432/dbname?sslmode=disable"
export DATABASE_TYPE=postgres
export DB_SCHEMA=content

# Test connection
./admin count
```

### Memory (Testing)

```bash
# Use in-memory database (empty by default)
export DATABASE_TYPE=memory

./admin list
# Returns empty list - useful for testing CLI without database
```

## JSON Output Format

All commands support `--json` flag for machine-readable output.

**List JSON:**
```json
{
  "contents": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "tenant_id": "...",
      "owner_id": "...",
      "name": "Document 1",
      "status": "uploaded",
      "document_type": "application/pdf",
      "created_at": "2024-01-15T10:30:45Z",
      "updated_at": "2024-01-15T10:30:45Z"
    }
  ],
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

**Count JSON:**
```json
{
  "count": 12345
}
```

**Stats JSON:**
```json
{
  "statistics": {
    "total_count": 12345,
    "by_status": { "uploaded": 10000 },
    "by_tenant": { "tenant-id": 8000 },
    "by_derivation_type": { "original": 10000 },
    "by_document_type": { "application/pdf": 5000 },
    "oldest_content": "2024-01-01T00:00:00Z",
    "newest_content": "2024-12-31T23:59:59Z"
  },
  "computed_at": "2024-12-31T23:59:59Z"
}
```

## Scripting Examples

### Bash - Daily Content Report

```bash
#!/bin/bash
# daily_report.sh

echo "Content Report - $(date)"
echo "================================"
echo ""

echo "Total Contents:"
./admin count

echo ""
echo "By Status:"
for status in created uploaded deleted; do
  count=$(./admin count --status=$status --json | jq -r '.count')
  echo "  $status: $count"
done

echo ""
echo "Recent Uploads (last 10):"
./admin list --status=uploaded --limit=10
```

### Python - Export to CSV

```python
#!/usr/bin/env python3
import subprocess
import json
import csv

# Get all content as JSON
result = subprocess.run(
    ['./admin', 'list', '--limit=10000', '--json'],
    capture_output=True,
    text=True
)

data = json.loads(result.stdout)

# Write to CSV
with open('contents.csv', 'w', newline='') as f:
    writer = csv.DictWriter(f, fieldnames=['id', 'name', 'tenant_id', 'status', 'created_at'])
    writer.writeheader()
    for content in data['contents']:
        writer.writerow({
            'id': content['id'],
            'name': content['name'],
            'tenant_id': content['tenant_id'],
            'status': content['status'],
            'created_at': content['created_at']
        })
```

## Performance Tips

1. **Use `count` before large `list` operations** to know data size
2. **Paginate large result sets** with `--limit` and `--offset`
3. **Filter early** to reduce result set size
4. **Use `--json` for scripting** to avoid parsing table output
5. **Direct database connection** is faster than HTTP API

## Troubleshooting

**Connection errors:**
```bash
# Check database connection
psql "$DATABASE_URL" -c "SELECT 1"

# Verify schema exists
psql "$DATABASE_URL" -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'content'"
```

**Empty results:**
```bash
# Check if content table exists
psql "$DATABASE_URL" -c "\dt content.*"

# Verify data exists
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM content.content"
```

**Permission issues:**
```bash
# Ensure database user has SELECT permission
psql "$DATABASE_URL" -c "GRANT SELECT ON ALL TABLES IN SCHEMA content TO your_user"
```

## Comparison with HTTP API

| Feature | Admin CLI | HTTP API |
|---------|-----------|----------|
| Setup | Database only | Full server + storage |
| Overhead | Minimal | HTTP/network |
| Output | Table or JSON | JSON only |
| Auth | Database auth | Application auth required |
| Use Case | Operations/scripts | Application integration |
| Performance | Direct DB | Network latency |

## Examples by Role

### For DevOps

```bash
# Quick health check
./admin count

# Monitor content growth
watch -n 60 './admin stats'

# Export metrics to monitoring system
./admin stats --json | telegraf --config monitoring.conf
```

### For Data Analysts

```bash
# Export all data
./admin list --limit=100000 --json > export.json

# Tenant usage analysis
for tenant in $(cat tenant_ids.txt); do
  echo "$tenant: $(./admin count --tenant-id=$tenant --json | jq '.count')"
done
```

### For Support Engineers

```bash
# Find customer's content
./admin list --tenant-id=<customer-tenant-id>

# Check content status
./admin list --owner-id=<user-id> --status=failed

# Verify upload completion
./admin count --tenant-id=<tenant-id> --status=uploaded
```

## Security Considerations

⚠️ **Important**: This tool has unrestricted access to all content across all tenants.

- Protect database credentials
- Use read-only database users when possible
- Audit CLI usage in production
- Consider VPN/bastion access for production databases
- Never commit database credentials to version control

## Future Enhancements

- Export to multiple formats (CSV, Excel, Parquet)
- Interactive mode with TUI
- Bulk update operations
- Content validation and repair
- Real-time monitoring mode
- Prometheus metrics export
