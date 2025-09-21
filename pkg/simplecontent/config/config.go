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
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
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
func (c *ServerConfig) BuildService() (simplecontent.Service, error) {
	var options []simplecontent.Option

	// Set up repository
	repo, err := c.buildRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to build repository: %w", err)
	}
	options = append(options, simplecontent.WithRepository(repo))

	// Set up storage backends
	for _, backendConfig := range c.StorageBackends {
		store, err := c.buildStorageBackend(backendConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build storage backend %s: %w", backendConfig.Name, err)
		}
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
