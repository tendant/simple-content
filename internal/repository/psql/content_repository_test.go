package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/domain"
	"golang.org/x/exp/slog"
)

func TestPSQLContentRepository_Create(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create a new content
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Description:    "Test Description",
			DocumentType:   "document",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}

		// Create the content
		err := repo.Create(ctx, content)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, content.ID)
		assert.False(t, content.CreatedAt.IsZero())
		assert.False(t, content.UpdatedAt.IsZero())
	})
}

func TestPSQLContentRepository_Get(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create a new content
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Description:    "Test Description",
			DocumentType:   "document",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}

		// Create the content
		err := repo.Create(ctx, content)
		require.NoError(t, err)

		// Get the content
		retrieved, err := repo.Get(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, content.ID, retrieved.ID)
		assert.Equal(t, content.TenantID, retrieved.TenantID)
		assert.Equal(t, content.OwnerID, retrieved.OwnerID)
		assert.Equal(t, content.OwnerType, retrieved.OwnerType)
		assert.Equal(t, content.Name, retrieved.Name)
		assert.Equal(t, content.Description, retrieved.Description)
		assert.Equal(t, content.DocumentType, retrieved.DocumentType)
		assert.Equal(t, content.Status, retrieved.Status)
		assert.Equal(t, content.DerivationType, retrieved.DerivationType)

		// Try to get a non-existent content
		_, err = repo.Get(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestPSQLContentRepository_Update(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create a new content
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Description:    "Test Description",
			DocumentType:   "document",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}

		// Create the content
		err := repo.Create(ctx, content)
		require.NoError(t, err)

		// Update the content
		originalUpdatedAt := content.UpdatedAt
		time.Sleep(1 * time.Millisecond) // Ensure timestamp changes
		content.Name = "Updated Content"
		content.Description = "Updated Description"
		content.Status = domain.ContentStatusUploaded
		content.UpdatedAt = time.Now().UTC()

		err = repo.Update(ctx, content)
		require.NoError(t, err)

		// Get the updated content
		updated, err := repo.Get(ctx, content.ID)
		slog.Info("Updated content", "updated", updated)
		require.NoError(t, err)
		assert.Equal(t, "Updated Content", updated.Name)
		assert.Equal(t, "Updated Description", updated.Description)
		assert.Equal(t, domain.ContentStatusUploaded, updated.Status)
		// The updated timestamp should be different from the original
		assert.NotEqual(t, originalUpdatedAt, updated.UpdatedAt)
	})
}

func TestPSQLContentRepository_Delete(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create a new content
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Description:    "Test Description",
			DocumentType:   "document",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}

		// Create the content
		err := repo.Create(ctx, content)
		require.NoError(t, err)

		// Delete the content
		err = repo.Delete(ctx, content.ID)
		require.NoError(t, err)

		// Try to get the deleted content
		_, err = repo.Get(ctx, content.ID)
		assert.Error(t, err)
	})
}

func TestPSQLContentRepository_List(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create tenant and owner IDs
		tenantID1 := uuid.New()
		tenantID2 := uuid.New()
		ownerID1 := uuid.New()
		ownerID2 := uuid.New()

		// Create test contents
		contents := []*domain.Content{
			{
				TenantID:       tenantID1,
				OwnerID:        ownerID1,
				OwnerType:      "user",
				Name:           "Content 1",
				Status:         domain.ContentStatusCreated,
				DerivationType: domain.ContentDerivationTypeOriginal,
			},
			{
				TenantID:       tenantID1,
				OwnerID:        ownerID2,
				OwnerType:      "user",
				Name:           "Content 2",
				Status:         domain.ContentStatusCreated,
				DerivationType: domain.ContentDerivationTypeOriginal,
			},
			{
				TenantID:       tenantID2,
				OwnerID:        ownerID1,
				OwnerType:      "user",
				Name:           "Content 3",
				Status:         domain.ContentStatusCreated,
				DerivationType: domain.ContentDerivationTypeOriginal,
			},
		}

		// Create all contents
		for _, c := range contents {
			err := repo.Create(ctx, c)
			require.NoError(t, err)
		}

		// Test list by tenant ID
		result, err := repo.List(ctx, uuid.Nil, tenantID1)
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Test list by owner ID
		result, err = repo.List(ctx, ownerID1, uuid.Nil)
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Test list by both tenant and owner ID
		result, err = repo.List(ctx, ownerID1, tenantID1)
		require.NoError(t, err)
		assert.Len(t, result, 1)

		// Test list all
		result, err = repo.List(ctx, uuid.Nil, uuid.Nil)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})
}
