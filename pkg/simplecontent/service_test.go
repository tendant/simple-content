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

// setupTestServiceWithStorage returns both Service and StorageService interfaces
func setupTestServiceWithStorage(t *testing.T) (simplecontent.Service, simplecontent.StorageService) {
	svc := setupTestService(t)
	storageSvc, ok := svc.(simplecontent.StorageService)
	require.True(t, ok, "Service should implement StorageService interface")
	return svc, storageSvc
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


func TestObjectOperations(t *testing.T) {
	svc, storageSvc := setupTestServiceWithStorage(t)
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

		object, err := storageSvc.CreateObject(ctx, objReq)
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
		created, err := storageSvc.CreateObject(ctx, objReq)
		require.NoError(t, err)

		// Retrieve object
		retrieved, err := storageSvc.GetObject(ctx, created.ID)
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
			_, err := storageSvc.CreateObject(ctx, objReq)
			require.NoError(t, err)
		}

		// Get objects by content ID
		objects, err := storageSvc.GetObjectsByContentID(ctx, content.ID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(objects), 3) // At least 3, might be more from other tests
	})
}

func TestGetObjectsByContentID_ServiceInterface(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create a content
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Content for GetObjectsByContentID",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(t, err)

	// Upload content to create an object
	uploadReq := simplecontent.UploadContentRequest{
		OwnerID:            content.OwnerID,
		TenantID:           content.TenantID,
		Name:               content.Name,
		StorageBackendName: "memory",
		Reader:             strings.NewReader("test data"),
		FileName:           "test.txt",
	}
	uploadedContent, err := svc.UploadContent(ctx, uploadReq)
	require.NoError(t, err)

	// Test GetObjectsByContentID via Service interface
	objects, err := svc.GetObjectsByContentID(ctx, uploadedContent.ID)
	assert.NoError(t, err)
	assert.NotNil(t, objects)
	assert.Equal(t, 1, len(objects))
	assert.Equal(t, uploadedContent.ID, objects[0].ContentID)
}

func TestObjectUploadDownload(t *testing.T) {
	svc, storageSvc := setupTestServiceWithStorage(t)
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
	object, err := storageSvc.CreateObject(ctx, objReq)
	require.NoError(t, err)

	testData := "Hello, World! This is test data for upload/download."

	t.Run("UploadObject", func(t *testing.T) {
		reader := strings.NewReader(testData)
		req := simplecontent.UploadObjectRequest{
			ObjectID: object.ID,
			Reader:   reader,
		}
		err := storageSvc.UploadObject(ctx, req)
		assert.NoError(t, err)

		// Verify object status was updated
		updated, err := storageSvc.GetObject(ctx, object.ID)
		assert.NoError(t, err)
        assert.Equal(t, string(simplecontent.ObjectStatusUploaded), updated.Status)
	})

	t.Run("DownloadObject", func(t *testing.T) {
		reader, err := storageSvc.DownloadObject(ctx, object.ID)
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
		object2, err := storageSvc.CreateObject(ctx, objReq2)
		require.NoError(t, err)

		reader := strings.NewReader("Test data with metadata")
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: object2.ID,
			Reader:   reader,
			MimeType: "text/plain",
		}

		err = storageSvc.UploadObject(ctx, uploadReq)
		assert.NoError(t, err)
	})
}

func TestErrorHandling(t *testing.T) {
	svc, storageSvc := setupTestServiceWithStorage(t)
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

		object, err := storageSvc.CreateObject(ctx, objReq)
		assert.Error(t, err)
		assert.Nil(t, object)
	})

	t.Run("UploadToNonExistentObject", func(t *testing.T) {
		reader := strings.NewReader("test data")
		req := simplecontent.UploadObjectRequest{
			ObjectID: uuid.New(),
			Reader:   reader,
		}
		err := storageSvc.UploadObject(ctx, req)
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

	// Update parent status to "uploaded" so derived content can be created
	parent.Status = string(simplecontent.ContentStatusUploaded)
	err = svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{Content: parent})
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

func TestUploadDerivedContentStatus(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create and upload parent content
	parentReq := simplecontent.UploadContentRequest{
		OwnerID:            uuid.New(),
		TenantID:           uuid.New(),
		Name:               "Parent Image",
		DocumentType:       "image/jpeg",
		StorageBackendName: "memory",
		Reader:             strings.NewReader("original image data"),
		FileName:           "original.jpg",
	}
	parent, err := svc.UploadContent(ctx, parentReq)
	require.NoError(t, err)
	require.Equal(t, string(simplecontent.ContentStatusUploaded), parent.Status, "Parent should have 'uploaded' status")

	// Upload derived content (thumbnail)
	derivedReq := simplecontent.UploadDerivedContentRequest{
		ParentID:           parent.ID,
		DerivationType:     "thumbnail",
		Variant:            "thumbnail_256",
		StorageBackendName: "memory",
		Reader:             strings.NewReader("thumbnail data"),
		FileName:           "thumb.jpg",
	}
	derived, err := svc.UploadDerivedContent(ctx, derivedReq)
	require.NoError(t, err)
	require.NotNil(t, derived)

	// Verify derived content has "processed" status (not "uploaded")
	assert.Equal(t, string(simplecontent.ContentStatusProcessed), derived.Status,
		"Derived content should have 'processed' status after upload")
	assert.Equal(t, "thumbnail", derived.DerivationType)

	// Verify GetContentDetails marks it as ready
	details, err := svc.GetContentDetails(ctx, derived.ID)
	require.NoError(t, err)
	assert.True(t, details.Ready, "Derived content with 'processed' status should be marked ready")
}

func TestGetContentDetailsReadyLogic(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	t.Run("OriginalContentReady", func(t *testing.T) {
		// Upload original content
		req := simplecontent.UploadContentRequest{
			OwnerID:            uuid.New(),
			TenantID:           uuid.New(),
			Name:               "Original",
			StorageBackendName: "memory",
			Reader:             strings.NewReader("data"),
			FileName:           "file.txt",
		}
		content, err := svc.UploadContent(ctx, req)
		require.NoError(t, err)

		// Check details - should be ready with "uploaded" status
		details, err := svc.GetContentDetails(ctx, content.ID)
		require.NoError(t, err)
		assert.True(t, details.Ready, "Original content with 'uploaded' status should be ready")
	})

	t.Run("DerivedContentReady", func(t *testing.T) {
		// Create parent
		parentReq := simplecontent.UploadContentRequest{
			OwnerID:            uuid.New(),
			TenantID:           uuid.New(),
			Name:               "Parent",
			StorageBackendName: "memory",
			Reader:             strings.NewReader("parent data"),
			FileName:           "parent.jpg",
		}
		parent, err := svc.UploadContent(ctx, parentReq)
		require.NoError(t, err)

		// Upload derived content
		derivedReq := simplecontent.UploadDerivedContentRequest{
			ParentID:           parent.ID,
			DerivationType:     "thumbnail",
			Variant:            "thumbnail_128",
			StorageBackendName: "memory",
			Reader:             strings.NewReader("thumb data"),
			FileName:           "thumb.jpg",
		}
		derived, err := svc.UploadDerivedContent(ctx, derivedReq)
		require.NoError(t, err)

		// Check details - should be ready with "processed" status
		details, err := svc.GetContentDetails(ctx, derived.ID)
		require.NoError(t, err)
		assert.True(t, details.Ready, "Derived content with 'processed' status should be ready")
	})
}

func TestUploadObjectForContent(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	// Create parent content first
	parentContent, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Parent Content",
		DocumentType: "text/plain",
		Reader:       strings.NewReader("parent data"),
		FileName:     "parent.txt",
	})
	require.NoError(t, err)
	require.NotNil(t, parentContent)

	// Create derived content (without object/data)
	derivedContent, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
		ParentID:       parentContent.ID,
		OwnerID:        parentContent.OwnerID,
		TenantID:       parentContent.TenantID,
		DerivationType: "thumbnail",
		Variant:        "thumbnail_256",
		InitialStatus:  simplecontent.ContentStatusProcessing,
	})
	require.NoError(t, err)
	require.NotNil(t, derivedContent)
	assert.Equal(t, string(simplecontent.ContentStatusProcessing), derivedContent.Status,
		"Initial status should be 'processing'")

	// Upload object for existing content
	thumbnailData := strings.NewReader("thumbnail image data")
	object, err := svc.UploadObjectForContent(ctx, simplecontent.UploadObjectForContentRequest{
		ContentID: derivedContent.ID,
		Reader:    thumbnailData,
		FileName:  "thumb.jpg",
		MimeType:  "image/jpeg",
	})
	require.NoError(t, err)
	require.NotNil(t, object)

	// Verify object was created correctly
	assert.Equal(t, derivedContent.ID, object.ContentID)
	assert.Equal(t, string(simplecontent.ObjectStatusUploaded), object.Status)
	assert.Equal(t, "thumb.jpg", object.FileName)
	assert.Equal(t, "image/jpeg", object.ObjectType)

	// Verify object can be retrieved
	objects, err := svc.GetObjectsByContentID(ctx, derivedContent.ID)
	require.NoError(t, err)
	assert.Len(t, objects, 1)
	assert.Equal(t, object.ID, objects[0].ID)
}

func TestAsyncWorkflow(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	t.Run("CompleteAsyncWorkflow", func(t *testing.T) {
		// Step 1: Create parent content
		parentContent, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "Source Image",
			DocumentType: "image/png",
			Reader:       strings.NewReader("source image data"),
			FileName:     "source.png",
		})
		require.NoError(t, err)
		assert.Equal(t, string(simplecontent.ContentStatusUploaded), parentContent.Status)

		// Step 2: Create derived content placeholder (before processing)
		derivedContent, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
			ParentID:       parentContent.ID,
			OwnerID:        parentContent.OwnerID,
			TenantID:       parentContent.TenantID,
			DerivationType: "thumbnail",
			Variant:        "thumbnail_256",
			InitialStatus:  simplecontent.ContentStatusProcessing,
			Metadata: map[string]interface{}{
				"target_size": "256x256",
				"format":      "jpeg",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, string(simplecontent.ContentStatusProcessing), derivedContent.Status,
			"Derived content should start in 'processing' state")

		// Step 3: Simulate worker processing (generate thumbnail)
		// In real workflow, worker would:
		// - Download parent content
		// - Generate thumbnail
		// - Upload thumbnail data
		thumbnailData := strings.NewReader("generated thumbnail data")

		// Step 4: Upload object for the derived content
		object, err := svc.UploadObjectForContent(ctx, simplecontent.UploadObjectForContentRequest{
			ContentID: derivedContent.ID,
			Reader:    thumbnailData,
			FileName:  "thumb_256.jpg",
			MimeType:  "image/jpeg",
		})
		require.NoError(t, err)
		require.NotNil(t, object)
		assert.Equal(t, string(simplecontent.ObjectStatusUploaded), object.Status)

		// Step 5: Update content status to processed
		err = svc.UpdateContentStatus(ctx, derivedContent.ID, simplecontent.ContentStatusProcessed)
		require.NoError(t, err)

		// Step 6: Verify final state
		finalContent, err := svc.GetContent(ctx, derivedContent.ID)
		require.NoError(t, err)
		assert.Equal(t, string(simplecontent.ContentStatusProcessed), finalContent.Status,
			"Final status should be 'processed'")

		// Verify object is accessible
		objects, err := svc.GetObjectsByContentID(ctx, derivedContent.ID)
		require.NoError(t, err)
		assert.Len(t, objects, 1)
		assert.Equal(t, object.ID, objects[0].ID)

		// Verify content can be downloaded
		reader, err := svc.DownloadContent(ctx, derivedContent.ID)
		require.NoError(t, err)
		defer reader.Close()

		downloadedData, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "generated thumbnail data", string(downloadedData))
	})

	t.Run("AsyncWorkflowWithFailure", func(t *testing.T) {
		// Create parent content
		parentContent, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "Source for Failed Processing",
			DocumentType: "image/png",
			Reader:       strings.NewReader("source data"),
		})
		require.NoError(t, err)

		// Create derived content placeholder
		derivedContent, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
			ParentID:       parentContent.ID,
			OwnerID:        parentContent.OwnerID,
			TenantID:       parentContent.TenantID,
			DerivationType: "thumbnail",
			Variant:        "thumbnail_512",
			InitialStatus:  simplecontent.ContentStatusProcessing,
		})
		require.NoError(t, err)

		// Simulate processing failure - content remains in "processing" state
		// In real workflow, worker would detect failure and either:
		// - Leave status as "processing" for retry
		// - Store error metadata
		err = svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
			ContentID: derivedContent.ID,
			CustomMetadata: map[string]interface{}{
				"last_error":   "processing failed: invalid image format",
				"error_count":  1,
				"last_attempt": time.Now().Format(time.RFC3339),
			},
		})
		require.NoError(t, err)

		// Verify content is still in processing state
		content, err := svc.GetContent(ctx, derivedContent.ID)
		require.NoError(t, err)
		assert.Equal(t, string(simplecontent.ContentStatusProcessing), content.Status)

		// Worker can query for stuck content and retry
		processingContent, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusProcessing)
		require.NoError(t, err)
		assert.NotEmpty(t, processingContent)

		// Find our content in the processing list
		found := false
		for _, c := range processingContent {
			if c.ID == derivedContent.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Content should be in processing state for retry")
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
	storageSvc, ok := svc.(simplecontent.StorageService)
	if !ok {
		b.Fatal("Service should implement StorageService interface")
	}
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
		object, err := storageSvc.CreateObject(ctx, objReq)
		if err != nil {
			b.Fatal(err)
		}

		// Upload
		reader := strings.NewReader(testData)
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: object.ID,
			Reader:   reader,
		}
		err = storageSvc.UploadObject(ctx, uploadReq)
		if err != nil {
			b.Fatal(err)
		}

		// Download
		downloadReader, err := storageSvc.DownloadObject(ctx, object.ID)
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

func TestGetContentDetails(t *testing.T) {
	ctx := context.Background()
	svc, storageSvc := setupTestServiceWithStorage(t)

	// Create owner and tenant IDs
	ownerID := uuid.New()
	tenantID := uuid.New()

	// Test 1: Content with no objects - should return basic details
	t.Run("ContentWithoutObjects", func(t *testing.T) {
		req := simplecontent.CreateContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
			Name:     "Test Content",
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		details, err := svc.GetContentDetails(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, content.ID.String(), details.ID)
		assert.Empty(t, details.Download)
		assert.Empty(t, details.Preview)
		assert.Empty(t, details.Thumbnail)
		assert.Empty(t, details.Thumbnails)
		assert.Empty(t, details.Previews)
		assert.Empty(t, details.Transcodes)
		assert.False(t, details.Ready) // Created content is not ready until uploaded
		assert.Equal(t, content.CreatedAt, details.CreatedAt)
		assert.Equal(t, content.UpdatedAt, details.UpdatedAt)
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
		object, err := storageSvc.CreateObject(ctx, objReq)
		require.NoError(t, err)

		// Upload some content to the object
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: object.ID,
			Reader:   strings.NewReader("test content"),
			MimeType: "text/plain",
		}
		err = storageSvc.UploadObject(ctx, uploadReq)
		require.NoError(t, err)

		// Get details
		details, err := svc.GetContentDetails(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, content.ID.String(), details.ID)
		// Memory backend doesn't generate URLs (returns empty strings)
		// This is expected behavior - it supports direct download only
		// Note: Content status is still "created" because UploadObject doesn't update content status
		// Ready flag reflects the actual content status, not object upload state
		assert.False(t, details.Ready)
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
		parentObject, err := storageSvc.CreateObject(ctx, objReq)
		require.NoError(t, err)

		// Upload content
		uploadReq := simplecontent.UploadObjectRequest{
			ObjectID: parentObject.ID,
			Reader:   strings.NewReader("parent content"),
			MimeType: "image/jpeg",
		}
		err = storageSvc.UploadObject(ctx, uploadReq)
		require.NoError(t, err)

		// Update parent status to "uploaded" so derived content can be created
		parent.Status = string(simplecontent.ContentStatusUploaded)
		err = svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{Content: parent})
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
		derivedObject, err := storageSvc.CreateObject(ctx, derivedObjReq)
		require.NoError(t, err)

		// Upload thumbnail content
		thumbUploadReq := simplecontent.UploadObjectRequest{
			ObjectID: derivedObject.ID,
			Reader:   strings.NewReader("thumbnail content"),
			MimeType: "image/jpeg",
		}
		err = storageSvc.UploadObject(ctx, thumbUploadReq)
		require.NoError(t, err)

		// Get details - should include both original and thumbnail URLs
		details, err := svc.GetContentDetails(ctx, parent.ID)
		require.NoError(t, err)
		assert.Equal(t, parent.ID.String(), details.ID)
		// Memory backend doesn't generate URLs, but structure should be correct
		assert.NotNil(t, details.Thumbnails)
		assert.NotNil(t, details.Previews)
		assert.NotNil(t, details.Transcodes)
		// Both parent and derived content are still in "created" status (legacy API)
		// Ready flag is false because content status was never updated to "uploaded"
		assert.False(t, details.Ready)
	})

	// Test 4: Non-existent content
	t.Run("NonExistentContent", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := svc.GetContentDetails(ctx, nonExistentID)
		assert.Error(t, err)
		// Should be a ContentError with not found
		var contentErr *simplecontent.ContentError
		assert.True(t, errors.As(err, &contentErr))
	})

	// Test 5: Ready flag - created status (not ready)
	t.Run("ReadyFlag_CreatedStatus", func(t *testing.T) {
		req := simplecontent.CreateContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
			Name:     "Created Content",
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)
		require.Equal(t, string(simplecontent.ContentStatusCreated), content.Status)

		details, err := svc.GetContentDetails(ctx, content.ID)
		require.NoError(t, err)
		assert.False(t, details.Ready, "Created content should not be ready")
	})

	// Test 6: Ready flag - uploaded status (ready)
	t.Run("ReadyFlag_UploadedStatus", func(t *testing.T) {
		// Use UploadContent to create and upload in one step
		uploadReq := simplecontent.UploadContentRequest{
			OwnerID:      ownerID,
			TenantID:     tenantID,
			Name:         "Uploaded Content",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("test content"),
			FileName:     "test.txt",
		}
		content, err := svc.UploadContent(ctx, uploadReq)
		require.NoError(t, err)
		require.Equal(t, string(simplecontent.ContentStatusUploaded), content.Status)

		details, err := svc.GetContentDetails(ctx, content.ID)
		require.NoError(t, err)
		assert.True(t, details.Ready, "Uploaded content should be ready")
	})

	// Test 7: Ready flag - uploaded parent with created derived content (not ready)
	t.Run("ReadyFlag_UploadedParentWithCreatedDerived", func(t *testing.T) {
		// Create and upload parent content
		parentReq := simplecontent.UploadContentRequest{
			OwnerID:      ownerID,
			TenantID:     tenantID,
			Name:         "Parent Content",
			DocumentType: "image/jpeg",
			Reader:       strings.NewReader("parent image data"),
			FileName:     "parent.jpg",
		}
		parent, err := svc.UploadContent(ctx, parentReq)
		require.NoError(t, err)
		require.Equal(t, string(simplecontent.ContentStatusUploaded), parent.Status)

		// Create derived content but don't upload it yet
		derivedReq := simplecontent.CreateDerivedContentRequest{
			ParentID:       parent.ID,
			OwnerID:        ownerID,
			TenantID:       tenantID,
			DerivationType: "thumbnail",
			Variant:        "thumbnail_256",
		}
		_, err = svc.CreateDerivedContent(ctx, derivedReq)
		require.NoError(t, err)

		// Get parent details - should not be ready because derived content is not uploaded
		details, err := svc.GetContentDetails(ctx, parent.ID)
		require.NoError(t, err)
		assert.False(t, details.Ready, "Parent with non-uploaded derived content should not be ready")
	})

	// Test 8: Ready flag - uploaded parent with uploaded derived content (ready)
	t.Run("ReadyFlag_UploadedParentWithUploadedDerived", func(t *testing.T) {
		// Create and upload parent content
		parentReq := simplecontent.UploadContentRequest{
			OwnerID:      ownerID,
			TenantID:     tenantID,
			Name:         "Parent Content",
			DocumentType: "image/jpeg",
			Reader:       strings.NewReader("parent image data"),
			FileName:     "parent.jpg",
		}
		parent, err := svc.UploadContent(ctx, parentReq)
		require.NoError(t, err)

		// Create and upload derived content
		derivedReq := simplecontent.UploadDerivedContentRequest{
			ParentID:       parent.ID,
			OwnerID:        ownerID,
			TenantID:       tenantID,
			DerivationType: "thumbnail",
			Variant:        "thumbnail_256",
			Reader:         strings.NewReader("thumbnail data"),
			FileName:       "thumb.jpg",
		}
		_, err = svc.UploadDerivedContent(ctx, derivedReq)
		require.NoError(t, err)

		// Get parent details - should be ready now
		details, err := svc.GetContentDetails(ctx, parent.ID)
		require.NoError(t, err)
		assert.True(t, details.Ready, "Parent with uploaded derived content should be ready")
	})
}
