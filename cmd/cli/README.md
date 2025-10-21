# Simple Content CLI - Content Management Tool

A command-line interface for content management using the `simplecontent` service directly.

## Overview

This CLI uses `pkg/simplecontent/service_impl.go` directly for all operations. By default, it runs with **in-memory storage** for quick testing and development.

## Installation

```bash
cd /Users/txgao/Desktop/simple-content
go build -o cli ./cmd/cli
```

## Usage

### Upload Content

The CLI supports multiple upload methods and file types.

#### Basic Upload

```bash
# Upload a single file
./cli upload document.pdf

# Upload an image
./cli upload photo.jpg

# Upload a text file
./cli upload notes.txt

# Verbose output (shows upload progress and details)
./cli upload myfile.pdf -v
```

#### Upload with Metadata

```bash
# Upload with custom metadata
./cli upload report.pdf --metadata '{"author":"John Doe","department":"Finance"}'

# Upload with tags
./cli upload presentation.pptx --metadata '{"tags":["Q4","sales","2024"]}'

# Upload with multiple metadata fields
./cli upload contract.pdf --metadata '{
  "title": "Service Agreement",
  "author": "Legal Team",
  "confidential": true,
  "expiry_date": "2025-12-31"
}'
```

#### Upload Different File Types

```bash
# Documents
./cli upload document.pdf
./cli upload spreadsheet.xlsx
./cli upload presentation.pptx

# Images
./cli upload photo.jpg
./cli upload diagram.png
./cli upload logo.svg

# Videos (large files will get upload URL)
./cli upload video.mp4
./cli upload recording.mov

# Archives
./cli upload backup.zip
./cli upload data.tar.gz

# Code files
./cli upload script.py
./cli upload config.json
```

#### Use Cases

**1. Document Management**
```bash
# Upload company documents with metadata
./cli upload policy.pdf --metadata '{
  "type": "policy",
  "department": "HR",
  "effective_date": "2024-01-01"
}'

# List all uploaded documents
./cli list

# Get document details
./cli metadata dd0a368a-de27-48bd-b8a6-100f4ff3e714
```

**2. Image Gallery**
```bash
# Upload multiple images
./cli upload photo1.jpg -v
./cli upload photo2.jpg -v
./cli upload photo3.jpg -v

# Upload with photographer metadata
./cli upload landscape.jpg --metadata '{
  "photographer": "Jane Smith",
  "location": "Yosemite",
  "date": "2024-10-15"
}'
```

**3. Backup and Archive**
```bash
# Upload backup files
./cli upload database-backup.sql -v
./cli upload config-backup.tar.gz -v

# Upload with backup metadata
./cli upload backup.zip --metadata '{
  "backup_date": "2024-10-21",
  "source": "production-db",
  "retention_days": 90
}'
```

**4. Large File Upload (with Upload URL)**
```bash
# For very large files, the service may return an upload URL
# The CLI will show the upload URL for manual upload
./cli upload large-video.mp4 -v

# Output example:
# Upload URL: https://s3.amazonaws.com/bucket/presigned-url
# You can upload directly to this URL using curl or other tools
```

**5. Development Workflow**
```bash
# Upload test files during development
./cli upload test-data.json -v
./cli upload sample.csv -v

# Upload with environment metadata
./cli upload build.zip --metadata '{
  "environment": "staging",
  "version": "1.2.3",
  "build_number": "456"
}'
```

### Download Content

```bash
# Download content by ID
./cli download <content-id>

# Download with custom output name
./cli download <content-id> -o myfile.pdf
```

### Content Management

```bash
# List all contents
./cli list

# List with pagination
./cli list --limit 50 --offset 100

# Get content metadata
./cli metadata <content-id>

# Delete content
./cli delete <content-id>
```

### Global Options

```bash
# Verbose output (shows service initialization)
./cli -v upload file.pdf

# Use config file (optional)
./cli --config config.yaml list
```

## How It Works

### Direct Service Usage

The CLI uses the simplecontent service directly:

```
User → cli → ServiceClient → service_impl.go → Repository/BlobStore
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
./cli --config config.yaml upload file.pdf
```

## Examples

### Basic Workflow

```bash
# 1. Upload a file
./cli upload photo.jpg
# Output: Content ID: 123e4567-e89b-12d3-a456-426614174000

# 2. List contents
./cli list
# Output: Total: 1
#         - ID: 123e4567-e89b-12d3-a456-426614174000

# 3. Download the file
./cli download 123e4567-e89b-12d3-a456-426614174000 -o downloaded.jpg

# 4. Delete when done
./cli delete 123e4567-e89b-12d3-a456-426614174000
```

### With Metadata

```bash
# Upload with custom metadata
./cli upload document.pdf --metadata '{
  "title": "Q4 Report",
  "author": "Jane Doe",
  "department": "Finance"
}'

# View metadata
./cli metadata <content-id>
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
