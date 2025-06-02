# Content Upload Server

A standalone HTTP server for content upload and download functionality using PostgreSQL and S3-compatible storage.

## Features

- Content creation, retrieval, and management
- File upload and download via S3-compatible storage (MinIO, AWS S3)
- Object metadata management
- Content derivation and relationships
- RESTful API endpoints

## Configuration

The server is configured via environment variables:

### Server Configuration
- `SERVER_HOST` - Server host (default: localhost)
- `SERVER_PORT` - Server port (default: 8080)

### Database Configuration
- `DB_HOST` - PostgreSQL host (default: localhost)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_NAME` - Database name (default: powercard_db)
- `DB_USER` - Database user (default: content)
- `DB_PASSWORD` - Database password (required)

### S3 Storage Configuration
- `S3_ENDPOINT` - S3 endpoint URL (required for MinIO)
- `S3_ACCESS_KEY_ID` - S3 access key (required)
- `S3_SECRET_ACCESS_KEY` - S3 secret key (required)
- `S3_BUCKET_NAME` - S3 bucket name (required)
- `S3_REGION` - S3 region (default: us-east-1)
- `S3_USE_SSL` - Use SSL for S3 connections (default: true)

## API Endpoints

### Content Management
- `POST /contents/` - Create new content
- `GET /contents/{id}` - Get content by ID
- `DELETE /contents/{id}` - Delete content
- `GET /contents/list` - List all contents

### Metadata Management
- `PUT /contents/{id}/metadata` - Update content metadata
- `GET /contents/{id}/metadata` - Get content metadata

### Object Management
- `POST /contents/{id}/objects` - Create object for content
- `GET /contents/{id}/objects` - List objects for content
- `GET /contents/{id}/download` - Get download URL for content

### Content Derivation
- `POST /contents/{id}/derive` - Create derived content
- `GET /contents/{id}/derived` - Get derived content
- `GET /contents/{id}/derived-tree` - Get derived content tree

### File Upload API
- `POST /files/` - Create file and get upload URL
- `POST /files/{content_id}/complete` - Mark upload complete
- `PATCH /files/{content_id}` - Update metadata
- `GET /files/{content_id}` - Get file info with URLs

## File Upload API Examples

### 1. Create File and Get Upload URL

```bash
# Create a new file content and get upload URL
curl -X POST http://localhost:8080/files/ \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "example.txt",
    "content_type": "text/plain",
    "file_size": 1024,
    "owner_id": "00000000-0000-0000-0000-000000000001",
    "tenant_id": "00000000-0000-0000-0000-000000000001"
  }'
```

Response:
```json
{
  "content_id": "123e4567-e89b-12d3-a456-426614174000",
  "upload_url": "https://minio:9000/content-bucket/object-key?X-Amz-Algorithm=...",
  "expires_at": "2025-06-01T16:35:33Z"
}
```

### 2. Upload File to Storage

Use the upload URL from step 1 to upload your file:

```bash
# Upload the actual file using the presigned URL
curl -X PUT "https://minio:9000/content-bucket/object-key?X-Amz-Algorithm=..." \
  -H "Content-Type: text/plain" \
  --data-binary @example.txt
```

### 3. Mark Upload Complete

After uploading the file, mark the upload as complete:

```bash
# Mark upload as complete
curl -X POST http://localhost:8080/files/123e4567-e89b-12d3-a456-426614174000/complete \
  -H "Content-Type: application/json"
```

Response:
```json
{
  "status": "completed"
}
```

### 4. Update File Metadata

```bash
# Update file metadata
curl -X PATCH http://localhost:8080/files/123e4567-e89b-12d3-a456-426614174000 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Example File",
    "description": "This is an example text file",
    "tags": ["example", "text", "demo"]
  }'
```

Response:
```json
{
  "status": "updated"
}
```

### 5. Get File Information

```bash
# Get file info with download and preview URLs
curl -X GET http://localhost:8080/files/123e4567-e89b-12d3-a456-426614174000
```

Response:
```json
{
  "content_id": "123e4567-e89b-12d3-a456-426614174000",
  "file_name": "example.txt",
  "preview_url": "https://minio:9000/content-bucket/object-key?X-Amz-Algorithm=...",
  "download_url": "https://minio:9000/content-bucket/object-key?X-Amz-Algorithm=...",
  "metadata": {
    "content_type": "text/plain",
    "title": "My Example File",
    "description": "This is an example text file",
    "file_name": "example.txt",
    "tags": ["example", "text", "demo"]
  },
  "created_at": "2025-06-01T15:30:00Z",
  "updated_at": "2025-06-01T15:35:00Z",
  "status": "uploaded"
}
```

### Complete Upload Workflow

Here's a complete example workflow:

```bash
# Step 1: Create file and get upload URL
RESPONSE=$(curl -s -X POST http://localhost:8080/files/ \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "document.pdf",
    "content_type": "application/pdf",
    "file_size": 2048576,
    "owner_id": "00000000-0000-0000-0000-000000000001",
    "tenant_id": "00000000-0000-0000-0000-000000000001"
  }')

# Extract content_id and upload_url from response
CONTENT_ID=$(echo $RESPONSE | jq -r '.content_id')
UPLOAD_URL=$(echo $RESPONSE | jq -r '.upload_url')

echo "Content ID: $CONTENT_ID"
echo "Upload URL: $UPLOAD_URL"

# Step 2: Upload file to storage
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: application/pdf" \
  --data-binary @document.pdf

# Step 3: Mark upload complete
curl -X POST http://localhost:8080/files/$CONTENT_ID/complete \
  -H "Content-Type: application/json"

# Step 4: Update metadata (optional)
curl -X PATCH http://localhost:8080/files/$CONTENT_ID \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Important Document",
    "description": "A critical business document",
    "tags": ["business", "document", "important"]
  }'

# Step 5: Get file info with download URLs
curl -X GET http://localhost:8080/files/$CONTENT_ID
```

## Building and Running

### Build
```bash
go build -o bin/files ./cmd/files
```

### Run
```bash
./bin/files
```

### Test
```bash
go test ./cmd/files -v
```

## Dependencies

- PostgreSQL database
- S3-compatible storage (MinIO or AWS S3)
- Go 1.21+

## Architecture

The server uses a layered architecture:
- **API Layer**: HTTP handlers and routing
- **Service Layer**: Business logic
- **Repository Layer**: Data access
- **Storage Layer**: File storage backends

The server supports multiple storage backends and can be extended with additional storage providers.
