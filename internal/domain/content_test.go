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
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		OwnerID:        uuid.New(),
		TenantID:       uuid.New(),
		Status:         "active",
		DerivationType: "original",
	}
	assert.Equal(t, "original", original.DerivationType)

	// Test derived content
	derived := &domain.Content{
		ID:             uuid.New(),
		CreatedAt:      now,
		UpdatedAt:      now,
		OwnerID:        uuid.New(),
		TenantID:       uuid.New(),
		Status:         "active",
		DerivationType: "derived",
	}
	assert.Equal(t, "derived", derived.DerivationType)
}

func TestContentMetadata_Validation(t *testing.T) {
	contentID := uuid.New()
	metadata := &domain.ContentMetadata{
		ContentID:         contentID,
		MimeType:          "video/mp4",
		FileName:          "test_video.mp4",
		Tags:              []string{"test", "video"},
		FileSize:          1024,
		Checksum:          "abc123",
		ChecksumAlgorithm: "md5",
		Metadata: map[string]interface{}{
			"duration":   "00:01:30",
			"resolution": "1920x1080",
			"title":      "Test Video",
			"description": "A test video description",
			"created_by":  "Test User",
		},
	}

	assert.Equal(t, contentID, metadata.ContentID)
	assert.Equal(t, "video/mp4", metadata.MimeType)
	assert.Equal(t, "test_video.mp4", metadata.FileName)
	assert.Equal(t, []string{"test", "video"}, metadata.Tags)
	assert.Equal(t, int64(1024), metadata.FileSize)
	assert.Equal(t, "abc123", metadata.Checksum)
	assert.Equal(t, "md5", metadata.ChecksumAlgorithm)
	assert.Equal(t, "00:01:30", metadata.Metadata["duration"])
	assert.Equal(t, "1920x1080", metadata.Metadata["resolution"])
	assert.Equal(t, "Test Video", metadata.Metadata["title"])
	assert.Equal(t, "A test video description", metadata.Metadata["description"])
	assert.Equal(t, "Test User", metadata.Metadata["created_by"])
}
