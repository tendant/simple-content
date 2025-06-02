package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tendant/simple-content/pkg/model"
	"github.com/tendant/simple-content/pkg/repository/memory"
	psqlrepo "github.com/tendant/simple-content/pkg/repository/psql"
	"github.com/tendant/simple-content/pkg/service"
	"github.com/tendant/simple-content/pkg/storage/s3"
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
		slog.Error("Failed to initialize S3 backend", "err", err)
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

	// Execute the complete content and object flow
	err = executeContentFlow(context.Background(), contentService, objectService)
	if err != nil {
		slog.Error("Content flow failed", "err", err)
	}

	slog.Info("Content flow completed successfully!")
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
	slog.Info("Initializing S3 backend with bucket '%s'...", "bucket", bucket)
	backend, err := s3.NewS3Backend(config)
	if err != nil {
		return nil, err
	}

	slog.Info("S3 backend initialized successfully!")
	return backend.(*s3.S3Backend), nil
}

func executeContentFlow(ctx context.Context, contentService *service.ContentService, objectService *service.ObjectService) error {
	// 1. Create a tenant and owner ID (in a real app, these would come from your auth system)
	tenantID := uuid.New()
	ownerID := uuid.New()
	slog.Info("Using tenant ID", "tenantID", tenantID)
	slog.Info("Using owner ID", "ownerID", ownerID)

	// 2. Create a new content
	slog.Info("Creating new content...")
	content, err := contentService.CreateContent(ctx, ownerID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create content: %w", err)
	}
	slog.Info("Content created with ID", "contentID", content.ID)

	// 3. Set content metadata
	slog.Info("Setting content metadata...")
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
	slog.Info("Creating new object...")
	object, err := objectService.CreateObject(
		ctx,
		content.ID,
		"s3-default", // Use the S3 storage backend
		1,            // Version 1
	)
	if err != nil {
		return fmt.Errorf("failed to create object: %w", err)
	}
	slog.Info("Object created with ID", "objectID", object.ID)

	// 5. Get a sample image file to upload
	filePath := getEnvOrDefault("SAMPLE_IMAGE_PATH", "./receipt.jpg")
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open sample file: %w", err)
	}
	defer file.Close()

	// 6. Upload the object to S3
	slog.Info("Uploading file to object...", "filePath", filePath)
	err = objectService.UploadObject(ctx, object.ID, file)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	slog.Info("Object uploaded successfully!")

	// 7. Get object metadata
	fileInfo, err := objectService.GetObjectMetadata(ctx, object.ID)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}
	slog.Info("Object metadata retrieved successfully!", "fileInfo", fileInfo)

	// 8. Update content status to uploaded
	slog.Info("Updating content status to uploaded...")
	content.Status = model.ContentStatusUploaded
	content.UpdatedAt = time.Now().UTC()
	err = contentService.UpdateContent(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to update content status: %w", err)
	}

	// 9. Get a download URL for the object
	slog.Info("Generating download URL...")
	downloadURL, err := objectService.GetDownloadURL(ctx, object.ID)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}
	slog.Info("Download URL", "downloadURL", downloadURL)

	// 10. Download the object to verify it was uploaded correctly
	slog.Info("Downloading object to verify upload...")
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
	slog.Info("Successfully read bytes from downloaded object", "n", n)

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
