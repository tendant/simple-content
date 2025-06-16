package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/pkg/repository/memory"
	"github.com/tendant/simple-content/pkg/service"
	memorystorage "github.com/tendant/simple-content/pkg/storage/memory"
)

func setupObjectService() (*service.ObjectService, *memorystorage.MemoryBackend) {
	objectRepo := memory.NewObjectRepository()
	objectMetadataRepo := memory.NewObjectMetadataRepository()
	contentRepo := memory.NewContentRepository()
	contentMetadataRepo := memory.NewContentMetadataRepository()
	backend := memorystorage.NewMemoryBackend().(*memorystorage.MemoryBackend)

	service := service.NewObjectService(objectRepo, objectMetadataRepo, contentRepo, contentMetadataRepo)
	service.RegisterBackend("memory", backend)
	return service, backend
}

func TestObjectService_CreateAndGetObject(t *testing.T) {
	svc, _ := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()
	version := 1

	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            version,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)
	assert.NotNil(t, object)
	assert.Equal(t, contentID, object.ContentID)
	assert.Equal(t, "memory", object.StorageBackendName)
	assert.Equal(t, version, object.Version)

	fetched, err := svc.GetObject(ctx, object.ID)
	assert.NoError(t, err)
	assert.Equal(t, object.ID, fetched.ID)
}

func TestObjectService_UpdateObject(t *testing.T) {
	svc, _ := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)

	object.Version = 2
	err = svc.UpdateObject(ctx, object)
	assert.NoError(t, err)

	updated, err := svc.GetObject(ctx, object.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, updated.Version)
}

func TestObjectService_DeleteObject(t *testing.T) {
	svc, backend := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)

	// Upload some data to storage
	backend.Upload(ctx, object.ObjectKey, bytes.NewReader([]byte("test data")))

	err = svc.DeleteObject(ctx, object.ID)
	assert.NoError(t, err)

	_, err = svc.GetObject(ctx, object.ID)
	assert.Error(t, err)
}

func TestObjectService_UploadAndDownloadObject(t *testing.T) {
	svc, _ := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)

	data := []byte("hello world")
	err = svc.UploadObject(ctx, object.ID, bytes.NewReader(data))
	assert.NoError(t, err)

	reader, err := svc.DownloadObject(ctx, object.ID)
	assert.NoError(t, err)
	readData, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, data, readData)
}

func TestObjectService_SetAndGetObjectMetadata(t *testing.T) {
	svc, _ := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)

	// Test with custom metadata fields
	customMeta := map[string]interface{}{
		"custom_field":  "custom_value",
		"another_field": 123,
	}
	err = svc.SetObjectMetadata(ctx, object.ID, customMeta)
	assert.NoError(t, err)

	// Retrieve and verify custom metadata
	metadata, err := svc.GetObjectMetadata(ctx, object.ID)
	assert.NoError(t, err)
	assert.Equal(t, "custom_value", metadata["custom_field"])
	assert.Equal(t, 123, metadata["another_field"])

	// Now test with standard metadata fields that should be extracted to specific struct fields
	standardMeta := map[string]interface{}{
		"etag":       "abc123",
		"size_bytes": int64(42),
		"mime_type":  "text/plain",
	}
	err = svc.SetObjectMetadata(ctx, object.ID, standardMeta)
	assert.NoError(t, err)

	// Retrieve and verify the metadata was set correctly
	metadata, err = svc.GetObjectMetadata(ctx, object.ID)
	assert.NoError(t, err)

	// Verify the standard metadata fields were set correctly
	assert.Equal(t, "abc123", metadata["etag"])
	assert.Equal(t, int64(42), metadata["size_bytes"])
	assert.Equal(t, "text/plain", metadata["mime_type"])

	// Also verify the custom fields are still there
	assert.Equal(t, "custom_value", metadata["custom_field"])
	assert.Equal(t, 123, metadata["another_field"])
}

func TestObjectService_GetObjectsByContentID(t *testing.T) {
	svc, _ := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()

	// Create multiple objects for the same content
	createObjectParams1 := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object1, err := svc.CreateObject(ctx, createObjectParams1)
	assert.NoError(t, err)

	createObjectParams2 := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            2,
	}
	object2, err := svc.CreateObject(ctx, createObjectParams2)
	assert.NoError(t, err)

	// Get objects by content ID
	objects, err := svc.GetObjectsByContentID(ctx, contentID)
	assert.NoError(t, err)
	assert.Len(t, objects, 2)

	// Verify the objects are the ones we created
	ids := []uuid.UUID{objects[0].ID, objects[1].ID}
	assert.Contains(t, ids, object1.ID)
	assert.Contains(t, ids, object2.ID)
}

func TestObjectService_GetObjectByObjectKeyAndStorageBackendName(t *testing.T) {
	svc, _ := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()

	// Create an object with a specific object key
	uniqueName := "memory"

	// Create the object first
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: uniqueName,
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)

	// Create another object with different key but same storage backend
	createObjectParams2 := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: uniqueName,
		Version:            2,
	}
	_, err = svc.CreateObject(ctx, createObjectParams2)
	assert.NoError(t, err)

	// Test cases
	t.Run("Find object by key and storage backend name", func(t *testing.T) {
		// Get the object by key and storage backend name
		params := service.GetObjectByObjectKeyAndStorageBackendNameParams{
			ObjectKey:          object.ObjectKey,
			StorageBackendName: uniqueName,
		}
		foundObject, err := svc.GetObjectByObjectKeyAndStorageBackendName(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, object.ID, foundObject)
	})

	t.Run("Object not found with incorrect key", func(t *testing.T) {
		// Try to get an object with a non-existent key
		params := service.GetObjectByObjectKeyAndStorageBackendNameParams{
			ObjectKey:          "non-existent-key",
			StorageBackendName: uniqueName,
		}
		foundObject, err := svc.GetObjectByObjectKeyAndStorageBackendName(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Nil, foundObject)
	})

	t.Run("Object not found with incorrect storage backend name", func(t *testing.T) {
		// Try to get an object with a non-existent storage backend name
		params := service.GetObjectByObjectKeyAndStorageBackendNameParams{
			ObjectKey:          object.ObjectKey,
			StorageBackendName: "non-existent-backend",
		}
		foundObject, err := svc.GetObjectByObjectKeyAndStorageBackendName(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Nil, foundObject)
	})

	t.Run("Deleted object not found", func(t *testing.T) {
		// Delete the object
		err := svc.DeleteObject(ctx, object.ID)
		assert.NoError(t, err)

		// Try to get the deleted object by key and storage backend name
		params := service.GetObjectByObjectKeyAndStorageBackendNameParams{
			ObjectKey:          object.ObjectKey,
			StorageBackendName: uniqueName,
		}
		foundObject, err := svc.GetObjectByObjectKeyAndStorageBackendName(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Nil, foundObject)
	})
}
