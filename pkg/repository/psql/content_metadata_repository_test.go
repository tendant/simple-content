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

func TestPSQLContentMetadataRepository_Set(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		metadataRepo := NewPSQLContentMetadataRepository(db.Pool)
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

		// Create metadata
		metadata := &domain.ContentMetadata{
			ContentID:         content.ID,
			MimeType:          "text/plain",
			FileName:          "test.txt",
			Checksum:          "abc123",
			ChecksumAlgorithm: "SHA-256",
			Tags:              []string{"test", "sample"},
			FileSize:          1024,
			Metadata: map[string]interface{}{
				"author":      "Test User",
				"description": "Test file description",
			},
		}

		// Set the metadata
		err = metadataRepo.Set(ctx, metadata)
		require.NoError(t, err)
		assert.False(t, metadata.CreatedAt.IsZero())
		assert.False(t, metadata.UpdatedAt.IsZero())
	})
}

func TestPSQLContentMetadataRepository_Get(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		metadataRepo := NewPSQLContentMetadataRepository(db.Pool)
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

		// Create metadata
		metadata := &domain.ContentMetadata{
			ContentID:         content.ID,
			MimeType:          "text/plain",
			FileName:          "test.txt",
			Checksum:          "abc123",
			ChecksumAlgorithm: "SHA-256",
			Tags:              []string{"test", "sample"},
			FileSize:          1024,
			Metadata: map[string]interface{}{
				"author":      "Test User",
				"description": "Test file description",
			},
		}

		// Set the metadata
		err = metadataRepo.Set(ctx, metadata)
		require.NoError(t, err)

		// Get the metadata
		retrieved, err := metadataRepo.Get(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, content.ID, retrieved.ContentID)
		assert.Equal(t, "text/plain", retrieved.MimeType)
		assert.Equal(t, "test.txt", retrieved.FileName)
		assert.Equal(t, "abc123", retrieved.Checksum)
		assert.Equal(t, "SHA-256", retrieved.ChecksumAlgorithm)
		assert.ElementsMatch(t, []string{"test", "sample"}, retrieved.Tags)
		assert.Equal(t, int64(1024), retrieved.FileSize)
		assert.Equal(t, "Test User", retrieved.Metadata["author"])
		assert.Equal(t, "Test file description", retrieved.Metadata["description"])

		// Update the metadata
		originalUpdatedAt := metadata.UpdatedAt
		time.Sleep(1 * time.Millisecond) // Ensure timestamp changes
		metadata.FileName = "updated.txt"
		metadata.FileSize = 2048
		metadata.Metadata["version"] = "1.1"

		err = metadataRepo.Set(ctx, metadata)
		require.NoError(t, err)

		// Get the updated metadata
		updated, err := metadataRepo.Get(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated.txt", updated.FileName)
		assert.Equal(t, int64(2048), updated.FileSize)
		assert.Equal(t, "1.1", updated.Metadata["version"])
		assert.True(t, updated.UpdatedAt.After(originalUpdatedAt))

		// Try to get metadata for non-existent content
		_, err = metadataRepo.Get(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestPSQLContentMetadataRepository_MultipleContents(t *testing.T) {
	RunTest(t, func(t *testing.T, db *TestDB) {
		// Create repositories
		contentRepo := NewPSQLContentRepository(db.Pool)
		metadataRepo := NewPSQLContentMetadataRepository(db.Pool)
		ctx := context.Background()

		// Create two contents
		tenantID := uuid.New()
		ownerID := uuid.New()
		
		// Content 1
		content1 := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Content 1",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err := contentRepo.Create(ctx, content1)
		require.NoError(t, err)
		
		// Content 2
		content2 := &domain.Content{
			TenantID:       tenantID,
			OwnerID:        ownerID,
			OwnerType:      "user",
			Name:           "Content 2",
			Status:         domain.ContentStatusCreated,
			DerivationType: domain.ContentDerivationTypeOriginal,
		}
		err = contentRepo.Create(ctx, content2)
		require.NoError(t, err)

		// Create metadata for content 1
		metadata1 := &domain.ContentMetadata{
			ContentID:         content1.ID,
			MimeType:          "video/mp4",
			FileName:          "video.mp4",
			Checksum:          "def456",
			ChecksumAlgorithm: "SHA-256",
			Tags:              []string{"video", "original"},
			FileSize:          10240,
			Metadata: map[string]interface{}{
				"duration":   "00:05:30",
				"resolution": "1920x1080",
			},
		}
		err = metadataRepo.Set(ctx, metadata1)
		require.NoError(t, err)

		// Create metadata for content 2
		metadata2 := &domain.ContentMetadata{
			ContentID:         content2.ID,
			MimeType:          "image/jpeg",
			FileName:          "image.jpg",
			Checksum:          "ghi789",
			ChecksumAlgorithm: "SHA-256",
			Tags:              []string{"image", "thumbnail"},
			FileSize:          512,
			Metadata: map[string]interface{}{
				"width":  1280,
				"height": 720,
			},
		}
		err = metadataRepo.Set(ctx, metadata2)
		require.NoError(t, err)

		// Get metadata for content 1
		retrieved1, err := metadataRepo.Get(ctx, content1.ID)
		require.NoError(t, err)
		assert.Equal(t, "video/mp4", retrieved1.MimeType)
		assert.Equal(t, "video.mp4", retrieved1.FileName)
		assert.ElementsMatch(t, []string{"video", "original"}, retrieved1.Tags)

		// Get metadata for content 2
		retrieved2, err := metadataRepo.Get(ctx, content2.ID)
		require.NoError(t, err)
		assert.Equal(t, "image/jpeg", retrieved2.MimeType)
		assert.Equal(t, "image.jpg", retrieved2.FileName)
		assert.ElementsMatch(t, []string{"image", "thumbnail"}, retrieved2.Tags)
	})
}
