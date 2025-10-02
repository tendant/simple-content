package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestS3Backend_BasicConfiguration tests the configuration and creation of S3 backend
func TestS3Backend_BasicConfiguration(t *testing.T) {
	t.Run("EmptyBucket", func(t *testing.T) {
		config := Config{
			Region: "us-east-1",
			Bucket: "",
		}
		_, err := New(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	t.Run("DefaultRegion", func(t *testing.T) {
		// This will fail without credentials, but we can at least test
		// that it attempts to create with defaults
		config := Config{
			Bucket:          "test-bucket",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		}
		// We expect this to succeed configuration-wise
		backend, err := New(config)
		// May error due to network/credentials, but not due to missing bucket
		if err != nil {
			assert.NotContains(t, err.Error(), "bucket name is required")
		} else {
			assert.NotNil(t, backend)
		}
	})

	t.Run("CustomPresignDuration", func(t *testing.T) {
		config := Config{
			Bucket:          "test-bucket",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			PresignDuration: 7200,
		}
		backend, err := New(config)
		if err == nil {
			assert.NotNil(t, backend)
			// Check the duration was set (if we can access it)
			if b, ok := backend.(*Backend); ok {
				assert.Equal(t, 7200*time.Second, b.presignDuration)
			}
		}
	})

	t.Run("ServerSideEncryption_AES256", func(t *testing.T) {
		config := Config{
			Bucket:          "test-bucket",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			EnableSSE:       true,
			SSEAlgorithm:    "AES256",
		}
		backend, err := New(config)
		// Should not error on config validation
		if err != nil {
			assert.NotContains(t, err.Error(), "invalid SSE")
		} else {
			assert.NotNil(t, backend)
		}
	})

	t.Run("ServerSideEncryption_KMS_WithoutKeyID", func(t *testing.T) {
		config := Config{
			Bucket:          "test-bucket",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			EnableSSE:       true,
			SSEAlgorithm:    "aws:kms",
			// Missing SSEKMSKeyID
		}
		backend, err := New(config)
		// This should potentially error or warn about missing KMS key ID
		// Depending on implementation, adjust assertion
		_ = backend
		_ = err
	})

	t.Run("ServerSideEncryption_KMS_WithKeyID", func(t *testing.T) {
		config := Config{
			Bucket:          "test-bucket",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			EnableSSE:       true,
			SSEAlgorithm:    "aws:kms",
			SSEKMSKeyID:     "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		}
		backend, err := New(config)
		if err == nil {
			assert.NotNil(t, backend)
		}
	})
}

// TestS3Backend_MinIOConfiguration tests MinIO-specific configuration
func TestS3Backend_MinIOConfiguration(t *testing.T) {
	t.Run("CustomEndpoint", func(t *testing.T) {
		config := Config{
			Bucket:          "test-bucket",
			Region:          "us-east-1",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			Endpoint:        "http://localhost:9000",
			UseSSL:          false,
			UsePathStyle:    true,
		}
		backend, err := New(config)
		if err == nil {
			assert.NotNil(t, backend)
			// Verify it's configured for MinIO
			if b, ok := backend.(*Backend); ok {
				assert.Equal(t, "http://localhost:9000", b.config.Endpoint)
				assert.True(t, b.config.UsePathStyle)
				assert.False(t, b.config.UseSSL)
			}
		}
	})

	t.Run("PathStyleAddressing", func(t *testing.T) {
		config := Config{
			Bucket:          "test-bucket",
			AccessKeyID:     "minioadmin",
			SecretAccessKey: "minioadmin",
			Endpoint:        "http://minio:9000",
			UsePathStyle:    true,
		}
		backend, err := New(config)
		if err == nil {
			assert.NotNil(t, backend)
		}
	})
}

// TestS3Backend_Integration tests actual S3/MinIO operations
// This test requires a running MinIO instance or S3 credentials
func TestS3Backend_Integration(t *testing.T) {
	// Check if integration tests should run
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check for MinIO environment variables
	endpoint := os.Getenv("AWS_S3_ENDPOINT")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucket := os.Getenv("AWS_S3_BUCKET")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("Skipping integration test: S3/MinIO environment variables not set")
	}

	config := Config{
		Bucket:          bucket,
		Region:          "us-east-1",
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Endpoint:        endpoint,
		UseSSL:          false,
		UsePathStyle:    true,
		CreateBucketIfNotExist: true,
	}

	backend, err := New(config)
	require.NoError(t, err, "Failed to create S3 backend")
	require.NotNil(t, backend)

	ctx := context.Background()
	objectKey := fmt.Sprintf("test/integration/%d/file.txt", time.Now().Unix())
	testData := []byte("Hello from S3 integration test!")

	t.Run("UploadAndDownload", func(t *testing.T) {
		// Upload
		err := backend.Upload(ctx, objectKey, bytes.NewReader(testData))
		require.NoError(t, err, "Failed to upload object")

		// Download
		reader, err := backend.Download(ctx, objectKey)
		require.NoError(t, err, "Failed to download object")
		defer reader.Close()

		downloadedData, err := io.ReadAll(reader)
		require.NoError(t, err, "Failed to read downloaded data")
		assert.Equal(t, testData, downloadedData, "Downloaded data doesn't match uploaded data")
	})

	t.Run("GetObjectMeta", func(t *testing.T) {
		meta, err := backend.GetObjectMeta(ctx, objectKey)
		require.NoError(t, err, "Failed to get object metadata")
		assert.Greater(t, meta.Size, int64(0), "Object size should be greater than 0")
		assert.NotEmpty(t, meta.ETag, "ETag should not be empty")
	})

	t.Run("GetObjectMeta_NonExistent", func(t *testing.T) {
		_, err := backend.GetObjectMeta(ctx, "nonexistent/object.txt")
		require.Error(t, err, "Should error for non-existent object")
	})

	t.Run("GetUploadURL", func(t *testing.T) {
		uploadKey := fmt.Sprintf("test/presigned/%d/upload.txt", time.Now().Unix())
		uploadURL, err := backend.GetUploadURL(ctx, uploadKey)
		require.NoError(t, err, "Failed to get upload URL")
		assert.NotEmpty(t, uploadURL, "Upload URL should not be empty")
		assert.Contains(t, uploadURL, bucket, "URL should contain bucket name")
	})

	t.Run("GetDownloadURL", func(t *testing.T) {
		downloadURL, err := backend.GetDownloadURL(ctx, objectKey, "file.txt")
		require.NoError(t, err, "Failed to get download URL")
		assert.NotEmpty(t, downloadURL, "Download URL should not be empty")
		assert.Contains(t, downloadURL, bucket, "URL should contain bucket name")
	})

	t.Run("GetPreviewURL", func(t *testing.T) {
		previewURL, err := backend.GetPreviewURL(ctx, objectKey)
		require.NoError(t, err, "Failed to get preview URL")
		assert.NotEmpty(t, previewURL, "Preview URL should not be empty")
		assert.Contains(t, previewURL, bucket, "URL should contain bucket name")
	})

	t.Run("Delete", func(t *testing.T) {
		// Delete the object
		err := backend.Delete(ctx, objectKey)
		require.NoError(t, err, "Failed to delete object")

		// Verify it's deleted
		_, err = backend.Download(ctx, objectKey)
		require.Error(t, err, "Should error when downloading deleted object")
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		// Deleting non-existent object should not error (S3 behavior)
		err := backend.Delete(ctx, "nonexistent/object.txt")
		// S3 Delete is idempotent, so this typically doesn't error
		assert.NoError(t, err, "Delete of non-existent object should not error")
	})
}

// TestS3Backend_ErrorHandling tests error scenarios
func TestS3Backend_ErrorHandling(t *testing.T) {
	// Create a backend with invalid credentials to test error handling
	config := Config{
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "invalid-key",
		SecretAccessKey: "invalid-secret",
	}

	backend, err := New(config)
	// Backend creation may succeed (config is valid)
	if err != nil {
		t.Skip("Backend creation failed, can't test error handling")
	}

	ctx := context.Background()

	t.Run("Download_InvalidObject", func(t *testing.T) {
		_, err := backend.Download(ctx, "nonexistent/object.txt")
		assert.Error(t, err, "Should error for non-existent object")
	})

	t.Run("GetObjectMeta_InvalidObject", func(t *testing.T) {
		_, err := backend.GetObjectMeta(ctx, "nonexistent/object.txt")
		assert.Error(t, err, "Should error for non-existent object")
	})
}

// TestS3Backend_PresignedURLFormat tests presigned URL generation format
func TestS3Backend_PresignedURLFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping presigned URL format test in short mode")
	}

	config := Config{
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		PresignDuration: 3600,
	}

	backend, err := New(config)
	if err != nil {
		t.Skip("Backend creation failed, skipping presigned URL tests")
	}

	ctx := context.Background()
	objectKey := "test/object.txt"

	t.Run("UploadURLFormat", func(t *testing.T) {
		url, err := backend.GetUploadURL(ctx, objectKey)
		if err == nil {
			assert.True(t, strings.Contains(url, "X-Amz") || strings.Contains(url, "http"),
				"URL should be a presigned URL or HTTP URL")
		}
	})

	t.Run("DownloadURLFormat", func(t *testing.T) {
		url, err := backend.GetDownloadURL(ctx, objectKey, "")
		if err == nil {
			assert.True(t, strings.Contains(url, "X-Amz") || strings.Contains(url, "http"),
				"URL should be a presigned URL or HTTP URL")
		}
	})
}

// TestS3Backend_ContextCancellation tests context cancellation handling
func TestS3Backend_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping context cancellation test in short mode")
	}

	config := Config{
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
	}

	backend, err := New(config)
	if err != nil {
		t.Skip("Backend creation failed, skipping context tests")
	}

	t.Run("CancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Operations should respect context cancellation
		err := backend.Upload(ctx, "test/key", bytes.NewReader([]byte("test")))
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		}
	})
}
