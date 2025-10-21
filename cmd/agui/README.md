# AGUI CLI - Simple Content Management

A command-line interface for content management using the `simplecontent` service directly.

## Overview

This CLI uses `pkg/simplecontent/service_impl.go` directly for all operations. By default, it runs with **in-memory storage** for quick testing and development.

## Installation

```bash
cd /Users/txgao/Desktop/simple-content
go build -o agui ./cmd/agui
```

## Usage

### Upload Content

```bash
# Upload a file
./agui upload myfile.pdf

# Upload with metadata
./agui upload myfile.pdf --metadata '{"author":"John","tags":["document"]}'

# Verbose output
./agui upload myfile.pdf -v
```

### Download Content

```bash
# Download content by ID
./agui download <content-id>

# Download with custom output name
./agui download <content-id> -o myfile.pdf
```

### Content Management

```bash
# List all contents
./agui list

# List with pagination
./agui list --limit 50 --offset 100

# Get content metadata
./agui metadata <content-id>

# Delete content
./agui delete <content-id>
```

### Global Options

```bash
# Verbose output (shows service initialization)
./agui -v upload file.pdf

# Use config file (optional)
./agui --config config.yaml list
```

## How It Works

### Direct Service Usage

The CLI uses the simplecontent service directly:

```
User → agui CLI → ServiceClient → service_impl.go → Repository/BlobStore
```

**No HTTP server required!** All operations are direct function calls.

### Default Configuration

By default, the CLI uses:
- **In-memory repository** - Content metadata stored in memory
- **In-memory blob store** - File data stored in memory
- **Content-based URL strategy** - URLs generated for content access

This means:
- ✅ Fast and lightweight
- ✅ No external dependencies
- ✅ Perfect for testing and development
- ⚠️ Data is lost when the CLI exits

### Custom Configuration

You can configure persistent storage using a config file:

```yaml
# config.yaml
database_type: postgres
database_url: postgres://user:pass@localhost/content

storage_backends:
  - name: s3
    type: s3
    config:
      bucket: my-bucket
      region: us-west-2

default_storage_backend: s3
```

Then use it:
```bash
./agui --config config.yaml upload file.pdf
```

## Examples

### Basic Workflow

```bash
# 1. Upload a file
./agui upload photo.jpg
# Output: Content ID: 123e4567-e89b-12d3-a456-426614174000

# 2. List contents
./agui list
# Output: Total: 1
#         - ID: 123e4567-e89b-12d3-a456-426614174000

# 3. Download the file
./agui download 123e4567-e89b-12d3-a456-426614174000 -o downloaded.jpg

# 4. Delete when done
./agui delete 123e4567-e89b-12d3-a456-426614174000
```

### With Metadata

```bash
# Upload with custom metadata
./agui upload document.pdf --metadata '{
  "title": "Q4 Report",
  "author": "Jane Doe",
  "department": "Finance"
}'

# View metadata
./agui metadata <content-id>
```

## Architecture

### Components

- **`main.go`** - CLI initialization, creates service with default config
- **`service_client.go`** - Wraps `simplecontent.Service` interface
- **`upload.go`, `download.go`, etc.** - Command implementations
- **`pkg/simplecontent/service_impl.go`** - Core service logic (used directly)

### Service Initialization

```go
// Load default in-memory configuration
cfg, _ := config.Load()

// Build service
service, _ := cfg.BuildService()

// Use service directly
content, _ := service.UploadContent(ctx, req)
```

## Benefits

1. **No Server Required** - Direct service usage, no HTTP overhead
2. **Fast Development** - In-memory storage for quick iteration
3. **Testable** - Easy to test without infrastructure
4. **Flexible** - Can be configured for production storage (S3, Postgres)
5. **Embeddable** - Same service used by the full server

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/tendant/simple-content/pkg/simplecontent` - Core service library
