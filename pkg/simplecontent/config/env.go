package config

import (
	"fmt"
	"os"
	"strconv"
)

// WithEnv applies environment variable overrides using the provided prefix.
//
// Simplified environment variable mapping:
//
// Server (cmd/server-configured only):
//   PORT - Server port (default: "8080")
//   ENVIRONMENT - Runtime environment (default: "development")
//
// Database:
//   DATABASE_URL - Connection string (e.g., "postgresql://user:pass@host/db")
//                  If set with "postgresql://" prefix, automatically sets DATABASE_TYPE=postgres
//                  If empty or "memory", uses in-memory database
//
// Storage:
//   STORAGE_URL - Storage connection string (one of):
//                 - "memory://" - In-memory storage (default)
//                 - "file:///path/to/data" - Filesystem storage
//                 - "s3://bucket?region=us-east-1" - S3 storage
//
// That's it! Use programmatic config for advanced features.
func WithEnv(prefix string) Option {
	return func(c *ServerConfig) error {
		// Server-level config (cmd/server-configured only)
		if v, ok := lookupEnv(prefix, "PORT"); ok && v != "" {
			c.Port = v
		}
		if v, ok := lookupEnv(prefix, "ENVIRONMENT"); ok && v != "" {
			c.Environment = v
		}

		// Database config
		if err := applyDatabaseEnv(prefix, c); err != nil {
			return err
		}

		// Storage config
		if err := applyStorageEnv(prefix, c); err != nil {
			return err
		}

		return nil
	}
}

// applyDatabaseEnv applies database configuration from environment
func applyDatabaseEnv(prefix string, c *ServerConfig) error {
	dbURL, hasURL := lookupEnv(prefix, "DATABASE_URL")

	if !hasURL || dbURL == "" || dbURL == "memory" {
		// Default to memory
		c.DatabaseType = "memory"
		c.DatabaseURL = ""
		return nil
	}

	// Auto-detect database type from URL
	if len(dbURL) > 13 && dbURL[:13] == "postgresql://" {
		c.DatabaseType = "postgres"
		c.DatabaseURL = dbURL
	} else if len(dbURL) > 11 && dbURL[:11] == "postgres://" {
		c.DatabaseType = "postgres"
		c.DatabaseURL = dbURL
	} else {
		return fmt.Errorf("unsupported DATABASE_URL format: %s (use 'memory' or 'postgresql://...')", dbURL)
	}

	return nil
}

// applyStorageEnv applies storage configuration from environment
func applyStorageEnv(prefix string, c *ServerConfig) error {
	storageURL, hasURL := lookupEnv(prefix, "STORAGE_URL")

	if !hasURL || storageURL == "" || storageURL == "memory" || storageURL == "memory://" {
		// Default to memory storage
		c.DefaultStorageBackend = "memory"
		backend := StorageBackendConfig{
			Name:   "memory",
			Type:   "memory",
			Config: map[string]interface{}{},
		}
		c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		return nil
	}

	// Parse storage URL
	if len(storageURL) > 7 && storageURL[:7] == "file://" {
		return applyFilesystemStorage(storageURL, c)
	} else if len(storageURL) > 5 && storageURL[:5] == "s3://" {
		return applyS3Storage(storageURL, prefix, c)
	}

	return fmt.Errorf("unsupported STORAGE_URL format: %s (use 'memory://', 'file://...', or 's3://...')", storageURL)
}

// applyFilesystemStorage configures filesystem storage from URL
// Format: file:///path/to/data
func applyFilesystemStorage(url string, c *ServerConfig) error {
	// Extract path (remove file:// prefix)
	path := url[7:] // Remove "file://"
	if path == "" {
		return fmt.Errorf("filesystem path cannot be empty in STORAGE_URL")
	}

	backend := StorageBackendConfig{
		Name: "fs",
		Type: "fs",
		Config: map[string]interface{}{
			"base_dir": path,
		},
	}

	c.DefaultStorageBackend = "fs"
	c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
	return nil
}

// applyS3Storage configures S3 storage from URL
// Format: s3://bucket?region=us-east-1&endpoint=http://localhost:9000
func applyS3Storage(url string, prefix string, c *ServerConfig) error {
	// Simple parsing: extract bucket name
	// Format: s3://bucket or s3://bucket?params
	bucket := url[5:] // Remove "s3://"

	// Find query string if present
	queryIdx := -1
	for i, ch := range bucket {
		if ch == '?' {
			queryIdx = i
			break
		}
	}

	bucketName := bucket
	if queryIdx > 0 {
		bucketName = bucket[:queryIdx]
	}

	if bucketName == "" {
		return fmt.Errorf("S3 bucket name cannot be empty in STORAGE_URL")
	}

	backend := StorageBackendConfig{
		Name: "s3",
		Type: "s3",
		Config: map[string]interface{}{
			"bucket": bucketName,
			"region": "us-east-1", // Default
		},
	}

	// Check for AWS credentials in environment
	if accessKey, ok := os.LookupEnv("AWS_ACCESS_KEY_ID"); ok && accessKey != "" {
		backend.Config["access_key_id"] = accessKey
	}
	if secretKey, ok := os.LookupEnv("AWS_SECRET_ACCESS_KEY"); ok && secretKey != "" {
		backend.Config["secret_access_key"] = secretKey
	}
	if region, ok := os.LookupEnv("AWS_REGION"); ok && region != "" {
		backend.Config["region"] = region
	}

	c.DefaultStorageBackend = "s3"
	c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
	return nil
}

func lookupEnv(prefix, key string) (string, bool) {
	return os.LookupEnv(prefix + key)
}

func parseBoolEnv(prefix, key string) (bool, bool, error) {
	raw, ok := lookupEnv(prefix, key)
	if !ok || raw == "" {
		return false, false, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, fmt.Errorf("invalid boolean for %s%s: %w", prefix, key, err)
	}
	return parsed, true, nil
}

func parseIntEnv(prefix, key string) (int, bool, error) {
	raw, ok := lookupEnv(prefix, key)
	if !ok || raw == "" {
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, fmt.Errorf("invalid integer for %s%s: %w", prefix, key, err)
	}
	return parsed, true, nil
}

func upsertStorageBackend(backends []StorageBackendConfig, backend StorageBackendConfig) []StorageBackendConfig {
	if backend.Config == nil {
		backend.Config = map[string]interface{}{}
	}
	for i := range backends {
		if backends[i].Name == backend.Name {
			backends[i] = backend
			return backends
		}
	}
	return append(backends, backend)
}
