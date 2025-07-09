package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/domain"
	repolib "github.com/tendant/simple-content/internal/repository"
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
				ParentIDs:      []uuid.UUID{parentContent.ID},
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL720},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, derivedContent1.ID, result[0].ID)
		})

		// Test case 2: List all derived content for parent content (multiple relationship types)
		t.Run("Filter by parent ID with multiple relationship types", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				ParentIDs:      []uuid.UUID{parentContent.ID},
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL720, domain.ContentDerivedTHUMBNAIL480},
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
				TenantID:       tenantID,
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL720},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, derivedContent1.ID, result[0].ID)
		})

		// Test case 4: Filter by multiple parent IDs
		t.Run("Filter by multiple parent IDs", func(t *testing.T) {
			params := repolib.ListDerivedContentParams{
				ParentIDs:      []uuid.UUID{parentContent.ID, otherParentContent.ID},
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL720, domain.ContentDerivedTHUMBNAIL480},
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
				ParentIDs:      []uuid.UUID{uuid.New()}, // Random non-existent ID
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL720},
			}

			result, err := repo.ListDerivedContent(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 0)
		})
	})
}

func TestPSQLContentRepository_CreateDerivedContentRelationship(t *testing.T) {
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

		// Create a derived content
		derivedContent := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Derived Content",
			Description:    "Derived Description",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, derivedContent)
		require.NoError(t, err)

		// Create relationship with derivation params and processing metadata
		t.Run("Create relationship with params and metadata", func(t *testing.T) {
			derivationParams := map[string]interface{}{
				"width":  720,
				"height": 480,
				"format": "jpg",
			}
			processingMetadata := map[string]interface{}{
				"processor":      "thumbnail-service",
				"processingTime": 1.5,
				"status":         "completed",
			}

			params := repolib.CreateDerivedContentParams{
				ParentID:           parentContent.ID,
				DerivedContentID:   derivedContent.ID,
				DerivationType:     domain.ContentDerivedTHUMBNAIL480,
				DerivationParams:   derivationParams,
				ProcessingMetadata: processingMetadata,
			}

			result, err := repo.CreateDerivedContentRelationship(ctx, params)
			require.NoError(t, err)
			assert.Equal(t, parentContent.ID, result.ParentID)
			assert.Equal(t, derivedContent.ID, result.ID)
			assert.Equal(t, domain.ContentDerivedTHUMBNAIL480, result.DerivationType)

			// Verify derivation params
			assert.NotNil(t, result.DerivationParams)
			assert.Equal(t, float64(720), result.DerivationParams["width"])
			assert.Equal(t, float64(480), result.DerivationParams["height"])
			assert.Equal(t, "jpg", result.DerivationParams["format"])

			// Verify processing metadata
			assert.NotNil(t, result.ProcessingMetadata)
			assert.Equal(t, "thumbnail-service", result.ProcessingMetadata["processor"])
			assert.Equal(t, float64(1.5), result.ProcessingMetadata["processingTime"])
			assert.Equal(t, "completed", result.ProcessingMetadata["status"])
		})

		// Test Error on duplicate relationship
		t.Run("Error on duplicate relationship", func(t *testing.T) {
			params := repolib.CreateDerivedContentParams{
				ParentID:         parentContent.ID,
				DerivedContentID: derivedContent.ID,
				DerivationType:   domain.ContentDerivedTHUMBNAIL720, // Same as first test case
			}

			_, err := repo.CreateDerivedContentRelationship(ctx, params)
			assert.Error(t, err) // Should fail due to unique constraint
		})
	})
}

func TestPSQLContentRepository_GetDerivedContentByLevel(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create a new repository
		repo := NewPSQLContentRepository(db.Pool)
		ctx := context.Background()

		// Create a tenant
		tenantID := uuid.New()

		// Create a root content (level 0)
		rootContent := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Root Content",
			Description:    "Root Description",
			DocumentType:   "document",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := repo.Create(ctx, rootContent)
		require.NoError(t, err)

		// Create level 1 derived contents (direct children of root)
		level1Content1 := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Level 1 Content 1",
			Description:    "Level 1 Description 1",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, level1Content1)
		require.NoError(t, err)

		level1Content2 := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Level 1 Content 2",
			Description:    "Level 1 Description 2",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, level1Content2)
		require.NoError(t, err)

		// Create level 2 derived content (child of level1Content1)
		level2Content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Level 2 Content",
			Description:    "Level 2 Description",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, level2Content)
		require.NoError(t, err)

		// Create relationships
		// Root -> Level 1 Content 1
		_, err = repo.CreateDerivedContentRelationship(ctx, repolib.CreateDerivedContentParams{
			ParentID:         rootContent.ID,
			DerivedContentID: level1Content1.ID,
			DerivationType:   domain.ContentDerivedTHUMBNAIL720,
		})
		require.NoError(t, err)

		// Root -> Level 1 Content 2
		_, err = repo.CreateDerivedContentRelationship(ctx, repolib.CreateDerivedContentParams{
			ParentID:         rootContent.ID,
			DerivedContentID: level1Content2.ID,
			DerivationType:   domain.ContentDerivedTHUMBNAIL480,
		})
		require.NoError(t, err)

		// Level 1 Content 1 -> Level 2 Content
		_, err = repo.CreateDerivedContentRelationship(ctx, repolib.CreateDerivedContentParams{
			ParentID:         level1Content1.ID,
			DerivedContentID: level2Content.ID,
			DerivationType:   domain.ContentDerivedTHUMBNAIL720,
		})
		require.NoError(t, err)

		// Test case 1: Get level 0 (root content only)
		t.Run("Get level 0 with parent (root content only)", func(t *testing.T) {
			params := repolib.GetDerivedContentByLevelParams{
				RootID: rootContent.ID,
				Level:  0,
			}

			result, err := repo.GetDerivedContentByLevel(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 1)

			// Root content should have nil parent ID
			assert.Equal(t, rootContent.ID, result[0].Content.ID)
			assert.Equal(t, uuid.Nil, result[0].ParentID)
			assert.Equal(t, 0, result[0].Level)
		})

		// Test case 2: Get level 1 (root + direct children of root)
		t.Run("Get level 1 with parent (root + direct children of root)", func(t *testing.T) {
			params := repolib.GetDerivedContentByLevelParams{
				RootID: rootContent.ID,
				Level:  1,
			}

			result, err := repo.GetDerivedContentByLevel(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 3) // Root + 2 children

			// Create maps for easier verification
			contentMap := make(map[uuid.UUID]repolib.ContentWithParent)
			for _, item := range result {
				contentMap[item.Content.ID] = item
			}

			// Verify root content
			rootItem, exists := contentMap[rootContent.ID]
			assert.True(t, exists)
			assert.Equal(t, uuid.Nil, rootItem.ParentID)
			assert.Equal(t, 0, rootItem.Level)

			// Verify level 1 content 1
			level1Item1, exists := contentMap[level1Content1.ID]
			assert.True(t, exists)
			assert.Equal(t, rootContent.ID, level1Item1.ParentID)
			assert.Equal(t, 1, level1Item1.Level)

			// Verify level 1 content 2
			level1Item2, exists := contentMap[level1Content2.ID]
			assert.True(t, exists)
			assert.Equal(t, rootContent.ID, level1Item2.ParentID)
			assert.Equal(t, 1, level1Item2.Level)
		})

		// Test case 3: Get level 2 (root + level 1 + level 2)
		t.Run("Get level 2 with parent (root + level 1 + level 2)", func(t *testing.T) {
			params := repolib.GetDerivedContentByLevelParams{
				RootID: rootContent.ID,
				Level:  2,
			}

			result, err := repo.GetDerivedContentByLevel(ctx, params)
			require.NoError(t, err)
			assert.Len(t, result, 4) // Root + 2 children + 1 grandchild

			// Create maps for easier verification
			contentMap := make(map[uuid.UUID]repolib.ContentWithParent)
			for _, item := range result {
				contentMap[item.Content.ID] = item
			}

			// Verify level 2 content
			level2Item, exists := contentMap[level2Content.ID]
			assert.True(t, exists)
			assert.Equal(t, level1Content1.ID, level2Item.ParentID)
			assert.Equal(t, 2, level2Item.Level)
		})
	})
}

func TestPSQLContentRepository_DeleteDerivedContentRelationship(t *testing.T) {
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

		// Create a derived content
		derivedContent := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        uuid.New(),
			OwnerType:      "user",
			Name:           "Derived Content",
			Description:    "Derived Description",
			DocumentType:   "thumbnail",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeDerived,
		}
		err = repo.Create(ctx, derivedContent)
		require.NoError(t, err)

		// Create another derived content for multiple relationship tests
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

		// Create relationships for testing deletion
		params1 := repolib.CreateDerivedContentParams{
			ParentID:         parentContent.ID,
			DerivedContentID: derivedContent.ID,
			DerivationType:   domain.ContentDerivedTHUMBNAIL720,
		}
		_, err = repo.CreateDerivedContentRelationship(ctx, params1)
		require.NoError(t, err)

		params2 := repolib.CreateDerivedContentParams{
			ParentID:         parentContent.ID,
			DerivedContentID: derivedContent2.ID,
			DerivationType:   domain.ContentDerivedTHUMBNAIL480,
		}
		_, err = repo.CreateDerivedContentRelationship(ctx, params2)
		require.NoError(t, err)

		// Test case 1: Delete an existing relationship
		t.Run("Delete existing relationship", func(t *testing.T) {
			deleteParams := repolib.DeleteDerivedContentParams{
				ParentID:         parentContent.ID,
				DerivedContentID: derivedContent.ID,
			}

			err := repo.DeleteDerivedContentRelationship(ctx, deleteParams)
			require.NoError(t, err)

			// Verify the relationship is deleted by trying to list it
			listParams := repolib.ListDerivedContentParams{
				ParentIDs:      []uuid.UUID{parentContent.ID},
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL720},
			}
			result, err := repo.ListDerivedContent(ctx, listParams)
			require.NoError(t, err)
			assert.Len(t, result, 0) // Should be empty since we deleted the relationship
		})

		// Test case 2: Deleting a non-existent relationship should not error
		t.Run("Delete non-existent relationship", func(t *testing.T) {
			deleteParams := repolib.DeleteDerivedContentParams{
				ParentID:         parentContent.ID,
				DerivedContentID: uuid.New(), // Random non-existent ID
			}

			err := repo.DeleteDerivedContentRelationship(ctx, deleteParams)
			require.NoError(t, err) // Should not error, just no rows affected
		})

		// Test case 3: Verify other relationships remain after deletion
		t.Run("Other relationships remain intact", func(t *testing.T) {
			// Verify the second relationship is still there
			listParams := repolib.ListDerivedContentParams{
				ParentIDs:      []uuid.UUID{parentContent.ID},
				DerivationType: []string{domain.ContentDerivedTHUMBNAIL480},
			}
			result, err := repo.ListDerivedContent(ctx, listParams)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, derivedContent2.ID, result[0].ID)
		})
	})
}
