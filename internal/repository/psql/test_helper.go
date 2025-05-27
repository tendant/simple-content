package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestDB represents a test database connection
type TestDB struct {
	Pool *pgxpool.Pool
}

// NewTestDB creates a new test database connection
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Get database connection string from environment variable or use a default for local testing
	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		connString = "postgres://powercard:pwd@localhost:5432/powercard_db?sslmode=disable"
	}

	// Connect to the database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err, "Failed to connect to test database")

	// Verify the connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping test database")

	return &TestDB{
		Pool: pool,
	}
}

// Setup initializes the test database with required schema and tables
func (db *TestDB) Setup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Create schema if it doesn't exist
	_, err := db.Pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS content")
	require.NoError(t, err, "Failed to create content schema")

	// // Create extension for UUID generation
	// _, err = db.Pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS uuid-ossp")
	// require.NoError(t, err, "Failed to create uuid-ossp extension")

	// Create content table
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS content.content (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID,
			owner_id UUID,
			owner_type VARCHAR(50),
			name VARCHAR(255),
			description TEXT,
			document_type VARCHAR(50),
			status VARCHAR(50) NOT NULL DEFAULT 'created',
			derivation_type VARCHAR(50),
			created_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			deleted_at TIMESTAMP
		)
	`)
	require.NoError(t, err, "Failed to create content table")

	// Create content_metadata table
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS content.content_metadata (
			content_id UUID PRIMARY KEY REFERENCES content.content(id),
			tags TEXT[],
			file_size BIGINT,
			file_name VARCHAR(255),
			mime_type VARCHAR(255) NOT NULL,
			checksum VARCHAR(128),
			checksum_algorithm VARCHAR(20) DEFAULT 'SHA256',
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			deleted_at TIMESTAMP
		)
	`)
	require.NoError(t, err, "Failed to create content_metadata table")

	// Create content_derived table
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS content.content_derived (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			parent_content_id UUID NOT NULL REFERENCES content.content(id),
			derived_content_id UUID NOT NULL REFERENCES content.content(id),
			derivation_type VARCHAR(100) NOT NULL,
			derivation_params JSONB,
			processing_status VARCHAR(50) NOT NULL DEFAULT 'pending',
			processing_metadata JSONB,
			created_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			deleted_at TIMESTAMP,
			CONSTRAINT unique_source_derivation UNIQUE (parent_content_id, derived_content_id),
			CONSTRAINT no_self_reference CHECK (parent_content_id != derived_content_id)
		)
	`)
	require.NoError(t, err, "Failed to create object_metadata table")

	// Create object table
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS content.object (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			content_id UUID NOT NULL REFERENCES content.content(id),
			storage_backend_name VARCHAR(100) NOT NULL,
			storage_class VARCHAR(100),
			object_key VARCHAR(1024) NOT NULL,
			file_name VARCHAR(1024),
			version_id VARCHAR(255),
			object_type VARCHAR(100),
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			deleted_at TIMESTAMP,
			CONSTRAINT unique_storage_path UNIQUE (storage_backend_name, object_key)
		)
	`)
	require.NoError(t, err, "Failed to create object table")

	// Create object_metadata table
	_, err = db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS content.object_metadata (
			object_id UUID PRIMARY KEY REFERENCES content.object(id),
			size_bytes BIGINT NOT NULL,
			mime_type VARCHAR(255) NOT NULL,
			etag VARCHAR(255),
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
			deleted_at TIMESTAMP
		)
	`)
	require.NoError(t, err, "Failed to create object_metadata table")
}

// Cleanup removes all test data from the database
func (db *TestDB) Cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Clean up tables in reverse order of dependencies
	_, err := db.Pool.Exec(ctx, "TRUNCATE content.content_derived CASCADE")
	require.NoError(t, err, "Failed to truncate content_derived table")

	_, err = db.Pool.Exec(ctx, "TRUNCATE content.object_metadata CASCADE")
	require.NoError(t, err, "Failed to truncate object_metadata table")

	_, err = db.Pool.Exec(ctx, "TRUNCATE content.object CASCADE")
	require.NoError(t, err, "Failed to truncate object table")

	_, err = db.Pool.Exec(ctx, "TRUNCATE content.content_metadata CASCADE")
	require.NoError(t, err, "Failed to truncate content_metadata table")

	_, err = db.Pool.Exec(ctx, "TRUNCATE content.content CASCADE")
	require.NoError(t, err, "Failed to truncate content table")
}

// Close closes the database connection
func (db *TestDB) Close(t *testing.T) {
	t.Helper()
	db.Pool.Close()
}

// RunTest runs a test with database setup and cleanup
func RunTest(t *testing.T, testFunc func(t *testing.T, db *TestDB)) {
	t.Helper()

	// Skip if in short mode or if the database connection is not available
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	// Setup test database
	db := NewTestDB(t)
	defer db.Close(t)

	// Setup schema and tables
	db.Setup(t)

	// Run the test
	t.Run("", func(t *testing.T) {
		// Clean up before the test
		db.Cleanup(t)

		// Run the test
		testFunc(t, db)
	})
}
