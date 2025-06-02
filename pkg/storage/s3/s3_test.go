package s3_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	s3backend "github.com/tendant/simple-content/pkg/storage/s3"
)

// Mock S3 client for unit testing
type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *mockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.HeadBucketOutput), args.Error(1)
}

func (m *mockS3Client) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.CreateBucketOutput), args.Error(1)
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

// Mock presign client
type mockPresignClient struct {
	mock.Mock
}

func (m *mockPresignClient) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*s3backend.PresignOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3backend.PresignOutput), args.Error(1)
}

func (m *mockPresignClient) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*s3backend.PresignOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3backend.PresignOutput), args.Error(1)
}

// TestS3Backend tests the basic functionality of the S3 backend
func TestS3Backend(t *testing.T) {
	// This test uses mocks to avoid actual S3 calls
	// For integration tests with real S3/MinIO, see tests/integration/s3_backend_test.go

	// Create mocks
	mockClient := new(mockS3Client)
	mockPresign := new(mockPresignClient)

	// Create test data
	objectKey := "test/object.txt"
	content := "Hello, World!"
	bucket := "test-bucket"

	// Set up mock expectations for Upload
	mockClient.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3.PutObjectOutput{}, nil)

	// Set up mock expectations for Download
	mockClient.On("GetObject", mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(content)),
	}, nil)

	// Set up mock expectations for Delete
	mockClient.On("DeleteObject", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3.DeleteObjectOutput{}, nil)

	// Set up mock expectations for GetUploadURL
	mockPresign.On("PresignPutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3backend.PresignOutput{
		URL: "https://test-bucket.s3.amazonaws.com/test/object.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
	}, nil)

	// Set up mock expectations for GetDownloadURL
	mockPresign.On("PresignGetObject", mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3backend.PresignOutput{
		URL: "https://test-bucket.s3.amazonaws.com/test/object.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
	}, nil)

	// Create a backend instance with the mocks
	backend := &s3backend.S3BackendForTesting{
		Client:          mockClient,
		PresignClient:   mockPresign,
		Bucket:          bucket,
		PresignDuration: 3600,
	}

	ctx := context.Background()

	// Test Upload
	err := backend.Upload(ctx, objectKey, strings.NewReader(content))
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "PutObject", mock.Anything, mock.Anything)

	// Test Download
	reader, err := backend.Download(ctx, objectKey)
	assert.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))
	mockClient.AssertCalled(t, "GetObject", mock.Anything, mock.Anything)

	// Test Delete
	err = backend.Delete(ctx, objectKey)
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "DeleteObject", mock.Anything, mock.Anything)

	// Test GetUploadURL
	uploadURL, err := backend.GetUploadURL(ctx, objectKey)
	assert.NoError(t, err)
	assert.Contains(t, uploadURL, "https://test-bucket.s3.amazonaws.com/test/object.txt")
	mockPresign.AssertCalled(t, "PresignPutObject", mock.Anything, mock.Anything)

	// Test GetPreviewURL
	previewURL, err := backend.GetPreviewURL(ctx, objectKey)
	assert.NoError(t, err)
	assert.Contains(t, previewURL, "https://test-bucket.s3.amazonaws.com/test/object.txt")
	mockPresign.AssertCalled(t, "PresignGetObject", mock.Anything, mock.Anything)
}

// TestS3BackendWithSSE tests the server-side encryption functionality
func TestS3BackendWithSSE(t *testing.T) {
	// Create mocks
	mockClient := new(mockS3Client)
	mockPresign := new(mockPresignClient)

	// Create test data
	objectKey := "test/object.txt"
	content := "Hello, World!"
	bucket := "test-bucket"

	// Set up mock expectations for Upload with SSE
	mockClient.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == bucket &&
			*input.Key == objectKey &&
			input.ServerSideEncryption == types.ServerSideEncryptionAes256
	})).Return(&s3.PutObjectOutput{}, nil)

	// Create a backend instance with the mocks and SSE enabled
	backend := &s3backend.S3BackendForTesting{
		Client:          mockClient,
		PresignClient:   mockPresign,
		Bucket:          bucket,
		PresignDuration: 3600,
		EnableSSE:       true,
		SSEAlgorithm:    "AES256",
	}

	ctx := context.Background()

	// Test Upload with SSE
	err := backend.Upload(ctx, objectKey, strings.NewReader(content))
	assert.NoError(t, err)
	mockClient.AssertCalled(t, "PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return input.ServerSideEncryption == types.ServerSideEncryptionAes256
	}))
}

// TestS3BackendErrors tests error handling in the S3 backend
func TestS3BackendErrors(t *testing.T) {
	// Create mocks
	mockClient := new(mockS3Client)
	mockPresign := new(mockPresignClient)

	// Create test data
	objectKey := "test/object.txt"
	bucket := "test-bucket"
	testError := fmt.Errorf("object not found: %s", objectKey)

	// Set up mock expectations for Download error
	mockClient.On("GetObject", mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3.GetObjectOutput{}, testError)

	// Set up mock expectations for Delete error
	mockClient.On("DeleteObject", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectInput) bool {
		return *input.Bucket == bucket && *input.Key == objectKey
	})).Return(&s3.DeleteObjectOutput{}, testError)

	// Create a backend instance with the mocks
	backend := &s3backend.S3BackendForTesting{
		Client:          mockClient,
		PresignClient:   mockPresign,
		Bucket:          bucket,
		PresignDuration: 3600,
	}

	ctx := context.Background()

	// Test Download error
	_, err := backend.Download(ctx, objectKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download object")

	// Test Delete error
	err = backend.Delete(ctx, objectKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete object")
}

// TestNewS3Backend tests the creation of a new S3 backend
func TestNewS3Backend(t *testing.T) {
	// Skip the first test that depends on AWS credentials
	t.Run("EmptyBucket", func(t *testing.T) {

		// Test with empty bucket
		config := s3backend.Config{
			Region: "us-east-1",
			Bucket: "",
		}
		_, err := s3backend.NewS3Backend(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	t.Run("KMSWithoutKeyID", func(t *testing.T) {
		// Test with KMS but no key ID
		config := s3backend.Config{
			Region:       "us-east-1",
			Bucket:       "test-bucket",
			EnableSSE:    true,
			SSEAlgorithm: "aws:kms",
		}
		_, err := s3backend.NewS3Backend(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "KMS key ID is required")
	})
}
