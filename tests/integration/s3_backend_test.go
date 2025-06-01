package integration

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/pkg/storage/s3"
)

// TestS3BackendWithMinIO tests the S3 backend with a MinIO server
// This test requires a running MinIO server
// You can start one with Docker:
// docker run -p 9000:9000 -p 9001:9001 minio/minio server /data --console-address ":9001"
func TestS3BackendWithMinIO(t *testing.T) {
	// Skip if MINIO_INTEGRATION_TEST environment variable is not set
	if os.Getenv("MINIO_INTEGRATION_TEST") == "" {
		t.Skip("Skipping MinIO integration test. Set MINIO_INTEGRATION_TEST=1 to run.")
	}

	// MinIO configuration
	config := s3.Config{
		Region:                 "us-east-1",
		Bucket:                 "test-bucket-" + time.Now().Format("20060102150405"),
		AccessKeyID:            "minioadmin",
		SecretAccessKey:        "minioadmin",
		Endpoint:               "http://localhost:9000",
		UseSSL:                 false,
		UsePathStyle:           true,
		PresignDuration:        3600,
		CreateBucketIfNotExist: true,
	}

	// Create S3 backend
	backend, err := s3.NewS3Backend(config)
	require.NoError(t, err)

	ctx := context.Background()
	objectKey := "test/integration-test.txt"
	content := "Hello, MinIO! This is an integration test."

	// Test Upload
	err = backend.Upload(ctx, objectKey, strings.NewReader(content))
	assert.NoError(t, err)

	// Test GetUploadURL
	uploadURL, err := backend.GetUploadURL(ctx, "test/presigned-upload.txt")
	assert.NoError(t, err)
	assert.Contains(t, uploadURL, config.Endpoint)
	assert.Contains(t, uploadURL, "test/presigned-upload.txt")
	assert.Contains(t, uploadURL, "X-Amz-Algorithm")

	// Test GetDownloadURL
	downloadURL, err := backend.GetDownloadURL(ctx, objectKey, "test/integration-test.txt")
	assert.NoError(t, err)
	assert.Contains(t, downloadURL, config.Endpoint)
	assert.Contains(t, downloadURL, objectKey)
	assert.Contains(t, downloadURL, "X-Amz-Algorithm")

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

	// Verify object is deleted
	_, err = backend.Download(ctx, objectKey)
	assert.Error(t, err)
}

// TestS3BackendWithMinIOAndSSE tests the S3 backend with a MinIO server and server-side encryption
func TestS3BackendWithMinIOAndSSE(t *testing.T) {
	// Skip if MINIO_INTEGRATION_TEST environment variable is not set
	if os.Getenv("MINIO_INTEGRATION_TEST") == "" {
		t.Skip("Skipping MinIO integration test. Set MINIO_INTEGRATION_TEST=1 to run.")
	}

	// MinIO configuration with SSE
	config := s3.Config{
		Region:                 "us-east-1",
		Bucket:                 "test-bucket-sse-" + time.Now().Format("20060102150405"),
		AccessKeyID:            "minioadmin",
		SecretAccessKey:        "minioadmin",
		Endpoint:               "http://localhost:9000",
		UseSSL:                 false,
		UsePathStyle:           true,
		PresignDuration:        3600,
		CreateBucketIfNotExist: true,
		EnableSSE:              true,
		SSEAlgorithm:           "AES256",
	}

	// Create S3 backend
	backend, err := s3.NewS3Backend(config)
	require.NoError(t, err)

	ctx := context.Background()
	objectKey := "test/integration-test-sse.txt"
	content := "Hello, MinIO with SSE! This is an integration test."

	// Test Upload with SSE
	err = backend.Upload(ctx, objectKey, strings.NewReader(content))
	assert.NoError(t, err)

	// Test Download with SSE
	reader, err := backend.Download(ctx, objectKey)
	assert.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Test Delete
	err = backend.Delete(ctx, objectKey)
	assert.NoError(t, err)
}
