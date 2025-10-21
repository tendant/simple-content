# AG-UI Server - Multimodal Content API

A REST API server implementing the AG-UI protocol for multimodal content handling (text + binary files).

## Overview

This server provides endpoints for:
- **File Upload** - Multiple formats (multipart, base64, URL reference)
- **Content Analysis** - Multimodal content analysis (text + files)
- **File Download** - Single or batch downloads
- **Content Management** - List, metadata, delete operations

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

### 1. Upload Content

**Multipart Upload:**
```bash
curl -X POST http://localhost:8080/api/v1/contents/upload \
  -F "file=@document.pdf" \
  -F "metadata={\"author\":\"John\"}"
```

**Response:**
```json
{
  "id": "dd0a368a-de27-48bd-b8a6-100f4ff3e714",
  "url": "http://localhost:9000/bucket/dd0a368a.../download"
}
```

---

### 2. Analyze Content (AG-UI Protocol)

**Submit multimodal analysis:**
```bash
curl -X POST http://localhost:8080/api/v1/contents/analysis \
  -H "Content-Type: application/json" \
  -d '{
    "content": [
      {
        "type": "text",
        "text": "Analyze this document and summarize key points"
      },
      {
        "type": "binary",
        "mime_type": "application/pdf",
        "id": "dd0a368a-de27-48bd-b8a6-100f4ff3e714",
        "filename": "report.pdf"
      }
    ],
    "analysis_type": "document_analysis"
  }'
```

**Response:**
```json
{
  "id": "analysis-123",
  "status": "pending",
  "created_at": "2024-10-21T14:30:00Z"
}
```

---

### 3. Get Analysis Status

```bash
curl http://localhost:8080/api/v1/contents/analysis/analysis-123
```

**Response:**
```json
{
  "id": "analysis-123",
  "status": "completed",
  "result": {
    "summary": "Document contains...",
    "key_points": ["Point 1", "Point 2"]
  },
  "created_at": "2024-10-21T14:30:00Z",
  "completed_at": "2024-10-21T14:31:00Z"
}
```

---

### 4. List Analyses

```bash
curl "http://localhost:8080/api/v1/contents/analysis?status=completed&limit=10"
```

---

### 5. Get Content Metadata

```bash
curl http://localhost:8080/api/v1/contents/{contentId}/metadata
```

**Response:**
```json
{
  "id": "dd0a368a-de27-48bd-b8a6-100f4ff3e714",
  "filename": "document.pdf",
  "mime_type": "application/pdf",
  "size": 1048576,
  "created_at": "2024-10-21T14:30:00Z"
}
```

---

### 6. Download Content

**Single file:**
```bash
curl -X POST http://localhost:8080/api/v1/contents/download \
  -H "Content-Type: application/json" \
  -d '{"content_ids": ["dd0a368a-de27-48bd-b8a6-100f4ff3e714"]}' \
  -o document.pdf
```

**Multiple files (zip):**
```bash
curl -X POST http://localhost:8080/api/v1/contents/download \
  -H "Content-Type: application/json" \
  -d '{"content_ids": ["id1", "id2", "id3"]}' \
  -o files.zip
```

---

### 7. List Contents

```bash
curl "http://localhost:8080/api/v1/contents?limit=10&offset=0"
```

**Response:**
```json
{
  "contents": [
    {
      "id": "dd0a368a-de27-48bd-b8a6-100f4ff3e714",
      "url": "http://localhost:9000/bucket/..."
    }
  ],
  "total": 1
}
```

---

### 8. Delete Content

```bash
curl -X DELETE http://localhost:8080/api/v1/contents/{contentId}
```

---

## AG-UI Protocol Examples

### Example 1: Upload then Analyze (ID Reference)

```bash
# Step 1: Upload file
CONTENT_ID=$(curl -X POST http://localhost:8080/api/v1/contents/upload \
  -F "file=@document.pdf" | jq -r '.id')

# Step 2: Analyze with text prompt
ANALYSIS_ID=$(curl -X POST http://localhost:8080/api/v1/contents/analysis \
  -H "Content-Type: application/json" \
  -d "{
    \"content\": [
      {\"type\": \"text\", \"text\": \"Summarize this document\"},
      {\"type\": \"binary\", \"mime_type\": \"application/pdf\", \"id\": \"$CONTENT_ID\"}
    ],
    \"analysis_type\": \"document_summary\"
  }" | jq -r '.id')

# Step 3: Check status
curl http://localhost:8080/api/v1/contents/analysis/$ANALYSIS_ID
```

---

### Example 2: Inline Data Analysis (Base64)

```bash
# Convert file to base64
BASE64_DATA=$(base64 -i image.png)

# Analyze directly
curl -X POST http://localhost:8080/api/v1/contents/analysis \
  -H "Content-Type: application/json" \
  -d "{
    \"content\": [
      {\"type\": \"text\", \"text\": \"What is in this image?\"},
      {
        \"type\": \"binary\",
        \"mime_type\": \"image/png\",
        \"filename\": \"image.png\",
        \"data\": \"$BASE64_DATA\"
      }
    ],
    \"analysis_type\": \"image_analysis\"
  }"
```

---

### Example 3: URL Reference Analysis

```bash
curl -X POST http://localhost:8080/api/v1/contents/analysis \
  -H "Content-Type: application/json" \
  -d '{
    "content": [
      {"type": "text", "text": "Analyze this public document"},
      {
        "type": "binary",
        "mime_type": "application/pdf",
        "filename": "report.pdf",
        "url": "https://example.com/public/report.pdf"
      }
    ],
    "analysis_type": "document_analysis"
  }'
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
| `POST /upload` (multipart) | ✅ Implemented |
| `POST /upload` (JSON base64) | ❌ TODO |
| `POST /upload` (JSON URL) | ❌ TODO |
| `POST /analysis` | ⚠️ Partial (accepts requests) |
| `GET /analysis/{id}` | ⚠️ Partial (mock response) |
| `GET /analysis` | ⚠️ Partial (mock response) |
| `GET /{id}/metadata` | ✅ Implemented |
| `POST /download` | ✅ Implemented (single file) |
| `GET /contents` | ✅ Implemented |
| `DELETE /{id}` | ✅ Implemented |

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

### Docker
```bash
docker build -t agui .
docker run -p 8080:8080 agui
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
Repository (database) + BlobStore (storage)
```

---

## Next Steps

1. **Complete Upload Endpoint**
   - Add base64 data upload
   - Add URL reference upload

2. **Implement Analysis Processing**
   - Create analysis database schema
   - Add job queue
   - Implement worker pool

3. **Add Authentication**
   - JWT token validation
   - User context extraction

4. **Add Batch Download**
   - Create zip archives for multiple files

---

## License

MIT
