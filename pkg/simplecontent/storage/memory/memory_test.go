package memory_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/pkg/simplecontent"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func TestMemoryBackend(t *testing.T) {
	backend := memorystorage.New()
	ctx := context.Background()
	testKey := "test/object/key"
	testData := "Hello, World! This is test data."
	testMimeType := "text/plain"

	t.Run("Upload", func(t *testing.T) {
		reader := strings.NewReader(testData)
		err := backend.Upload(ctx, testKey, reader)
		assert.NoError(t, err)
	})

	t.Run("GetObjectMeta", func(t *testing.T) {
		meta, err := backend.GetObjectMeta(ctx, testKey)
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, testKey, meta.Key)
		assert.Equal(t, int64(len(testData)), meta.Size)
		assert.Equal(t, "application/octet-stream", meta.ContentType) // Default content type
		assert.Contains(t, meta.Metadata, "mime_type")
	})

	t.Run("Download", func(t *testing.T) {
		reader, err := backend.Download(ctx, testKey)
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		defer reader.Close()

		downloadedData, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, testData, string(downloadedData))
	})

	t.Run("UploadWithParams", func(t *testing.T) {
		testKey2 := "test/object/key2"
		params := simplecontent.UploadParams{
			ObjectKey: testKey2,
			MimeType:  testMimeType,
		}

		reader := strings.NewReader(testData)
		err := backend.UploadWithParams(ctx, reader, params)
		assert.NoError(t, err)

		// Verify the mime type was stored
		meta, err := backend.GetObjectMeta(ctx, testKey2)
		assert.NoError(t, err)
		assert.Equal(t, testMimeType, meta.ContentType)
	})

	t.Run("Delete", func(t *testing.T) {
		testKey3 := "test/object/key3"
		
		// Upload first
		reader := strings.NewReader(testData)
		err := backend.Upload(ctx, testKey3, reader)
		assert.NoError(t, err)

		// Delete
		err = backend.Delete(ctx, testKey3)
		assert.NoError(t, err)

		// Verify deletion
		_, err = backend.GetObjectMeta(ctx, testKey3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})

	t.Run("GetUploadURL_ShouldReturnError", func(t *testing.T) {
		url, err := backend.GetUploadURL(ctx, "test/key")
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "direct upload required")
	})

	t.Run("GetDownloadURL_ShouldReturnError", func(t *testing.T) {
		url, err := backend.GetDownloadURL(ctx, "test/key", "filename.txt")
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "direct download required")
	})

	t.Run("GetPreviewURL_ShouldReturnError", func(t *testing.T) {
		url, err := backend.GetPreviewURL(ctx, "test/key")
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "direct preview required")
	})

	t.Run("ErrorCases", func(t *testing.T) {
		nonExistentKey := "nonexistent/key"

		// GetObjectMeta for non-existent object
		meta, err := backend.GetObjectMeta(ctx, nonExistentKey)
		assert.Error(t, err)
		assert.Nil(t, meta)

		// Download non-existent object
		reader, err := backend.Download(ctx, nonExistentKey)
		assert.Error(t, err)
		assert.Nil(t, reader)

		// Delete non-existent object
		err = backend.Delete(ctx, nonExistentKey)
		assert.Error(t, err)
	})
}

func TestMemoryBackendConcurrency(t *testing.T) {
	backend := memorystorage.New()
	ctx := context.Background()

	// Test concurrent uploads and downloads
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				testKey := fmt.Sprintf("concurrent/test/%d/%d", goroutineID, j)
				testData := fmt.Sprintf("Test data from goroutine %d, operation %d", goroutineID, j)

				// Upload
				reader := strings.NewReader(testData)
				err := backend.Upload(ctx, testKey, reader)
				require.NoError(t, err)

				// Download and verify
				downloadReader, err := backend.Download(ctx, testKey)
				require.NoError(t, err)

				downloadedData, err := io.ReadAll(downloadReader)
				require.NoError(t, err)
				downloadReader.Close()

				assert.Equal(t, testData, string(downloadedData))

				// Delete
				err = backend.Delete(ctx, testKey)
				require.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func BenchmarkMemoryBackend(b *testing.B) {
	backend := memorystorage.New()
	ctx := context.Background()
	testData := strings.Repeat("benchmark data ", 100) // ~1.4KB

	b.Run("Upload", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testKey := fmt.Sprintf("benchmark/upload/%d", i)
			reader := strings.NewReader(testData)
			err := backend.Upload(ctx, testKey, reader)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Download", func(b *testing.B) {
		// Setup: upload test data
		for i := 0; i < b.N; i++ {
			testKey := fmt.Sprintf("benchmark/download/%d", i)
			reader := strings.NewReader(testData)
			err := backend.Upload(ctx, testKey, reader)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testKey := fmt.Sprintf("benchmark/download/%d", i)
			reader, err := backend.Download(ctx, testKey)
			if err != nil {
				b.Fatal(err)
			}
			_, err = io.ReadAll(reader)
			if err != nil {
				b.Fatal(err)
			}
			reader.Close()
		}
	})

	b.Run("GetObjectMeta", func(b *testing.B) {
		// Setup: upload test data
		testKey := "benchmark/meta/object"
		reader := strings.NewReader(testData)
		err := backend.Upload(ctx, testKey, reader)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := backend.GetObjectMeta(ctx, testKey)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}