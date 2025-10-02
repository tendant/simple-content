#!/bin/bash
set -e

# Database connection parameters (defaults for docker-compose setup)
DB_HOST="${CONTENT_PG_HOST:-localhost}"
DB_PORT="${CONTENT_PG_PORT:-5433}"
DB_NAME="${CONTENT_PG_NAME:-simple_content}"
DB_USER="${CONTENT_PG_USER:-content}"
DB_PASSWORD="${CONTENT_PG_PASSWORD:-contentpass}"
DB_SCHEMA="${CONTENT_DB_SCHEMA:-content}"

# Construct connection string
DATABASE_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable&search_path=${DB_SCHEMA}"

echo "Running migrations..."
echo "Database: ${DB_NAME}"
echo "Schema: ${DB_SCHEMA}"
echo "Host: ${DB_HOST}:${DB_PORT}"

# Check if goose is installed
if ! command -v goose &> /dev/null; then
    echo "Error: goose is not installed. Install with: go install github.com/pressly/goose/v3/cmd/goose@latest"
    exit 1
fi

# Run migrations
goose -dir ./migrations/postgres postgres "$DATABASE_URL" "$@"

echo "Migrations completed successfully!"
