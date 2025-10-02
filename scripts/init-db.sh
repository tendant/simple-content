#!/bin/bash
set -e

echo "Initializing Simple Content database..."

# Create the content schema
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create content schema
    CREATE SCHEMA IF NOT EXISTS content;

    -- Set search path
    ALTER DATABASE $POSTGRES_DB SET search_path TO content,public;

    -- Grant privileges
    GRANT ALL PRIVILEGES ON SCHEMA content TO $POSTGRES_USER;
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA content TO $POSTGRES_USER;
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA content TO $POSTGRES_USER;
EOSQL

echo "Database initialized successfully!"
echo "Note: Run migrations with 'goose -dir ./migrations/postgres postgres \"postgresql://$POSTGRES_USER:$POSTGRES_PASSWORD@localhost:5433/$POSTGRES_DB?sslmode=disable&search_path=content\" up'"
