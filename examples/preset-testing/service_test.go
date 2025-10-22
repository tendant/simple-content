package presettest

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/presets"
)

// TestUploadAndDownload demonstrates the testing preset for unit tests
func TestUploadAndDownload(t *testing.T) {
	// Create service with testing preset
	// - In-memory database (isolated per test)
	// - In-memory storage (fast, no disk I/O)
	// - Automatic cleanup via t.Cleanup()
	svc := presets.NewTesting(t)

	ctx := context.Background()

	// Upload content
	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Test Document",
		DocumentType: "text/plain",
		Reader:       strings.NewReader("Test content"),
		FileName:     "test.txt",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, content.ID)
	assert.Equal(t, "Test Document", content.Name)

	// Download content
	reader, err := svc.DownloadContent(ctx, content.ID)
	require.NoError(t, err)
	defer reader.Close()

	// Verify content
	buf := new(strings.Builder)
	_, err = io.Copy(buf, reader)
	require.NoError(t, err)
	assert.Equal(t, "Test content", buf.String())
}

// TestDerivedContent demonstrates creating and querying derived content
func TestDerivedContent(t *testing.T) {
	svc := presets.NewTesting(t)
	ctx := context.Background()

	// Upload original image
	original, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Original Image",
		DocumentType: "image/jpeg",
		Reader:       strings.NewReader("fake-image-data"),
		FileName:     "photo.jpg",
	})
	require.NoError(t, err)

	// Create thumbnail
	thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
		ParentID:       original.ID,
		DerivationType: "thumbnail",
		Variant:        "thumbnail_128",
		Reader:         strings.NewReader("fake-thumbnail-data"),
		FileName:       "photo_thumb.jpg",
	})
	require.NoError(t, err)
	assert.Equal(t, "thumbnail", thumbnail.DerivationType)

	// Get content details
	details, err := svc.GetContentDetails(ctx, original.ID)
	require.NoError(t, err)
	assert.True(t, details.Ready)

	// Verify thumbnail was created correctly
	thumbnailContent, err := svc.GetContent(ctx, thumbnail.ID)
	require.NoError(t, err)
	assert.Equal(t, "thumbnail", thumbnailContent.DerivationType)
}

// TestMetadata demonstrates content metadata operations
func TestMetadata(t *testing.T) {
	svc := presets.NewTesting(t)
	ctx := context.Background()

	// Upload content
	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Document with Metadata",
		DocumentType: "application/pdf",
		Reader:       strings.NewReader("fake-pdf-data"),
		FileName:     "document.pdf",
	})
	require.NoError(t, err)

	// Set custom metadata
	err = svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
		ContentID: content.ID,
		CustomMetadata: map[string]interface{}{
			"author":      "John Doe",
			"pages":       42,
			"confidential": true,
		},
	})
	require.NoError(t, err)

	// Get metadata
	metadata, err := svc.GetContentMetadata(ctx, content.ID)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", metadata.Metadata["author"])
	assert.Equal(t, 42, metadata.Metadata["pages"])
	assert.True(t, metadata.Metadata["confidential"].(bool))
}

// TestParallelExecution demonstrates that tests can run in parallel
func TestParallelExecution(t *testing.T) {
	t.Run("test1", func(t *testing.T) {
		t.Parallel()
		svc := presets.NewTesting(t)
		ctx := context.Background()

		content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "Test 1",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Data 1"),
			FileName:     "test1.txt",
		})
		require.NoError(t, err)
		assert.Equal(t, "Test 1", content.Name)
	})

	t.Run("test2", func(t *testing.T) {
		t.Parallel()
		svc := presets.NewTesting(t)
		ctx := context.Background()

		content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "Test 2",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Data 2"),
			FileName:     "test2.txt",
		})
		require.NoError(t, err)
		assert.Equal(t, "Test 2", content.Name)
	})
}

// TestIsolation demonstrates that each test gets its own isolated service
func TestIsolation(t *testing.T) {
	// Create two services in the same test
	svc1 := presets.NewTesting(t)
	svc2 := presets.NewTesting(t)

	ctx := context.Background()

	// Upload to svc1
	content1, err := svc1.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Service 1 Content",
		DocumentType: "text/plain",
		Reader:       strings.NewReader("Data from service 1"),
		FileName:     "svc1.txt",
	})
	require.NoError(t, err)

	// Content should NOT exist in svc2 (isolated)
	_, err = svc2.GetContent(ctx, content1.ID)
	assert.Error(t, err, "content from svc1 should not exist in svc2")
}

// TestConvenienceFunction demonstrates the TestService helper
func TestConvenienceFunction(t *testing.T) {
	// TestService is an alias for NewTesting with no options
	svc := presets.TestService(t)
	ctx := context.Background()

	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Convenience Test",
		DocumentType: "text/plain",
		Reader:       strings.NewReader("Quick test"),
		FileName:     "quick.txt",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, content.ID)
}
