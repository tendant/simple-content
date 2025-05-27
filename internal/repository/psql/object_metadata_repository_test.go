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

func TestPSQLObjectMetadataRepository_Set(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		metadataRepo := NewPSQLObjectMetadataRepository(db.Pool)
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
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)

		// Create metadata
		metadata := &domain.ObjectMetadata{
			ObjectID:  object.ID,
			SizeBytes: 1024,
			MimeType:  "text/plain",
			ETag:      "abc123",
			Metadata: map[string]interface{}{
				"author":      "Test User",
				"description": "Test file description",
				"tags":        []string{"test", "example"},
			},
		}

		// Set the metadata
		err = metadataRepo.Set(ctx, metadata)
		require.NoError(t, err)
		assert.False(t, metadata.CreatedAt.IsZero())
		assert.False(t, metadata.UpdatedAt.IsZero())
	})
}

func TestPSQLObjectMetadataRepository_Get(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		metadataRepo := NewPSQLObjectMetadataRepository(db.Pool)
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
		err = objectRepo.Create(ctx, object)
		require.NoError(t, err)

		// Create metadata
		metadata := &domain.ObjectMetadata{
			ObjectID:  object.ID,
			SizeBytes: 1024,
			MimeType:  "text/plain",
			ETag:      "abc123",
			Metadata: map[string]interface{}{
				"author":      "Test User",
				"description": "Test file description",
				"tags":        []string{"test", "example"},
			},
		}

		// Set the metadata
		err = metadataRepo.Set(ctx, metadata)
		require.NoError(t, err)

		// Get the metadata
		retrieved, err := metadataRepo.Get(ctx, object.ID)
		require.NoError(t, err)
		assert.Equal(t, object.ID, retrieved.ObjectID)
		assert.Equal(t, int64(1024), retrieved.SizeBytes)
		assert.Equal(t, "text/plain", retrieved.MimeType)
		assert.Equal(t, "abc123", retrieved.ETag)
		assert.Equal(t, "Test User", retrieved.Metadata["author"])
		assert.Equal(t, "Test file description", retrieved.Metadata["description"])
		assert.Equal(t, []interface{}{"test", "example"}, retrieved.Metadata["tags"])

		// Update the metadata
		originalUpdatedAt := metadata.UpdatedAt
		time.Sleep(1 * time.Millisecond) // Ensure timestamp changes
		metadata.SizeBytes = 2048
		metadata.MimeType = "application/json"
		metadata.Metadata["version"] = "1.1"

		err = metadataRepo.Set(ctx, metadata)
		require.NoError(t, err)

		// Get the updated metadata
		updated, err := metadataRepo.Get(ctx, object.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2048), updated.SizeBytes)
		assert.Equal(t, "application/json", updated.MimeType)
		assert.Equal(t, "1.1", updated.Metadata["version"])
		assert.True(t, updated.UpdatedAt.After(originalUpdatedAt))

		// Try to get metadata for non-existent object
		_, err = metadataRepo.Get(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestPSQLObjectMetadataRepository_MultipleObjects(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		objectRepo := NewPSQLObjectRepository(db.Pool)
		metadataRepo := NewPSQLObjectMetadataRepository(db.Pool)
		ctx := context.Background()

		// Create a content
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

		// Create two objects for the same content
		object1 := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: "test-backend",
			StorageClass:       "standard",
			ObjectKey:          "test-key-1",
			FileName:           "test-file-1.txt",
			Version:            1,
			ObjectType:         "file",
			Status:             domain.ObjectStatusUploaded,
		}
		err = objectRepo.Create(ctx, object1)
		require.NoError(t, err)

		object2 := &domain.Object{
			ContentID:          content.ID,
			StorageBackendName: "test-backend",
			StorageClass:       "standard",
			ObjectKey:          "test-key-2",
			FileName:           "test-file-2.jpg",
			Version:            2,
			ObjectType:         "image",
			Status:             domain.ObjectStatusUploaded,
		}
		err = objectRepo.Create(ctx, object2)
		require.NoError(t, err)

		// Create metadata for object 1
		metadata1 := &domain.ObjectMetadata{
			ObjectID:  object1.ID,
			SizeBytes: 1024,
			MimeType:  "text/plain",
			ETag:      "abc123",
			Metadata: map[string]interface{}{
				"author":      "User 1",
				"description": "Text file",
			},
		}
		err = metadataRepo.Set(ctx, metadata1)
		require.NoError(t, err)

		// Create metadata for object 2
		metadata2 := &domain.ObjectMetadata{
			ObjectID:  object2.ID,
			SizeBytes: 2048,
			MimeType:  "image/jpeg",
			ETag:      "def456",
			Metadata: map[string]interface{}{
				"width":  1280,
				"height": 720,
			},
		}
		err = metadataRepo.Set(ctx, metadata2)
		require.NoError(t, err)

		// Get metadata for object 1
		retrieved1, err := metadataRepo.Get(ctx, object1.ID)
		require.NoError(t, err)
		assert.Equal(t, "text/plain", retrieved1.MimeType)
		assert.Equal(t, int64(1024), retrieved1.SizeBytes)
		assert.Equal(t, "User 1", retrieved1.Metadata["author"])

		// Get metadata for object 2
		retrieved2, err := metadataRepo.Get(ctx, object2.ID)
		require.NoError(t, err)
		assert.Equal(t, "image/jpeg", retrieved2.MimeType)
		assert.Equal(t, int64(2048), retrieved2.SizeBytes)
		assert.Equal(t, float64(1280), retrieved2.Metadata["width"])
		assert.Equal(t, float64(720), retrieved2.Metadata["height"])
	})
}
