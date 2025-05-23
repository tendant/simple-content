package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestObject(t *testing.T) {
	id := uuid.New()
	contentID := uuid.New()
	now := time.Now().UTC()

	obj := &Object{
		ID:                 id,
		ContentID:          contentID,
		StorageBackendName: "test-backend",
		StorageClass:       "standard",
		ObjectKey:          "test-key",
		FileName:           "test-file.txt",
		Version:            1,
		VersionID:          "v1",
		ObjectType:         "file",
		Status:             ObjectStatusCreated,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	assert.Equal(t, id, obj.ID)
	assert.Equal(t, contentID, obj.ContentID)
	assert.Equal(t, "test-backend", obj.StorageBackendName)
	assert.Equal(t, "standard", obj.StorageClass)
	assert.Equal(t, "test-key", obj.ObjectKey)
	assert.Equal(t, "test-file.txt", obj.FileName)
	assert.Equal(t, 1, obj.Version)
	assert.Equal(t, "v1", obj.VersionID)
	assert.Equal(t, "file", obj.ObjectType)
	assert.Equal(t, ObjectStatusCreated, obj.Status)
	assert.Equal(t, now, obj.CreatedAt)
	assert.Equal(t, now, obj.UpdatedAt)
}

func TestObjectMetadata(t *testing.T) {
	objectID := uuid.New()
	now := time.Now().UTC()

	metadata := &ObjectMetadata{
		ObjectID:  objectID,
		SizeBytes: 1024,
		MimeType:  "text/plain",
		ETag:      "abc123",
		Metadata: map[string]interface{}{
			"author":      "Test User",
			"description": "Test file description",
			"tags":        []string{"test", "example"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, objectID, metadata.ObjectID)
	assert.Equal(t, int64(1024), metadata.SizeBytes)
	assert.Equal(t, "text/plain", metadata.MimeType)
	assert.Equal(t, "abc123", metadata.ETag)
	assert.Equal(t, "Test User", metadata.Metadata["author"])
	assert.Equal(t, "Test file description", metadata.Metadata["description"])
	assert.Equal(t, []string{"test", "example"}, metadata.Metadata["tags"])
	assert.Equal(t, now, metadata.CreatedAt)
	assert.Equal(t, now, metadata.UpdatedAt)
}
