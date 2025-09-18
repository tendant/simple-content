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

// PSQLContentMetadataRepository implements the ContentMetadataRepository interface
type PSQLContentMetadataRepository struct {
	BaseRepository
}

// NewPSQLContentMetadataRepository creates a new PostgreSQL content metadata repository
func NewPSQLContentMetadataRepository(db DBTX) *PSQLContentMetadataRepository {
	return &PSQLContentMetadataRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Set implements ContentMetadataRepository.Set
func (r *PSQLContentMetadataRepository) Set(ctx context.Context, metadata *domain.ContentMetadata) error {
	// Check if the metadata already exists
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM content.content_metadata WHERE content_id = $1)",
		metadata.ContentID).Scan(&exists)
	if err != nil {
		return err
	}

	// Convert metadata map to JSONB
	metadataJSON, err := json.Marshal(metadata.Metadata)
	if err != nil {
		return err
	}

	var query string
	if exists {
		// Update existing metadata
		query = `
			UPDATE content.content_metadata
			SET 
				tags = $2,
				file_size = $3,
				file_name = $4,
				mime_type = $5,
				checksum = $6,
				checksum_algorithm = $7,
				metadata = $8,
				updated_at = $9
			WHERE content_id = $1
			RETURNING updated_at
		`
	} else {
		// Insert new metadata
		query = `
			INSERT INTO content.content_metadata (
				content_id, tags, file_size, file_name, mime_type, checksum, checksum_algorithm, 
				metadata, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $9
			) RETURNING updated_at
		`
	}

	now := time.Now().UTC()
	var updatedAt time.Time
	err = r.db.QueryRow(
		ctx,
		query,
		metadata.ContentID,
		metadata.Tags,
		metadata.FileSize,
		metadata.FileName,
		metadata.MimeType,
		metadata.Checksum,
		metadata.ChecksumAlgorithm,
		metadataJSON,
		now,
	).Scan(&updatedAt)

	metadata.UpdatedAt = updatedAt
	if !exists {
		metadata.CreatedAt = updatedAt
	}

	return err
}

// Get implements ContentMetadataRepository.Get
func (r *PSQLContentMetadataRepository) Get(ctx context.Context, contentID uuid.UUID) (*domain.ContentMetadata, error) {
	query := `
		SELECT 
			content_id, tags, file_size, file_name, mime_type, checksum, checksum_algorithm,
			metadata, created_at, updated_at
		FROM content.content_metadata
		WHERE content_id = $1 AND deleted_at IS NULL
	`

	metadata := &domain.ContentMetadata{
		ContentID: contentID,
		Metadata:  make(map[string]interface{}),
	}

	var tags []string
	var metadataJSON []byte
	var nullableFileSize *int64
	var nullableFileName, nullableMimeType, nullableChecksum, nullableChecksumAlgo *string

	err := r.db.QueryRow(ctx, query, contentID).Scan(
		&metadata.ContentID,
		&tags,
		&nullableFileSize,
		&nullableFileName,
		&nullableMimeType,
		&nullableChecksum,
		&nullableChecksumAlgo,
		&metadataJSON,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("content metadata not found: %w", err)
		}
		return nil, err
	}

	metadata.Tags = tags
	if nullableFileSize != nil {
		metadata.FileSize = *nullableFileSize
	}
	if nullableFileName != nil {
		metadata.FileName = *nullableFileName
	}
	if nullableMimeType != nil {
		metadata.MimeType = *nullableMimeType
	}
	if nullableChecksum != nil {
		metadata.Checksum = *nullableChecksum
	}
	if nullableChecksumAlgo != nil {
		metadata.ChecksumAlgorithm = *nullableChecksumAlgo
	}

	// Parse the JSON metadata
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &metadata.Metadata); err != nil {
			return nil, err
		}
	}

	return metadata, nil
}
