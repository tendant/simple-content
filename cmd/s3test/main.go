package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tendant/simple-content/pkg/simplecontent"
	s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)

// Helper functions to get environment variables with fallbacks to command-line flags
func getEnvOrFlag(envName, defaultValue string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	result := flag.String(strings.ToLower(strings.TrimPrefix(envName, "S3_")), defaultValue, "")
	return *result
}

func getEnvBoolOrFlag(envName string, defaultValue bool) bool {
	if value := os.Getenv(envName); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	flagName := strings.ToLower(strings.TrimPrefix(envName, "S3_"))
	result := flag.Bool(flagName, defaultValue, "")
	return *result
}

func getEnvIntOrFlag(envName string, defaultValue int) int {
	if value := os.Getenv(envName); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	flagName := strings.ToLower(strings.TrimPrefix(envName, "S3_"))
	result := flag.Int(flagName, defaultValue, "")
	return *result
}
func main() {

	var (
		region          = getEnvOrFlag("S3_REGION", "us-east-1")
		bucket          = getEnvOrFlag("S3_BUCKET", "mymusic")
		accessKey       = getEnvOrFlag("S3_ACCESS_KEY", "minioadmin")
		secretKey       = getEnvOrFlag("S3_SECRET_KEY", "minioadmin")
		endpoint        = getEnvOrFlag("S3_ENDPOINT", "http://localhost:9000")
		useSSL          = getEnvBoolOrFlag("S3_USE_SSL", true)
		usePathStyle    = getEnvBoolOrFlag("S3_USE_PATH_STYLE", false)
		enableSSE       = getEnvBoolOrFlag("S3_ENABLE_SSE", false)
		sseAlgorithm    = getEnvOrFlag("S3_SSE_ALGORITHM", "AES256")
		sseKMSKeyID     = getEnvOrFlag("S3_SSE_KMS_KEY_ID", "")
		presignDuration = getEnvIntOrFlag("S3_PRESIGN_DURATION", 3600)
		createBucket    = getEnvBoolOrFlag("S3_CREATE_BUCKET", false)
	)

	// Define commands
	command := flag.String("command", "help", "Command to execute: upload, download, delete, url-upload, url-download, help")
	objectKey := flag.String("key", "", "Object key for operations")
	filePath := flag.String("file", "", "File path for upload/download")

	// MinIO shortcut
	useMinio := flag.Bool("use-minio", false, "Use MinIO defaults (sets endpoint, path-style, etc.)")
	minioEndpoint := flag.String("minio-endpoint", "http://localhost:9000", "MinIO server endpoint")

	flag.Parse()

	// Apply MinIO defaults if requested
	if *useMinio {
		endpoint = *minioEndpoint
		useSSL = false
		usePathStyle = true
		createBucket = true
		if accessKey == "" {
			accessKey = "minioadmin"
		}
		if secretKey == "" {
			secretKey = "minioadmin"
		}
	}

	// Check for required parameters
	if bucket == "" && *command != "help" && *command != "" {
		log.Fatal("Bucket name is required")
	}

	// Check for environment variables if flags not provided
	if accessKey == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	}

	if secretKey == "" {
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	// Initialize S3 backend
	config := s3storage.Config{
		Region:                 region,
		Bucket:                 bucket,
		AccessKeyID:            accessKey,
		SecretAccessKey:        secretKey,
		Endpoint:               endpoint,
		UseSSL:                 useSSL,
		UsePathStyle:           usePathStyle,
		PresignDuration:        presignDuration,
		EnableSSE:              enableSSE,
		SSEAlgorithm:           sseAlgorithm,
		SSEKMSKeyID:            sseKMSKeyID,
		CreateBucketIfNotExist: createBucket,
	}

	// Skip backend initialization for help command
	var backend simplecontent.BlobStore
	var ctx context.Context

	if *command != "help" && *command != "" {
		fmt.Println("Initializing S3 backend with the following configuration:")
		fmt.Printf("  Region: %s\n", config.Region)
		fmt.Printf("  Bucket: %s\n", config.Bucket)
		fmt.Printf("  Endpoint: %s\n", config.Endpoint)
		fmt.Printf("  Use SSL: %v\n", config.UseSSL)
		fmt.Printf("  Use Path Style: %v\n", config.UsePathStyle)
		fmt.Printf("  Create Bucket If Not Exist: %v\n", config.CreateBucketIfNotExist)
		fmt.Printf("  Server-side Encryption: %v\n", config.EnableSSE)
		if config.EnableSSE {
			fmt.Printf("  SSE Algorithm: %s\n", config.SSEAlgorithm)
		}
		fmt.Println()

		var err error
		backend, err = s3storage.New(config)
		if err != nil {
			log.Fatalf("Failed to initialize S3 backend: %v", err)
		}

		ctx = context.Background()
	}

	// Execute command
	switch strings.ToLower(*command) {
	case "upload":
		if *objectKey == "" || *filePath == "" {
			log.Fatal("Object key and file path are required for upload")
		}

		file, err := os.Open(*filePath)
		if err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		fmt.Printf("Uploading %s to %s...\n", *filePath, *objectKey)
		startTime := time.Now()
		err = backend.Upload(ctx, *objectKey, file)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Upload failed: %v", err)
		}
		fmt.Printf("Upload successful (took %v)\n", duration)

	case "download":
		if *objectKey == "" || *filePath == "" {
			log.Fatal("Object key and file path are required for download")
		}

		fmt.Printf("Downloading %s to %s...\n", *objectKey, *filePath)
		startTime := time.Now()
		reader, err := backend.Download(ctx, *objectKey)
		if err != nil {
			log.Fatalf("Download failed: %v", err)
		}
		defer reader.Close()

		file, err := os.Create(*filePath)
		if err != nil {
			log.Fatalf("Failed to create file: %v", err)
		}
		defer file.Close()

		bytesWritten, err := io.Copy(file, reader)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Failed to write file: %v", err)
		}
		fmt.Printf("Download successful: %d bytes (took %v)\n", bytesWritten, duration)

	case "object-meta":
		if *objectKey == "" {
			log.Fatal("Object key is required for object metadata")
		}

		fmt.Printf("Getting metadata for %s...\n", *objectKey)
		startTime := time.Now()
		meta, err := backend.GetObjectMeta(ctx, *objectKey)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Failed to get object metadata: %v", err)
		}

		fmt.Println("Object Metadata:")
		fmt.Printf("  Key: %s\n", meta.Key)
		fmt.Printf("  Size: %d bytes\n", meta.Size)
		fmt.Printf("  Content Type: %s\n", meta.ContentType)
		fmt.Printf("  Last Modified: %v\n", meta.UpdatedAt)
		fmt.Printf("  ETag: %s\n", meta.ETag)

		if len(meta.Metadata) > 0 {
			fmt.Println("  Custom Metadata:")
			for k, v := range meta.Metadata {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		fmt.Printf("Retrieved in %v\n", duration)

	case "delete":
		if *objectKey == "" {
			log.Fatal("Object key is required for delete")
		}

		fmt.Printf("Deleting %s...\n", *objectKey)
		startTime := time.Now()
		var err error
		err = backend.Delete(ctx, *objectKey)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Delete failed: %v", err)
		}
		fmt.Printf("Delete successful (took %v)\n", duration)

	case "url-upload":
		if *objectKey == "" {
			log.Fatal("Object key is required for upload URL")
		}

		startTime := time.Now()
		url, err := backend.GetUploadURL(ctx, *objectKey)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Failed to get upload URL: %v", err)
		}
		fmt.Printf("Upload URL for %s (valid for %d seconds):\n%s\n",
			*objectKey, presignDuration, url)
		fmt.Printf("Generated in %v\n", duration)
		fmt.Println("\nTo use this URL with curl:")
		fmt.Printf("curl -X PUT -T your-file.txt \"%s\"\n", url)

	case "url-download":
		if *objectKey == "" {
			log.Fatal("Object key is required for download URL")
		}

		startTime := time.Now()
		url, err := backend.GetDownloadURL(ctx, *objectKey, "test-download.jpg")
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Failed to get download URL: %v", err)
		}
		fmt.Printf("Download URL for %s (valid for %d seconds):\n%s\n",
			*objectKey, presignDuration, url)
		fmt.Printf("Generated in %v\n", duration)
		fmt.Println("\nTo use this URL with curl:")
		fmt.Printf("curl \"%s\" -o downloaded-file.txt\n", url)

	case "help", "":
		fmt.Println("S3 Backend Test Application")
		fmt.Println("\nCommands:")
		fmt.Println("  upload        Upload a file to S3")
		fmt.Println("  download      Download a file from S3")
		fmt.Println("  delete        Delete an object from S3")
		fmt.Println("  url-upload    Generate a pre-signed upload URL")
		fmt.Println("  url-download  Generate a pre-signed download URL")
		fmt.Println("  help          Show this help message")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  Upload a file to AWS S3:")
		fmt.Println("    s3test -bucket my-bucket -access-key AKIAXXXX -secret-key XXXX -command upload -key test/file.txt -file ./local-file.txt")
		fmt.Println("\n  Upload a file to MinIO:")
		fmt.Println("    s3test -use-minio -bucket my-bucket -command upload -key test/file.txt -file ./local-file.txt")
		fmt.Println("\n  Generate a pre-signed download URL:")
		fmt.Println("    s3test -bucket my-bucket -command url-download -key test/file.txt")

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}
