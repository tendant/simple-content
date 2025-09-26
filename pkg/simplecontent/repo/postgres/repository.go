package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// DBTX is an interface that allows us to use either a database connection or a transaction
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

// Repository implements simplecontent.Repository using PostgreSQL
type Repository struct {
	db DBTX
}

// New creates a new PostgreSQL repository
func New(db DBTX) simplecontent.Repository {
	return &Repository{db: db}
}

// NewWithPool creates a new PostgreSQL repository with connection pool
func NewWithPool(pool *pgxpool.Pool) simplecontent.Repository {
	return &Repository{db: pool}
}

// Error handling helper
func (r *Repository) handlePostgresError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			if strings.Contains(pgErr.ConstraintName, "content") {
				return fmt.Errorf("content already exists")
			}
			if strings.Contains(pgErr.ConstraintName, "object") {
				return fmt.Errorf("object already exists")
			}
			return fmt.Errorf("duplicate entry")
		case "23503": // foreign_key_violation
			return fmt.Errorf("referenced record not found")
		case "23502": // not_null_violation
			return fmt.Errorf("required field %s is missing", pgErr.ColumnName)
		case "42P01": // undefined_table
			return fmt.Errorf("table does not exist - database migration required")
		default:
			return fmt.Errorf("database error in %s: %s (code: %s)", operation, pgErr.Message, pgErr.Code)
		}
	}

	// Handle other common errors
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("record not found")
	}

	return fmt.Errorf("database error in %s: %w", operation, err)
}

// Content operations

func (r *Repository) CreateContent(ctx context.Context, content *simplecontent.Content) error {
	query := `
		INSERT INTO content (
			id, tenant_id, owner_id, owner_type, name, description, 
			document_type, status, derivation_type, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Exec(ctx, query,
		content.ID, content.TenantID, content.OwnerID, content.OwnerType,
		content.Name, content.Description, content.DocumentType,
		content.Status, content.DerivationType, content.CreatedAt, content.UpdatedAt)

	if err != nil {
		return r.handlePostgresError("create content", err)
	}

	return nil
}

func (r *Repository) GetContent(ctx context.Context, id uuid.UUID) (*simplecontent.Content, error) {
	query := `
        SELECT id, tenant_id, owner_id, owner_type, name, description,
               document_type, status, derivation_type, created_at, updated_at
        FROM content WHERE id = $1 AND deleted_at IS NULL`

	var content simplecontent.Content
	err := r.db.QueryRow(ctx, query, id).Scan(
		&content.ID, &content.TenantID, &content.OwnerID, &content.OwnerType,
		&content.Name, &content.Description, &content.DocumentType,
		&content.Status, &content.DerivationType, &content.CreatedAt, &content.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, simplecontent.ErrContentNotFound
		}
		return nil, err
	}

	return &content, nil
}

func (r *Repository) UpdateContent(ctx context.Context, content *simplecontent.Content) error {
	query := `
		UPDATE content SET
			tenant_id = $2, owner_id = $3, owner_type = $4, name = $5,
			description = $6, document_type = $7, status = $8,
			derivation_type = $9, updated_at = $10
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		content.ID, content.TenantID, content.OwnerID, content.OwnerType,
		content.Name, content.Description, content.DocumentType,
		content.Status, content.DerivationType, content.UpdatedAt)

	return err
}

func (r *Repository) DeleteContent(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE content SET status = 'deleted', deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *Repository) ListContent(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*simplecontent.Content, error) {
	query := `
        SELECT id, tenant_id, owner_id, owner_type, name, description,
               document_type, status, derivation_type, created_at, updated_at
        FROM content WHERE owner_id = $1 AND tenant_id = $2 AND deleted_at IS NULL
        ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, ownerID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []*simplecontent.Content
	for rows.Next() {
		var content simplecontent.Content
		if err := rows.Scan(
			&content.ID, &content.TenantID, &content.OwnerID, &content.OwnerType,
			&content.Name, &content.Description, &content.DocumentType,
			&content.Status, &content.DerivationType, &content.CreatedAt, &content.UpdatedAt); err != nil {
			return nil, err
		}
		contents = append(contents, &content)
	}

	return contents, nil
}

// Content metadata operations

func (r *Repository) SetContentMetadata(ctx context.Context, metadata *simplecontent.ContentMetadata) error {
	query := `
		INSERT INTO content_metadata (
			content_id, tags, file_size, file_name, mime_type,
			checksum, checksum_algorithm, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (content_id) DO UPDATE SET
			tags = EXCLUDED.tags,
			file_size = EXCLUDED.file_size,
			file_name = EXCLUDED.file_name,
			mime_type = EXCLUDED.mime_type,
			checksum = EXCLUDED.checksum,
			checksum_algorithm = EXCLUDED.checksum_algorithm,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		metadata.ContentID, metadata.Tags, metadata.FileSize, metadata.FileName,
		metadata.MimeType, metadata.Checksum, metadata.ChecksumAlgorithm,
		metadata.Metadata, metadata.CreatedAt, metadata.UpdatedAt)

	return err
}

func (r *Repository) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*simplecontent.ContentMetadata, error) {
	query := `
		SELECT content_id, tags, file_size, file_name, mime_type,
			   checksum, checksum_algorithm, metadata, created_at, updated_at
		FROM content_metadata WHERE content_id = $1`

	var metadata simplecontent.ContentMetadata
	err := r.db.QueryRow(ctx, query, contentID).Scan(
		&metadata.ContentID, &metadata.Tags, &metadata.FileSize, &metadata.FileName,
		&metadata.MimeType, &metadata.Checksum, &metadata.ChecksumAlgorithm,
		&metadata.Metadata, &metadata.CreatedAt, &metadata.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("content metadata not found for content %s", contentID)
		}
		return nil, err
	}

	return &metadata, nil
}

// Object operations

func (r *Repository) CreateObject(ctx context.Context, object *simplecontent.Object) error {
	query := `
		INSERT INTO object (
			id, content_id, storage_backend_name, storage_class, object_key,
			file_name, version, object_type, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Exec(ctx, query,
		object.ID, object.ContentID, object.StorageBackendName, object.StorageClass,
		object.ObjectKey, object.FileName, object.Version, object.ObjectType,
		object.Status, object.CreatedAt, object.UpdatedAt)

	return err
}

func (r *Repository) GetObject(ctx context.Context, id uuid.UUID) (*simplecontent.Object, error) {
	query := `
        SELECT id, content_id, storage_backend_name, storage_class, object_key,
               file_name, version, object_type, status, created_at, updated_at
        FROM object WHERE id = $1 AND deleted_at IS NULL`

	var object simplecontent.Object
	err := r.db.QueryRow(ctx, query, id).Scan(
		&object.ID, &object.ContentID, &object.StorageBackendName, &object.StorageClass,
		&object.ObjectKey, &object.FileName, &object.Version, &object.ObjectType,
		&object.Status, &object.CreatedAt, &object.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, simplecontent.ErrObjectNotFound
		}
		return nil, err
	}

	return &object, nil
}

func (r *Repository) GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*simplecontent.Object, error) {
	query := `
        SELECT id, content_id, storage_backend_name, storage_class, object_key,
               file_name, version, object_type, status, created_at, updated_at
        FROM object WHERE content_id = $1 AND deleted_at IS NULL ORDER BY version DESC`

	rows, err := r.db.Query(ctx, query, contentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*simplecontent.Object
	for rows.Next() {
		var object simplecontent.Object
		if err := rows.Scan(
			&object.ID, &object.ContentID, &object.StorageBackendName, &object.StorageClass,
			&object.ObjectKey, &object.FileName, &object.Version, &object.ObjectType,
			&object.Status, &object.CreatedAt, &object.UpdatedAt); err != nil {
			return nil, err
		}
		objects = append(objects, &object)
	}

	return objects, nil
}

func (r *Repository) GetObjectByObjectKeyAndStorageBackendName(ctx context.Context, objectKey, storageBackendName string) (*simplecontent.Object, error) {
	query := `
		SELECT id, content_id, storage_backend_name, storage_class, object_key,
			   file_name, version, object_type, status, created_at, updated_at
		FROM object WHERE object_key = $1 AND storage_backend_name = $2 AND deleted_at IS NULL`

	var object simplecontent.Object
	err := r.db.QueryRow(ctx, query, objectKey, storageBackendName).Scan(
		&object.ID, &object.ContentID, &object.StorageBackendName, &object.StorageClass,
		&object.ObjectKey, &object.FileName, &object.Version, &object.ObjectType,
		&object.Status, &object.CreatedAt, &object.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, simplecontent.ErrObjectNotFound
		}
		return nil, err
	}

	return &object, nil
}

func (r *Repository) UpdateObject(ctx context.Context, object *simplecontent.Object) error {
	query := `
		UPDATE object SET
			content_id = $2, storage_backend_name = $3, storage_class = $4,
			object_key = $5, file_name = $6, version = $7, object_type = $8,
			status = $9, updated_at = $10
		WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		object.ID, object.ContentID, object.StorageBackendName, object.StorageClass,
		object.ObjectKey, object.FileName, object.Version, object.ObjectType,
		object.Status, object.UpdatedAt)

	return err
}

func (r *Repository) DeleteObject(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE object SET status = 'deleted', deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Object metadata operations

func (r *Repository) SetObjectMetadata(ctx context.Context, metadata *simplecontent.ObjectMetadata) error {
	query := `
		INSERT INTO object_metadata (
			object_id, size_bytes, mime_type, etag, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (object_id) DO UPDATE SET
			size_bytes = EXCLUDED.size_bytes,
			mime_type = EXCLUDED.mime_type,
			etag = EXCLUDED.etag,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		metadata.ObjectID, metadata.SizeBytes, metadata.MimeType,
		metadata.ETag, metadata.Metadata, metadata.CreatedAt, metadata.UpdatedAt)

	return err
}

func (r *Repository) GetObjectMetadata(ctx context.Context, objectID uuid.UUID) (*simplecontent.ObjectMetadata, error) {
	query := `
		SELECT object_id, size_bytes, mime_type, etag, metadata, created_at, updated_at
		FROM object_metadata WHERE object_id = $1`

	var metadata simplecontent.ObjectMetadata
	err := r.db.QueryRow(ctx, query, objectID).Scan(
		&metadata.ObjectID, &metadata.SizeBytes, &metadata.MimeType,
		&metadata.ETag, &metadata.Metadata, &metadata.CreatedAt, &metadata.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("object metadata not found for object %s", objectID)
		}
		return nil, err
	}

	return &metadata, nil
}

// Derived content operations (simplified implementations)

func (r *Repository) CreateDerivedContentRelationship(ctx context.Context, params simplecontent.CreateDerivedContentParams) (*simplecontent.DerivedContent, error) {
	query := `
	        INSERT INTO content_derived (
	            parent_id, content_id, derivation_type, variant, derivation_params,
	            processing_metadata, created_at, updated_at, status
	        ) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), 'created')
	        RETURNING parent_id, content_id, derivation_type, variant, derivation_params,
	                  processing_metadata, created_at, updated_at, status`

	var derived simplecontent.DerivedContent

	err := r.db.QueryRow(ctx, query,
		params.ParentID, params.DerivedContentID, params.DerivationType, params.Variant,
		params.DerivationParams, params.ProcessingMetadata).Scan(
		&derived.ParentID, &derived.ContentID, &derived.DerivationType, &derived.Variant,
		&derived.DerivationParams, &derived.ProcessingMetadata,
		&derived.CreatedAt, &derived.UpdatedAt, &derived.Status)

	if err != nil {
		return nil, err
	}

	// Populate document type from the content row so callers have immediate context.
	if docErr := r.db.QueryRow(ctx,
		`SELECT document_type FROM content WHERE id = $1`,
		params.DerivedContentID,
	).Scan(&derived.DocumentType); docErr != nil && docErr != sql.ErrNoRows {
		return nil, docErr
	}

	return &derived, nil
}

func (r *Repository) ListDerivedContent(ctx context.Context, params simplecontent.ListDerivedContentParams) ([]*simplecontent.DerivedContent, error) {
	query, args := r.buildEnhancedQuery(params)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query derived content: %w", err)
	}
	defer rows.Close()

	var result []*simplecontent.DerivedContent
	for rows.Next() {
		derived := &simplecontent.DerivedContent{}
		err := rows.Scan(
			&derived.ParentID, &derived.ContentID,
			&derived.DerivationType, &derived.Variant,
			&derived.DerivationParams, &derived.ProcessingMetadata,
			&derived.CreatedAt, &derived.UpdatedAt,
			&derived.DocumentType, &derived.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan derived content: %w", err)
		}
		result = append(result, derived)
	}

	return result, nil
}

func (r *Repository) GetDerivedRelationshipByContentID(ctx context.Context, contentID uuid.UUID) (*simplecontent.DerivedContent, error) {
	query := `
        SELECT parent_id, content_id, variant as derivation_type, derivation_params,
               processing_metadata, created_at, updated_at, status
        FROM content_derived WHERE content_id = $1`

	var derived simplecontent.DerivedContent
	err := r.db.QueryRow(ctx, query, contentID).Scan(
		&derived.ParentID, &derived.ContentID, &derived.DerivationType,
		&derived.DerivationParams, &derived.ProcessingMetadata,
		&derived.CreatedAt, &derived.UpdatedAt, &derived.Status,
	)
	if err != nil {
		return nil, r.handlePostgresError("get derived relationship by content id", err)
	}
	return &derived, nil
}

// buildEnhancedQuery builds a PostgreSQL query with enhanced filtering capabilities
func (r *Repository) buildEnhancedQuery(params simplecontent.ListDerivedContentParams) (string, []interface{}) {
	query := `
		SELECT cd.parent_id, cd.content_id, cd.derivation_type, cd.variant, cd.derivation_params,
			   cd.processing_metadata, cd.created_at, cd.updated_at,
			   COALESCE(c.document_type, '') as document_type, cd.status
		FROM content_derived cd
		LEFT JOIN content c ON cd.content_id = c.id
		WHERE cd.deleted_at IS NULL
	`

	var args []interface{}
	argIndex := 1

	// Backward compatible filtering
	if params.ParentID != nil {
		query += fmt.Sprintf(" AND cd.parent_id = $%d", argIndex)
		args = append(args, *params.ParentID)
		argIndex++
	}
	if params.DerivationType != nil {
		query += fmt.Sprintf(" AND cd.derivation_type = $%d", argIndex)
		args = append(args, *params.DerivationType)
		argIndex++
	}

	// NEW: Enhanced filtering
	if len(params.ParentIDs) > 0 {
		placeholders := make([]string, len(params.ParentIDs))
		for i, parentID := range params.ParentIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, parentID)
			argIndex++
		}
		query += fmt.Sprintf(" AND cd.parent_id IN (%s)", strings.Join(placeholders, ","))
	}

	if len(params.DerivationTypes) > 0 {
		placeholders := make([]string, len(params.DerivationTypes))
		for i, dtype := range params.DerivationTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, dtype)
			argIndex++
		}
		query += fmt.Sprintf(" AND cd.derivation_type IN (%s)", strings.Join(placeholders, ","))
	}

	if params.Variant != nil {
		query += fmt.Sprintf(" AND cd.variant = $%d", argIndex)
		args = append(args, *params.Variant)
		argIndex++
	}

	if len(params.Variants) > 0 {
		placeholders := make([]string, len(params.Variants))
		for i, variant := range params.Variants {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, variant)
			argIndex++
		}
		query += fmt.Sprintf(" AND cd.variant IN (%s)", strings.Join(placeholders, ","))
	}

	// TypeVariantPair filtering
	if len(params.TypeVariantPairs) > 0 {
		conditions := make([]string, len(params.TypeVariantPairs))
		for i, pair := range params.TypeVariantPairs {
			conditions[i] = fmt.Sprintf("(cd.derivation_type = $%d AND cd.variant = $%d)", argIndex, argIndex+1)
			args = append(args, pair.DerivationType, pair.Variant)
			argIndex += 2
		}
		query += fmt.Sprintf(" AND (%s)", strings.Join(conditions, " OR "))
	}

	// Content status filtering
	if params.ContentStatus != nil {
		query += fmt.Sprintf(" AND cd.status = $%d", argIndex)
		args = append(args, *params.ContentStatus)
		argIndex++
	}

	// Temporal filtering
	if params.CreatedAfter != nil {
		query += fmt.Sprintf(" AND cd.created_at > $%d", argIndex)
		args = append(args, *params.CreatedAfter)
		argIndex++
	}

	if params.CreatedBefore != nil {
		query += fmt.Sprintf(" AND cd.created_at < $%d", argIndex)
		args = append(args, *params.CreatedBefore)
		argIndex++
	}

	// Sorting
	switch {
	case params.SortBy == nil || *params.SortBy == "" || *params.SortBy == "created_at_desc":
		query += " ORDER BY cd.created_at DESC"
	case *params.SortBy == "created_at_asc":
		query += " ORDER BY cd.created_at ASC"
	case *params.SortBy == "type_variant":
		query += " ORDER BY cd.derivation_type, cd.variant"
	default:
		query += " ORDER BY cd.created_at DESC"
	}

	// Pagination
	if params.Limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, *params.Limit)
		argIndex++
	}
	if params.Offset != nil {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, *params.Offset)
		argIndex++
	}

	return query, args
}
