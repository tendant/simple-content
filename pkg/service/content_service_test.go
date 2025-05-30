package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/internal/repository/memory"
	"github.com/tendant/simple-content/pkg/service"
)

func setupContentService() *service.ContentService {
	contentRepo := memory.NewContentRepository()
	metadataRepo := memory.NewContentMetadataRepository()
	objectRepo := memory.NewObjectRepository()
	return service.NewContentService(contentRepo, metadataRepo, objectRepo)
}

func TestContentService_CreateContent(t *testing.T) {
	svc := setupContentService()
	ctx := context.Background()

	ownerID := uuid.New()
	tenantID := uuid.New()

	content, err := svc.CreateContent(ctx, ownerID, tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.Equal(t, ownerID, content.OwnerID)
	assert.Equal(t, tenantID, content.TenantID)
	assert.Equal(t, "original", content.DerivationType)
	// Note: DerivationLevel and ParentID have been removed from the Content struct
}

func TestContentService_CreateDerivedContent(t *testing.T) {
	svc := setupContentService()
	ctx := context.Background()

	// Create parent content
	ownerID := uuid.New()
	tenantID := uuid.New()
	parent, err := svc.CreateContent(ctx, ownerID, tenantID)
	assert.NoError(t, err)

	// Create derived content
	derived, err := svc.CreateDerivedContent(ctx, parent.ID, ownerID, tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, derived)
	// Parent relationship is now tracked in ContentDerived table
	assert.Equal(t, "derived", derived.DerivationType)

	// Create second-level derived content
	secondLevel, err := svc.CreateDerivedContent(ctx, derived.ID, ownerID, tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, secondLevel)
	// Parent relationship is now tracked in ContentDerived table
	assert.Equal(t, "derived", secondLevel.DerivationType)
}

func TestContentService_CreateDerivedContent_MaxDepthLimit(t *testing.T) {
	svc := setupContentService()
	ctx := context.Background()

	// Create a chain of derived content up to the max depth
	ownerID := uuid.New()
	tenantID := uuid.New()

	// Level 0 (original)
	content, err := svc.CreateContent(ctx, ownerID, tenantID)
	assert.NoError(t, err)
	// DerivationLevel is now tracked in the ContentDerived table

	// Levels 1-5
	currentID := content.ID
	for i := 1; i <= 5; i++ {
		derived, err := svc.CreateDerivedContent(ctx, currentID, ownerID, tenantID)
		assert.NoError(t, err)
		// Note: DerivationLevel has been removed from the Content struct
		currentID = derived.ID
	}

	// Attempt to create level 6 (should fail)
	_, err = svc.CreateDerivedContent(ctx, currentID, ownerID, tenantID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum derivation depth")
}

func TestContentService_SetContentMetadata(t *testing.T) {
	svc := setupContentService()
	ctx := context.Background()

	// Create content
	ownerID := uuid.New()
	tenantID := uuid.New()
	content, err := svc.CreateContent(ctx, ownerID, tenantID)
	assert.NoError(t, err)

	// Set metadata
	contentType := "video/mp4"
	title := "Test Video"
	description := "A test video"
	tags := []string{"test", "video"}
	fileSize := int64(1024)
	createdBy := "Test User"
	customMetadata := map[string]interface{}{
		"duration": "00:01:30",
	}

	err = svc.SetContentMetadata(
		ctx,
		content.ID,
		contentType,
		title,
		description,
		tags,
		fileSize,
		createdBy,
		customMetadata,
	)
	assert.NoError(t, err)

	// Get metadata
	metadata, err := svc.GetContentMetadata(ctx, content.ID)
	assert.NoError(t, err)
	// ContentType, Title, Description, and CreatedBy are now stored in the Metadata map
	assert.Equal(t, contentType, metadata.Metadata["content_type"])
	assert.Equal(t, title, metadata.Metadata["title"])
	assert.Equal(t, description, metadata.Metadata["description"])
	assert.Equal(t, tags, metadata.Tags)
	assert.Equal(t, fileSize, metadata.FileSize)
	assert.Equal(t, createdBy, metadata.Metadata["created_by"])
	assert.Equal(t, "00:01:30", metadata.Metadata["duration"])
}

func TestContentService_IndependentMetadata(t *testing.T) {
	svc := setupContentService()
	ctx := context.Background()

	// Create original content
	ownerID := uuid.New()
	tenantID := uuid.New()
	original, err := svc.CreateContent(ctx, ownerID, tenantID)
	assert.NoError(t, err)

	// Set metadata for original content
	err = svc.SetContentMetadata(
		ctx,
		original.ID,
		"video/mp4",
		"Original Video",
		"An original video",
		[]string{"original", "video"},
		int64(2048),
		"User 1",
		map[string]interface{}{
			"duration":   "00:05:30",
			"resolution": "1920x1080",
		},
	)
	assert.NoError(t, err)

	// Create derived content
	derived, err := svc.CreateDerivedContent(ctx, original.ID, ownerID, tenantID)
	assert.NoError(t, err)

	// Set different metadata for derived content
	err = svc.SetContentMetadata(
		ctx,
		derived.ID,
		"image/jpeg",
		"Thumbnail",
		"A thumbnail from the original video",
		[]string{"derived", "image", "thumbnail"},
		int64(512),
		"System",
		map[string]interface{}{
			"width":  1280,
			"height": 720,
		},
	)
	assert.NoError(t, err)

	// Get and verify original metadata
	originalMetadata, err := svc.GetContentMetadata(ctx, original.ID)
	assert.NoError(t, err)
	assert.Equal(t, "video/mp4", originalMetadata.Metadata["content_type"])
	assert.Equal(t, "Original Video", originalMetadata.Metadata["title"])
	assert.Equal(t, "00:05:30", originalMetadata.Metadata["duration"])

	// Get and verify derived metadata
	derivedMetadata, err := svc.GetContentMetadata(ctx, derived.ID)
	assert.NoError(t, err)
	assert.Equal(t, "image/jpeg", derivedMetadata.Metadata["content_type"])
	assert.Equal(t, "Thumbnail", derivedMetadata.Metadata["title"])
	// Check that the width value exists and is correct, regardless of type
	width, ok := derivedMetadata.Metadata["width"]
	assert.True(t, ok, "width field should exist in metadata")

	// Convert to int for comparison, handling both int and float64 cases
	var widthValue int
	switch v := width.(type) {
	case int:
		widthValue = v
	case float64:
		widthValue = int(v)
	default:
		t.Fatalf("unexpected type for width: %T", width)
	}

	assert.Equal(t, 1280, widthValue)
}
