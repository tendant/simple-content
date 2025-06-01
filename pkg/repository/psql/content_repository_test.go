package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/domain"
	repolib "github.com/tendant/simple-content/pkg/repository"
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

		// Test list empty
		result, err = repo.List(ctx, uuid.Nil, uuid.Nil)
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestPSQLContentRepository_ListDerivedContent(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create a tenant
		tenantID := uuid.New()

		// Create a parent content
		parentContent := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Parent Content",
			Description:    "Parent Description",
			DocumentType:   "receipt",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := repo.Create(ctx, parentContent)
		require.NoError(t, err)

		// Create another parent content with different tenant
		otherTenantID := uuid.New()
		otherParentContent := &domain.Content{
			TenantID:       otherTenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Other Parent Content",
			Description:    "Other Parent Description",
			DocumentType:   "receipt",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err = repo.Create(ctx, otherParentContent)
		require.NoError(t, err)

		// Create derived contents
		derivedContent1 := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Derived Content 1",
			Description:    "Derived Description 1",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, derivedContent1)
		require.NoError(t, err)

		derivedContent2 := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Derived Content 2",
			Description:    "Derived Description 2",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, derivedContent2)
		require.NoError(t, err)

		derivedContent3 := &domain.Content{
			TenantID:       otherTenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Derived Content 3",
			Description:    "Derived Description 3",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, derivedContent3)
		require.NoError(t, err)

		// Create content_derived relationships
		// First derived content - translation relationship
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO content.content_derived (
				parent_content_id, derived_content_id, derivation_type
			) VALUES ($1, $2, $3)
		`, parentContent.ID, derivedContent1.ID, domain.ContentDerivedTHUMBNAIL720)
		require.NoError(t, err)

		// Second derived content - summary relationship
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO content.content_derived (
				parent_content_id, derived_content_id, derivation_type
			) VALUES ($1, $2, $3)
		`, parentContent.ID, derivedContent2.ID, domain.ContentDerivedTHUMBNAIL480)
		require.NoError(t, err)

		// Third derived content - translation relationship but different tenant
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO content.content_derived (
				parent_content_id, derived_content_id, derivation_type
			) VALUES ($1, $2, $3)
		`, otherParentContent.ID, derivedContent3.ID, domain.ContentDerivedTHUMBNAIL720)
		require.NoError(t, err)

		// Test case 1: List derived content for parent content with specific relationship type
		t.Run("Filter by parent ID and relationship type", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				ParentIDs:    []uuid.UUID{parentContent.ID},
				Relationship: []string{domain.ContentDerivedTHUMBNAIL720},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, derivedContent1.ID, result[0].ID)
		})

		// Test case 2: List all derived content for parent content (multiple relationship types)
		t.Run("Filter by parent ID with multiple relationship types", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				ParentIDs:    []uuid.UUID{parentContent.ID},
				Relationship: []string{domain.ContentDerivedTHUMBNAIL720, domain.ContentDerivedTHUMBNAIL480},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 2)

			// Create a map of IDs for easier verification
			idsMap := make(map[uuid.UUID]bool)
			for _, content := range result {
				idsMap[content.ID] = true
			}

			// Verify both derived contents are in the result
			assert.True(t, idsMap[derivedContent1.ID])
			assert.True(t, idsMap[derivedContent2.ID])
		})

		// Test case 3: Filter by tenant ID
		t.Run("Filter by tenant ID", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				TenantID:     tenantID,
				Relationship: []string{domain.ContentDerivedTHUMBNAIL720},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, derivedContent1.ID, result[0].ID)
		})

		// Test case 4: Filter by multiple parent IDs
		t.Run("Filter by multiple parent IDs", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				ParentIDs:    []uuid.UUID{parentContent.ID, otherParentContent.ID},
				Relationship: []string{domain.ContentDerivedTHUMBNAIL720, domain.ContentDerivedTHUMBNAIL480},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 3)

			// Create a map of IDs for easier verification
			idsMap := make(map[uuid.UUID]bool)
			for _, content := range result {
				idsMap[content.ID] = true
			}

			// Verify both derived contents are in the result
			assert.True(t, idsMap[derivedContent1.ID])
			assert.True(t, idsMap[derivedContent3.ID])
		})

		// Test case 5: No results when filtering by non-existent parent ID
		t.Run("No results for non-existent parent ID", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				ParentIDs:    []uuid.UUID{uuid.New()}, // Random non-existent ID
				Relationship: []string{domain.ContentDerivedTHUMBNAIL720},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 0)
		})
	})
}
