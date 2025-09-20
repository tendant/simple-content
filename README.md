# Simple Content Management System

A flexible content management system that supports multi-backend storage, versioning, and metadata management.

## Features

- Store and manage content with metadata
- Support for multiple storage backends (currently in-memory, with extensibility for file system and S3)
- Content versioning
- Metadata management for both content and objects
- RESTful API for content and object operations

## Getting Started

### Prerequisites

- Go 1.24 or higher

### Installation

1. Clone the repository:

```bash
git clone https://github.com/tendant/simple-content.git
cd simple-content
```

2. Build the application:

```bash
go build -o simple-content ./cmd/server
```

3. Run the server:

```bash
./simple-content
```

The server will start on port 8080 by default. You can change the port by setting the `PORT` environment variable.

### Database schema

- Postgres deployments default to using a dedicated `content` schema. Provision it ahead of goose migrations (see `migrations/manual/000_create_schema.sql` for a helper snippet).
- Configure the database connection that runs migrations to set `search_path` to the target schema (for example, append `?search_path=content` to the Postgres connection string).

## Docker Deployment

For a complete development environment with PostgreSQL and MinIO, you can use Docker Compose:

### Prerequisites

- Docker and Docker Compose installed

### Quick Start with Docker

1. Clone the repository:

```bash
git clone https://github.com/tendant/simple-content.git
cd simple-content
```

2. Start all services:

```bash
docker network create simple-content-network
```

```bash
docker compose up --build
```

This will start:
- **PostgreSQL** database on port 5432
- **MinIO** object storage on ports 9000 (API) and 9001 (Console)
- **Files API** server on port 8080

3. Access the services:
- Files API: http://localhost:8080
- MinIO Console: http://localhost:9001 (admin/minioadmin)
- Health check: http://localhost:8080/health

4. Stop all services:

```bash
docker compose down
```

### Environment Variables

The Docker Compose setup uses the following environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | 0.0.0.0 | Server bind address |
| `SERVER_PORT` | 8080 | Server port |
| `DB_HOST` | postgres | Database host |
| `DB_PORT` | 5432 | Database port |
| `DB_NAME` | powercard_db | Database name |
| `DB_USER` | content | Database user |
| `DB_PASSWORD` | pwd | Database password |
| `S3_ENDPOINT` | minio:9000 | S3 endpoint |
| `S3_ACCESS_KEY_ID` | minioadmin | S3 access key |
| `S3_SECRET_ACCESS_KEY` | minioadmin | S3 secret key |
| `S3_BUCKET_NAME` | content-bucket | S3 bucket name |
| `S3_REGION` | us-east-1 | S3 region |
| `S3_USE_SSL` | false | Use SSL for S3 |

### Docker Services

- **postgres**: PostgreSQL 15 database with persistent storage
- **minio**: MinIO object storage compatible with S3 API
- **files-api**: The Simple Content Files API server

## API Usage

### New HTTP API (configured server)

The configured server under `cmd/server-configured` exposes a library-first API under `/api/v1`:

- Contents
  - `POST /api/v1/contents` — create content
  - `GET /api/v1/contents/{contentID}` — get content (includes `derivation_type` for derived, and `variant` when available)
  - `PUT /api/v1/contents/{contentID}` — update content
  - `DELETE /api/v1/contents/{contentID}` — delete content
  - `GET /api/v1/contents?owner_id=&tenant_id=` — list contents
  - `POST /api/v1/contents/{parentID}/derived` — create derived content (body: `owner_id`, `tenant_id`, `derivation_type`, `variant`, `metadata`)
  - `GET /api/v1/contents/{contentID}/derived` — list all derived contents for a parent (each item includes `derivation_type` and `variant`)

- Content metadata
  - `POST /api/v1/contents/{contentID}/metadata` — set metadata
  - `GET /api/v1/contents/{contentID}/metadata` — get metadata

- Objects
  - `POST /api/v1/contents/{contentID}/objects` — create object
  - `GET /api/v1/objects/{objectID}` — get object
  - `DELETE /api/v1/objects/{objectID}` — delete object
  - `GET /api/v1/contents/{contentID}/objects` — list objects by content

- Upload/Download
  - `POST /api/v1/objects/{objectID}/upload` — direct upload (uses `Content-Type` when present)
  - `GET /api/v1/objects/{objectID}/download` — stream download
  - `GET /api/v1/objects/{objectID}/upload-url` — presigned upload URL
  - `GET /api/v1/objects/{objectID}/download-url` — presigned download URL
  - `GET /api/v1/objects/{objectID}/preview-url` — preview URL

Derivation semantics:

- `derivation_type` on Content is a user-facing type for derived items (e.g., `thumbnail`, `preview`, `transcode`) and is omitted for originals.
- `variant` (specific) is stored on the derived relationship (e.g., `thumbnail_256`, `mp4_1080p`). If only `variant` is provided at creation time, the server infers `derivation_type` from its prefix.

### Storage Backends

Before uploading content, you need to create a storage backend:

```bash
# Create a memory storage backend
curl -X POST http://localhost:8080/storage-backend \
  -H "Content-Type: application/json" \
  -d '{
    "name": "memory-backend",
    "type": "memory",
    "config": {}
  }'
```
Response:

```json
{
  "name":"memory-backend",
  "type":"memory",
  "config":{},
  "is_active":true,
  "created_at":"2025-05-23T14:33:59.118817-07:00",
  "updated_at":"2025-05-23T14:33:59.118817-07:00"
}
```

### Content Management

#### Create Content

```bash
# Create a new content
curl -X POST http://localhost:8080/content \
  -H "Content-Type: application/json" \
  -d '{
    "owner_id": "00000000-0000-0000-0000-000000000001",
    "tenant_id": "00000000-0000-0000-0000-000000000001"
  }'
```

Response:

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "created_at": "2025-05-21T15:04:05Z",
  "updated_at": "2025-05-21T15:04:05Z",
  "owner_id": "00000000-0000-0000-0000-000000000001",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "status": "active"
}
```

#### Add Metadata to Content

```bash
# Add metadata to content
curl -X PUT http://localhost:8080/content/123e4567-e89b-12d3-a456-426614174000/metadata \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "example.txt",
    "description": "An example text file",
    "tags": ["example", "text"]
  }'
```

#### Create an Object for Content

```bash
# Create an object for content (replace storage_backend_name with the name from your storage backend creation)
curl -X POST http://localhost:8080/content/123e4567-e89b-12d3-a456-426614174000/objects \
  -H "Content-Type: application/json" \
  -d '{
    "storage_backend_name": "memory-backend",
    "version": 1
  }'
```

Response:

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174002",
  "content_id": "123e4567-e89b-12d3-a456-426614174000",
  "storage_backend_name": "memory-backend",
  "version": 1,
  "object_key": "123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174002",
  "status": "pending",
  "created_at": "2025-05-21T15:04:05Z",
  "updated_at": "2025-05-21T15:04:05Z"
}
```

#### Upload Content to Object

```bash
# Upload content to object
curl -X POST http://localhost:8080/object/123e4567-e89b-12d3-a456-426614174002/upload \
  -H "Content-Type: application/octet-stream" \
  --data-binary @example.txt
```

#### Download Content

```bash
# Download content
curl -X GET http://localhost:8080/object/123e4567-e89b-12d3-a456-426614174002/download \
  -o downloaded_example.txt
```

#### Add Metadata to Object

```bash
# Add metadata to object
curl -X PUT http://localhost:8080/object/123e4567-e89b-12d3-a456-426614174002/metadata \
  -H "Content-Type: application/json" \
  -d '{
    "content_type": "text/plain",
    "size": 1024,
    "checksum": "d41d8cd98f00b204e9800998ecf8427e"
  }'
```

### List Operations

#### List Content

```bash
# List all content
curl -X GET http://localhost:8080/content/list

# List content by owner
curl -X GET http://localhost:8080/content/list?owner_id=00000000-0000-0000-0000-000000000001

# List content by tenant
curl -X GET http://localhost:8080/content/list?tenant_id=00000000-0000-0000-0000-000000000001
```

#### List Objects for Content

```bash
# List objects for content
curl -X GET http://localhost:8080/content/123e4567-e89b-12d3-a456-426614174000/objects
```

#### List Storage Backends

```bash
# List all storage backends
curl -X GET http://localhost:8080/storage-backend
```

### Delete Operations

#### Delete Content

```bash
# Delete content (this will also delete all associated objects)
curl -X DELETE http://localhost:8080/content/123e4567-e89b-12d3-a456-426614174000
```

#### Delete Object

```bash
# Delete object
curl -X DELETE http://localhost:8080/object/123e4567-e89b-12d3-a456-426614174002
```

#### Delete Storage Backend

```bash
# Delete storage backend
curl -X DELETE http://localhost:8080/storage-backend/123e4567-e89b-12d3-a456-426614174001
```

## Architecture

The system is designed with a clean architecture approach:

- **Domain Layer**: Core business entities and interfaces
- **Repository Layer**: Data access interfaces and implementations
- **Service Layer**: Business logic and operations
- **API Layer**: HTTP handlers and routes

## Future Enhancements

- Persistent storage with PostgreSQL
- File system storage backend
- S3 storage backend
- Preview generation
- Audit trail implementation
- Event system for lifecycle events

## License

This project is licensed under the MIT License - see the LICENSE file for details.
