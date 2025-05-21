package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/internal/domain"
)

func TestContent_DerivationValidation(t *testing.T) {
	// Test original content
	now := time.Now()
	original := &domain.Content{
		ID:              uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
		OwnerID:         uuid.New(),
		TenantID:        uuid.New(),
		Status:          "active",
		DerivationType:  "original",
		DerivationLevel: 0,
	}
	assert.Equal(t, "original", original.DerivationType)
	assert.Equal(t, 0, original.DerivationLevel)
	assert.Nil(t, original.ParentID)

	// Test derived content
	parentID := uuid.New()
	derived := &domain.Content{
		ID:              uuid.New(),
		ParentID:        &parentID,
		CreatedAt:       now,
		UpdatedAt:       now,
		OwnerID:         uuid.New(),
		TenantID:        uuid.New(),
		Status:          "active",
		DerivationType:  "derived",
		DerivationLevel: 1,
	}
	assert.Equal(t, parentID, *derived.ParentID)
	assert.Equal(t, "derived", derived.DerivationType)
	assert.Equal(t, 1, derived.DerivationLevel)
}

func TestContentMetadata_Validation(t *testing.T) {
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

	assert.Equal(t, contentID, metadata.ContentID)
	assert.Equal(t, "video/mp4", metadata.ContentType)
	assert.Equal(t, "Test Video", metadata.Title)
	assert.Equal(t, "A test video description", metadata.Description)
	assert.Equal(t, []string{"test", "video"}, metadata.Tags)
	assert.Equal(t, int64(1024), metadata.FileSize)
	assert.Equal(t, "Test User", metadata.CreatedBy)
	assert.Equal(t, "00:01:30", metadata.Metadata["duration"])
	assert.Equal(t, "1920x1080", metadata.Metadata["resolution"])
}
