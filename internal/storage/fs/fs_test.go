package fs_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/storage/fs"
)

func TestFSBackend(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fs-backend-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a new file system backend
	config := fs.Config{
		BaseDir: tempDir,
	}
	backend, err := fs.NewFSBackend(config)
	require.NoError(t, err)

	ctx := context.Background()
	objectKey := "test/object.txt"
	content := "Hello, World!"

	// Test Upload
	err = backend.Upload(ctx, objectKey, strings.NewReader(content))
	assert.NoError(t, err)

	// Verify file exists
	filePath := filepath.Join(tempDir, objectKey)
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Test Download
	reader, err := backend.Download(ctx, objectKey)
	assert.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Test Delete
	err = backend.Delete(ctx, objectKey)
	assert.NoError(t, err)

	// Verify file no longer exists
	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestFSBackendWithURLPrefix(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fs-backend-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a new file system backend with URL prefix
	config := fs.Config{
		BaseDir:   tempDir,
		URLPrefix: "http://localhost:8080",
	}
	backend, err := fs.NewFSBackend(config)
	require.NoError(t, err)

	ctx := context.Background()
	objectKey := "test/object.txt"

	// Test GetUploadURL
	uploadURL, err := backend.GetUploadURL(ctx, objectKey)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/upload/test/object.txt", uploadURL)

	// Test GetDownloadURL
	downloadURL, err := backend.GetDownloadURL(ctx, objectKey, "test/object.txt")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/download/test/object.txt", downloadURL)
}

func TestFSBackendErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fs-backend-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a new file system backend
	config := fs.Config{
		BaseDir: tempDir,
	}
	backend, err := fs.NewFSBackend(config)
	require.NoError(t, err)

	ctx := context.Background()
	objectKey := "test/object.txt"

	// Test GetUploadURL with no URL prefix
	_, err = backend.GetUploadURL(ctx, objectKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "direct upload required")

	// Test GetDownloadURL with no URL prefix
	_, err = backend.GetDownloadURL(ctx, objectKey, "test/object.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "direct download required")

	// Test Download non-existent file
	_, err = backend.Download(ctx, objectKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object not found")

	// Test Delete non-existent file
	err = backend.Delete(ctx, objectKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object not found")
}

func TestNewFSBackendErrors(t *testing.T) {
	// Test with empty base directory
	config := fs.Config{
		BaseDir: "",
	}
	_, err := fs.NewFSBackend(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base directory is required")

	// Test with invalid base directory (use a file as a directory)
	tempFile, err := os.CreateTemp("", "fs-backend-test")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	config = fs.Config{
		BaseDir: tempFile.Name(),
	}
	_, err = fs.NewFSBackend(config)
	assert.Error(t, err)
}
