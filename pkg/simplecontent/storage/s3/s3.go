package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// Config options for the S3 backend
type Config struct {
	Region          string // AWS region
	Bucket          string // S3 bucket name
	AccessKeyID     string // AWS access key ID
	SecretAccessKey string // AWS secret access key
	Endpoint        string // Optional custom endpoint for S3-compatible services
	UseSSL          bool   // Use SSL for connections (default: true)
	UsePathStyle    bool   // Use path-style addressing (default: false)
	PresignDuration int    // Duration in seconds for presigned URLs (default: 3600)

	// Server-side encryption options
	EnableSSE    bool   // Enable server-side encryption
	SSEAlgorithm string // SSE algorithm (AES256 or aws:kms)
	SSEKMSKeyID  string // Optional KMS key ID for aws:kms algorithm

	// MinIO/S3-compatible service options
	CreateBucketIfNotExist bool // Create bucket if it doesn't exist
}

// Backend is an S3-compatible implementation of the simplecontent.BlobStore interface
type Backend struct {
	client          *s3.Client
	bucket          string
	presignClient   *s3.PresignClient
	presignDuration time.Duration
	config          Config
}

// New creates a new S3-compatible storage backend
func New(config Config) (simplecontent.BlobStore, error) {
	if config.Bucket == "" {
		return nil, errors.New("bucket name is required")
	}

	if config.Region == "" {
		config.Region = "us-east-1"
	}

	if config.PresignDuration == 0 {
		config.PresignDuration = 3600 // 1 hour default
	}

	// Set up AWS config
	var awsCfg aws.Config
	var err error

	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		// Use provided credentials
		awsCfg, err = awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(config.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				config.AccessKeyID,
				config.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Use default credential chain
		awsCfg, err = awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(config.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Configure S3 client options
	var s3Options []func(*s3.Options)

	// Custom endpoint for S3-compatible services (MinIO, etc.)
	if config.Endpoint != "" {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(config.Endpoint)
			o.UsePathStyle = config.UsePathStyle
			// Note: Removed custom EndpointResolverV2 for better MinIO compatibility
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Options...)

	// Create presign client
	presignClient := s3.NewPresignClient(client)

	backend := &Backend{
		client:          client,
		bucket:          config.Bucket,
		presignClient:   presignClient,
		presignDuration: time.Duration(config.PresignDuration) * time.Second,
		config:          config,
	}

	// Create bucket if requested
	if config.CreateBucketIfNotExist {
		if err := backend.createBucketIfNotExists(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return backend, nil
}

// createBucketIfNotExists creates the bucket if it doesn't exist
func (b *Backend) createBucketIfNotExists(ctx context.Context) error {
	// Check if bucket exists
	_, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.bucket),
	})

	if err == nil {
		// Bucket exists
		return nil
	}

	// Check if error indicates bucket doesn't exist (handle multiple error types for MinIO compatibility)
	var notFound *types.NotFound
	var noSuchBucket *types.NoSuchBucket

	if !errors.As(err, &notFound) && !errors.As(err, &noSuchBucket) &&
		!strings.Contains(err.Error(), "BadRequest") &&
		!strings.Contains(err.Error(), "NoSuchBucket") {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	// Create bucket
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(b.bucket),
	}

	// Add location constraint for regions other than us-east-1
	if b.config.Region != "us-east-1" {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(b.config.Region),
		}
	}

	_, err = b.client.CreateBucket(ctx, createInput)
	if err != nil {
		// Handle bucket already exists gracefully
		if strings.Contains(err.Error(), "BucketAlreadyExists") ||
			strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			return nil
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// GetObjectMeta retrieves metadata for an object in S3
func (b *Backend) GetObjectMeta(ctx context.Context, objectKey string) (*simplecontent.ObjectMeta, error) {
	result, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return nil, errors.New("object not found")
		}
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	metadata := make(map[string]string)
	for k, v := range result.Metadata {
		metadata[k] = v
	}
	metadata["content_type"] = contentType

	meta := &simplecontent.ObjectMeta{
		Key:         objectKey,
		Size:        *result.ContentLength,
		ContentType: contentType,
		UpdatedAt:   *result.LastModified,
		ETag:        strings.Trim(*result.ETag, "\""),
		Metadata:    metadata,
	}

	return meta, nil
}

// GetUploadURL returns a presigned URL for uploading content
func (b *Backend) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	// Add server-side encryption if enabled
	if b.config.EnableSSE {
		switch b.config.SSEAlgorithm {
		case "AES256":
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		case "aws:kms":
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.config.SSEKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.config.SSEKMSKeyID)
			}
		}
	}

	result, err := b.presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = b.presignDuration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return result.URL, nil
}

// Upload uploads content directly to S3
func (b *Backend) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	uploader := manager.NewUploader(b.client)

	input := &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
		Body:   reader,
	}

	// Add server-side encryption if enabled
	if b.config.EnableSSE {
		switch b.config.SSEAlgorithm {
		case "AES256":
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		case "aws:kms":
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.config.SSEKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.config.SSEKMSKeyID)
			}
		}
	}

	_, err := uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// UploadWithParams uploads content with additional parameters
func (b *Backend) UploadWithParams(ctx context.Context, reader io.Reader, params simplecontent.UploadParams) error {
	uploader := manager.NewUploader(b.client)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(params.ObjectKey),
		Body:        reader,
		ContentType: aws.String(params.MimeType),
	}

	// Add server-side encryption if enabled
	if b.config.EnableSSE {
		switch b.config.SSEAlgorithm {
		case "AES256":
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		case "aws:kms":
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.config.SSEKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.config.SSEKMSKeyID)
			}
		}
	}

	_, err := uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3 with params: %w", err)
	}

	return nil
}

// GetDownloadURL returns a presigned URL for downloading content
func (b *Backend) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	// Set response content disposition if filename is provided
	if downloadFilename != "" {
		input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=\"%s\"", downloadFilename))
	}

	result, err := b.presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = b.presignDuration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return result.URL, nil
}

// GetPreviewURL returns a presigned URL for previewing content (inline display)
func (b *Backend) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket:                     aws.String(b.bucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String("inline"),
	}

	result, err := b.presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = b.presignDuration
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned preview URL: %w", err)
	}

	return result.URL, nil
}

// Download downloads content directly from S3
func (b *Backend) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		var notFound *types.NoSuchKey
		if errors.As(err, &notFound) {
			return nil, errors.New("object not found")
		}
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

// Delete deletes content from S3
func (b *Backend) Delete(ctx context.Context, objectKey string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}
