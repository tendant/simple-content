package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tendant/simple-content/internal/storage"
	"github.com/tendant/simple-content/internal/storage/s3"
)

func main() {
	// Define command-line flags
	region := flag.String("region", "us-east-1", "AWS region")
	bucket := flag.String("bucket", "", "S3 bucket name")
	accessKey := flag.String("access-key", "", "AWS access key ID")
	secretKey := flag.String("secret-key", "", "AWS secret access key")
	endpoint := flag.String("endpoint", "", "Custom S3 endpoint (for MinIO, etc.)")
	useSSL := flag.Bool("use-ssl", true, "Use SSL for connections")
	usePathStyle := flag.Bool("use-path-style", false, "Use path-style addressing")
	enableSSE := flag.Bool("enable-sse", false, "Enable server-side encryption")
	sseAlgorithm := flag.String("sse-algorithm", "AES256", "SSE algorithm (AES256 or aws:kms)")
	sseKMSKeyID := flag.String("sse-kms-key-id", "", "KMS key ID for aws:kms algorithm")
	presignDuration := flag.Int("presign-duration", 3600, "Duration in seconds for presigned URLs")
	createBucket := flag.Bool("create-bucket", false, "Create bucket if it doesn't exist")

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
		*endpoint = *minioEndpoint
		*useSSL = false
		*usePathStyle = true
		*createBucket = true
		if *accessKey == "" {
			*accessKey = "minioadmin"
		}
		if *secretKey == "" {
			*secretKey = "minioadmin"
		}
	}

	// Check for required parameters
	if *bucket == "" && *command != "help" && *command != "" {
		log.Fatal("Bucket name is required")
	}

	// Check for environment variables if flags not provided
	if *accessKey == "" {
		*accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	}

	if *secretKey == "" {
		*secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	// Initialize S3 backend
	config := s3.Config{
		Region:                 *region,
		Bucket:                 *bucket,
		AccessKeyID:            *accessKey,
		SecretAccessKey:        *secretKey,
		Endpoint:               *endpoint,
		UseSSL:                 *useSSL,
		UsePathStyle:           *usePathStyle,
		PresignDuration:        *presignDuration,
		EnableSSE:              *enableSSE,
		SSEAlgorithm:           *sseAlgorithm,
		SSEKMSKeyID:            *sseKMSKeyID,
		CreateBucketIfNotExist: *createBucket,
	}

	// Skip backend initialization for help command
	var backend storage.Backend
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
		backend, err = s3.NewS3Backend(config)
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
			*objectKey, *presignDuration, url)
		fmt.Printf("Generated in %v\n", duration)
		fmt.Println("\nTo use this URL with curl:")
		fmt.Printf("curl -X PUT -T your-file.txt \"%s\"\n", url)

	case "url-download":
		if *objectKey == "" {
			log.Fatal("Object key is required for download URL")
		}

		startTime := time.Now()
		url, err := backend.GetDownloadURL(ctx, *objectKey)
		duration := time.Since(startTime)
		if err != nil {
			log.Fatalf("Failed to get download URL: %v", err)
		}
		fmt.Printf("Download URL for %s (valid for %d seconds):\n%s\n",
			*objectKey, *presignDuration, url)
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
