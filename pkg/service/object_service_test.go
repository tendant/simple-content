package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/internal/storage"
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
	backend.Upload(ctx, object.ObjectKey, object.ObjectType, bytes.NewReader([]byte("test data")))

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

func TestObjectService_GetObjectMetaFromStorage(t *testing.T) {
	svc, backend := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()

	// Create an object
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)
	assert.NotNil(t, object)

	// Test cases
	t.Run("Object exists - successful retrieval", func(t *testing.T) {
		// Upload some data to the object so we have metadata
		testData := []byte("test data content")
		err := backend.Upload(ctx, object.ObjectKey, "text/plain", bytes.NewReader(testData))
		assert.NoError(t, err)

		// Get object metadata from storage
		objectMeta, err := svc.GetObjectMetaFromStorage(ctx, object.ID)
		assert.NoError(t, err)
		assert.NotNil(t, objectMeta)

		// Verify metadata matches what we expect from storage.ObjectMeta
		assert.Equal(t, object.ObjectKey, objectMeta.Key)
		assert.Equal(t, int64(len(testData)), objectMeta.Size)

		// Verify the object has the expected storage metadata structure
		assert.IsType(t, &storage.ObjectMeta{}, objectMeta)
	})

	t.Run("Object does not exist in storage", func(t *testing.T) {
		// Create another object but don't upload any data
		createObjectParams := service.CreateObjectParams{
			ContentID:          contentID,
			StorageBackendName: "memory",
			Version:            2,
		}
		objectWithoutData, err := svc.CreateObject(ctx, createObjectParams)
		assert.NoError(t, err)

		// Try to get metadata for an object that doesn't exist in storage
		_, err = svc.GetObjectMetaFromStorage(ctx, objectWithoutData.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})

	t.Run("Object ID does not exist", func(t *testing.T) {
		// Try to get metadata for a non-existent object ID
		_, err := svc.GetObjectMetaFromStorage(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})
}

// TestObjectService_GetObjectMetaFromStorage_NonExistentKey tests specifically the scenario
// where an object key doesn't exist in storage, simulating a Minio/S3 backend behavior
func TestObjectService_GetObjectMetaFromStorage_NonExistentKey(t *testing.T) {
	// Set up the service with memory backend
	svc, backend := setupObjectService()
	ctx := context.Background()
	contentID := uuid.New()

	// Create an object with a specific object key
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "memory",
		Version:            1,
	}
	object, err := svc.CreateObject(ctx, createObjectParams)
	assert.NoError(t, err)

	// Verify the object was created successfully
	assert.NotNil(t, object)
	assert.NotEmpty(t, object.ObjectKey)

	// Test when the object key doesn't exist in storage
	// Note: We don't upload any data to the backend, so the object key doesn't exist

	// Attempt to get metadata for the object that doesn't exist in storage
	_, err = svc.GetObjectMetaFromStorage(ctx, object.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object not found")

	// Verify the error comes from the storage backend by trying to download the object
	// This should also fail since the object doesn't exist in storage
	_, downloadErr := backend.Download(ctx, object.ObjectKey)
	assert.Error(t, downloadErr)
	assert.Contains(t, downloadErr.Error(), "object not found", "Download should also fail for non-existent object")
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

func TestObjectService_GetObjectMetaFromStorageByObjectKeyAndStorageBackendName(t *testing.T) {
	ctx := context.Background()

	t.Run("Success case", func(t *testing.T) {
		// Setup
		svc, backend := setupObjectService()
		objectKey := "test-object-key"

		// Upload test data to the backend
		testData := []byte("test data content")
		err := backend.Upload(ctx, objectKey, "text/plain", bytes.NewReader(testData))
		assert.NoError(t, err)

		// Call the function being tested
		objectMeta, err := svc.GetObjectMetaFromStorageByObjectKeyAndStorageBackendName(ctx, objectKey, "memory")

		// Verify results
		assert.NoError(t, err)
		assert.NotNil(t, objectMeta)
		assert.Equal(t, int64(len(testData)), objectMeta.Size)
		assert.Equal(t, objectKey, objectMeta.Key)
	})

	t.Run("Backend not found", func(t *testing.T) {
		// Setup
		svc, _ := setupObjectService()
		objectKey := "test-object-key"

		// Call with non-existent backend
		objectMeta, err := svc.GetObjectMetaFromStorageByObjectKeyAndStorageBackendName(ctx, objectKey, "non-existent-backend")

		// Verify error
		assert.Error(t, err)
		assert.Nil(t, objectMeta)
		assert.Contains(t, err.Error(), "failed to get backend")
	})

	t.Run("Object key not found", func(t *testing.T) {
		// Setup
		svc, _ := setupObjectService()
		nonExistentKey := "non-existent-key"

		// Call the function being tested with a key that doesn't exist
		objectMeta, err := svc.GetObjectMetaFromStorageByObjectKeyAndStorageBackendName(ctx, nonExistentKey, "memory")

		// Verify error
		assert.Error(t, err)
		assert.Nil(t, objectMeta)
		assert.Contains(t, err.Error(), "object not found")
	})
}
