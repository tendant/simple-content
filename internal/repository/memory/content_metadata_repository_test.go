package memory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository/memory"
)

func TestContentMetadataRepository_Set(t *testing.T) {
	repo := memory.NewContentMetadataRepository()
	ctx := context.Background()

	contentID := uuid.New()
	metadata := &domain.ContentMetadata{
		ContentID:   contentID,
		ContentType: "video/mp4",
		Title:       "Test Video",
		Description: "A test video description",
		Tags:        []string{"test", "video"},
		FileSize:    1024,
		CreatedBy:   "Test User",
		Metadata: map[string]interface{}{
			"duration":   "00:01:30",
			"resolution": "1920x1080",
		},
	}

	err := repo.Set(ctx, metadata)
	assert.NoError(t, err)

	// Update the metadata
	metadata.Title = "Updated Title"
	metadata.Metadata["duration"] = "00:02:00"

	err = repo.Set(ctx, metadata)
	assert.NoError(t, err)
}

func TestContentMetadataRepository_Get(t *testing.T) {
	repo := memory.NewContentMetadataRepository()
	ctx := context.Background()

	contentID := uuid.New()
	metadata := &domain.ContentMetadata{
		ContentID:   contentID,
		ContentType: "video/mp4",
		Title:       "Test Video",
		Description: "A test video description",
		Tags:        []string{"test", "video"},
		FileSize:    1024,
		CreatedBy:   "Test User",
		Metadata: map[string]interface{}{
			"duration":   "00:01:30",
			"resolution": "1920x1080",
		},
	}

	err := repo.Set(ctx, metadata)
	assert.NoError(t, err)

	// Get the metadata
	retrieved, err := repo.Get(ctx, contentID)
	assert.NoError(t, err)
	assert.Equal(t, contentID, retrieved.ContentID)
	assert.Equal(t, "video/mp4", retrieved.ContentType)
	assert.Equal(t, "Test Video", retrieved.Title)
	assert.Equal(t, "A test video description", retrieved.Description)
	assert.Equal(t, []string{"test", "video"}, retrieved.Tags)
	assert.Equal(t, int64(1024), retrieved.FileSize)
	assert.Equal(t, "Test User", retrieved.CreatedBy)
	assert.Equal(t, "00:01:30", retrieved.Metadata["duration"])
	assert.Equal(t, "1920x1080", retrieved.Metadata["resolution"])

	// Try to get non-existent metadata
	_, err = repo.Get(ctx, uuid.New())
	assert.Error(t, err)
}

func TestContentMetadataRepository_MultipleContents(t *testing.T) {
	repo := memory.NewContentMetadataRepository()
	ctx := context.Background()

	// Create metadata for original content
	originalID := uuid.New()
	originalMetadata := &domain.ContentMetadata{
		ContentID:   originalID,
		ContentType: "video/mp4",
		Title:       "Original Video",
		Description: "An original video",
		Tags:        []string{"original", "video"},
		FileSize:    2048,
		CreatedBy:   "User 1",
		Metadata: map[string]interface{}{
			"duration":   "00:05:30",
			"resolution": "1920x1080",
		},
	}

	err := repo.Set(ctx, originalMetadata)
	assert.NoError(t, err)

	// Create metadata for derived content
	derivedID := uuid.New()
	derivedMetadata := &domain.ContentMetadata{
		ContentID:   derivedID,
		ContentType: "image/jpeg",
		Title:       "Thumbnail",
		Description: "A thumbnail from the original video",
		Tags:        []string{"derived", "image", "thumbnail"},
		FileSize:    512,
		CreatedBy:   "System",
		Metadata: map[string]interface{}{
			"width":  1280,
			"height": 720,
		},
	}

	err = repo.Set(ctx, derivedMetadata)
	assert.NoError(t, err)

	// Get and verify original metadata
	retrievedOriginal, err := repo.Get(ctx, originalID)
	assert.NoError(t, err)
	assert.Equal(t, "video/mp4", retrievedOriginal.ContentType)
	assert.Equal(t, "Original Video", retrievedOriginal.Title)

	// Get and verify derived metadata
	retrievedDerived, err := repo.Get(ctx, derivedID)
	assert.NoError(t, err)
	assert.Equal(t, "image/jpeg", retrievedDerived.ContentType)
	assert.Equal(t, "Thumbnail", retrievedDerived.Title)
}
