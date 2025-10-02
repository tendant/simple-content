# Docker Development Setup

Quick reference for local development with Docker Compose.

## Quick Start

```bash
# 1. Start services (Postgres + MinIO)
./scripts/docker-dev.sh start

# 2. Run database migrations
./scripts/run-migrations.sh up

# 3. (Optional) Create S3 bucket for MinIO storage
aws --endpoint-url http://localhost:9000 s3 mb s3://content-bucket

# 4. Run the application
ENVIRONMENT=development \
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
STORAGE_BACKEND=memory \
go run ./cmd/server-configured
```

## Service Details

### PostgreSQL
- **Host**: `localhost:5433` (mapped from container port 5432)
- **Database**: `simple_content`
- **User**: `content`
- **Password**: `contentpass`
- **Schema**: `content` (created automatically)

**Connection String:**
```
postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content
```

### MinIO (S3-Compatible Storage)
- **API Endpoint**: `http://localhost:9000`
- **Console**: `http://localhost:9001`
- **Access Key**: `minioadmin`
- **Secret Key**: `minioadmin`
- **Region**: `us-east-1`

**Configuration for MinIO:**
```bash
export AWS_S3_ENDPOINT=http://localhost:9000
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_BUCKET=content-bucket
export AWS_S3_REGION=us-east-1
export AWS_S3_USE_SSL=false
```

## Helper Scripts

### Docker Service Management

```bash
# Start Postgres and MinIO
./scripts/docker-dev.sh start

# Stop services
./scripts/docker-dev.sh stop

# Restart services
./scripts/docker-dev.sh restart

# View logs
./scripts/docker-dev.sh logs

# Check service status
./scripts/docker-dev.sh status

# Clean up (removes data volumes!)
./scripts/docker-dev.sh clean
```

### Database Migrations

```bash
# Run all pending migrations
./scripts/run-migrations.sh up

# Rollback last migration
./scripts/run-migrations.sh down

# Check migration status
./scripts/run-migrations.sh status

# Create new migration
./scripts/run-migrations.sh create add_new_feature sql

# Reset database (down then up)
./scripts/run-migrations.sh reset
```

**Custom Database Connection:**
```bash
# Override default connection settings
CONTENT_PG_HOST=custom-host \
CONTENT_PG_PORT=5432 \
CONTENT_PG_NAME=my_db \
./scripts/run-migrations.sh up
```

## Running Tests

### Unit Tests (Memory Backend)
```bash
go test ./pkg/simplecontent/...
```

### Integration Tests (Postgres + MinIO)
```bash
# Start services
./scripts/docker-dev.sh start
./scripts/run-migrations.sh up

# Run tests
DATABASE_TYPE=postgres \
DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' \
go test -tags=integration ./pkg/simplecontent/...

# Stop services
./scripts/docker-dev.sh stop
```

## Running the Full Stack

To run the API server along with Postgres and MinIO:

```bash
# Start all services (Postgres + MinIO + API)
docker-compose up --build

# Or in detached mode
docker-compose up -d --build

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Stop and remove volumes (clean slate)
docker-compose down -v
```

**API Server Access:**
- Base URL: `http://localhost:4000`
- Health: `http://localhost:4000/health`
- API: `http://localhost:4000/api/v1`

## Troubleshooting

### Port Already in Use

If ports are already in use, modify `docker-compose.yml`:

```yaml
postgres:
  ports:
    - "5434:5432"  # Change 5433 to another port

minio:
  ports:
    - "9010:9000"  # Change 9000 to another port
    - "9011:9001"  # Change 9001 to another port
```

### Database Connection Issues

Check if Postgres is healthy:
```bash
docker ps  # Look for simple-content-postgres
docker logs simple-content-postgres
```

Test connection:
```bash
PGPASSWORD=contentpass psql -h localhost -p 5433 -U content -d simple_content -c "SELECT 1"
```

### MinIO Connection Issues

Check if MinIO is healthy:
```bash
docker ps  # Look for simple-content-minio
docker logs simple-content-minio
```

Test MinIO:
```bash
aws --endpoint-url http://localhost:9000 s3 ls
```

### Clean Slate

To completely reset the development environment:

```bash
# Stop and remove all containers and volumes
./scripts/docker-dev.sh clean

# Start fresh
./scripts/docker-dev.sh start

# Run migrations
./scripts/run-migrations.sh up
```

## Environment Configuration Examples

### Development (Memory Storage)
```bash
export ENVIRONMENT=development
export DATABASE_TYPE=postgres
export DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content'
export STORAGE_BACKEND=memory
```

### Development (Filesystem Storage)
```bash
export ENVIRONMENT=development
export DATABASE_TYPE=postgres
export DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content'
export STORAGE_BACKEND=fs
export FS_BASE_PATH=./data
```

### Development (MinIO S3 Storage)
```bash
export ENVIRONMENT=development
export DATABASE_TYPE=postgres
export DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content'
export STORAGE_BACKEND=s3
export AWS_S3_ENDPOINT=http://localhost:9000
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_BUCKET=content-bucket
export AWS_S3_REGION=us-east-1
export AWS_S3_USE_SSL=false
```

## Data Persistence

### Volumes

Docker volumes persist data between container restarts:
- `postgres-data`: PostgreSQL database files
- `minio-data`: MinIO object storage files

**View volumes:**
```bash
docker volume ls | grep simple-content
```

**Inspect volume:**
```bash
docker volume inspect simple-content_postgres-data
docker volume inspect simple-content_minio-data
```

**Backup volume:**
```bash
# Backup Postgres
docker run --rm \
  -v simple-content_postgres-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres-backup.tar.gz /data

# Backup MinIO
docker run --rm \
  -v simple-content_minio-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/minio-backup.tar.gz /data
```

## Next Steps

1. **Start coding**: Services are ready for development
2. **Add migrations**: Create new migrations in `migrations/postgres/`
3. **Run tests**: Use integration tests to verify changes
4. **Extend API**: Add new endpoints in `cmd/server-configured/`
5. **Deploy**: Use the same docker-compose pattern for staging/production

## Additional Resources

- [README.md](./README.md) - Full project documentation
- [PROGRAMMATIC_USAGE.md](./PROGRAMMATIC_USAGE.md) - Library usage guide
- [CLAUDE.md](./CLAUDE.md) - Architecture and conventions
- [migrations/postgres/](./migrations/postgres/) - Database schema migrations
