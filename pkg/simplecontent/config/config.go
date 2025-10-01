package config

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/objectkey"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
	"github.com/tendant/simple-content/pkg/simplecontent/urlstrategy"
)

// Option applies configuration to a ServerConfig instance.
type Option func(*ServerConfig) error

// Load constructs a ServerConfig by applying the supplied options on top of library defaults.
func Load(opts ...Option) (*ServerConfig, error) {
	cfg := defaults()

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadServerConfig is a convenience function that loads configuration from environment variables.
// It applies the default config and then overlays environment variable settings.
func LoadServerConfig() (*ServerConfig, error) {
	return Load(WithEnv(""))
}

func defaults() ServerConfig {
	return ServerConfig{
		Port:                  "8080",
		Environment:           "development",
		DatabaseType:          "memory",
		DBSchema:              "content",
		DefaultStorageBackend: "memory",
		StorageBackends: []StorageBackendConfig{
			{
				Name:   "memory",
				Type:   "memory",
				Config: map[string]interface{}{},
			},
		},
		EnableEventLogging: true,
		EnablePreviews:     true,
		URLStrategy:        "content-based", // Default URL strategy
		APIBaseURL:         "/api/v1",       // Default API base URL
		ObjectKeyGenerator: "git-like",      // Default to Git-like for better performance
	}
}

// ServerConfig represents server configuration for the simple-content service
type ServerConfig struct {
	Port        string
	Environment string // development, production, testing

	// Database configuration
	DatabaseURL  string
	DatabaseType string // "memory", "postgres"
	DBSchema     string // Postgres schema to use (default: content)

	// Storage configuration
	DefaultStorageBackend string
	StorageBackends       []StorageBackendConfig

	// Server options
	EnableEventLogging bool
	EnablePreviews     bool
	EnableAdminAPI     bool // Enable admin API endpoints (requires authentication in production)

	// URL generation
	URLStrategy     string // "cdn", "content-based", "storage-delegated"
	CDNBaseURL      string // Base URL for CDN strategy downloads (e.g., "https://cdn.example.com")
	UploadBaseURL   string // Base URL for CDN strategy uploads (e.g., "https://api.example.com" or "/api/v1")
	APIBaseURL      string // Base URL for content-based strategy (e.g., "/api/v1")

	// Object key generation
	ObjectKeyGenerator string // "default", "git-like", "tenant-aware", "legacy"
}

// StorageBackendConfig represents configuration for a storage backend
type StorageBackendConfig struct {
	Name   string
	Type   string // "memory", "fs", "s3"
	Config map[string]interface{}
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c.Port == "" {
		return errors.New("port is required")
	}

	if c.DatabaseType != "memory" && c.DatabaseType != "postgres" {
		return errors.New("database_type must be 'memory' or 'postgres'")
	}

	if c.DatabaseType == "postgres" && c.DatabaseURL == "" {
		return errors.New("database_url is required when using postgres")
	}

	// Ensure default storage backend exists in configured backends
	found := false
	for _, backend := range c.StorageBackends {
		if backend.Name == c.DefaultStorageBackend {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("default storage backend '%s' not found in configured backends", c.DefaultStorageBackend)
	}

	return nil
}

// BuildService creates a Service instance from the server configuration
// BuildRepository builds just the repository from configuration
func (c *ServerConfig) BuildRepository() (simplecontent.Repository, error) {
	return c.buildRepository()
}

func (c *ServerConfig) BuildService() (simplecontent.Service, error) {
	var options []simplecontent.Option

	// Set up repository
	repo, err := c.buildRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to build repository: %w", err)
	}
	options = append(options, simplecontent.WithRepository(repo))

	// Set up storage backends
	blobStores := make(map[string]simplecontent.BlobStore)
	for _, backendConfig := range c.StorageBackends {
		store, err := c.buildStorageBackend(backendConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build storage backend %s: %w", backendConfig.Name, err)
		}
		blobStores[backendConfig.Name] = store
		options = append(options, simplecontent.WithBlobStore(backendConfig.Name, store))
	}

	// Set up event sink
	if c.EnableEventLogging {
		eventSink := simplecontent.NewNoopEventSink() // In a real implementation, you'd use a proper logger
		options = append(options, simplecontent.WithEventSink(eventSink))
	}

	// Set up previewer
	if c.EnablePreviews {
		previewer := simplecontent.NewBasicImagePreviewer()
		options = append(options, simplecontent.WithPreviewer(previewer))
	}

	// Set up object key generator
	keyGenerator, err := c.buildObjectKeyGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to build object key generator: %w", err)
	}
	options = append(options, simplecontent.WithObjectKeyGenerator(keyGenerator))

	// Set up URL strategy
	urlStrategy, err := c.buildURLStrategyWithBlobStores(blobStores)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL strategy: %w", err)
	}
	options = append(options, simplecontent.WithURLStrategy(urlStrategy))

	return simplecontent.New(options...)
}

// buildRepository creates a Repository based on the configuration
func (c *ServerConfig) buildRepository() (simplecontent.Repository, error) {
	switch c.DatabaseType {
	case "memory":
		return memory.New(), nil
	case "postgres":
		if c.DatabaseURL == "" {
			return nil, errors.New("database_url is required for postgres")
		}
		cfg, err := pgxpool.ParseConfig(c.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
		}
		// Optionally set search_path for the connection
		schema := c.DBSchema
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			if schema == "" {
				return nil
			}
			// set search_path for this session
			_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema))
			return err
		}
		pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create pgx pool: %w", err)
		}
		return repopg.NewWithPool(pool), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.DatabaseType)
	}
}

// PingPostgres verifies connectivity to Postgres and optionally sets search_path for the session.
// It fails if the schema (when provided) does not exist.
func PingPostgres(databaseURL, schema string) error {
	if databaseURL == "" {
		return errors.New("database_url is required")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}
	if schema != "" {
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema))
			return err
		}
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("failed to create pgx pool: %w", err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

// buildStorageBackend creates a BlobStore based on the backend configuration
func (c *ServerConfig) buildStorageBackend(config StorageBackendConfig) (simplecontent.BlobStore, error) {
	switch config.Type {
	case "memory":
		return memorystorage.New(), nil

	case "fs":
		fsConfig := fsstorage.Config{
			BaseDir:   getString(config.Config, "base_dir", "./data/storage"),
			URLPrefix: getString(config.Config, "url_prefix", ""),
		}
		return fsstorage.New(fsConfig)

	case "s3":
		s3Config := s3storage.Config{
			Region:                 getString(config.Config, "region", "us-east-1"),
			Bucket:                 getString(config.Config, "bucket", ""),
			AccessKeyID:            getString(config.Config, "access_key_id", ""),
			SecretAccessKey:        getString(config.Config, "secret_access_key", ""),
			Endpoint:               getString(config.Config, "endpoint", ""),
			UseSSL:                 getBool(config.Config, "use_ssl", true),
			UsePathStyle:           getBool(config.Config, "use_path_style", false),
			PresignDuration:        getInt(config.Config, "presign_duration", 3600),
			EnableSSE:              getBool(config.Config, "enable_sse", false),
			SSEAlgorithm:           getString(config.Config, "sse_algorithm", "AES256"),
			SSEKMSKeyID:            getString(config.Config, "sse_kms_key_id", ""),
			CreateBucketIfNotExist: getBool(config.Config, "create_bucket_if_not_exist", false),
		}
		return s3storage.New(s3Config)

	default:
		return nil, fmt.Errorf("unsupported storage backend type: %s", config.Type)
	}
}

func getString(config map[string]interface{}, key string, defaultValue string) string {
	if value, exists := config[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := config[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
		if str, ok := value.(string); ok {
			if b, err := strconv.ParseBool(str); err == nil {
				return b
			}
		}
	}
	return defaultValue
}

func getInt(config map[string]interface{}, key string, defaultValue int) int {
	if value, exists := config[key]; exists {
		if i, ok := value.(int); ok {
			return i
		}
		if str, ok := value.(string); ok {
			if i, err := strconv.Atoi(str); err == nil {
				return i
			}
		}
		if f, ok := value.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

// buildObjectKeyGenerator creates an ObjectKey Generator based on the configuration
func (c *ServerConfig) buildObjectKeyGenerator() (objectkey.Generator, error) {
	switch c.ObjectKeyGenerator {
	case "legacy":
		return objectkey.NewLegacyGenerator(), nil
	case "git-like", "default", "":
		return objectkey.NewGitLikeGenerator(), nil
	case "tenant-aware":
		return objectkey.NewTenantAwareGitLikeGenerator(), nil
	case "high-performance":
		return objectkey.NewHighPerformanceGenerator(), nil
	default:
		return nil, fmt.Errorf("unsupported object key generator: %s", c.ObjectKeyGenerator)
	}
}

// buildURLStrategyWithBlobStores creates a URL strategy based on the configuration with blob stores
func (c *ServerConfig) buildURLStrategyWithBlobStores(blobStores map[string]simplecontent.BlobStore) (urlstrategy.URLStrategy, error) {
	switch c.URLStrategy {
	case "cdn":
		if c.CDNBaseURL == "" {
			return nil, fmt.Errorf("CDN base URL is required for CDN strategy")
		}
		if c.UploadBaseURL != "" {
			return urlstrategy.NewCDNStrategyWithUpload(c.CDNBaseURL, c.UploadBaseURL), nil
		}
		return urlstrategy.NewCDNStrategy(c.CDNBaseURL), nil

	case "content-based", "default", "":
		apiBaseURL := c.APIBaseURL
		if apiBaseURL == "" {
			apiBaseURL = "/api/v1" // Default fallback
		}
		return urlstrategy.NewContentBasedStrategy(apiBaseURL), nil

	case "storage-delegated":
		// Convert simplecontent.BlobStore to urlstrategy.BlobStore
		urlBlobStores := make(map[string]urlstrategy.BlobStore)
		for name, store := range blobStores {
			urlBlobStores[name] = store // This works because the interface methods match
		}
		return urlstrategy.NewStorageDelegatedStrategy(urlBlobStores), nil

	default:
		return nil, fmt.Errorf("unsupported URL strategy: %s", c.URLStrategy)
	}
}
