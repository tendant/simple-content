// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tendant/simple-content/internal/domain"
)

// PSQLObjectRepository implements the ObjectRepository interface
type PSQLObjectRepository struct {
	BaseRepository
}

// NewPSQLObjectRepository creates a new PostgreSQL object repository
func NewPSQLObjectRepository(db DBTX) *PSQLObjectRepository {
	return &PSQLObjectRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create implements ObjectRepository.Create
func (r *PSQLObjectRepository) Create(ctx context.Context, object *domain.Object) error {
	query := `
		INSERT INTO content.object (
			id, content_id, storage_backend_name, storage_class, object_key, 
			file_name, version, object_type, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at, updated_at
	`

	// If ID is not provided, generate one
	if object.ID == uuid.Nil {
		object.ID = uuid.New()
	}

	// Set timestamps if not provided
	now := time.Now().UTC()
	if object.CreatedAt.IsZero() {
		object.CreatedAt = now
	}
	if object.UpdatedAt.IsZero() {
		object.UpdatedAt = now
	}

	// Default status if not provided
	if object.Status == "" {
		object.Status = domain.ObjectStatusCreated
	}

	err := r.db.QueryRow(
		ctx,
		query,
		object.ID,
		object.ContentID,
		object.StorageBackendName,
		object.StorageClass,
		object.ObjectKey,
		object.FileName, // Use the provided FileName
		object.Version,
		object.ObjectType, // Default object_type
		object.Status,
		object.CreatedAt,
		object.UpdatedAt,
	).Scan(&object.ID, &object.CreatedAt, &object.UpdatedAt)

	return err
}

// Get implements ObjectRepository.Get
func (r *PSQLObjectRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Object, error) {
	query := `
		SELECT 
			id, content_id, storage_backend_name, storage_class, object_key, file_name, version, object_type,
			status, created_at, updated_at
		FROM content.object
		WHERE id = $1 AND deleted_at IS NULL
	`

	object := &domain.Object{}

	var nullableStorageClass, nullableFileName, nullableObjectType *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&object.ID,
		&object.ContentID,
		&object.StorageBackendName,
		&nullableStorageClass,
		&object.ObjectKey,
		&nullableFileName,
		&object.Version,
		&nullableObjectType,
		&object.Status,
		&object.CreatedAt,
		&object.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("object not found: %w", err)
		}
		return nil, err
	}

	if nullableStorageClass != nil {
		object.StorageClass = *nullableStorageClass
	}

	if nullableFileName != nil {
		object.FileName = *nullableFileName
	}

	if nullableObjectType != nil {
		object.ObjectType = *nullableObjectType
	}

	// Note: Version is now used instead of VersionID

	return object, nil
}

// GetByContentID implements ObjectRepository.GetByContentID
func (r *PSQLObjectRepository) GetByContentID(ctx context.Context, contentID uuid.UUID) ([]*domain.Object, error) {
	query := `
		SELECT 
			id, content_id, storage_backend_name, storage_class, object_key, file_name, version, object_type,
			status, created_at, updated_at
		FROM content.object
		WHERE content_id = $1 AND deleted_at IS NULL
		ORDER BY version, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := []*domain.Object{}

	for rows.Next() {
		object := &domain.Object{}
		var nullableStorageClass, nullableFileName, nullableObjectType *string

		err := rows.Scan(
			&object.ID,
			&object.ContentID,
			&object.StorageBackendName,
			&nullableStorageClass,
			&object.ObjectKey,
			&nullableFileName,
			&object.Version,
			&nullableObjectType,
			&object.Status,
			&object.CreatedAt,
			&object.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if nullableStorageClass != nil {
			object.StorageClass = *nullableStorageClass
		}

		if nullableFileName != nil {
			object.FileName = *nullableFileName
		}

		if nullableObjectType != nil {
			object.ObjectType = *nullableObjectType
		}

		objects = append(objects, object)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return objects, nil
}

// Update implements ObjectRepository.Update
func (r *PSQLObjectRepository) Update(ctx context.Context, object *domain.Object) error {
	query := `
		UPDATE content.object
		SET 
			storage_backend_name = $2,
			storage_class = $3,
			object_key = $4,
			file_name = $5,
			version = $6,
			object_type = $7,
			status = $8,
			updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	// Update timestamp
	object.UpdatedAt = time.Now().UTC()

	err := r.db.QueryRow(
		ctx,
		query,
		object.ID,
		object.StorageBackendName,
		object.StorageClass,
		object.ObjectKey,
		object.FileName,
		object.Version,
		object.ObjectType,
		object.Status,
		object.UpdatedAt,
	).Scan(&object.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("object not found: %w", err)
		}
		return err
	}

	return nil
}

// Delete implements ObjectRepository.Delete
func (r *PSQLObjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE content.object
		SET deleted_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, time.Now().UTC(), id)
	return err
}

// GetByObjectKeyAndStorageBackend retrieves a non-deleted object by object key and storage backend name
func (r *PSQLObjectRepository) GetByObjectKeyAndStorageBackendName(ctx context.Context, objectKey string, storageBackendName string) (*domain.Object, error) {
	query := `
		SELECT 
			id, content_id, storage_backend_name, status, object_key
		FROM content.object
		WHERE object_key = $1 AND storage_backend_name = $2 AND deleted_at IS NULL
		ORDER BY version DESC
		LIMIT 1
	`

	object := &domain.Object{}

	err := r.db.QueryRow(ctx, query, objectKey, storageBackendName).Scan(
		&object.ID,
		&object.ContentID,
		&object.StorageBackendName,
		&object.Status,
		&object.ObjectKey,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("object not found")
		}
		return nil, err
	}

	return object, nil
}
