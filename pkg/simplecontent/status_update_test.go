package simplecontent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	simplecontent "github.com/tendant/simple-content/pkg/simplecontent"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func setupStatusTestService(t *testing.T) simplecontent.Service {
	repo := memoryrepo.New()
	memBackend := memorystorage.New()

	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	require.NoError(t, err)

	return svc
}

// TestUpdateContentStatus tests the UpdateContentStatus method
func TestUpdateContentStatus(t *testing.T) {
	svc := setupStatusTestService(t)
	ctx := context.Background()

	// Create content with initial status
	req := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Content",
	}
	content, err := svc.CreateContent(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, string(simplecontent.ContentStatusCreated), content.Status)

	t.Run("ValidStatusUpdate", func(t *testing.T) {
		// Update to uploaded status
		err := svc.UpdateContentStatus(ctx, content.ID, simplecontent.ContentStatusUploaded)
		assert.NoError(t, err)

		// Verify status changed
		updated, err := svc.GetContent(ctx, content.ID)
		require.NoError(t, err)
		assert.Equal(t, string(simplecontent.ContentStatusUploaded), updated.Status)
		assert.True(t, updated.UpdatedAt.After(content.UpdatedAt))
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		// Try to update to invalid status
		err := svc.UpdateContentStatus(ctx, content.ID, simplecontent.ContentStatus("invalid"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, simplecontent.ErrInvalidContentStatus))
	})

	t.Run("NonExistentContent", func(t *testing.T) {
		// Try to update non-existent content
		err := svc.UpdateContentStatus(ctx, uuid.New(), simplecontent.ContentStatusProcessed)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, simplecontent.ErrContentNotFound))
	})
}

// TestUpdateObjectStatus tests the UpdateObjectStatus method
func TestUpdateObjectStatus(t *testing.T) {
	svc := setupStatusTestService(t)
	storageSvc := svc.(simplecontent.StorageService)
	ctx := context.Background()

	// Create content and object
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Content",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(t, err)

	objectReq := simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "memory",
		ObjectKey:          "test-key",
		Version:            1,
	}
	object, err := storageSvc.CreateObject(ctx, objectReq)
	require.NoError(t, err)
	assert.Equal(t, string(simplecontent.ObjectStatusCreated), object.Status)

	t.Run("ValidStatusUpdate", func(t *testing.T) {
		// Update to uploaded status
		err := svc.UpdateObjectStatus(ctx, object.ID, simplecontent.ObjectStatusUploaded)
		assert.NoError(t, err)

		// Verify status changed
		updated, err := storageSvc.GetObject(ctx, object.ID)
		require.NoError(t, err)
		assert.Equal(t, string(simplecontent.ObjectStatusUploaded), updated.Status)
		assert.True(t, updated.UpdatedAt.After(object.UpdatedAt))
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		// Try to update to invalid status
		err := svc.UpdateObjectStatus(ctx, object.ID, simplecontent.ObjectStatus("invalid"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, simplecontent.ErrInvalidObjectStatus))
	})

	t.Run("NonExistentObject", func(t *testing.T) {
		// Try to update non-existent object
		err := svc.UpdateObjectStatus(ctx, uuid.New(), simplecontent.ObjectStatusProcessed)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, simplecontent.ErrObjectNotFound))
	})
}

// TestGetContentByStatus tests the GetContentByStatus method
func TestGetContentByStatus(t *testing.T) {
	svc := setupStatusTestService(t)
	ctx := context.Background()

	ownerID := uuid.New()
	tenantID := uuid.New()

	// Create multiple content items with different statuses
	createContent := func(name string, status simplecontent.ContentStatus) *simplecontent.Content {
		req := simplecontent.CreateContentRequest{
			OwnerID:  ownerID,
			TenantID: tenantID,
			Name:     name,
		}
		content, err := svc.CreateContent(ctx, req)
		require.NoError(t, err)

		if status != simplecontent.ContentStatusCreated {
			err = svc.UpdateContentStatus(ctx, content.ID, status)
			require.NoError(t, err)
		}
		return content
	}

	created1 := createContent("Created 1", simplecontent.ContentStatusCreated)
	created2 := createContent("Created 2", simplecontent.ContentStatusCreated)
	uploaded1 := createContent("Uploaded 1", simplecontent.ContentStatusUploaded)
	_ = createContent("Processed 1", simplecontent.ContentStatusProcessed)

	t.Run("FindCreatedContent", func(t *testing.T) {
		results, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusCreated)
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		ids := []uuid.UUID{results[0].ID, results[1].ID}
		assert.Contains(t, ids, created1.ID)
		assert.Contains(t, ids, created2.ID)
	})

	t.Run("FindUploadedContent", func(t *testing.T) {
		results, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusUploaded)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, uploaded1.ID, results[0].ID)
	})

	t.Run("NoMatches", func(t *testing.T) {
		results, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusFailed)
		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		_, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatus("invalid"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, simplecontent.ErrInvalidContentStatus))
	})

	t.Run("ExcludesDeletedContent", func(t *testing.T) {
		// Delete one of the created content
		err := svc.DeleteContent(ctx, created1.ID)
		require.NoError(t, err)

		// Should now only find one created content
		results, err := svc.GetContentByStatus(ctx, simplecontent.ContentStatusCreated)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, created2.ID, results[0].ID)
	})
}

// TestGetObjectsByStatus tests the GetObjectsByStatus method
func TestGetObjectsByStatus(t *testing.T) {
	svc := setupStatusTestService(t)
	storageSvc := svc.(simplecontent.StorageService)
	ctx := context.Background()

	// Create content for objects
	contentReq := simplecontent.CreateContentRequest{
		OwnerID:  uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Content",
	}
	content, err := svc.CreateContent(ctx, contentReq)
	require.NoError(t, err)

	// Create multiple objects with different statuses
	createObject := func(key string, status simplecontent.ObjectStatus) *simplecontent.Object {
		objectReq := simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "memory",
			ObjectKey:          key,
			Version:            1,
		}
		object, err := storageSvc.CreateObject(ctx, objectReq)
		require.NoError(t, err)

		if status != simplecontent.ObjectStatusCreated {
			err = svc.UpdateObjectStatus(ctx, object.ID, status)
			require.NoError(t, err)
		}
		return object
	}

	created1 := createObject("obj-created-1", simplecontent.ObjectStatusCreated)
	created2 := createObject("obj-created-2", simplecontent.ObjectStatusCreated)
	uploaded1 := createObject("obj-uploaded-1", simplecontent.ObjectStatusUploaded)
	_ = createObject("obj-processed-1", simplecontent.ObjectStatusProcessed)

	t.Run("FindCreatedObjects", func(t *testing.T) {
		results, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatusCreated)
		assert.NoError(t, err)
		assert.Len(t, results, 2)

		ids := []uuid.UUID{results[0].ID, results[1].ID}
		assert.Contains(t, ids, created1.ID)
		assert.Contains(t, ids, created2.ID)
	})

	t.Run("FindUploadedObjects", func(t *testing.T) {
		results, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatusUploaded)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, uploaded1.ID, results[0].ID)
	})

	t.Run("NoMatches", func(t *testing.T) {
		results, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatusFailed)
		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("InvalidStatus", func(t *testing.T) {
		_, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatus("invalid"))
		assert.Error(t, err)
		assert.True(t, errors.Is(err, simplecontent.ErrInvalidObjectStatus))
	})

	t.Run("ExcludesDeletedObjects", func(t *testing.T) {
		// Delete one of the created objects
		err := storageSvc.DeleteObject(ctx, created1.ID)
		require.NoError(t, err)

		// Should now only find one created object
		results, err := svc.GetObjectsByStatus(ctx, simplecontent.ObjectStatusCreated)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, created2.ID, results[0].ID)
	})
}
