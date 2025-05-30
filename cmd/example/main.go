package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
	"github.com/tendant/simple-content/internal/repository/memory"
	psqlrepo "github.com/tendant/simple-content/internal/repository/psql"
	"github.com/tendant/simple-content/internal/service"
	"github.com/tendant/simple-content/internal/storage/s3"
)

type DbConfig struct {
	Port     uint16 `env:"PC_CONTENT_PG_PORT" env-default:"5432"`
	Host     string `env:"PC_CONTENT_PG_HOST" env-default:"localhost"`
	Name     string `env:"PC_CONTENT_PG_NAME" env-default:"powercard_db"`
	User     string `env:"PC_CONTENT_PG_USER" env-default:"content"`
	Password string `env:"PC_CONTENT_PG_PASSWORD" env-default:"pwd"`
}

func (c DbConfig) toDatabaseUrl() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Name,
	}
	return u.String()
}

func NewDbPool(ctx context.Context, dbConfig DbConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dbConfig.toDatabaseUrl())
	return pool, err
}

func main() {

	// 1. Initialize database connection
	var config DbConfig
	cleanenv.ReadEnv(&config)

	dbconn, err := NewDbPool(context.Background(), config)
	if err != nil {
		slog.Error("Failed to connect to app db", "err", err)
		os.Exit(-1)
	}

	// 2. Initialize repositories using the repository factory
	repoFactory := psqlrepo.NewRepositoryFactory(dbconn)
	contentRepo := repoFactory.NewContentRepository()
	contentMetadataRepo := repoFactory.NewContentMetadataRepository()
	objectRepo := repoFactory.NewObjectRepository()
	objectMetadataRepo := repoFactory.NewObjectMetadataRepository()

	// Create a custom implementation for storage backend repository
	// since it's not provided by the factory
	storageBackendRepo := memory.NewStorageBackendRepository()

	// 3. Initialize S3 storage backend
	s3Backend, err := initializeS3Backend()
	if err != nil {
		log.Fatalf("Failed to initialize S3 backend: %v", err)
	}

	// 4. Initialize services
	contentService := service.NewContentService(
		contentRepo,
		contentMetadataRepo,
		objectRepo,
	)

	objectService := service.NewObjectService(
		objectRepo,
		objectMetadataRepo,
		storageBackendRepo,
		s3Backend,
	)

	// Register the S3 backend with the object service
	objectService.RegisterBackend("s3-default", s3Backend)

	// 5. Create storage backend in the database if it doesn't exist
	err = ensureStorageBackendExists(context.Background(), storageBackendRepo)
	if err != nil {
		log.Fatalf("Failed to ensure storage backend exists: %v", err)
	}

	// 6. Execute the complete content and object flow
	err = executeContentFlow(context.Background(), contentService, objectService)
	if err != nil {
		log.Fatalf("Content flow failed: %v", err)
	}

	log.Println("Content flow completed successfully!")
}
func initializeS3Backend() (*s3.S3Backend, error) {
	// Get S3 configuration from environment variables or use defaults
	region := getEnvOrDefault("S3_REGION", "us-east-1")
	bucket := getEnvOrDefault("S3_BUCKET", "mymusic")
	accessKey := getEnvOrDefault("S3_ACCESS_KEY", "minioadmin")
	secretKey := getEnvOrDefault("S3_SECRET_KEY", "minioadmin")
	endpoint := getEnvOrDefault("S3_ENDPOINT", "http://localhost:9000")
	useSSL := getEnvOrDefaultBool("S3_USE_SSL", false)
	usePathStyle := getEnvOrDefaultBool("S3_USE_PATH_STYLE", true)
	createBucket := getEnvOrDefaultBool("S3_CREATE_BUCKET", true)

	// Create S3 backend configuration
	config := s3.Config{
		Region:                 region,
		Bucket:                 bucket,
		AccessKeyID:            accessKey,
		SecretAccessKey:        secretKey,
		Endpoint:               endpoint,
		UseSSL:                 useSSL,
		UsePathStyle:           usePathStyle,
		CreateBucketIfNotExist: createBucket,
		PresignDuration:        3600, // 1 hour
	}

	// Initialize the S3 backend
	log.Printf("Initializing S3 backend with bucket '%s'...", bucket)
	backend, err := s3.NewS3Backend(config)
	if err != nil {
		return nil, err
	}

	log.Println("S3 backend initialized successfully!")
	return backend.(*s3.S3Backend), nil
}

func ensureStorageBackendExists(ctx context.Context, repo repository.StorageBackendRepository) error {
	// Check if the S3 storage backend already exists
	_, err := repo.Get(ctx, "s3-default")
	if err == nil {
		// Backend already exists
		log.Println("S3 storage backend already exists")
		return nil
	}

	// Create the S3 storage backend
	now := time.Now().UTC()
	s3Config := map[string]interface{}{
		"region":                     getEnvOrDefault("S3_REGION", "us-east-1"),
		"bucket":                     getEnvOrDefault("S3_BUCKET", "mymusic"),
		"endpoint":                   getEnvOrDefault("S3_ENDPOINT", "http://localhost:9000"),
		"use_ssl":                    getEnvOrDefaultBool("S3_USE_SSL", false),
		"use_path_style":             getEnvOrDefaultBool("S3_USE_PATH_STYLE", true),
		"create_bucket_if_not_exist": getEnvOrDefaultBool("S3_CREATE_BUCKET", true),
	}

	backend := &domain.StorageBackend{
		Name:      "s3-default",
		Type:      "s3",
		Config:    s3Config,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	log.Println("Creating S3 storage backend in the database...")
	err = repo.Create(ctx, backend)
	if err != nil {
		return err
	}

	log.Println("S3 storage backend created successfully!")
	return nil
}

func executeContentFlow(ctx context.Context, contentService *service.ContentService, objectService *service.ObjectService) error {
	// 1. Create a tenant and owner ID (in a real app, these would come from your auth system)
	tenantID := uuid.New()
	ownerID := uuid.New()
	log.Printf("Using tenant ID: %s", tenantID)
	log.Printf("Using owner ID: %s", ownerID)

	// 2. Create a new content
	log.Println("Creating new content...")
	content, err := contentService.CreateContent(ctx, ownerID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create content: %w", err)
	}
	log.Printf("Content created with ID: %s", content.ID)

	// 3. Set content metadata
	log.Println("Setting content metadata...")
	err = contentService.SetContentMetadata(
		ctx,
		content.ID,
		"image/jpeg",
		"Example Image",
		"This is an example image uploaded through the content flow",
		[]string{"example", "image", "test"},
		0, // File size will be updated later
		"example-user",
		map[string]interface{}{
			"source": "example-app",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to set content metadata: %w", err)
	}

	// 4. Create a new object for the content
	log.Println("Creating new object...")
	object, err := objectService.CreateObject(
		ctx,
		content.ID,
		"s3-default", // Use the S3 storage backend
		1,            // Version 1
	)
	if err != nil {
		return fmt.Errorf("failed to create object: %w", err)
	}
	log.Printf("Object created with ID: %s", object.ID)

	// 5. Get a sample image file to upload
	filePath := getEnvOrDefault("SAMPLE_IMAGE_PATH", "./receipt.jpg")
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open sample file: %w", err)
	}
	defer file.Close()

	// 6. Upload the object to S3
	log.Printf("Uploading file '%s' to object...", filePath)
	err = objectService.UploadObject(ctx, object.ID, file)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	log.Println("Object uploaded successfully!")

	// 7. Get object metadata
	fileInfo, err := objectService.GetObjectMetadata(ctx, object.ID)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}
	log.Println("Object metadata retrieved successfully!", fileInfo)

	// 8. Update content status to uploaded
	log.Println("Updating content status to uploaded...")
	content.Status = domain.ContentStatusUploaded
	content.UpdatedAt = time.Now().UTC()
	err = contentService.UpdateContent(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to update content status: %w", err)
	}

	// 9. Get a download URL for the object
	log.Println("Generating download URL...")
	downloadURL, err := objectService.GetDownloadURL(ctx, object.ID)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}
	log.Printf("Download URL: %s", downloadURL)

	// 10. Download the object to verify it was uploaded correctly
	log.Println("Downloading object to verify upload...")
	reader, err := objectService.DownloadObject(ctx, object.ID)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}
	defer reader.Close()

	// Read a small amount of data to verify the download works
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read from downloaded object: %w", err)
	}
	log.Printf("Successfully read %d bytes from downloaded object", n)

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}
