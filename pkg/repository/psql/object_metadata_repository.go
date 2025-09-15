// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tendant/simple-content/internal/domain"
)

// PSQLObjectMetadataRepository implements the ObjectMetadataRepository interface
type PSQLObjectMetadataRepository struct {
	BaseRepository
}

// NewPSQLObjectMetadataRepository creates a new PostgreSQL object metadata repository
func NewPSQLObjectMetadataRepository(db DBTX) *PSQLObjectMetadataRepository {
	return &PSQLObjectMetadataRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Set implements ObjectMetadataRepository.Set
func (r *PSQLObjectMetadataRepository) Set(ctx context.Context, objectMetadata *domain.ObjectMetadata) error {
	// Check if the object exists
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM content.object WHERE id = $1 AND deleted_at IS NULL)",
		objectMetadata.ObjectID).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("object not found")
	}

	// Check if metadata already exists
	err = r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM content.object_metadata WHERE object_id = $1)",
		objectMetadata.ObjectID).Scan(&exists)
	if err != nil {
		return err
	}

	// Convert metadata map to JSONB
	metadataJSON, err := json.Marshal(objectMetadata.Metadata)
	if err != nil {
		return err
	}

	var query string
	if exists {
		// Update existing metadata
		query = `
			UPDATE content.object_metadata
			SET 
				size_bytes = $2,
				mime_type = $3,
				etag = $4,
				metadata = $5,
				updated_at = $6
			WHERE object_id = $1
			RETURNING updated_at
		`
	} else {
		// Insert new metadata
		query = `
			INSERT INTO content.object_metadata (
				object_id, size_bytes, mime_type, etag, metadata, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $6
			) RETURNING created_at, updated_at
		`
	}

	now := time.Now().UTC()
	if objectMetadata.UpdatedAt.IsZero() {
		objectMetadata.UpdatedAt = now
	}
	if !exists && objectMetadata.CreatedAt.IsZero() {
		objectMetadata.CreatedAt = now
	}

	if exists {
		var updatedAt time.Time
		err = r.db.QueryRow(
			ctx,
			query,
			objectMetadata.ObjectID,
			objectMetadata.SizeBytes,
			objectMetadata.MimeType,
			objectMetadata.ETag,
			metadataJSON,
			objectMetadata.UpdatedAt,
		).Scan(&updatedAt)
		objectMetadata.UpdatedAt = updatedAt
	} else {
		var createdAt, updatedAt time.Time
		err = r.db.QueryRow(
			ctx,
			query,
			objectMetadata.ObjectID,
			objectMetadata.SizeBytes,
			objectMetadata.MimeType,
			objectMetadata.ETag,
			metadataJSON,
			objectMetadata.CreatedAt,
		).Scan(&createdAt, &updatedAt)
		objectMetadata.CreatedAt = createdAt
		objectMetadata.UpdatedAt = updatedAt
	}

	return err
}

// Get implements ObjectMetadataRepository.Get
func (r *PSQLObjectMetadataRepository) Get(ctx context.Context, objectID uuid.UUID) (*domain.ObjectMetadata, error) {
	query := `
		SELECT 
			object_id, size_bytes, mime_type, etag, metadata, created_at, updated_at
		FROM content.object_metadata
		WHERE object_id = $1 AND deleted_at IS NULL
	`

	metadata := &domain.ObjectMetadata{
		ObjectID: objectID,
		Metadata: make(map[string]interface{}),
	}

	var metadataJSON []byte
	var nullableETag *string

	err := r.db.QueryRow(ctx, query, objectID).Scan(
		&metadata.ObjectID,
		&metadata.SizeBytes,
		&metadata.MimeType,
		&nullableETag,
		&metadataJSON,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("object metadata not found: %w", err)
		}
		return nil, err
	}

	if nullableETag != nil {
		metadata.ETag = *nullableETag
	}

	// Parse the JSON metadata
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &metadata.Metadata); err != nil {
			return nil, err
		}
	}

	return metadata, nil
}
