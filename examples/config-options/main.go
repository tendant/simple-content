package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

func main() {
	fmt.Println("=== Configuration Options Example ===\n")

	// Example 1: Development configuration (programmatic)
	fmt.Println("1. Development Configuration (Programmatic)")
	devConfig, err := config.Load(
		config.WithPort("8080"),
		config.WithEnvironment("development"),
		config.WithDatabase("memory", ""),
		config.WithFilesystemStorage("fs", "./tmp/dev-storage", "/api/v1", "dev-secret"),
		config.WithDefaultStorage("fs"),
		config.WithContentBasedURLs("/api/v1"),
		config.WithObjectKeyGenerator("git-like"),
		config.WithEventLogging(true),
		config.WithPreviews(true),
		config.WithAdminAPI(true),
	)
	if err != nil {
		log.Fatalf("Failed to create dev config: %v", err)
	}
	fmt.Printf("   ✓ Port: %s\n", devConfig.Port)
	fmt.Printf("   ✓ Environment: %s\n", devConfig.Environment)
	fmt.Printf("   ✓ Database: %s\n", devConfig.DatabaseType)
	fmt.Printf("   ✓ Storage: %s\n", devConfig.DefaultStorageBackend)
	fmt.Printf("   ✓ URL Strategy: %s\n", devConfig.URLStrategy)
	fmt.Println()

	// Example 2: Testing configuration (minimal)
	fmt.Println("2. Testing Configuration (Minimal)")
	testConfig, err := config.Load(
		config.WithPort("0"), // Random port
		config.WithEnvironment("testing"),
		config.WithDatabase("memory", ""),
		config.WithMemoryStorage(""),
		config.WithDefaultStorage("memory"),
		config.WithContentBasedURLs("/api/v1"),
		config.WithEventLogging(false),
	)
	if err != nil {
		log.Fatalf("Failed to create test config: %v", err)
	}
	fmt.Printf("   ✓ Environment: %s\n", testConfig.Environment)
	fmt.Printf("   ✓ Database: %s\n", testConfig.DatabaseType)
	fmt.Printf("   ✓ Storage: %s\n", testConfig.DefaultStorageBackend)
	fmt.Println()

	// Example 3: S3 configuration (production-like)
	fmt.Println("3. S3 Configuration (Production-like)")
	s3Config, err := config.Load(
		config.WithPort("8080"),
		config.WithEnvironment("production"),
		config.WithDatabase("memory", ""), // Would be postgres in real production
		config.WithS3Storage("s3", "my-bucket", "us-west-2"),
		config.WithS3Endpoint("s3", "http://localhost:9000", false, true), // MinIO
		config.WithS3Credentials("s3", "minioadmin", "minioadmin"),
		config.WithS3PresignDuration("s3", 1800), // 30 minutes
		config.WithDefaultStorage("s3"),
		config.WithStorageDelegatedURLs(), // Use S3 presigned URLs
		config.WithObjectKeyGenerator("git-like"),
	)
	if err != nil {
		log.Fatalf("Failed to create S3 config: %v", err)
	}
	fmt.Printf("   ✓ Storage: %s\n", s3Config.DefaultStorageBackend)
	fmt.Printf("   ✓ URL Strategy: %s\n", s3Config.URLStrategy)
	fmt.Printf("   ✓ Storage backends: %d\n", len(s3Config.StorageBackends))
	fmt.Println()

	// Example 4: Actually use the service
	fmt.Println("4. Using the Service")
	svc, err := devConfig.BuildService()
	if err != nil {
		log.Fatalf("Failed to build service: %v", err)
	}

	ctx := context.Background()

	// Upload some content
	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:            uuid.New(),
		TenantID:           uuid.New(),
		Name:               "Example Document",
		Description:        "Configured via programmatic options",
		DocumentType:       "text/plain",
		StorageBackendName: "fs",
		Reader:             strings.NewReader("Hello from config options example!"),
		FileName:           "example.txt",
	})
	if err != nil {
		log.Fatalf("Failed to upload content: %v", err)
	}
	fmt.Printf("   ✓ Content uploaded: %s\n", content.ID)

	// Get content details
	details, err := svc.GetContentDetails(ctx, content.ID)
	if err != nil {
		log.Fatalf("Failed to get content details: %v", err)
	}
	fmt.Printf("   ✓ Download URL: %s\n", details.Download)
	fmt.Printf("   ✓ Status: Ready=%t\n", details.Ready)
	fmt.Println()

	// Example 5: Mixed configuration (programmatic + environment)
	fmt.Println("5. Mixed Configuration (Programmatic + Environment Override)")
	fmt.Println("   Set PORT=9090 in environment to see override")
	mixedConfig, err := config.Load(
		// Programmatic defaults
		config.WithPort("8080"),
		config.WithEnvironment("development"),
		config.WithDatabase("memory", ""),
		config.WithMemoryStorage(""),
		config.WithDefaultStorage("memory"),

		// Environment variables override programmatic settings
		config.WithEnv(""),
	)
	if err != nil {
		log.Fatalf("Failed to create mixed config: %v", err)
	}
	fmt.Printf("   ✓ Port: %s (from env if PORT is set, otherwise 8080)\n", mixedConfig.Port)
	fmt.Println()

	fmt.Println("=== Example Completed Successfully! ===")
}
