package memory_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func TestMemoryRepository_ContentOperations(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	t.Run("CreateContent", func(t *testing.T) {
		content := &simplecontent.Content{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			OwnerID:      uuid.New(),
			Name:         "Test Content",
			Description:  "A test content",
			DocumentType: "text/plain",
			Status:       string(simplecontent.ContentStatusCreated),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := repo.CreateContent(ctx, content)
		assert.NoError(t, err)
	})

	t.Run("GetContent", func(t *testing.T) {
		// Create content first
		content := &simplecontent.Content{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			OwnerID:     uuid.New(),
			Name:        "Test Content for Get",
			Description: "A test content for retrieval",
			Status:      string(simplecontent.ContentStatusCreated),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.CreateContent(ctx, content)
		require.NoError(t, err)

		// Retrieve it
		retrieved, err := repo.GetContent(ctx, content.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, content.ID, retrieved.ID)
		assert.Equal(t, content.Name, retrieved.Name)
		assert.Equal(t, content.Description, retrieved.Description)
	})

	t.Run("GetContent_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		content, err := repo.GetContent(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, content)
		assert.Equal(t, simplecontent.ErrContentNotFound, err)
	})

	t.Run("UpdateContent", func(t *testing.T) {
		// Create content
		content := &simplecontent.Content{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			OwnerID:     uuid.New(),
			Name:        "Original Name",
			Description: "Original Description",
			Status:      string(simplecontent.ContentStatusCreated),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.CreateContent(ctx, content)
		require.NoError(t, err)

		// Update content
		content.Name = "Updated Name"
		content.Description = "Updated Description"
		content.UpdatedAt = time.Now().Add(time.Hour)

		err = repo.UpdateContent(ctx, content)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.GetContent(ctx, content.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "Updated Description", updated.Description)
	})

	t.Run("DeleteContent", func(t *testing.T) {
		// Create content
		content := &simplecontent.Content{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			OwnerID:   uuid.New(),
			Name:      "Content to Delete",
			Status:    string(simplecontent.ContentStatusCreated),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.CreateContent(ctx, content)
		require.NoError(t, err)

		// Delete content
		err = repo.DeleteContent(ctx, content.ID)
		assert.NoError(t, err)

		// Verify deletion
		deleted, err := repo.GetContent(ctx, content.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
		assert.Equal(t, simplecontent.ErrContentNotFound, err)
	})

	t.Run("ListContent", func(t *testing.T) {
		ownerID := uuid.New()
		tenantID := uuid.New()

		// Create multiple contents
		var createdContents []*simplecontent.Content
		for i := 0; i < 3; i++ {
			content := &simplecontent.Content{
				ID:        uuid.New(),
				TenantID:  tenantID,
				OwnerID:   ownerID,
				Name:      fmt.Sprintf("List Test Content %d", i+1),
				Status:    string(simplecontent.ContentStatusCreated),
				CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
				UpdatedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			err := repo.CreateContent(ctx, content)
			require.NoError(t, err)
			createdContents = append(createdContents, content)
		}

		// List contents
		contents, err := repo.ListContent(ctx, ownerID, tenantID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(contents), 3)

		// Verify they're sorted by creation time (newest first)
		for i := 0; i < len(contents)-1; i++ {
			assert.True(t, contents[i].CreatedAt.After(contents[i+1].CreatedAt) ||
				contents[i].CreatedAt.Equal(contents[i+1].CreatedAt))
		}
	})
}

func TestMemoryRepository_ContentMetadataOperations(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	// Create content first
	content := &simplecontent.Content{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		OwnerID:   uuid.New(),
		Name:      "Content with Metadata",
		Status:    string(simplecontent.ContentStatusCreated),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.CreateContent(ctx, content)
	require.NoError(t, err)

	t.Run("SetContentMetadata", func(t *testing.T) {
		metadata := &simplecontent.ContentMetadata{
			ContentID: content.ID,
			Tags:      []string{"test", "metadata"},
			FileSize:  1024,
			FileName:  "test.txt",
			MimeType:  "text/plain",
			Metadata: map[string]interface{}{
				"category": "testing",
				"priority": "high",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.SetContentMetadata(ctx, metadata)
		assert.NoError(t, err)
	})

	t.Run("GetContentMetadata", func(t *testing.T) {
		metadata, err := repo.GetContentMetadata(ctx, content.ID)
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, content.ID, metadata.ContentID)
		assert.Equal(t, []string{"test", "metadata"}, metadata.Tags)
		assert.Equal(t, int64(1024), metadata.FileSize)
		assert.Equal(t, "test.txt", metadata.FileName)
		assert.Equal(t, "text/plain", metadata.MimeType)
		assert.Contains(t, metadata.Metadata, "category")
		assert.Contains(t, metadata.Metadata, "priority")
	})

	t.Run("SetContentMetadata_ContentNotFound", func(t *testing.T) {
		metadata := &simplecontent.ContentMetadata{
			ContentID: uuid.New(), // Non-existent content
			Tags:      []string{"test"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.SetContentMetadata(ctx, metadata)
		assert.Error(t, err)
		assert.Equal(t, simplecontent.ErrContentNotFound, err)
	})
}

func TestMemoryRepository_ObjectOperations(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	// Create content first
	content := &simplecontent.Content{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		OwnerID:   uuid.New(),
		Name:      "Content for Objects",
		Status:    string(simplecontent.ContentStatusCreated),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.CreateContent(ctx, content)
	require.NoError(t, err)

	t.Run("CreateObject", func(t *testing.T) {
		object := &simplecontent.Object{
			ID:                 uuid.New(),
			ContentID:          content.ID,
			StorageBackendName: "memory",
			ObjectKey:          "test/object/key",
			Version:            1,
			Status:             string(simplecontent.ObjectStatusCreated),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		err := repo.CreateObject(ctx, object)
		assert.NoError(t, err)
	})

	t.Run("GetObject", func(t *testing.T) {
		// Create object first
		object := &simplecontent.Object{
			ID:                 uuid.New(),
			ContentID:          content.ID,
			StorageBackendName: "memory",
			ObjectKey:          "test/object/get",
			Version:            1,
			Status:             string(simplecontent.ObjectStatusCreated),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		err := repo.CreateObject(ctx, object)
		require.NoError(t, err)

		// Retrieve it
		retrieved, err := repo.GetObject(ctx, object.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, object.ID, retrieved.ID)
		assert.Equal(t, object.ContentID, retrieved.ContentID)
		assert.Equal(t, object.StorageBackendName, retrieved.StorageBackendName)
	})

	t.Run("GetObjectsByContentID", func(t *testing.T) {
		// Create multiple objects for the same content
		var createdObjects []*simplecontent.Object
		for i := 0; i < 3; i++ {
			object := &simplecontent.Object{
				ID:                 uuid.New(),
				ContentID:          content.ID,
				StorageBackendName: "memory",
				ObjectKey:          fmt.Sprintf("test/object/list/%d", i),
				Version:            i + 2,
				Status:             string(simplecontent.ObjectStatusCreated),
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			err := repo.CreateObject(ctx, object)
			require.NoError(t, err)
			createdObjects = append(createdObjects, object)
		}

		// Get objects by content ID
		objects, err := repo.GetObjectsByContentID(ctx, content.ID)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(objects), 3)

		// Verify they're sorted by version (descending)
		for i := 0; i < len(objects)-1; i++ {
			assert.GreaterOrEqual(t, objects[i].Version, objects[i+1].Version)
		}
	})

	t.Run("GetObjectByObjectKeyAndStorageBackendName", func(t *testing.T) {
		// Create object
		objectKey := "test/object/bykey"
		backendName := "memory"
		object := &simplecontent.Object{
			ID:                 uuid.New(),
			ContentID:          content.ID,
			StorageBackendName: backendName,
			ObjectKey:          objectKey,
			Version:            1,
			Status:             string(simplecontent.ObjectStatusCreated),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		err := repo.CreateObject(ctx, object)
		require.NoError(t, err)

		// Retrieve by key and backend
		retrieved, err := repo.GetObjectByObjectKeyAndStorageBackendName(ctx, objectKey, backendName)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, object.ID, retrieved.ID)
		assert.Equal(t, objectKey, retrieved.ObjectKey)
		assert.Equal(t, backendName, retrieved.StorageBackendName)
	})

	t.Run("UpdateObject", func(t *testing.T) {
		// Create object
		object := &simplecontent.Object{
			ID:                 uuid.New(),
			ContentID:          content.ID,
			StorageBackendName: "memory",
			ObjectKey:          "test/object/update",
			Version:            1,
			Status:             string(simplecontent.ObjectStatusCreated),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		err := repo.CreateObject(ctx, object)
		require.NoError(t, err)

		// Update object
		object.Status = string(simplecontent.ObjectStatusUploaded)
		object.UpdatedAt = time.Now().Add(time.Hour)

		err = repo.UpdateObject(ctx, object)
		assert.NoError(t, err)

		// Verify update
		updated, err := repo.GetObject(ctx, object.ID)
		assert.NoError(t, err)
		assert.Equal(t, string(simplecontent.ObjectStatusUploaded), updated.Status)
	})

	t.Run("DeleteObject", func(t *testing.T) {
		// Create object
		object := &simplecontent.Object{
			ID:                 uuid.New(),
			ContentID:          content.ID,
			StorageBackendName: "memory",
			ObjectKey:          "test/object/delete",
			Version:            1,
			Status:             string(simplecontent.ObjectStatusCreated),
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
		err := repo.CreateObject(ctx, object)
		require.NoError(t, err)

		// Delete object
		err = repo.DeleteObject(ctx, object.ID)
		assert.NoError(t, err)

		// Verify deletion
		deleted, err := repo.GetObject(ctx, object.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
		assert.Equal(t, simplecontent.ErrObjectNotFound, err)
	})
}

func TestMemoryRepository_ObjectMetadataOperations(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	// Create content and object first
	content := &simplecontent.Content{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		OwnerID:   uuid.New(),
		Name:      "Content for Object Metadata",
		Status:    string(simplecontent.ContentStatusCreated),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.CreateContent(ctx, content)
	require.NoError(t, err)

	object := &simplecontent.Object{
		ID:                 uuid.New(),
		ContentID:          content.ID,
		StorageBackendName: "memory",
		ObjectKey:          "test/object/metadata",
		Version:            1,
		Status:             string(simplecontent.ObjectStatusCreated),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = repo.CreateObject(ctx, object)
	require.NoError(t, err)

	t.Run("SetObjectMetadata", func(t *testing.T) {
		metadata := &simplecontent.ObjectMetadata{
			ObjectID:  object.ID,
			SizeBytes: 2048,
			MimeType:  "application/json",
			ETag:      "etag123",
			Metadata: map[string]interface{}{
				"custom_field": "custom_value",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.SetObjectMetadata(ctx, metadata)
		assert.NoError(t, err)
	})

	t.Run("GetObjectMetadata", func(t *testing.T) {
		metadata, err := repo.GetObjectMetadata(ctx, object.ID)
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, object.ID, metadata.ObjectID)
		assert.Equal(t, int64(2048), metadata.SizeBytes)
		assert.Equal(t, "application/json", metadata.MimeType)
		assert.Equal(t, "etag123", metadata.ETag)
		assert.Contains(t, metadata.Metadata, "custom_field")
	})

	t.Run("SetObjectMetadata_ObjectNotFound", func(t *testing.T) {
		metadata := &simplecontent.ObjectMetadata{
			ObjectID:  uuid.New(), // Non-existent object
			SizeBytes: 1024,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.SetObjectMetadata(ctx, metadata)
		assert.Error(t, err)
		assert.Equal(t, simplecontent.ErrObjectNotFound, err)
	})
}

func TestMemoryRepository_DerivedContentOperations(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	// Create parent and derived content
	parent := &simplecontent.Content{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		OwnerID:   uuid.New(),
		Name:      "Parent Content",
		Status:    string(simplecontent.ContentStatusCreated),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.CreateContent(ctx, parent)
	require.NoError(t, err)

	derived := &simplecontent.Content{
		ID:        uuid.New(),
		TenantID:  parent.TenantID,
		OwnerID:   parent.OwnerID,
		Name:      "Derived Content",
		Status:    string(simplecontent.ContentStatusCreated),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = repo.CreateContent(ctx, derived)
	require.NoError(t, err)

	t.Run("CreateDerivedContentRelationship", func(t *testing.T) {
		params := simplecontent.CreateDerivedContentParams{
			ParentID:         parent.ID,
			DerivedContentID: derived.ID,
			DerivationType:   "thumbnail_256",
			DerivationParams: map[string]interface{}{
				"width":  256,
				"height": 256,
			},
			ProcessingMetadata: map[string]interface{}{
				"processor": "imagemagick",
			},
		}

		result, err := repo.CreateDerivedContentRelationship(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, parent.ID, result.ParentID)
		assert.Equal(t, derived.ID, result.ContentID)
		assert.Equal(t, "thumbnail_256", result.DerivationType)
	})

	t.Run("ListDerivedContent", func(t *testing.T) {
		params := simplecontent.ListDerivedContentParams{
			ParentID: &parent.ID,
		}

		results, err := repo.ListDerivedContent(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, parent.ID, results[0].ParentID)
		assert.Equal(t, derived.ID, results[0].ContentID)
	})

	t.Run("ListDerivedContent_WithFilters", func(t *testing.T) {
		derivationType := "thumbnail_256"
		limit := 10
		offset := 0

		params := simplecontent.ListDerivedContentParams{
			ParentID:       &parent.ID,
			DerivationType: &derivationType,
			Limit:          &limit,
			Offset:         &offset,
		}

		results, err := repo.ListDerivedContent(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, derivationType, results[0].DerivationType)
	})
}

func TestMemoryRepository_BatchOperations(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	t.Run("GetContentsByIDs", func(t *testing.T) {
		// Create multiple contents
		var contentIDs []uuid.UUID
		for i := 0; i < 5; i++ {
			content := &simplecontent.Content{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				OwnerID:   uuid.New(),
				Name:      fmt.Sprintf("Batch Content %d", i),
				Status:    string(simplecontent.ContentStatusCreated),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := repo.CreateContent(ctx, content)
			require.NoError(t, err)
			contentIDs = append(contentIDs, content.ID)
		}

		// Get all contents by IDs
		contents, err := repo.GetContentsByIDs(ctx, contentIDs)
		assert.NoError(t, err)
		assert.Len(t, contents, 5)

		// Verify all IDs are present
		foundIDs := make(map[uuid.UUID]bool)
		for _, content := range contents {
			foundIDs[content.ID] = true
		}
		for _, id := range contentIDs {
			assert.True(t, foundIDs[id], "Expected to find content ID %s", id)
		}
	})

	t.Run("GetContentsByIDs_EmptyArray", func(t *testing.T) {
		contents, err := repo.GetContentsByIDs(ctx, []uuid.UUID{})
		assert.NoError(t, err)
		assert.Empty(t, contents)
	})

	t.Run("GetContentsByIDs_PartialMatch", func(t *testing.T) {
		// Create one content
		content := &simplecontent.Content{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			OwnerID:   uuid.New(),
			Name:      "Partial Match Content",
			Status:    string(simplecontent.ContentStatusCreated),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.CreateContent(ctx, content)
		require.NoError(t, err)

		// Query with one existing and one non-existing ID
		ids := []uuid.UUID{content.ID, uuid.New()}
		contents, err := repo.GetContentsByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, contents, 1)
		assert.Equal(t, content.ID, contents[0].ID)
	})

	t.Run("GetContentsByIDs_ExcludesDeleted", func(t *testing.T) {
		// Create and delete one content
		deletedContent := &simplecontent.Content{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			OwnerID:   uuid.New(),
			Name:      "Deleted Content",
			Status:    string(simplecontent.ContentStatusCreated),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.CreateContent(ctx, deletedContent)
		require.NoError(t, err)
		err = repo.DeleteContent(ctx, deletedContent.ID)
		require.NoError(t, err)

		// Create an active content
		activeContent := &simplecontent.Content{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			OwnerID:   uuid.New(),
			Name:      "Active Content",
			Status:    string(simplecontent.ContentStatusCreated),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = repo.CreateContent(ctx, activeContent)
		require.NoError(t, err)

		// Query both
		ids := []uuid.UUID{deletedContent.ID, activeContent.ID}
		contents, err := repo.GetContentsByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, contents, 1)
		assert.Equal(t, activeContent.ID, contents[0].ID)
	})

	t.Run("GetContentMetadataByContentIDs", func(t *testing.T) {
		// Create contents with metadata
		var contentIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			content := &simplecontent.Content{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				OwnerID:   uuid.New(),
				Name:      fmt.Sprintf("Content with Metadata %d", i),
				Status:    string(simplecontent.ContentStatusCreated),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := repo.CreateContent(ctx, content)
			require.NoError(t, err)

			metadata := &simplecontent.ContentMetadata{
				ContentID: content.ID,
				FileName:  fmt.Sprintf("file%d.txt", i),
				FileSize:  int64((i + 1) * 1024),
				MimeType:  "text/plain",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = repo.SetContentMetadata(ctx, metadata)
			require.NoError(t, err)

			contentIDs = append(contentIDs, content.ID)
		}

		// Get all metadata by content IDs
		metadataMap, err := repo.GetContentMetadataByContentIDs(ctx, contentIDs)
		assert.NoError(t, err)
		assert.Len(t, metadataMap, 3)

		// Verify all metadata is present
		for _, contentID := range contentIDs {
			assert.Contains(t, metadataMap, contentID)
			assert.NotNil(t, metadataMap[contentID])
		}
	})

	t.Run("GetObjectsByContentIDs", func(t *testing.T) {
		// Create contents and objects
		var contentIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			content := &simplecontent.Content{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				OwnerID:   uuid.New(),
				Name:      fmt.Sprintf("Content for Batch Objects %d", i),
				Status:    string(simplecontent.ContentStatusCreated),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := repo.CreateContent(ctx, content)
			require.NoError(t, err)
			contentIDs = append(contentIDs, content.ID)

			// Create 2 objects per content
			for j := 0; j < 2; j++ {
				object := &simplecontent.Object{
					ID:                 uuid.New(),
					ContentID:          content.ID,
					StorageBackendName: "memory",
					ObjectKey:          fmt.Sprintf("batch/object/%d/%d", i, j),
					Version:            j + 1,
					Status:             string(simplecontent.ObjectStatusCreated),
					CreatedAt:          time.Now(),
					UpdatedAt:          time.Now(),
				}
				err = repo.CreateObject(ctx, object)
				require.NoError(t, err)
			}
		}

		// Get all objects by content IDs
		objectsMap, err := repo.GetObjectsByContentIDs(ctx, contentIDs)
		assert.NoError(t, err)
		assert.Len(t, objectsMap, 3)

		// Verify each content has 2 objects
		for _, contentID := range contentIDs {
			assert.Contains(t, objectsMap, contentID)
			assert.Len(t, objectsMap[contentID], 2)

			// Verify objects are sorted by version descending
			objects := objectsMap[contentID]
			for k := 0; k < len(objects)-1; k++ {
				assert.GreaterOrEqual(t, objects[k].Version, objects[k+1].Version)
			}
		}
	})

	t.Run("GetObjectMetadataByObjectIDs", func(t *testing.T) {
		// Create content and objects with metadata
		content := &simplecontent.Content{
			ID:        uuid.New(),
			TenantID:  uuid.New(),
			OwnerID:   uuid.New(),
			Name:      "Content for Object Metadata Batch",
			Status:    string(simplecontent.ContentStatusCreated),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.CreateContent(ctx, content)
		require.NoError(t, err)

		var objectIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			object := &simplecontent.Object{
				ID:                 uuid.New(),
				ContentID:          content.ID,
				StorageBackendName: "memory",
				ObjectKey:          fmt.Sprintf("batch/metadata/%d", i),
				Version:            i + 1,
				Status:             string(simplecontent.ObjectStatusCreated),
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			err = repo.CreateObject(ctx, object)
			require.NoError(t, err)

			metadata := &simplecontent.ObjectMetadata{
				ObjectID:  object.ID,
				SizeBytes: int64((i + 1) * 2048),
				MimeType:  "application/octet-stream",
				ETag:      fmt.Sprintf("etag%d", i),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = repo.SetObjectMetadata(ctx, metadata)
			require.NoError(t, err)

			objectIDs = append(objectIDs, object.ID)
		}

		// Get all object metadata by object IDs
		metadataMap, err := repo.GetObjectMetadataByObjectIDs(ctx, objectIDs)
		assert.NoError(t, err)
		assert.Len(t, metadataMap, 3)

		// Verify all metadata is present
		for _, objectID := range objectIDs {
			assert.Contains(t, metadataMap, objectID)
			assert.NotNil(t, metadataMap[objectID])
		}
	})
}

func TestMemoryRepositoryConcurrency(t *testing.T) {
	repo := memory.New()
	ctx := context.Background()

	const numGoroutines = 10
	const numOperations = 50

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				// Create content
				content := &simplecontent.Content{
					ID:        uuid.New(),
					TenantID:  uuid.New(),
					OwnerID:   uuid.New(),
					Name:      fmt.Sprintf("Concurrent Content %d-%d", goroutineID, j),
					Status:    string(simplecontent.ContentStatusCreated),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := repo.CreateContent(ctx, content)
				require.NoError(t, err)

				// Create object
				object := &simplecontent.Object{
					ID:                 uuid.New(),
					ContentID:          content.ID,
					StorageBackendName: "memory",
					ObjectKey:          fmt.Sprintf("concurrent/object/%d-%d", goroutineID, j),
					Version:            1,
					Status:             string(simplecontent.ObjectStatusCreated),
					CreatedAt:          time.Now(),
					UpdatedAt:          time.Now(),
				}
				err = repo.CreateObject(ctx, object)
				require.NoError(t, err)

				// Update and retrieve
				retrieved, err := repo.GetContent(ctx, content.ID)
				require.NoError(t, err)
				assert.Equal(t, content.Name, retrieved.Name)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
