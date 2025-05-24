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
	content.UpdatedAt = time.Now()

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

	result, err := r.db.Exec(ctx, query, id, time.Now())
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
