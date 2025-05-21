# Content System Design Document

## Overview
This document outlines the architecture and design for a generic content management system that supports multi-backend storage, versioning, preview generation, lifecycle events, and audit trails. The system is designed to be flexible, extensible, and integration-friendly with external systems handling access control (ACL).

## Objectives
- Store and manage content with metadata.
- Support storing content across multiple storage backends.
- Enable versioning of content objects.
- Allow preview generation for supported content types.
- Emit lifecycle events for system observability and async processing.
- Track audit trails for create, update, and access actions.
- Delegate access control (ACL) to external systems.

## Core Concepts

### Content
- Identified by an externally supplied `content_id` (UUID).
- Can have multiple versions.
- Each version can be stored in one or more storage backends.

### Object
- Represents the actual binary stored in a storage backend.
- One content version may have multiple associated objects (across backends).

### Metadata
- **Content Metadata**: General attributes (e.g., source system, filename).
- **Object Metadata**: Backend-specific or technical attributes (e.g., content type, checksum).

### Preview
- Previews are generated asynchronously for supported types.
- Multiple types of previews are supported (image thumbnail, PDF render, etc.).

### Audit Trail
- Captures create, update, access events.
- Stores actor identity, timestamp, and optional metadata.

### Lifecycle Events
- Published on a message bus (e.g., Kafka, NATS, or SQS).
- Used to trigger preview generation, cleanup, notifications, etc.


## Data Model

### Tables

\`\`\`sql
-- Logical content entity
content {
  id UUID PRIMARY KEY,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  owner_id UUID,
  tenant_id UUID,
  status TEXT
}

-- Custom metadata for a content
content_metadata {
  content_id UUID FK,
  metadata JSONB
}

-- Physical object stored in a backend
object {
  id UUID PRIMARY KEY,
  content_id UUID FK REFERENCES content(id),
  storage_backend_id UUID FK REFERENCES storage_backend(id),
  version INT DEFAULT 1,
  object_key TEXT,
  status TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
}

-- Metadata about the object (e.g., checksum, mime type)
object_metadata {
  object_id UUID FK REFERENCES object(id),
  metadata JSONB
}

-- Preview generated from an object
object_preview {
  id UUID PRIMARY KEY,
  object_id UUID FK REFERENCES object(id),
  preview_type TEXT,
  preview_url TEXT,
  status TEXT,
  created_at TIMESTAMP
}

-- Configurable storage backends
storage_backend {
  id UUID PRIMARY KEY,
  name TEXT UNIQUE,               -- e.g. 's3-us-west'
  type TEXT,                      -- ENUM: 's3', 'gcs', 'azure', 'local'
  config JSONB,                   -- e.g. bucket, region
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
}

-- Event log for audits
audit_event {
  id UUID PRIMARY KEY,
  content_id UUID FK REFERENCES content(id),
  object_id UUID FK REFERENCES object(id),
  actor_id UUID,
  action TEXT,
  metadata JSONB,
  created_at TIMESTAMP
}

-- Optional access log
access_log {
  id UUID PRIMARY KEY,
  content_id UUID FK REFERENCES content(id),
  actor_id UUID,
  method TEXT,
  storage_backend TEXT,
  created_at TIMESTAMP
}
\`\`\`

## APIs

### Content Interface
- `POST /content` → Create content
- `GET /content/{id}/upload-url` → Get upload URL
- `PUT /content/{id}/metadata` → Update metadata
- `GET /content/{id}/metadata` → Get metadata
- `GET /content/{id}/preview` → Get preview URL
- `GET /content/{id}/download` → Get download URL
- `DELETE /content/{id}` → Delete content

### Storage Interface
- `GET /object/{id}/upload-url`
- `GET /object/{id}/download-url`
- `POST /object/{id}/upload`
- `GET /object/{id}/download`
- `GET /object/{id}/metadata`
- `PUT /object/{id}/metadata`

## External ACL Integration
- Identity propagated via headers (`x-user-id`, `x-tenant-id`, etc.).
- Ownership fields (`owner_id`, `tenant_id`) stored in DB.
- External systems responsible for enforcing access rights.

## Optional Enhancements
- Retention policies / TTL
- Soft delete with purge logic
- Tagging / labeling
- Quota enforcement
- Deduplication by content fingerprint
- Chunked uploads for large files

## Deployment Considerations
- Deploy preview workers as a queue consumer.
- Use observability tools to track lifecycle events and failures.
- Scale storage backend interface using adapter pattern.
