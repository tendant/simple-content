# Environment Variable Configuration

## Overview

The configuration system has been **dramatically simplified**. Only **3 environment variables** are needed for most use cases:

```bash
PORT=8080                                          # Server port
DATABASE_URL=postgresql://user:pass@host/db       # Database
STORAGE_URL=file:///var/data                      # Storage
```

That's it! For advanced features, use [programmatic configuration](./README.md).

## Complete Reference

### Server Configuration (cmd/server-configured only)

```bash
PORT=8080                    # HTTP server port (default: "8080")
ENVIRONMENT=production       # Runtime environment: development, production, testing
```

**Note:** Library users should ignore these - you control your own server.

### Database Configuration

```bash
DATABASE_URL=<connection-string>
```

**Supported values:**
- `memory` or empty - In-memory database (default)
- `postgresql://user:pass@host/db` - PostgreSQL database
- `postgres://user:pass@host/db` - PostgreSQL database (alt syntax)

**Examples:**
```bash
# Development (in-memory)
DATABASE_URL=memory

# Production (PostgreSQL)
DATABASE_URL=postgresql://myapp:secret@db.example.com:5432/myapp_prod
```

### Storage Configuration

```bash
STORAGE_URL=<storage-url>
```

**Supported formats:**
- `memory` or `memory://` - In-memory storage (default)
- `file:///path/to/data` - Filesystem storage
- `s3://bucket-name` - S3 storage

**Examples:**
```bash
# Development (in-memory)
STORAGE_URL=memory

# Filesystem storage
STORAGE_URL=file:///var/data/storage

# S3 storage (uses AWS_* env vars for credentials)
STORAGE_URL=s3://my-app-bucket
AWS_REGION=us-west-2
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

## Complete Examples

### Development

```bash
#!/bin/bash
# Minimal development setup - everything in memory

export PORT=8080
export DATABASE_URL=memory
export STORAGE_URL=memory

./server-configured
```

### Production with PostgreSQL + Filesystem

```bash
#!/bin/bash
# Production setup with database and filesystem storage

export PORT=8080
export ENVIRONMENT=production
export DATABASE_URL=postgresql://myapp:${DB_PASSWORD}@db.prod.example.com/myapp
export STORAGE_URL=file:///var/data/storage

./server-configured
```

### Production with PostgreSQL + S3

```bash
#!/bin/bash
# Production setup with S3 storage

export PORT=8080
export ENVIRONMENT=production

# Database
export DATABASE_URL=postgresql://myapp:${DB_PASSWORD}@db.prod.example.com/myapp

# S3 Storage
export STORAGE_URL=s3://my-app-production-bucket
export AWS_REGION=us-west-2
export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_KEY}

./server-configured
```

### Docker Compose

```yaml
version: '3.8'
services:
  app:
    image: my-app:latest
    ports:
      - "8080:8080"
    environment:
      PORT: "8080"
      ENVIRONMENT: "production"
      DATABASE_URL: "postgresql://myapp:secret@db:5432/myapp"
      STORAGE_URL: "s3://my-app-bucket"
      AWS_REGION: "us-west-2"
      AWS_ACCESS_KEY_ID: "${AWS_ACCESS_KEY_ID}"
      AWS_SECRET_ACCESS_KEY: "${AWS_SECRET_ACCESS_KEY}"

  db:
    image: postgres:16
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: myapp
      POSTGRES_PASSWORD: secret
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  PORT: "8080"
  ENVIRONMENT: "production"
  DATABASE_URL: "postgresql://myapp:secret@postgres-service:5432/myapp"
  STORAGE_URL: "s3://my-app-bucket"
  AWS_REGION: "us-west-2"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: app
        image: my-app:latest
        envFrom:
        - configMapRef:
            name: app-config
        env:
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: secret-access-key
```

## What Happened to All the Other Variables?

**They're gone!** The old configuration had 20+ environment variables:

```bash
# ❌ OLD (complex, confusing)
FS_BASE_DIR=/var/data
FS_URL_PREFIX=http://localhost:8080
FS_SIGNATURE_SECRET_KEY=secret
FS_PRESIGN_EXPIRES_SECONDS=3600
DEFAULT_STORAGE_BACKEND=fs
OBJECT_KEY_GENERATOR=git-like
URL_STRATEGY=storage-delegated
# ... and many more
```

```bash
# ✅ NEW (simple, clear)
STORAGE_URL=file:///var/data
```

**For advanced features**, use [programmatic configuration](./README.md):

```go
cfg, err := config.Load(
    config.WithFilesystemStorageFull("fs", "/var/data", "/api/v1", "secret", 3600),
    config.WithObjectKeyGenerator("git-like"),
    config.WithStorageDelegatedURLs(),
    config.WithEnv(""),  // Still load PORT, DATABASE_URL, STORAGE_URL from env
)
```

## Migration Guide

### From Old Environment Variables

**Old configuration:**
```bash
DATABASE_TYPE=postgres
DATABASE_URL=postgresql://localhost/db
FS_BASE_DIR=/var/data
DEFAULT_STORAGE_BACKEND=fs
```

**New configuration:**
```bash
DATABASE_URL=postgresql://localhost/db  # Type auto-detected
STORAGE_URL=file:///var/data            # Backend auto-configured
```

### From Programmatic to Environment

**Before (programmatic only):**
```go
cfg, err := config.Load(
    config.WithDatabase("postgres", "postgresql://localhost/db"),
    config.WithFilesystemStorage("fs", "/var/data", "", ""),
)
```

**After (environment + programmatic):**
```bash
# .env
DATABASE_URL=postgresql://localhost/db
STORAGE_URL=file:///var/data
```

```go
cfg, err := config.Load(config.WithEnv(""))
```

## Benefits

1. **Simpler** - 3 variables instead of 20+
2. **Standard** - Uses industry-standard URL formats (DATABASE_URL, STORAGE_URL)
3. **12-Factor compliant** - Follows 12-factor app principles
4. **Container-friendly** - Perfect for Docker/Kubernetes
5. **Clear separation** - Environment for infrastructure, code for business logic

## See Also

- [Programmatic Configuration](./README.md) - For advanced features
- [Library Usage](./LIBRARY_USAGE.md) - Using as a library
- [Examples](/examples) - Working examples
