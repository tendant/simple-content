package simplecontent_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func TestServiceCreation(t *testing.T) {
	tests := []struct {
		name        string
		options     []simplecontent.Option
		expectError bool
	}{
		{
			name:        "no options should fail",
			options:     []simplecontent.Option{},
			expectError: true,
		},
		{
			name: "with repository should succeed",
			options: []simplecontent.Option{
				simplecontent.WithRepository(memory.New()),
			},
			expectError: false,
		},
		{
			name: "with repository and blob store should succeed",
			options: []simplecontent.Option{
				simplecontent.WithRepository(memory.New()),
				simplecontent.WithBlobStore("memory", memorystorage.New()),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := simplecontent.New(tt.options...)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
			}
		})
	}
}

func setupTestService(t *testing.T) simplecontent.Service {
	repo := memory.New()
	store := memorystorage.New()
	eventSink := simplecontent.NewNoopEventSink()
	previewer := simplecontent.NewBasicImagePreviewer()

	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", store),
		simplecontent.WithEventSink(eventSink),
		simplecontent.WithPreviewer(previewer),
	)
	require.NoError(t, err)
	require.NotNil(t, svc)

	return svc
}

func TestContentOperations(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	t.Run("CreateContent", func(t *testing.T) {
		req := simplecontent.CreateContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "Test Content",
			Description:  "A test content item",
			DocumentType: "text/plain",
		}

		content, err := svc.CreateContent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, content)
		assert.Equal(t, req.Name, content.Name)
		assert.Equal(t, req.Description, content.Description)
		assert.Equal(t, req.DocumentType, content.DocumentType)
        assert.Equal(t, string(simplecontent.ContentStatusCreated), content.Status)
		assert.False(t, content.CreatedAt.IsZero())
		assert.False(t, content.UpdatedAt.IsZero())
	})

	t.Run("GetContent", func(t *testing.T) {
		// First create a content
		req := simplecontent.CreateContentRequest{
			OwnerID:     uuid.New(),
			TenantID:    uuid.New(),
			Name:        "Test Content for Get",
			Description: "A test content for retrieval",
		}

		created, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		// Then retrieve it
		retrieved, err := svc.GetContent(ctx, created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Name, retrieved.Name)
		assert.Equal(t, created.Description, retrieved.Description)
	})

	t.Run("GetContent_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		content, err := svc.GetContent(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, content)
	})

	t.Run("ListContent", func(t *testing.T) {
		ownerID := uuid.New()
		tenantID := uuid.New()

		// Create multiple contents
		for i := 0; i < 3; i++ {
			req := simplecontent.CreateContentRequest{
				OwnerID:     ownerID,
				TenantID:    tenantID,
				Name:        fmt.Sprintf("Test Content %d", i+1),
				Description: fmt.Sprintf("Description %d", i+1),
			}
			_, err := svc.CreateContent(ctx, req)
			require.NoError(t, err)
		}

		// List contents
		listReq := simplecontent.ListContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
		}
		contents, err := svc.ListContent(ctx, listReq)
		assert.NoError(t, err)
		assert.Len(t, contents, 3)

		// Verify they're sorted by creation time (newest first)
		for i := 0; i < len(contents)-1; i++ {
			assert.True(t, contents[i].CreatedAt.After(contents[i+1].CreatedAt) || 
				contents[i].CreatedAt.Equal(contents[i+1].CreatedAt))
		}
	})

	t.Run("UpdateContent", func(t *testing.T) {
		// Create content
		req := simplecontent.CreateContentRequest{
			OwnerID:     uuid.New(),
			TenantID:    uuid.New(),
			Name:        "Original Name",
			Description: "Original Description",
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		// Small delay to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		// Update content
		content.Name = "Updated Name"
		content.Description = "Updated Description"
		updateReq := simplecontent.UpdateContentRequest{Content: content}

		err = svc.UpdateContent(ctx, updateReq)
		assert.NoError(t, err)

		// Verify update
		updated, err := svc.GetContent(ctx, content.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "Updated Description", updated.Description)
		assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
	})

	t.Run("DeleteContent", func(t *testing.T) {
		// Create content
		req := simplecontent.CreateContentRequest{
			OwnerID:  uuid.New(),
			TenantID: uuid.New(),
			Name:     "Content to Delete",
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		// Delete content
		err = svc.DeleteContent(ctx, content.ID)
		assert.NoError(t, err)

		// Verify deletion
		deleted, err := svc.GetContent(ctx, content.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})
}

func TestContentMetadataOperations(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create a content first
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Content with Metadata",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(t, err)

	t.Run("SetContentMetadata", func(t *testing.T) {
		metadataReq := simplecontent.SetContentMetadataRequest{
			ContentID:   content.ID,
			ContentType: "text/plain",
			Title:       "Test Document",
			Description: "A test document",
			Tags:        []string{"test", "document"},
			FileName:    "test.txt",
			FileSize:    1024,
			CreatedBy:   "test-user",
			CustomMetadata: map[string]interface{}{
				"category": "testing",
				"priority": "high",
			},
		}

		err := svc.SetContentMetadata(ctx, metadataReq)
		assert.NoError(t, err)
	})

	t.Run("GetContentMetadata", func(t *testing.T) {
		metadata, err := svc.GetContentMetadata(ctx, content.ID)
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, content.ID, metadata.ContentID)
		assert.Equal(t, "text/plain", metadata.MimeType)
		assert.Equal(t, "test.txt", metadata.FileName)
		assert.Equal(t, int64(1024), metadata.FileSize)
		assert.Equal(t, []string{"test", "document"}, metadata.Tags)
		assert.Contains(t, metadata.Metadata, "category")
		assert.Contains(t, metadata.Metadata, "priority")
	})
}

func TestObjectOperations(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create a content first
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Content for Objects",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(t, err)

	t.Run("CreateObject", func(t *testing.T) {
		objReq := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "memory",
			Version:            1,
		}

		object, err := svc.CreateObject(ctx, objReq)
		assert.NoError(t, err)
		assert.NotNil(t, object)
		assert.Equal(t, content.ID, object.ContentID)
		assert.Equal(t, "memory", object.StorageBackendName)
		assert.Equal(t, 1, object.Version)
        assert.Equal(t, string(simplecontent.ObjectStatusCreated), object.Status)
		assert.NotEmpty(t, object.ObjectKey)
	})

	t.Run("GetObject", func(t *testing.T) {
		// Create object
		objReq := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "memory",
			Version:            1,
		}
		created, err := svc.CreateObject(ctx, objReq)
		require.NoError(t, err)

		// Retrieve object
		retrieved, err := svc.GetObject(ctx, created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.ContentID, retrieved.ContentID)
	})

	t.Run("GetObjectsByContentID", func(t *testing.T) {
		// Create multiple objects for the same content
		for i := 0; i < 3; i++ {
			objReq := simplecontent.CreateObjectRequest{
				ContentID:          content.ID,
				StorageBackendName: "memory",
				Version:            i + 2, // Start from version 2
			}
			_, err := svc.CreateObject(ctx, objReq)
			require.NoError(t, err)
		}

		// Get objects by content ID
		objects, err := svc.GetObjectsByContentID(ctx, content.ID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(objects), 3) // At least 3, might be more from other tests
	})
}

func TestObjectUploadDownload(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create content and object
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Content for Upload Test",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(t, err)

	objReq := simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, objReq)
	require.NoError(t, err)

	testData := "Hello, World! This is test data for upload/download."

	t.Run("UploadObject", func(t *testing.T) {
		reader := strings.NewReader(testData)
		req := simplecontent.UploadObjectRequest{
			ObjectID: object.ID,
			Reader:   reader,
		}
		err := svc.UploadObject(ctx, req)
		assert.NoError(t, err)

		// Verify object status was updated
		updated, err := svc.GetObject(ctx, object.ID)
		assert.NoError(t, err)
        assert.Equal(t, string(simplecontent.ObjectStatusUploaded), updated.Status)
	})

	t.Run("DownloadObject", func(t *testing.T) {
		reader, err := svc.DownloadObject(ctx, object.ID)
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		defer reader.Close()

		downloadedData, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, testData, string(downloadedData))
	})

	t.Run("UploadObjectWithMetadata", func(t *testing.T) {
		// Create another object for this test
		objReq2 := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "memory",
			Version:            2,
		}
		object2, err := svc.CreateObject(ctx, objReq2)
		require.NoError(t, err)

		reader := strings.NewReader("Test data with metadata")
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: object2.ID,
			Reader:   reader,
			MimeType: "text/plain",
		}

		err = svc.UploadObject(ctx, uploadReq)
		assert.NoError(t, err)
	})
}

func TestErrorHandling(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	t.Run("GetNonExistentContent", func(t *testing.T) {
		content, err := svc.GetContent(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, content)
	})

	t.Run("CreateObjectWithInvalidBackend", func(t *testing.T) {
		// Create content first
		contentReq := simplecontent.CreateContentRequest{
			OwnerID:  uuid.New(),
			TenantID: uuid.New(),
			Name:     "Test Content",
		}
		content, err := svc.CreateContent(ctx, contentReq)
		require.NoError(t, err)

		// Try to create object with non-existent backend
		objReq := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "nonexistent",
			Version:            1,
		}

		object, err := svc.CreateObject(ctx, objReq)
		assert.Error(t, err)
		assert.Nil(t, object)
	})

	t.Run("UploadToNonExistentObject", func(t *testing.T) {
		reader := strings.NewReader("test data")
		req := simplecontent.UploadObjectRequest{
			ObjectID: uuid.New(),
			Reader:   reader,
		}
		err := svc.UploadObject(ctx, req)
		assert.Error(t, err)
	})
}

func TestDerivedContent(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create parent content
	parentReq := simplecontent.CreateContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Parent Content",
		Description:  "Original content",
		DocumentType: "image/jpeg",
	}
	parent, err := svc.CreateContent(ctx, parentReq)
	require.NoError(t, err)

    t.Run("CreateDerivedContent", func(t *testing.T) {
        derivedReq := simplecontent.CreateDerivedContentRequest{
            ParentID:       parent.ID,
            OwnerID:        parent.OwnerID,
            TenantID:       parent.TenantID,
            DerivationType: "thumbnail",
            Variant:        "thumbnail_256",
            Metadata: map[string]interface{}{
                "width":  256,
                "height": 256,
                "format": "jpeg",
            },
        }

        derived, err := svc.CreateDerivedContent(ctx, derivedReq)
        assert.NoError(t, err)
        assert.NotNil(t, derived)
        assert.Equal(t, parent.OwnerID, derived.OwnerID)
        assert.Equal(t, parent.TenantID, derived.TenantID)
        assert.Equal(t, "thumbnail", derived.DerivationType)
        assert.Equal(t, string(simplecontent.ContentStatusCreated), derived.Status)
    })

    t.Run("CreateDerivedContentWithInvalidParent", func(t *testing.T) {
        derivedReq := simplecontent.CreateDerivedContentRequest{
            ParentID:       uuid.New(), // Non-existent parent
            OwnerID:        uuid.New(),
            TenantID:       uuid.New(),
            DerivationType: "thumbnail",
            Variant:        "thumbnail_256",
        }

        derived, err := svc.CreateDerivedContent(ctx, derivedReq)
        assert.Error(t, err)
        assert.Nil(t, derived)
    })
}

// Benchmark tests
func BenchmarkCreateContent(b *testing.B) {
	svc := setupBenchmarkService(b)
	ctx := context.Background()

	req := simplecontent.CreateContentRequest{
		OwnerID:     uuid.New(),
		TenantID:    uuid.New(),
		Name:        "Benchmark Content",
		Description: "Content for benchmarking",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.Name = fmt.Sprintf("Benchmark Content %d", i)
		_, err := svc.CreateContent(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUploadDownload(b *testing.B) {
	svc := setupBenchmarkService(b)
	ctx := context.Background()

	// Setup
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Benchmark Content",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(b, err)

	testData := strings.Repeat("test data ", 1000) // ~9KB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create object
		objReq := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "memory",
			Version:            i + 1,
		}
		object, err := svc.CreateObject(ctx, objReq)
		if err != nil {
			b.Fatal(err)
		}

		// Upload
		reader := strings.NewReader(testData)
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: object.ID,
			Reader:   reader,
		}
		err = svc.UploadObject(ctx, uploadReq)
		if err != nil {
			b.Fatal(err)
		}

		// Download
		downloadReader, err := svc.DownloadObject(ctx, object.ID)
		if err != nil {
			b.Fatal(err)
		}
		_, err = io.ReadAll(downloadReader)
		if err != nil {
			b.Fatal(err)
		}
		downloadReader.Close()
	}
}

// Helper interface that both *testing.T and *testing.B implement
type testingInterface interface {
	Helper()
	Errorf(format string, args ...interface{})
	FailNow()
}

// Helper function for benchmark tests
func setupBenchmarkService(b *testing.B) simplecontent.Service {
	repo := memory.New()
	store := memorystorage.New()
	eventSink := simplecontent.NewNoopEventSink()
	previewer := simplecontent.NewBasicImagePreviewer()

	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", store),
		simplecontent.WithEventSink(eventSink),
		simplecontent.WithPreviewer(previewer),
	)
	if err != nil {
		b.Fatal(err)
	}
	if svc == nil {
		b.Fatal("service is nil")
	}

	return svc
}

func TestGetContentURLs(t *testing.T) {
	ctx := context.Background()
	svc := setupTestService(t)

	// Create owner and tenant IDs
	ownerID := uuid.New()
	tenantID := uuid.New()

	// Test 1: Content with no objects - should return empty URLs but not error
	t.Run("ContentWithoutObjects", func(t *testing.T) {
		req := simplecontent.CreateContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
			Name:     "Test Content",
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		urls, err := svc.GetContentURLs(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, content.ID.String(), urls.ID)
		assert.Empty(t, urls.Download)
		assert.Empty(t, urls.Preview)
		assert.Empty(t, urls.Thumbnail)
		assert.Empty(t, urls.Thumbnails)
		assert.Empty(t, urls.Previews)
		assert.Empty(t, urls.Transcodes)
		assert.True(t, urls.Ready) // Content exists and is ready
	})

	// Test 2: Content with objects - should return download/preview URLs
	t.Run("ContentWithObjects", func(t *testing.T) {
		// Create content
		req := simplecontent.CreateContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
			Name:     "Test Content with Objects",
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		// Create object
		objReq := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "memory",
			Version:            1,
			ObjectKey:          "test-object-key",
		}
		object, err := svc.CreateObject(ctx, objReq)
		require.NoError(t, err)

		// Upload some content to the object
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: object.ID,
			Reader:   strings.NewReader("test content"),
			MimeType: "text/plain",
		}
		err = svc.UploadObject(ctx, uploadReq)
		require.NoError(t, err)

		// Get URLs
		urls, err := svc.GetContentURLs(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, content.ID.String(), urls.ID)
		// Memory backend doesn't generate URLs (returns empty strings)
		// This is expected behavior - it supports direct download only
		assert.True(t, urls.Ready)
	})

	// Test 3: Content with derived content (thumbnails)
	t.Run("ContentWithDerivedContent", func(t *testing.T) {
		// Create parent content
		req := simplecontent.CreateContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
			Name:     "Parent Content",
		}
		parent, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		// Create parent object
		objReq := simplecontent.CreateObjectRequest{
			ContentID:          parent.ID,
			StorageBackendName: "memory",
			Version:            1,
			ObjectKey:          "parent-object-key",
		}
		parentObject, err := svc.CreateObject(ctx, objReq)
		require.NoError(t, err)

		// Upload content
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: parentObject.ID,
			Reader:   strings.NewReader("parent content"),
			MimeType: "image/jpeg",
		}
		err = svc.UploadObject(ctx, uploadReq)
		require.NoError(t, err)

		// Create derived content (thumbnail)
		derivedReq := simplecontent.CreateDerivedContentRequest{
			ParentID:       parent.ID,
			OwnerID:        ownerID,
			TenantID:       tenantID,
			DerivationType: "thumbnail",
			Variant:        "thumbnail_256",
		}
		derivedContent, err := svc.CreateDerivedContent(ctx, derivedReq)
		require.NoError(t, err)

		// Create object for derived content
		derivedObjReq := simplecontent.CreateObjectRequest{
			ContentID:          derivedContent.ID,
			StorageBackendName: "memory",
			Version:            1,
			ObjectKey:          "thumbnail-256-key",
		}
		derivedObject, err := svc.CreateObject(ctx, derivedObjReq)
		require.NoError(t, err)

		// Upload thumbnail content
		thumbUploadReq := simplecontent.UploadObjectRequest{
			ObjectID: derivedObject.ID,
			Reader:   strings.NewReader("thumbnail content"),
			MimeType: "image/jpeg",
		}
		err = svc.UploadObject(ctx, thumbUploadReq)
		require.NoError(t, err)

		// Get URLs - should include both original and thumbnail URLs
		urls, err := svc.GetContentURLs(ctx, parent.ID)
		require.NoError(t, err)
		assert.Equal(t, parent.ID.String(), urls.ID)
		// Memory backend doesn't generate URLs, but structure should be correct
		assert.NotNil(t, urls.Thumbnails)
		assert.NotNil(t, urls.Previews)
		assert.NotNil(t, urls.Transcodes)
		assert.True(t, urls.Ready)
	})

	// Test 4: Non-existent content
	t.Run("NonExistentContent", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := svc.GetContentURLs(ctx, nonExistentID)
		assert.Error(t, err)
		// Should be a ContentError with not found
		var contentErr *simplecontent.ContentError
		assert.True(t, errors.As(err, &contentErr))
	})
}
