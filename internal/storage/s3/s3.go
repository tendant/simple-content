package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/tendant/simple-content/internal/storage"
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

	// MinIO-specific options
	CreateBucketIfNotExist bool // Create bucket if it doesn't exist
}

// S3Backend is an AWS S3 implementation of the storage.Backend interface
type S3Backend struct {
	client          *s3.Client
	bucket          string
	presignClient   *s3.PresignClient
	presignDuration time.Duration
	enableSSE       bool
	sseAlgorithm    string
	sseKMSKeyID     string
}

// NewS3Backend creates a new S3 storage backend
func NewS3Backend(config Config) (storage.Backend, error) {
	// Validate config
	if config.Bucket == "" {
		return nil, errors.New("bucket name is required")
	}

	if config.Region == "" {
		config.Region = "us-east-1" // Default region
	}

	if config.PresignDuration <= 0 {
		config.PresignDuration = 3600 // Default to 1 hour
	}

	// Configure AWS SDK
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(config.Region),
	}

	// Add credentials if provided
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				config.AccessKeyID,
				config.SecretAccessKey,
				"",
			),
		))
	}

	// Add custom endpoint if provided (for S3-compatible services like MinIO)
	if config.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               config.Endpoint,
					SigningRegion:     config.Region,
					HostnameImmutable: true,
				}, nil
			})
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(customResolver))
	}

	// Create AWS config
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if config.UsePathStyle {
			o.UsePathStyle = true
		}
	})

	// Create presign client
	presignClient := s3.NewPresignClient(s3Client)

	// Validate SSE options
	if config.EnableSSE && config.SSEAlgorithm == "" {
		config.SSEAlgorithm = "AES256" // Default to AES256
	}

	if config.SSEAlgorithm == "aws:kms" && config.SSEKMSKeyID == "" {
		return nil, errors.New("KMS key ID is required when using aws:kms encryption")
	}

	// Create bucket if it doesn't exist and the option is enabled
	if config.CreateBucketIfNotExist {
		_, err := s3Client.HeadBucket(context.Background(), &s3.HeadBucketInput{
			Bucket: aws.String(config.Bucket),
		})

		if err != nil {
			// Create the bucket if it doesn't exist
			_, err = s3Client.CreateBucket(context.Background(), &s3.CreateBucketInput{
				Bucket: aws.String(config.Bucket),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create bucket: %w", err)
			}
		}
	}

	return &S3Backend{
		client:          s3Client,
		bucket:          config.Bucket,
		presignClient:   presignClient,
		presignDuration: time.Duration(config.PresignDuration) * time.Second,
		enableSSE:       config.EnableSSE,
		sseAlgorithm:    config.SSEAlgorithm,
		sseKMSKeyID:     config.SSEKMSKeyID,
	}, nil
}

// GetUploadURL returns a pre-signed URL for uploading content
func (b *S3Backend) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	// Add server-side encryption if enabled
	if b.enableSSE {
		if b.sseAlgorithm == "AES256" {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		} else if b.sseAlgorithm == "aws:kms" {
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.sseKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.sseKMSKeyID)
			}
		}
	}

	result, err := b.presignClient.PresignPutObject(ctx, input,
		s3.WithPresignExpires(b.presignDuration))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// Upload uploads content directly to S3
func (b *S3Backend) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	uploader := manager.NewUploader(b.client)

	input := &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
		Body:   reader,
	}

	// Add server-side encryption if enabled
	if b.enableSSE {
		if b.sseAlgorithm == "AES256" {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		} else if b.sseAlgorithm == "aws:kms" {
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.sseKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.sseKMSKeyID)
			}
		}
	}

	_, err := uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// GetDownloadURL returns a pre-signed URL for downloading content
func (b *S3Backend) GetDownloadURL(ctx context.Context, objectKey string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	result, err := b.presignClient.PresignGetObject(ctx, input,
		s3.WithPresignExpires(b.presignDuration))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// Download downloads content directly from S3
func (b *S3Backend) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	result, err := b.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	return result.Body, nil
}

// Delete deletes content from S3
func (b *S3Backend) Delete(ctx context.Context, objectKey string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	_, err := b.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
