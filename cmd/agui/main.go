package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/objectkey"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
	"github.com/tendant/simple-content/pkg/simplecontent/urlstrategy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewRootCommand() *cobra.Command {
	var configFile string
	var verbose bool

	rootCmd := &cobra.Command{
		Use:   "agui",
		Short: "AGUI Protocol CLI - Content Management Client",
		Long: `AGUI Protocol Command Line Interface
		
A CLI tool for content management using the simplecontent service directly.
Supports file upload, download, and content management operations.

Uses in-memory storage by default for quick testing and development.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (optional)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(NewUploadCommand())
	rootCmd.AddCommand(NewDownloadCommand())
	rootCmd.AddCommand(NewListCommand())
	rootCmd.AddCommand(NewDeleteCommand())
	rootCmd.AddCommand(NewMetadataCommand())

	return rootCmd
}

// NewServiceClientFromFlags creates a service client based on command flags and environment variables
func NewServiceClientFromFlags(cmd *cobra.Command) (*ServiceClient, error) {
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		log.Println("Initializing simplecontent service")
	}

	// Read environment variables (compatible with pkg/simplecontent/config)
	databaseURL := getEnv("DATABASE_URL", "memory")
	dbSchema := getEnv("DB_SCHEMA", "content")
	storageURL := getEnv("STORAGE_URL", "memory://")
	storageName := getEnv("STORAGE_NAME", "default")
	urlStrategyType := getEnv("URL_STRATEGY", "content-based")
	apiBaseURL := getEnv("API_BASE_URL", "/api/v1")

	// Parse database type from URL
	var databaseType string
	if databaseURL == "" || databaseURL == "memory" {
		databaseType = "memory"
		databaseURL = ""
	} else if len(databaseURL) >= 13 && databaseURL[:13] == "postgresql://" {
		databaseType = "postgres"
	} else if len(databaseURL) >= 11 && databaseURL[:11] == "postgres://" {
		databaseType = "postgres"
	} else {
		return nil, fmt.Errorf("unsupported DATABASE_URL format: %s", databaseURL)
	}

	// Parse storage type from URL
	storageType := "memory"
	if storageURL != "" && storageURL != "memory" && storageURL != "memory://" {
		if len(storageURL) >= 7 && storageURL[:7] == "file://" {
			storageType = "fs"
		} else if len(storageURL) >= 5 && storageURL[:5] == "s3://" {
			storageType = "s3"
		} else {
			return nil, fmt.Errorf("unsupported STORAGE_URL format: %s", storageURL)
		}
	}

	if verbose {
		log.Printf("Configuration from environment:")
		log.Printf("  Database: %s", databaseType)
		log.Printf("  Storage: %s", storageType)
		log.Printf("  Storage Name: %s", storageName)
		log.Printf("  URL Strategy: %s", urlStrategyType)
		if databaseURL != "" {
			log.Printf("  Database URL: %s", maskPassword(databaseURL))
		}
		if storageURL != "" && storageURL != "memory://" {
			log.Printf("  Storage URL: %s", storageURL)
		}
	}

	// 1. Initialize Repository (Database)
	var repo simplecontent.Repository
	var err error

	switch databaseType {
	case "postgres":
		if databaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required for postgres")
		}
		repo, err = initPostgresRepo(databaseURL, dbSchema, verbose)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize postgres repository: %w", err)
		}
	case "memory":
		repo = memory.New()
		if verbose {
			log.Println("Using in-memory repository")
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}

	// 2. Initialize Storage Backend (BlobStore)
	blobStore, err := initStorageBackend(storageType, storageURL, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage backend: %w", err)
	}

	// 3. Initialize Object Key Generator
	keyGenerator := objectkey.NewGitLikeGenerator()
	if verbose {
		log.Println("Using Git-like object key generator")
	}

	// 4. Initialize URL Strategy
	var urlStrategy urlstrategy.URLStrategy
	switch urlStrategyType {
	case "content-based":
		urlStrategy = urlstrategy.NewContentBasedStrategy(apiBaseURL)
		if verbose {
			log.Printf("Using content-based URL strategy with base URL: %s", apiBaseURL)
		}
	case "cdn":
		cdnBaseURL := getEnv("CDN_BASE_URL", "")
		if cdnBaseURL == "" {
			return nil, fmt.Errorf("CDN_BASE_URL is required for cdn strategy")
		}
		uploadBaseURL := getEnv("UPLOAD_BASE_URL", "")
		if uploadBaseURL != "" {
			urlStrategy = urlstrategy.NewCDNStrategyWithUpload(cdnBaseURL, uploadBaseURL)
		} else {
			urlStrategy = urlstrategy.NewCDNStrategy(cdnBaseURL)
		}
		if verbose {
			log.Printf("Using CDN URL strategy with base URL: %s", cdnBaseURL)
		}
	case "storage-delegated":
		blobStores := map[string]urlstrategy.BlobStore{
			storageName: blobStore,
		}
		urlStrategy = urlstrategy.NewStorageDelegatedStrategy(blobStores)
		if verbose {
			log.Println("Using storage-delegated URL strategy")
		}
	default:
		return nil, fmt.Errorf("unsupported URL strategy: %s", urlStrategyType)
	}

	// 5. Initialize Event Sink (optional)
	eventSink := simplecontent.NewNoopEventSink()

	// 6. Initialize Previewer (optional)
	previewer := simplecontent.NewBasicImagePreviewer()

	// 7. Create Service with all components
	service, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore(storageName, blobStore),
		simplecontent.WithObjectKeyGenerator(keyGenerator),
		simplecontent.WithURLStrategy(urlStrategy),
		simplecontent.WithEventSink(eventSink),
		simplecontent.WithPreviewer(previewer),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	if verbose {
		log.Println("Service initialized successfully")
	}

	return NewServiceClient(service, verbose), nil
}

// initPostgresRepo initializes a PostgreSQL repository
func initPostgresRepo(databaseURL, schema string, verbose bool) (simplecontent.Repository, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	// Set search_path if schema is provided
	if schema != "" {
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema))
			return err
		}
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	if verbose {
		log.Printf("Connected to PostgreSQL database (schema: %s)", schema)
	}

	return repopg.NewWithPool(pool), nil
}

// initStorageBackend initializes a storage backend based on type and URL
func initStorageBackend(storageType, storageURL string, verbose bool) (simplecontent.BlobStore, error) {
	switch storageType {
	case "memory":
		if verbose {
			log.Println("Using in-memory storage backend")
		}
		return memorystorage.New(), nil

	case "fs":
		// Parse file:///path from storageURL
		baseDir := "./data/storage"
		if len(storageURL) >= 7 && storageURL[:7] == "file://" {
			baseDir = storageURL[7:] // Remove "file://"
		}

		// Allow override via environment variables
		if envBaseDir := os.Getenv("FS_BASE_DIR"); envBaseDir != "" {
			baseDir = envBaseDir
		}

		urlPrefix := getEnv("FS_URL_PREFIX", "")
		secretKey := getEnv("FS_SIGNATURE_SECRET_KEY", "")
		presignExpires := getEnvInt("FS_PRESIGN_EXPIRES_SECONDS", 3600)

		fsConfig := fsstorage.Config{
			BaseDir:            baseDir,
			URLPrefix:          urlPrefix,
			SignatureSecretKey: secretKey,
			PresignExpires:     time.Duration(presignExpires) * time.Second,
		}

		if verbose {
			log.Printf("Using filesystem storage backend (base_dir: %s)", baseDir)
		}

		return fsstorage.New(fsConfig)

	case "s3":
		// Parse s3://bucket from storageURL
		bucket := ""
		if len(storageURL) >= 5 && storageURL[:5] == "s3://" {
			bucketPart := storageURL[5:] // Remove "s3://"
			// Find query string if present
			for i, ch := range bucketPart {
				if ch == '?' {
					bucket = bucketPart[:i]
					break
				}
			}
			if bucket == "" {
				bucket = bucketPart
			}
		}

		// Use standard AWS environment variables (compatible with pkg/simplecontent/config)
		s3Config := s3storage.Config{
			Region:                 getEnv("AWS_REGION", "us-east-1"),
			Bucket:                 bucket,
			AccessKeyID:            getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:        getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Endpoint:               getEnv("AWS_S3_ENDPOINT", ""),
			UseSSL:                 getEnvBool("AWS_S3_USE_SSL", true),
			UsePathStyle:           getEnvBool("AWS_S3_USE_PATH_STYLE", false),
			PresignDuration:        getEnvInt("AWS_S3_PRESIGN_DURATION", 3600),
			EnableSSE:              getEnvBool("AWS_S3_ENABLE_SSE", false),
			SSEAlgorithm:           getEnv("AWS_S3_SSE_ALGORITHM", "AES256"),
			SSEKMSKeyID:            getEnv("AWS_S3_SSE_KMS_KEY_ID", ""),
			CreateBucketIfNotExist: getEnvBool("AWS_S3_CREATE_BUCKET_IF_NOT_EXIST", false),
		}

		if s3Config.Bucket == "" {
			return nil, fmt.Errorf("S3 bucket is required (set via STORAGE_URL=s3://bucket or AWS_S3_BUCKET)")
		}

		if verbose {
			log.Printf("Using S3 storage backend (bucket: %s, region: %s)", s3Config.Bucket, s3Config.Region)
			if s3Config.Endpoint != "" {
				log.Printf("  S3 Endpoint: %s", s3Config.Endpoint)
			}
		}

		return s3storage.New(s3Config)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// maskPassword masks the password in a database URL for logging
func maskPassword(url string) string {
	// Simple masking for postgres URLs: postgresql://user:password@host/db
	// Find the password part and replace it with ***
	start := 0
	for i := 0; i < len(url); i++ {
		if url[i] == ':' && i > 0 {
			// Check if this is the password separator (after //)
			if i >= 2 && url[i-2:i] == "//" {
				continue
			}
			start = i + 1
		}
		if url[i] == '@' && start > 0 {
			return url[:start] + "***" + url[i:]
		}
	}
	return url
}
