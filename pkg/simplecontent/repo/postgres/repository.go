package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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
	// Soft delete: set deleted_at timestamp, keep status at last operational state
	query := `UPDATE content SET deleted_at = NOW() WHERE id = $1`
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

// Status query operations

func (r *Repository) GetContentByStatus(ctx context.Context, status string) ([]*simplecontent.Content, error) {
	query := `
		SELECT id, tenant_id, owner_id, owner_type, name, description, document_type,
			   status, derivation_type, created_at, updated_at, deleted_at
		FROM content
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, r.handlePostgresError("get content by status", err)
	}
	defer rows.Close()

	var results []*simplecontent.Content
	for rows.Next() {
		var content simplecontent.Content
		err := rows.Scan(
			&content.ID, &content.TenantID, &content.OwnerID, &content.OwnerType,
			&content.Name, &content.Description, &content.DocumentType,
			&content.Status, &content.DerivationType, &content.CreatedAt,
			&content.UpdatedAt, &content.DeletedAt)
		if err != nil {
			return nil, r.handlePostgresError("scan content", err)
		}
		results = append(results, &content)
	}

	if err = rows.Err(); err != nil {
		return nil, r.handlePostgresError("iterate content rows", err)
	}

	return results, nil
}

func (r *Repository) GetObjectsByStatus(ctx context.Context, status string) ([]*simplecontent.Object, error) {
	query := `
		SELECT id, content_id, storage_backend_name, storage_class, object_key,
			   file_name, version, object_type, status, created_at, updated_at, deleted_at
		FROM object
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, r.handlePostgresError("get objects by status", err)
	}
	defer rows.Close()

	var results []*simplecontent.Object
	for rows.Next() {
		var object simplecontent.Object
		err := rows.Scan(
			&object.ID, &object.ContentID, &object.StorageBackendName, &object.StorageClass,
			&object.ObjectKey, &object.FileName, &object.Version, &object.ObjectType,
			&object.Status, &object.CreatedAt, &object.UpdatedAt, &object.DeletedAt)
		if err != nil {
			return nil, r.handlePostgresError("scan object", err)
		}
		results = append(results, &object)
	}

	if err = rows.Err(); err != nil {
		return nil, r.handlePostgresError("iterate object rows", err)
	}

	return results, nil
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
	// Soft delete: set deleted_at timestamp, keep status at last operational state
	query := `UPDATE object SET deleted_at = NOW() WHERE id = $1`
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
	// Note: Status is tracked in content.status, not content_derived (avoid duplication)
	query := `
	        INSERT INTO content_derived (
	            parent_id, content_id, derivation_type, variant, derivation_params,
	            processing_metadata, created_at, updated_at
	        ) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	        RETURNING parent_id, content_id, derivation_type, variant, derivation_params,
	                  processing_metadata, created_at, updated_at`

	var derived simplecontent.DerivedContent

	err := r.db.QueryRow(ctx, query,
		params.ParentID, params.DerivedContentID, params.DerivationType, params.Variant,
		params.DerivationParams, params.ProcessingMetadata).Scan(
		&derived.ParentID, &derived.ContentID, &derived.DerivationType, &derived.Variant,
		&derived.DerivationParams, &derived.ProcessingMetadata,
		&derived.CreatedAt, &derived.UpdatedAt)

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
               processing_metadata, created_at, updated_at
        FROM content_derived WHERE content_id = $1`

	var derived simplecontent.DerivedContent
	err := r.db.QueryRow(ctx, query, contentID).Scan(
		&derived.ParentID, &derived.ContentID, &derived.DerivationType,
		&derived.DerivationParams, &derived.ProcessingMetadata,
		&derived.CreatedAt, &derived.UpdatedAt,
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
			   COALESCE(c.document_type, '') as document_type,
			   COALESCE(c.status, '') as status
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

	// Content status filtering (uses content.status, not content_derived.status)
	if params.ContentStatus != nil {
		query += fmt.Sprintf(" AND c.status = $%d", argIndex)
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

// Admin operations - for administrative tasks without owner/tenant restrictions

func (r *Repository) ListContentWithFilters(ctx context.Context, filters simplecontent.ContentListFilters) ([]*simplecontent.Content, error) {
	query := `
        SELECT id, tenant_id, owner_id, owner_type, name, description,
               document_type, status, derivation_type, created_at, updated_at
        FROM content WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Build dynamic WHERE clause
	if !filters.IncludeDeleted {
		query += " AND deleted_at IS NULL"
	}

	if filters.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *filters.TenantID)
		argIndex++
	}
	if len(filters.TenantIDs) > 0 {
		query += fmt.Sprintf(" AND tenant_id = ANY($%d)", argIndex)
		args = append(args, filters.TenantIDs)
		argIndex++
	}

	if filters.OwnerID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *filters.OwnerID)
		argIndex++
	}
	if len(filters.OwnerIDs) > 0 {
		query += fmt.Sprintf(" AND owner_id = ANY($%d)", argIndex)
		args = append(args, filters.OwnerIDs)
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filters.Status)
		argIndex++
	}
	if len(filters.Statuses) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, filters.Statuses)
		argIndex++
	}

	if filters.DerivationType != nil {
		query += fmt.Sprintf(" AND derivation_type = $%d", argIndex)
		args = append(args, *filters.DerivationType)
		argIndex++
	}
	if len(filters.DerivationTypes) > 0 {
		query += fmt.Sprintf(" AND derivation_type = ANY($%d)", argIndex)
		args = append(args, filters.DerivationTypes)
		argIndex++
	}

	if filters.DocumentType != nil {
		query += fmt.Sprintf(" AND document_type = $%d", argIndex)
		args = append(args, *filters.DocumentType)
		argIndex++
	}
	if len(filters.DocumentTypes) > 0 {
		query += fmt.Sprintf(" AND document_type = ANY($%d)", argIndex)
		args = append(args, filters.DocumentTypes)
		argIndex++
	}

	if filters.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filters.CreatedAfter)
		argIndex++
	}
	if filters.CreatedBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filters.CreatedBefore)
		argIndex++
	}

	if filters.UpdatedAfter != nil {
		query += fmt.Sprintf(" AND updated_at >= $%d", argIndex)
		args = append(args, *filters.UpdatedAfter)
		argIndex++
	}
	if filters.UpdatedBefore != nil {
		query += fmt.Sprintf(" AND updated_at <= $%d", argIndex)
		args = append(args, *filters.UpdatedBefore)
		argIndex++
	}

	// Sorting
	sortBy := "created_at"
	sortOrder := "DESC"
	if filters.SortBy != nil {
		switch *filters.SortBy {
		case "created_at", "updated_at", "name", "status":
			sortBy = *filters.SortBy
		}
	}
	if filters.SortOrder != nil {
		if strings.ToUpper(*filters.SortOrder) == "ASC" {
			sortOrder = "ASC"
		}
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Pagination
	if filters.Limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, *filters.Limit)
		argIndex++
	}
	if filters.Offset != nil {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, *filters.Offset)
		argIndex++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, r.handlePostgresError("list content with filters", err)
	}
	defer rows.Close()

	var contents []*simplecontent.Content
	for rows.Next() {
		var content simplecontent.Content
		if err := rows.Scan(
			&content.ID, &content.TenantID, &content.OwnerID, &content.OwnerType,
			&content.Name, &content.Description, &content.DocumentType,
			&content.Status, &content.DerivationType, &content.CreatedAt, &content.UpdatedAt); err != nil {
			return nil, r.handlePostgresError("scan content", err)
		}
		contents = append(contents, &content)
	}

	if err := rows.Err(); err != nil {
		return nil, r.handlePostgresError("iterate content rows", err)
	}

	return contents, nil
}

func (r *Repository) CountContentWithFilters(ctx context.Context, filters simplecontent.ContentCountFilters) (int64, error) {
	query := "SELECT COUNT(*) FROM content WHERE 1=1"

	args := []interface{}{}
	argIndex := 1

	// Build dynamic WHERE clause
	if !filters.IncludeDeleted {
		query += " AND deleted_at IS NULL"
	}

	if filters.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *filters.TenantID)
		argIndex++
	}
	if len(filters.TenantIDs) > 0 {
		query += fmt.Sprintf(" AND tenant_id = ANY($%d)", argIndex)
		args = append(args, filters.TenantIDs)
		argIndex++
	}

	if filters.OwnerID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *filters.OwnerID)
		argIndex++
	}
	if len(filters.OwnerIDs) > 0 {
		query += fmt.Sprintf(" AND owner_id = ANY($%d)", argIndex)
		args = append(args, filters.OwnerIDs)
		argIndex++
	}

	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filters.Status)
		argIndex++
	}
	if len(filters.Statuses) > 0 {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, filters.Statuses)
		argIndex++
	}

	if filters.DerivationType != nil {
		query += fmt.Sprintf(" AND derivation_type = $%d", argIndex)
		args = append(args, *filters.DerivationType)
		argIndex++
	}
	if len(filters.DerivationTypes) > 0 {
		query += fmt.Sprintf(" AND derivation_type = ANY($%d)", argIndex)
		args = append(args, filters.DerivationTypes)
		argIndex++
	}

	if filters.DocumentType != nil {
		query += fmt.Sprintf(" AND document_type = $%d", argIndex)
		args = append(args, *filters.DocumentType)
		argIndex++
	}
	if len(filters.DocumentTypes) > 0 {
		query += fmt.Sprintf(" AND document_type = ANY($%d)", argIndex)
		args = append(args, filters.DocumentTypes)
		argIndex++
	}

	if filters.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filters.CreatedAfter)
		argIndex++
	}
	if filters.CreatedBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filters.CreatedBefore)
		argIndex++
	}

	if filters.UpdatedAfter != nil {
		query += fmt.Sprintf(" AND updated_at >= $%d", argIndex)
		args = append(args, *filters.UpdatedAfter)
		argIndex++
	}
	if filters.UpdatedBefore != nil {
		query += fmt.Sprintf(" AND updated_at <= $%d", argIndex)
		args = append(args, *filters.UpdatedBefore)
		argIndex++
	}

	var count int64
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, r.handlePostgresError("count content with filters", err)
	}

	return count, nil
}

func (r *Repository) GetContentStatistics(ctx context.Context, filters simplecontent.ContentCountFilters, options simplecontent.ContentStatisticsOptions) (*simplecontent.ContentStatisticsResult, error) {
	result := &simplecontent.ContentStatisticsResult{
		ByStatus:         make(map[string]int64),
		ByTenant:         make(map[string]int64),
		ByDerivationType: make(map[string]int64),
		ByDocumentType:   make(map[string]int64),
	}

	// Get total count
	totalCount, err := r.CountContentWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}
	result.TotalCount = totalCount

	// Build base WHERE clause for all statistics queries
	baseWhere, baseArgs := r.buildStatisticsWhereClause(filters)

	// Get status breakdown
	if options.IncludeStatusBreakdown {
		query := "SELECT status, COUNT(*) FROM content WHERE " + baseWhere + " GROUP BY status"
		rows, err := r.db.Query(ctx, query, baseArgs...)
		if err != nil {
			return nil, r.handlePostgresError("get status breakdown", err)
		}
		defer rows.Close()

		for rows.Next() {
			var status string
			var count int64
			if err := rows.Scan(&status, &count); err != nil {
				return nil, r.handlePostgresError("scan status breakdown", err)
			}
			result.ByStatus[status] = count
		}
	}

	// Get tenant breakdown
	if options.IncludeTenantBreakdown {
		query := "SELECT tenant_id, COUNT(*) FROM content WHERE " + baseWhere + " GROUP BY tenant_id"
		rows, err := r.db.Query(ctx, query, baseArgs...)
		if err != nil {
			return nil, r.handlePostgresError("get tenant breakdown", err)
		}
		defer rows.Close()

		for rows.Next() {
			var tenantID uuid.UUID
			var count int64
			if err := rows.Scan(&tenantID, &count); err != nil {
				return nil, r.handlePostgresError("scan tenant breakdown", err)
			}
			result.ByTenant[tenantID.String()] = count
		}
	}

	// Get derivation type breakdown
	if options.IncludeDerivationBreakdown {
		query := "SELECT COALESCE(derivation_type, ''), COUNT(*) FROM content WHERE " + baseWhere + " GROUP BY derivation_type"
		rows, err := r.db.Query(ctx, query, baseArgs...)
		if err != nil {
			return nil, r.handlePostgresError("get derivation breakdown", err)
		}
		defer rows.Close()

		for rows.Next() {
			var derivationType string
			var count int64
			if err := rows.Scan(&derivationType, &count); err != nil {
				return nil, r.handlePostgresError("scan derivation breakdown", err)
			}
			if derivationType == "" {
				derivationType = "original"
			}
			result.ByDerivationType[derivationType] = count
		}
	}

	// Get document type breakdown
	if options.IncludeDocumentTypeBreakdown {
		query := "SELECT COALESCE(document_type, ''), COUNT(*) FROM content WHERE " + baseWhere + " GROUP BY document_type"
		rows, err := r.db.Query(ctx, query, baseArgs...)
		if err != nil {
			return nil, r.handlePostgresError("get document type breakdown", err)
		}
		defer rows.Close()

		for rows.Next() {
			var documentType string
			var count int64
			if err := rows.Scan(&documentType, &count); err != nil {
				return nil, r.handlePostgresError("scan document type breakdown", err)
			}
			if documentType == "" {
				documentType = "unknown"
			}
			result.ByDocumentType[documentType] = count
		}
	}

	// Get time range
	if options.IncludeTimeRange {
		query := "SELECT MIN(created_at), MAX(created_at) FROM content WHERE " + baseWhere
		var oldest, newest *time.Time
		err := r.db.QueryRow(ctx, query, baseArgs...).Scan(&oldest, &newest)
		if err != nil && err != sql.ErrNoRows {
			return nil, r.handlePostgresError("get time range", err)
		}
		result.OldestContent = oldest
		result.NewestContent = newest
	}

	return result, nil
}

// buildStatisticsWhereClause builds the WHERE clause for statistics queries
func (r *Repository) buildStatisticsWhereClause(filters simplecontent.ContentCountFilters) (string, []interface{}) {
	where := "1=1"
	args := []interface{}{}
	argIndex := 1

	if !filters.IncludeDeleted {
		where += " AND deleted_at IS NULL"
	}

	if filters.TenantID != nil {
		where += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *filters.TenantID)
		argIndex++
	}
	if len(filters.TenantIDs) > 0 {
		where += fmt.Sprintf(" AND tenant_id = ANY($%d)", argIndex)
		args = append(args, filters.TenantIDs)
		argIndex++
	}

	if filters.OwnerID != nil {
		where += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *filters.OwnerID)
		argIndex++
	}
	if len(filters.OwnerIDs) > 0 {
		where += fmt.Sprintf(" AND owner_id = ANY($%d)", argIndex)
		args = append(args, filters.OwnerIDs)
		argIndex++
	}

	if filters.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filters.Status)
		argIndex++
	}
	if len(filters.Statuses) > 0 {
		where += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, filters.Statuses)
		argIndex++
	}

	if filters.DerivationType != nil {
		where += fmt.Sprintf(" AND derivation_type = $%d", argIndex)
		args = append(args, *filters.DerivationType)
		argIndex++
	}
	if len(filters.DerivationTypes) > 0 {
		where += fmt.Sprintf(" AND derivation_type = ANY($%d)", argIndex)
		args = append(args, filters.DerivationTypes)
		argIndex++
	}

	if filters.DocumentType != nil {
		where += fmt.Sprintf(" AND document_type = $%d", argIndex)
		args = append(args, *filters.DocumentType)
		argIndex++
	}
	if len(filters.DocumentTypes) > 0 {
		where += fmt.Sprintf(" AND document_type = ANY($%d)", argIndex)
		args = append(args, filters.DocumentTypes)
		argIndex++
	}

	if filters.CreatedAfter != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filters.CreatedAfter)
		argIndex++
	}
	if filters.CreatedBefore != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filters.CreatedBefore)
		argIndex++
	}

	if filters.UpdatedAfter != nil {
		where += fmt.Sprintf(" AND updated_at >= $%d", argIndex)
		args = append(args, *filters.UpdatedAfter)
		argIndex++
	}
	if filters.UpdatedBefore != nil {
		where += fmt.Sprintf(" AND updated_at <= $%d", argIndex)
		args = append(args, *filters.UpdatedBefore)
		argIndex++
	}

	return where, args
}
