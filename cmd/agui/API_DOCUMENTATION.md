# AG-UI Content API Documentation

## Overview

The AG-UI Content API provides endpoints for managing file uploads and downloads using presigned URLs for efficient handling of large files.

**Base URL:** `http://localhost:8080/api/v5`

---

## Authentication

Currently, the API does not require authentication. Future versions will support JWT-based authentication.

---

## Endpoints

### 1. Request Upload URL

**POST** `/content/upload`

Creates a content record and returns a presigned upload URL.

#### Request Body

```json
{
  "mime_type": "image/jpeg",
  "filename": "photo.jpg",
  "size": 123456
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mime_type` | string | Yes | MIME type of the file |
| `filename` | string | Yes | Name of the file |
| `size` | integer | No | File size in bytes |

#### Response

```json
{
  "content_id": "550e8400-e29b-41d4-a716-446655440000",
  "upload_url": "https://s3.amazonaws.com/bucket/path?signature=..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `content_id` | string (UUID) | Unique identifier for the content |
| `upload_url` | string (URL) | Presigned URL for uploading the file |

#### Example

```bash
curl -X POST http://localhost:8080/api/v5/content/upload \
  -H "Content-Type: application/json" \
  -d '{
    "mime_type": "image/jpeg",
    "filename": "photo.jpg",
    "size": 123456
  }'
```

---

### 2. Mark Upload Complete

**POST** `/content/upload/done`

Marks a content upload as complete after the file has been uploaded to the presigned URL.

#### Request Body

```json
{
  "content_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content_id` | string (UUID) | Yes | Content ID from the upload response |

#### Response

```json
{
  "content_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processed"
}
```

#### Example

```bash
curl -X POST http://localhost:8080/api/v5/content/upload/done \
  -H "Content-Type: application/json" \
  -d '{
    "content_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

---

### 3. Get Content Details

**GET** `/content/contents/{contentId}`

Retrieves metadata and download URL for a specific content.

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `contentId` | string (UUID) | Yes | Content ID |

#### Response

```json
{
  "content_id": "550e8400-e29b-41d4-a716-446655440000",
  "file_name": "photo.jpg",
  "file_size": 123456,
  "download_url": "https://s3.amazonaws.com/bucket/path?signature=..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `content_id` | string (UUID) | Unique identifier for the content |
| `file_name` | string | Name of the file |
| `file_size` | integer | File size in bytes |
| `download_url` | string (URL) | Presigned URL for downloading the file |

#### Example

```bash
curl http://localhost:8080/api/v5/content/contents/550e8400-e29b-41d4-a716-446655440000
```

---

### 4. List Contents

**GET** `/content/contents`

Retrieves a list of all contents with their metadata and download URLs.

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `limit` | integer | No | 100 | Maximum number of items to return |
| `offset` | integer | No | 0 | Number of items to skip |

#### Response

```json
{
  "contents": [
    {
      "content_id": "550e8400-e29b-41d4-a716-446655440000",
      "file_name": "photo.jpg",
      "file_size": 123456,
      "download_url": "https://s3.amazonaws.com/bucket/path?signature=..."
    },
    {
      "content_id": "660e8400-e29b-41d4-a716-446655440001",
      "file_name": "document.pdf",
      "file_size": 789012,
      "download_url": "https://s3.amazonaws.com/bucket/path2?signature=..."
    }
  ],
  "total": 2
}
```

#### Example

```bash
curl "http://localhost:8080/api/v5/content/contents?limit=10&offset=0"
```

---

### 5. Delete Content

**DELETE** `/content/contents/{contentId}`

Deletes a content and its associated files.

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `contentId` | string (UUID) | Yes | Content ID |

#### Response

```json
{
  "message": "Content deleted"
}
```

#### Example

```bash
curl -X DELETE http://localhost:8080/api/v5/content/contents/550e8400-e29b-41d4-a716-446655440000
```

---

## Upload Workflow

### Complete Upload Flow

1. **Request Upload URL**
   ```bash
   POST /content/upload
   {
     "mime_type": "image/jpeg",
     "filename": "photo.jpg",
     "size": 123456
   }
   ```
   Response: `{ "content_id": "...", "upload_url": "..." }`

2. **Upload File to S3**
   ```bash
   curl -X PUT "<upload_url>" \
     -H "Content-Type: image/jpeg" \
     --data-binary @photo.jpg
   ```

3. **Mark Upload Complete**
   ```bash
   POST /content/upload/done
   {
     "content_id": "..."
   }
   ```

4. **Get Content Details**
   ```bash
   GET /content/contents/{contentId}
   ```
   Response includes `download_url` for accessing the file

---

## Error Responses

All endpoints return errors in the following format:

```json
{
  "error": "Error message description"
}
```

### Common HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid input |
| 404 | Not Found - Resource doesn't exist |
| 500 | Internal Server Error |

---

## Examples

### Complete Example: Upload and Download

```bash
# 1. Request upload URL
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v5/content/upload \
  -H "Content-Type: application/json" \
  -d '{"mime_type":"image/jpeg","filename":"photo.jpg","size":123456}')

CONTENT_ID=$(echo $RESPONSE | jq -r '.content_id')
UPLOAD_URL=$(echo $RESPONSE | jq -r '.upload_url')

echo "Content ID: $CONTENT_ID"
echo "Upload URL: $UPLOAD_URL"

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

# 5. Download the file
DOWNLOAD_RESPONSE=$(curl -s http://localhost:8080/api/v5/content/contents/$CONTENT_ID)
DOWNLOAD_URL=$(echo $DOWNLOAD_RESPONSE | jq -r '.download_url')
curl -o downloaded_photo.jpg "$DOWNLOAD_URL"
```

---

## Configuration

The API can be configured using environment variables. See `.env.example` for available options:

- `DATABASE_URL` - PostgreSQL connection string
- `STORAGE_URL` - S3 bucket URL
- `S3_ENDPOINT` - S3 endpoint (for MinIO or compatible services)
- `AWS_REGION` - AWS region
- `PORT` - Server port (default: 8080)

---

## Health Check

**GET** `/health`

Returns the health status of the API.

#### Response

```json
{
  "status": "ok"
}
```

#### Example

```bash
curl http://localhost:8080/health
```
