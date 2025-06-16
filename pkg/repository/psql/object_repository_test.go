package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/domain"
)

func TestPSQLObjectRepository_Create(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		ctx := context.Background()

		// Create a content first
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content)
		require.NoError(t, err)

		// Create an object
		object := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: "test-backend",
			StorageClass:       "standard",
			ObjectKey:          "test-key",
			FileName:           "test-file.txt",
			Version:            1,
			ObjectType:         "file",
			Status:             domain.ObjectStatusCreated,
		}

		// Create the object
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, object.ID)
		assert.False(t, object.CreatedAt.IsZero())
		assert.False(t, object.UpdatedAt.IsZero())
	})
}

func TestPSQLObjectRepository_Get(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		ctx := context.Background()

		// Create a content first
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content)
		require.NoError(t, err)

		// Create an object
		object := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: "test-backend",
			StorageClass:       "standard",
			ObjectKey:          "test-key",
			FileName:           "test-file.txt",
			ObjectType:         "file",
			Status:             domain.ObjectStatusCreated,
			Version:            1,
		}

		// Create the object
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)

		// Get the object
		retrieved, err := objectRepo.Get(ctx, object.ID)
		require.NoError(t, err)
		assert.Equal(t, object.ID, retrieved.ID)
		assert.Equal(t, content.ID, retrieved.ContentID)
		assert.Equal(t, "test-backend", retrieved.StorageBackendName)
		assert.Equal(t, "standard", retrieved.StorageClass)
		assert.Equal(t, "test-key", retrieved.ObjectKey)
		assert.Equal(t, "test-file.txt", retrieved.FileName)
		assert.Equal(t, 1, retrieved.Version)
		assert.Equal(t, "file", retrieved.ObjectType)
		assert.Equal(t, domain.ObjectStatusCreated, retrieved.Status)

		// Try to get a non-existent object
		_, err = objectRepo.Get(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestPSQLObjectRepository_GetByContentID(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		ctx := context.Background()

		// Create a content first
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content)
		require.NoError(t, err)

		// Create multiple objects for the same content
		objects := []*domain.Object{
			{
				ContentID:          content.ID,
				StorageBackendName: "test-backend",
				StorageClass:       "standard",
				ObjectKey:          "test-key-1",
				FileName:           "test-file-1.txt",
				Version:            1,

				ObjectType: "file",
				Status:     domain.ObjectStatusCreated,
			},
			{
				ContentID:          content.ID,
				StorageBackendName: "test-backend",
				StorageClass:       "standard",
				ObjectKey:          "test-key-2",
				FileName:           "test-file-2.txt",
				Version:            2,

				ObjectType: "file",
				Status:     domain.ObjectStatusUploaded,
			},
		}

		// Create all objects
		for _, obj := range objects {
			err := objectRepo.Create(ctx, obj)
			require.NoError(t, err)
		}

		// Get objects by content ID
		result, err := objectRepo.GetByContentID(ctx, content.ID)
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Verify objects are returned in the correct order (by version)
		assert.Equal(t, "test-key-2", result[0].ObjectKey)
		assert.Equal(t, "test-key-1", result[1].ObjectKey)

		// Try to get objects for a non-existent content
		result, err = objectRepo.GetByContentID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestPSQLObjectRepository_Update(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		ctx := context.Background()

		// Create a content first
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content)
		require.NoError(t, err)

		// Create an object
		object := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: "test-backend",
			StorageClass:       "standard",
			ObjectKey:          "test-key",
			FileName:           "test-file.txt",
			ObjectType:         "file",
			Status:             domain.ObjectStatusCreated,
		}

		// Create the object
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)

		// Update the object
		originalUpdatedAt := object.UpdatedAt
		time.Sleep(100 * time.Millisecond) // Ensure timestamp changes
		object.FileName = "updated-file.txt"
		object.Status = domain.ObjectStatusUploaded

		err = objectRepo.Update(ctx, object)
		require.NoError(t, err)

		// Get the updated object
		updated, err := objectRepo.Get(ctx, object.ID)
		require.NoError(t, err)

		assert.Equal(t, "updated-file.txt", updated.FileName)
		assert.Equal(t, domain.ObjectStatusUploaded, updated.Status)
		assert.True(t, updated.UpdatedAt.After(originalUpdatedAt))
	})
}

func TestPSQLObjectRepository_Delete(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		ctx := context.Background()

		// Create a content first
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content)
		require.NoError(t, err)

		// Create an object
		object := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: "test-backend",
			StorageClass:       "standard",
			ObjectKey:          "test-key",
			FileName:           "test-file.txt",
			ObjectType:         "file",
			Status:             domain.ObjectStatusCreated,
			Version:            1,
		}

		// Create the object
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)

		// Delete the object
		err = objectRepo.Delete(ctx, object.ID)
		require.NoError(t, err)

		// Try to get the deleted object
		_, err = objectRepo.Get(ctx, object.ID)
		assert.Error(t, err)
	})
}

func TestPSQLObjectRepository_GetByObjectKeyAndStorageBackendName(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		ctx := context.Background()

		// Create a content first
		tenantID := uuid.New()
		ownerID := uuid.New()
		content := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Test Content",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content)
		require.NoError(t, err)

		// Create an object with specific object key and storage backend name
		uniqueName := "unique-storage-backend"
		uniqueKey := "unique-object-key"
		object := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: uniqueName,
			StorageClass:       "standard",
			ObjectKey:          uniqueKey,
			FileName:           "test-file.txt",
			Version:            1,
			ObjectType:         "file",
			Status:             domain.ObjectStatusCreated,
		}

		// Create the object
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)

		// Create another object with same content ID but different object key
		object2 := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: uniqueName,
			StorageClass:       "standard",
			ObjectKey:          "different-key",
			FileName:           "another-file.txt",
			Version:            1,
			ObjectType:         "file",
			Status:             domain.ObjectStatusCreated,
		}

		// Create the second object
		err = objectRepo.Create(ctx, object2)
		require.NoError(t, err)

		// Test cases
		t.Run("Find object by key and storage backend name", func(t *testing.T) {
			// Get the object by key and storage backend name
			foundObject, err := objectRepo.GetByObjectKeyAndStorageBackendName(ctx, uniqueKey, uniqueName)
			require.NoError(t, err)
			assert.NotNil(t, foundObject)
			assert.Equal(t, object.ID, foundObject.ID)
			assert.Equal(t, object.ContentID, foundObject.ContentID)
			assert.Equal(t, uniqueKey, foundObject.ObjectKey)
			assert.Equal(t, uniqueName, foundObject.StorageBackendName)
		})

		t.Run("Object not found with incorrect key", func(t *testing.T) {
			// Try to get an object with a non-existent key
			_, err := objectRepo.GetByObjectKeyAndStorageBackendName(ctx, "non-existent-key", uniqueName)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "object not found")
		})

		t.Run("Object not found with incorrect storage backend name", func(t *testing.T) {
			// Try to get an object with a non-existent storage backend name
			_, err := objectRepo.GetByObjectKeyAndStorageBackendName(ctx, uniqueKey, "non-existent-backend")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "object not found")
		})

		t.Run("Deleted object not found", func(t *testing.T) {
			// Delete the object
			err := objectRepo.Delete(ctx, object.ID)
			require.NoError(t, err)

			// Try to get the deleted object by key and storage backend name
			_, err = objectRepo.GetByObjectKeyAndStorageBackendName(ctx, uniqueKey, uniqueName)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "object not found")
		})
	})
}
