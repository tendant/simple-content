// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tendant/simple-content/internal/domain"
	repo "github.com/tendant/simple-content/internal/repository"
)

// PSQLContentRepository implements the ContentRepository interface
type PSQLContentRepository struct {
	BaseRepository
}

// NewPSQLContentRepository creates a new PostgreSQL content repository
func NewPSQLContentRepository(db DBTX) *PSQLContentRepository {
	return &PSQLContentRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create implements ContentRepository.Create
func (r *PSQLContentRepository) Create(ctx context.Context, content *domain.Content) error {
	query := `
		INSERT INTO content.content (
			id, tenant_id, owner_id, owner_type, name, description, document_type,
			status, derivation_type, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at, updated_at
	`

	// If ID is not provided, generate one
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}

	// Set timestamps if not provided
	now := time.Now().UTC()
	if content.CreatedAt.IsZero() {
		content.CreatedAt = now
	}
	if content.UpdatedAt.IsZero() {
		content.UpdatedAt = now
	}

	// Default status if not provided
	if content.Status == "" {
		content.Status = domain.ContentStatusCreated
	}

	err := r.db.QueryRow(
		ctx,
		query,
		content.ID,
		content.TenantID,
		content.OwnerID,
		content.OwnerType,
		content.Name,
		content.Description,
		content.DocumentType,
		content.Status,
		content.DerivationType,
		content.CreatedAt,
		content.UpdatedAt,
	).Scan(&content.ID, &content.CreatedAt, &content.UpdatedAt)

	return err
}

// Get implements ContentRepository.Get
func (r *PSQLContentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Content, error) {
	query := `
		SELECT 
			id, tenant_id, owner_id, owner_type, name, description, document_type,
			status, derivation_type, created_at, updated_at
		FROM content.content
		WHERE id = $1 AND deleted_at IS NULL
	`

	content := &domain.Content{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&content.ID,
		&content.TenantID,
		&content.OwnerID,
		&content.OwnerType,
		&content.Name,
		&content.Description,
		&content.DocumentType,
		&content.Status,
		&content.DerivationType,
		&content.CreatedAt,
		&content.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("content not found: %w", err)
		}
		return nil, err
	}

	return content, nil
}

// Update implements ContentRepository.Update
func (r *PSQLContentRepository) Update(ctx context.Context, content *domain.Content) error {
	query := `
		UPDATE content.content
		SET 
			tenant_id = $2,
			owner_id = $3,
			owner_type = $4,
			name = $5,
			description = $6,
			document_type = $7,
			status = $8,
			derivation_type = $9,
			updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	// Update timestamp
	content.UpdatedAt = time.Now().UTC()

	err := r.db.QueryRow(
		ctx,
		query,
		content.ID,
		content.TenantID,
		content.OwnerID,
		content.OwnerType,
		content.Name,
		content.Description,
		content.DocumentType,
		content.Status,
		content.DerivationType,
		content.UpdatedAt,
	).Scan(&content.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("content not found: %w", err)
		}
		return err
	}

	return nil
}

// Delete implements ContentRepository.Delete
func (r *PSQLContentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE content.content
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("content not found or already deleted")
	}

	return nil
}

// List implements ContentRepository.List
func (r *PSQLContentRepository) List(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*domain.Content, error) {
	// Build the query based on provided parameters
	baseQuery := `
		SELECT 
			id, tenant_id, owner_id, owner_type, name, description, document_type,
			status, derivation_type, created_at, updated_at
		FROM content.content
		WHERE deleted_at IS NULL
	`

	whereClause := ""
	args := []interface{}{}
	paramCount := 1

	// Validate that at least one of ownerID or tenantID is provided
	if ownerID == uuid.Nil && tenantID == uuid.Nil {
		return nil, nil
	}

	// Add owner_id filter if provided
	if ownerID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND owner_id = $%d", paramCount)
		args = append(args, ownerID)
		paramCount++
	}

	// Add tenant_id filter if provided
	if tenantID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND tenant_id = $%d", paramCount)
		args = append(args, tenantID)
		paramCount++
	}

	// Complete the query
	query := baseQuery + whereClause + "\n\t\tORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contents []*domain.Content
	for rows.Next() {
		content := &domain.Content{}
		err := rows.Scan(
			&content.ID,
			&content.TenantID,
			&content.OwnerID,
			&content.OwnerType,
			&content.Name,
			&content.Description,
			&content.DocumentType,
			&content.Status,
			&content.DerivationType,
			&content.CreatedAt,
			&content.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return contents, nil
}

// ListDerivedContent implements ContentRepository.ListDerivedContent
func (r *PSQLContentRepository) ListDerivedContent(ctx context.Context, params repo.ListDerivedContentParams) ([]*domain.DerivedContent, error) {
	// Build the base query to join content_derived with content tables
	baseQuery := `
		SELECT 
			cd.parent_content_id, cd.derived_content_id, cd.derivation_type, cd.derivation_params, cd.processing_metadata, c.created_at, c.updated_at, c.document_type, c.status
		FROM content.content_derived cd
		JOIN content.content c ON cd.derived_content_id = c.id
		WHERE c.deleted_at IS NULL AND cd.deleted_at IS NULL
		AND cd.derivation_type = ANY($1)
	`

	// Initialize parameters for the query
	args := []interface{}{params.DerivationType}
	paramCount := 2

	// Initialize where clause
	whereClause := ""

	// Filter by parent content IDs if provided
	if len(params.ParentIDs) > 0 {
		// If there's only one parent ID, use a simple equality check
		if len(params.ParentIDs) == 1 {
			whereClause += fmt.Sprintf(" AND cd.parent_content_id = $%d", paramCount)
			args = append(args, params.ParentIDs[0])
			paramCount++
		} else {
			// For multiple parent IDs, use the ANY operator
			whereClause += fmt.Sprintf(" AND cd.parent_content_id = ANY($%d)", paramCount)
			args = append(args, params.ParentIDs)
			paramCount++
		}
	}
	// Filter by tenant ID if provided
	if params.TenantID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND c.tenant_id = $%d", paramCount)
		args = append(args, params.TenantID)
		paramCount++
	}

	// Combine the base query with the where clause
	query := baseQuery + whereClause + " ORDER BY c.created_at DESC"

	// Execute the query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process the results
	result := []*domain.DerivedContent{}
	for rows.Next() {
		content := &domain.DerivedContent{}
		err := rows.Scan(
			&content.ParentID,
			&content.ContentID,
			&content.DerivationType,
			&content.DerivationParams,
			&content.ProcessingMetadata,
			&content.CreatedAt,
			&content.UpdatedAt,
			&content.DocumentType,
			&content.Status,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, content)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Create implements ContentRepository.CreateDerivedContentRelationship
func (r *PSQLContentRepository) CreateDerivedContentRelationship(ctx context.Context, params repo.CreateDerivedContentParams) (domain.DerivedContent, error) {

	// Check if the parent and derived content IDs are the same
	if params.ParentID == params.DerivedContentID {
		return domain.DerivedContent{}, errors.New("invalid content ID")
	}

	query := `
		INSERT INTO content.content_derived (
			parent_content_id, derived_content_id, derivation_type, derivation_params, processing_metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id, parent_content_id, derived_content_id, derivation_type, derivation_params, processing_metadata
	`

	// Set timestamps if not provided
	now := time.Now().UTC()
	row := r.db.QueryRow(
		ctx,
		query,
		params.ParentID,
		params.DerivedContentID,
		params.DerivationType,
		params.DerivationParams,
		params.ProcessingMetadata,
		now,
		now,
	)
	// Create a variable to hold the ID since we don't need to store it in params
	var id uuid.UUID
	var derivationParams map[string]interface{}
	var processingMetadata map[string]interface{}
	err := row.Scan(&id, &params.ParentID, &params.DerivedContentID, &params.DerivationType, &derivationParams, &processingMetadata)

	return domain.DerivedContent{
		ParentID:           params.ParentID,
		ContentID:          params.DerivedContentID,
		DerivationType:     params.DerivationType,
		DerivationParams:   derivationParams,
		ProcessingMetadata: processingMetadata,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, err
}

// Delete implements ContentRepository.DeleteDerivedContentRelationship
func (r *PSQLContentRepository) DeleteDerivedContentRelationship(ctx context.Context, params repo.DeleteDerivedContentParams) error {
	query := `
		UPDATE content.content_derived
		SET deleted_at = $3
		WHERE parent_content_id = $1 AND derived_content_id = $2
		AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, query, params.ParentID, params.DerivedContentID, now)
	return err
}

// GetDerivedContentByLevel implements ContentRepository.GetDerivedContentByLevel
// Returns all contents up to and including the specified level in the derivation hierarchy
// along with their parent information
func (r *PSQLContentRepository) GetDerivedContentByLevel(ctx context.Context, params repo.GetDerivedContentByLevelParams) ([]repo.ContentWithParent, error) {
	// Set default max depth if not provided
	maxDepth := params.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 10 // Default max depth
	}

	// Build recursive CTE query to traverse the derivation hierarchy
	query := `
		WITH RECURSIVE derivation_tree AS (
			-- Base case: the root content
			SELECT 
				c.id, c.tenant_id, c.owner_id, c.owner_type, c.name, c.description, 
				c.document_type, c.status, c.derivation_type, c.created_at, c.updated_at,
				0 AS level, NULL::uuid AS parent_id
			FROM content.content c
			WHERE c.id = $1 AND c.deleted_at IS NULL
			
			UNION ALL
			
			-- Recursive case: derived content
			SELECT 
				c.id, c.tenant_id, c.owner_id, c.owner_type, c.name, c.description, 
				c.document_type, c.status, c.derivation_type, c.created_at, c.updated_at,
				dt.level + 1, cd.parent_content_id
			FROM content.content c
			JOIN content.content_derived cd ON c.id = cd.derived_content_id
			JOIN derivation_tree dt ON cd.parent_content_id = dt.id
			WHERE c.deleted_at IS NULL AND cd.deleted_at IS NULL
			AND dt.level < $2
		)
		SELECT * FROM derivation_tree WHERE level <= $3
	`

	// Initialize parameters for the query
	args := []interface{}{params.RootID, maxDepth, params.Level}

	// Add tenant filter if provided
	paramIndex := 4
	if params.TenantID != uuid.Nil {
		query += " AND tenant_id = $" + strconv.Itoa(paramIndex)
		args = append(args, params.TenantID)
		paramIndex++
	}

	// Execute the query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process the results
	contentsWithParent := []repo.ContentWithParent{}
	for rows.Next() {
		content := &domain.Content{}
		var level int
		var parentID *uuid.UUID
		err := rows.Scan(
			&content.ID,
			&content.TenantID,
			&content.OwnerID,
			&content.OwnerType,
			&content.Name,
			&content.Description,
			&content.DocumentType,
			&content.Status,
			&content.DerivationType,
			&content.CreatedAt,
			&content.UpdatedAt,
			&level,
			&parentID,
		)
		if err != nil {
			return nil, err
		}

		// Create ContentWithParent struct
		contentWithParent := repo.ContentWithParent{
			Content: content,
			Level:   level,
		}

		// Set parent ID if not null
		if parentID != nil {
			contentWithParent.ParentID = *parentID
		}

		contentsWithParent = append(contentsWithParent, contentWithParent)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return contentsWithParent, nil
}
