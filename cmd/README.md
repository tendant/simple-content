# Simple Content - Command Line Tools

This directory contains executable commands for the simple-content library. Each command serves a specific purpose in the content management ecosystem.

## Server Variants

### standalone-server (Testing) ⭐
**Fastest way to test - no setup required**

All-in-one server with in-memory repository and filesystem storage.

**Features:**
- In-memory repository (no database needed)
- Filesystem storage (persists at ./dev-data)
- Built-in test endpoint (/api/v1/test)
- Zero configuration required
- Perfect for quick testing and development
- Single process - everything embedded

**Use when:**
- Quick testing and iteration
- Learning the API
- No database available
- Need fastest setup possible

**Example:**
```bash
cd cmd/standalone-server
go run main.go
# Server starts on port 4000

# With custom port and data directory:
go run main.go -port 5000 -data-dir /tmp/content-data

# In another terminal - run quick test:
curl http://localhost:4000/api/v1/test
```

**Configuration:**
- CLI: `-port <port>` - HTTP port (default: `4000`)
- CLI: `-data-dir <path>` - Storage directory (default: `./dev-data`)
- ENV: `PORT` - HTTP port (overridden by -port flag)
- ENV: `STORAGE_DIR` - Storage directory (overridden by -data-dir flag)

**Priority:** CLI args > environment variables > defaults

### server-configured (Production)
**Recommended for production use**

Full-featured HTTP server with environment-based configuration.

**Features:**
- Environment variable configuration
- PostgreSQL or in-memory repository
- Multiple storage backends (S3, filesystem, memory)
- Presigned URL support
- Admin API endpoints
- Database connectivity checks
- Graceful shutdown

**Use when:**
- Deploying to production
- Need PostgreSQL persistence
- Require S3 or filesystem storage
- Need presigned URLs for direct uploads/downloads
- Want environment-based configuration

**Configuration:** Via environment variables (see pkg/simplecontent/config)

**Example:**
```bash
cd cmd/server-configured
CONTENT_DATABASE_TYPE=postgres \
CONTENT_DATABASE_URL=postgres://user:pass@localhost:5432/db \
CONTENT_STORAGE_TYPE=s3 \
AWS_S3_BUCKET=my-bucket \
go run main.go
```

### server
**Development/testing server with multiple backends**

HTTP server with in-memory repository and multiple storage backends.

**Features:**
- In-memory repository (no database required)
- Multiple storage backends (memory, filesystem)
- No external dependencies
- Simple to run locally

**Use when:**
- Local development and testing
- Testing multiple storage backends
- No database available
- Quick prototyping

**Example:**
```bash
cd cmd/server
go run main.go
# Server starts on port 8080
```

### server-simple
**Minimal example server**

Simplest possible HTTP server implementation with only memory storage.

**Features:**
- In-memory repository
- In-memory storage only
- Minimal dependencies
- Clean example code

**Use when:**
- Learning the library
- Simple examples/demos
- Testing basic functionality
- Reference implementation

**Example:**
```bash
cd cmd/server-simple
PORT=8080 go run main.go
```

### mcpserver
**AI Integration Server**

MCP (Model Context Protocol) server for AI assistant integrations.

**Features:**
- PostgreSQL repository
- S3 storage backend
- MCP protocol support (stdio, SSE, HTTP modes)
- Environment-based configuration
- Designed for Claude Code and similar AI tools

**Use when:**
- Integrating with AI assistants
- Need MCP protocol support
- Building AI-powered workflows

**Example:**
```bash
cd cmd/mcpserver
# Create .env file with database and S3 configuration
go run main.go --mode stdio
```

## Command-Line Tools

### admin
Admin CLI tool for content management operations.

**Features:**
- List content by owner/tenant
- Get content details
- Delete content
- PostgreSQL support

**Example:**
```bash
cd cmd/admin
go run main.go --owner-id <uuid> --tenant-id <uuid>
```

### admin-shell
Interactive admin shell for content operations.

Provides an interactive REPL for managing content.

### example
Demonstration of the library's core functionality.

Shows complete workflow: create content, upload data, create derived content.

**Use when:**
- Learning the library API
- Understanding content workflows
- Testing integration

### files
HTTP server for file operations with PostgreSQL and S3.

Production-ready file server with authentication support.

### s3test
S3 storage backend testing tool.

**Features:**
- Upload/download files
- Generate presigned URLs
- Test S3 configuration
- MinIO support

**Example:**
```bash
cd cmd/s3test
go run main.go -use-minio -bucket my-bucket -command upload -key test.txt -file local.txt
```

## Quick Reference

| Command | Purpose | Repository | Storage | Production Ready |
|---------|---------|------------|---------|------------------|
| standalone-server ⭐ | Quickest testing | Memory | Filesystem | ⚠️ Testing only |
| server-configured | Production server | Postgres/Memory | S3/FS/Memory | ✅ Yes |
| server | Development server | Memory | FS/Memory | ⚠️ Dev only |
| server-simple | Minimal example | Memory | Memory | ❌ No |
| mcpserver | AI integration | Postgres | S3 | ✅ Yes |
| files | File operations | Postgres | S3 | ✅ Yes |
| admin | CLI management | Postgres | - | ✅ Yes |
| admin-shell | Interactive shell | Postgres | - | ✅ Yes |
| example | Learning/demo | Postgres | S3 | ❌ No |
| s3test | S3 testing | - | S3 | ⚠️ Testing |

## Getting Started

### For Quick Testing ⚡
Start with `standalone-server` - fastest way to test with zero setup required.

### For Development
Start with `server-simple` to understand the basics, then try `server` for multiple storage backends.

### For Production
Use `server-configured` with PostgreSQL and S3 storage.

### For AI Integration
Use `mcpserver` with appropriate MCP client configuration.

### For Administration
Use `admin` or `admin-shell` for command-line management.

## Common Configuration

Most production servers require:

**Database (PostgreSQL):**
```bash
CONTENT_DATABASE_TYPE=postgres
CONTENT_DATABASE_URL=postgres://user:pass@host:5432/dbname
CONTENT_DB_SCHEMA=public
```

**Storage (S3):**
```bash
CONTENT_STORAGE_TYPE=s3
AWS_S3_BUCKET=my-bucket
AWS_S3_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
```

**Storage (Filesystem):**
```bash
CONTENT_STORAGE_TYPE=fs
CONTENT_FS_BASE_DIR=/path/to/storage
```

See individual command directories and `pkg/simplecontent/config` for detailed configuration options.
