// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/tendant/simple-content/internal/storage"
)

const (
	LOCAL_S3_ENDPOINT = "http://localhost:9000"
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

type resolverV2 struct {
	s3Endpoint string
	s3Region   string
}

// S3BackendForTesting is a version of S3Backend that can be used for testing
// It allows injecting mock clients
type S3BackendForTesting struct {
	Client          s3ClientInterface
	PresignClient   presignClientInterface
	Bucket          string
	PresignDuration time.Duration
	EnableSSE       bool
	SSEAlgorithm    string
	SSEKMSKeyID     string
}

// Interfaces for mocking
type s3ClientInterface interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

type presignClientInterface interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*PresignOutput, error)
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*PresignOutput, error)
}

// PresignOutput is a simplified version of the AWS SDK's PresignedHTTPRequest
type PresignOutput struct {
	URL string
}

// GetUploadURL returns a pre-signed URL for uploading content
func (b *S3BackendForTesting) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(objectKey),
	}

	// Add server-side encryption if enabled
	if b.EnableSSE {
		if b.SSEAlgorithm == "AES256" {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		} else if b.SSEAlgorithm == "aws:kms" {
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.SSEKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.SSEKMSKeyID)
			}
		}
	}

	result, err := b.PresignClient.PresignPutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// Upload uploads content directly to S3
func (b *S3BackendForTesting) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(objectKey),
		Body:   reader,
	}

	// Add server-side encryption if enabled
	if b.EnableSSE {
		if b.SSEAlgorithm == "AES256" {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		} else if b.SSEAlgorithm == "aws:kms" {
			input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			if b.SSEKMSKeyID != "" {
				input.SSEKMSKeyId = aws.String(b.SSEKMSKeyID)
			}
		}
	}

	_, err := b.Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// GetPreviewURL returns a pre-signed URL for previewing content
func (b *S3BackendForTesting) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(objectKey),
	}

	result, err := b.PresignClient.PresignGetObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return result.URL, nil
}

// GetDownloadURL returns a pre-signed URL for downloading content
func (b *S3BackendForTesting) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {
	var dispositionFilename string
	if downloadFilename != "" {
		dispositionFilename = fmt.Sprintf(`filename="%s"`, downloadFilename)
	}

	contentDisposition := fmt.Sprintf("attachment;%s", dispositionFilename)
	input := &s3.GetObjectInput{
		Bucket:                     aws.String(b.Bucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String(contentDisposition),
	}

	result, err := b.PresignClient.PresignGetObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return result.URL, nil
}

// Download downloads content directly from S3
func (b *S3BackendForTesting) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(objectKey),
	}

	result, err := b.Client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	return result.Body, nil
}

// Delete deletes content from S3
func (b *S3BackendForTesting) Delete(ctx context.Context, objectKey string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(objectKey),
	}

	_, err := b.Client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GetObjectMeta retrieves metadata for an object in S3
func (b *S3BackendForTesting) GetObjectMeta(ctx context.Context, objectKey string) (*storage.ObjectMeta, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(objectKey),
	}

	result, err := b.Client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Convert S3-specific metadata to generic ObjectMeta
	meta := &storage.ObjectMeta{
		Key:      objectKey,
		Metadata: make(map[string]string),
	}

	// Handle nil pointers safely
	if result.ContentLength != nil {
		meta.Size = *result.ContentLength
	}

	if result.LastModified != nil {
		meta.UpdatedAt = *result.LastModified
	}

	if result.ContentType != nil {
		meta.ContentType = *result.ContentType
	}

	if result.ETag != nil {
		meta.ETag = *result.ETag
	}

	// Convert S3 metadata to generic metadata map
	for k, v := range result.Metadata {
		meta.Metadata[k] = v
	}

	return meta, nil
}

// Reference: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/#v2-endpointresolverv2--baseendpoint
func (r *resolverV2) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (smithyendpoints.Endpoint, error) {

	if params.Region != nil && *params.Region == r.s3Region {
		base, err := url.Parse(r.s3Endpoint)
		u := base.JoinPath(*params.Bucket)

		if err != nil {
			return smithyendpoints.Endpoint{}, err
		}
		return smithyendpoints.Endpoint{
			URI: *u,
		}, nil
	}

	return s3.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
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

	// Create AWS config
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with appropriate options
	s3ClientOptions := []func(*s3.Options){
		func(o *s3.Options) {
			if config.UsePathStyle {
				o.UsePathStyle = true
			}
		},
	}

	// Add custom endpoint resolver if endpoint is specified
	if config.Endpoint == LOCAL_S3_ENDPOINT {
		s3ClientOptions = append(s3ClientOptions, func(o *s3.Options) {
			o.EndpointResolverV2 = &resolverV2{
				s3Endpoint: config.Endpoint,
				s3Region:   config.Region,
			}
		})
	}

	// Create the S3 client with all options
	s3Client := s3.NewFromConfig(cfg, s3ClientOptions...)

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

// GetObjectMeta retrieves metadata for an object in S3
func (b *S3Backend) GetObjectMeta(ctx context.Context, objectKey string) (*storage.ObjectMeta, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(objectKey),
	}

	result, err := b.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Convert S3-specific metadata to generic ObjectMeta
	meta := &storage.ObjectMeta{
		Key:      objectKey,
		Metadata: make(map[string]string),
	}

	// Handle nil pointers safely
	if result.ContentLength != nil {
		meta.Size = *result.ContentLength
	}

	if result.LastModified != nil {
		meta.UpdatedAt = *result.LastModified
	}

	if result.ContentType != nil {
		meta.ContentType = *result.ContentType
	}

	if result.ETag != nil {
		meta.ETag = *result.ETag
	}

	// Convert S3 metadata to generic metadata map
	for k, v := range result.Metadata {
		meta.Metadata[k] = v
	}

	return meta, nil
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

func (b *S3Backend) UploadWithParams(ctx context.Context, reader io.Reader, params storage.UploadParams) error {
	uploader := manager.NewUploader(b.client)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(params.ObjectKey),
		Body:        reader,
		ContentType: aws.String(params.MimeType),
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
func (b *S3Backend) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {

	var dispositionFilename string
	if downloadFilename != "" {
		dispositionFilename = fmt.Sprintf(`filename="%s"`, downloadFilename)
	}

	contentDisposition := fmt.Sprintf("attachment;%s", dispositionFilename)
	input := &s3.GetObjectInput{
		Bucket:                     aws.String(b.bucket),
		Key:                        aws.String(objectKey),
		ResponseContentDisposition: aws.String(contentDisposition),
	}

	result, err := b.presignClient.PresignGetObject(ctx, input,
		s3.WithPresignExpires(b.presignDuration))
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return result.URL, nil
}

// GetPreviewURL returns a URL for previewing content
func (b *S3Backend) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
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
