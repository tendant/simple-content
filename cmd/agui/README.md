# AG-UI Server - Content API

A REST API server for content management with presigned URL-based uploads and downloads.

## Overview

This server provides endpoints for:
- **File Upload** - Presigned URL-based uploads for efficient large file handling
- **File Download** - Presigned URL-based downloads
- **Content Management** - List, get details, and delete operations

## Quick Start

### 1. Setup Configuration

Copy the example environment file:
```bash
cd /Users/txgao/Desktop/simple-content/cmd/agui
cp .env.example .env
```

Edit `.env` with your configuration:
```bash
# For development with PostgreSQL and MinIO
DATABASE_URL=postgresql://content:password@localhost:5432/powercard_db
STORAGE_URL=s3://mymusic
S3_ENDPOINT=http://localhost:9000
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin
```

### 2. Build
```bash
go build -o agui
```

### 3. Run
```bash
# Reads from .env file automatically
./agui

# Or with environment variables
DATABASE_URL="postgresql://user:pass@localhost:5432/db" \
STORAGE_URL="s3://my-bucket" \
./agui

# Custom port
PORT=3000 ./agui
```

Server starts on `http://localhost:8080` by default.

## API Endpoints

For complete API documentation, see [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) or [ag-ui-content-v5.yaml](./ag-ui-content-v5.yaml).

### 1. Request Upload URL

```bash
curl -X POST http://localhost:8080/api/v5/content/upload \
  -H "Content-Type: application/json" \
  -d '{"mime_type":"image/jpeg","filename":"photo.jpg","size":123456}'
```

### 2. Mark Upload Complete

```bash
curl -X POST http://localhost:8080/api/v5/content/upload/done \
  -H "Content-Type: application/json" \
  -d '{"content_id":"550e8400-e29b-41d4-a716-446655440000"}'
```

### 3. Get Content Details

```bash
curl http://localhost:8080/api/v5/content/contents/550e8400-e29b-41d4-a716-446655440000
```

### 4. List Contents

```bash
curl http://localhost:8080/api/v5/content/contents
```

### 5. Delete Content

```bash
curl -X DELETE http://localhost:8080/api/v5/content/contents/550e8400-e29b-41d4-a716-446655440000
```

## Complete Upload Workflow

```bash
# 1. Request upload URL
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v5/content/upload \
  -H "Content-Type: application/json" \
  -d '{"mime_type":"image/jpeg","filename":"photo.jpg","size":123456}')

CONTENT_ID=$(echo $RESPONSE | jq -r '.content_id')
UPLOAD_URL=$(echo $RESPONSE | jq -r '.upload_url')

# 2. Upload file to S3
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: image/jpeg" \
  --data-binary @photo.jpg

# 3. Mark upload complete
curl -X POST http://localhost:8080/api/v5/content/upload/done \
  -H "Content-Type: application/json" \
  -d "{\"content_id\":\"$CONTENT_ID\"}"

# 4. Get content details with download URL
curl http://localhost:8080/api/v5/content/contents/$CONTENT_ID
```

---

## Configuration

### Environment Variables

```bash
# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/db
DB_SCHEMA=content

# Storage
STORAGE_URL=s3://my-bucket
STORAGE_NAME=s3-default

# Server
PORT=8080

# URL Strategy
URL_STRATEGY=storage-delegated
API_BASE_URL=/api/v1
```

### Using Config File

Create `config.yaml`:
```yaml
database:
  url: postgresql://user:pass@localhost:5432/db
  schema: content

storage:
  type: s3
  url: s3://my-bucket
  name: s3-default

url_strategy: storage-delegated
api_base_url: /api/v1
```

Run with config:
```bash
CONFIG_FILE=config.yaml ./agui
```

---

## Health Check

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok"
}
```

---

## Implementation Status

| Endpoint | Status |
|----------|--------|
| `POST /content/upload` | ✅ Implemented |
| `POST /content/upload/done` | ✅ Implemented |
| `GET /content/contents/{id}` | ✅ Implemented |
| `GET /content/contents` | ✅ Implemented |
| `DELETE /content/contents/{id}` | ✅ Implemented |

---

## Development

### Run Tests
```bash
go test ./...
```

### Build for Production
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o agui
```

---

## Architecture

```
Client Request
    ↓
AG-UI Server (main.go)
    ↓
Handler (routes + validation)
    ↓
Service Layer (pkg/simplecontent)
    ↓
Repository (database) + BlobStore (S3)
```

---

## License

MIT
