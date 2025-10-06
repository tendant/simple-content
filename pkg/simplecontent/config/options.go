package config

import (
	"fmt"
)

// WithPort sets the server port
func WithPort(port string) Option {
	return func(c *ServerConfig) error {
		if port == "" {
			return fmt.Errorf("port cannot be empty")
		}
		c.Port = port
		return nil
	}
}

// WithEnvironment sets the environment (development, production, testing)
func WithEnvironment(env string) Option {
	return func(c *ServerConfig) error {
		if env == "" {
			return fmt.Errorf("environment cannot be empty")
		}
		c.Environment = env
		return nil
	}
}

// WithDatabase configures the database backend
func WithDatabase(dbType, url string) Option {
	return func(c *ServerConfig) error {
		if dbType != "memory" && dbType != "postgres" {
			return fmt.Errorf("database type must be 'memory' or 'postgres', got: %s", dbType)
		}
		if dbType == "postgres" && url == "" {
			return fmt.Errorf("database URL is required for postgres")
		}
		c.DatabaseType = dbType
		c.DatabaseURL = url
		return nil
	}
}

// WithDatabaseSchema sets the database schema (for Postgres)
func WithDatabaseSchema(schema string) Option {
	return func(c *ServerConfig) error {
		c.DBSchema = schema
		return nil
	}
}

// WithDefaultStorage sets the default storage backend name
func WithDefaultStorage(name string) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			return fmt.Errorf("default storage backend name cannot be empty")
		}
		c.DefaultStorageBackend = name
		return nil
	}
}

// WithFilesystemStorage adds a filesystem storage backend
// If name is empty, defaults to "fs"
func WithFilesystemStorage(name, baseDir, urlPrefix, secretKey string) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "fs"
		}
		if baseDir == "" {
			return fmt.Errorf("filesystem base directory cannot be empty")
		}

		backend := StorageBackendConfig{
			Name: name,
			Type: "fs",
			Config: map[string]interface{}{
				"base_dir": baseDir,
			},
		}

		if urlPrefix != "" {
			backend.Config["url_prefix"] = urlPrefix
		}
		if secretKey != "" {
			backend.Config["signature_secret_key"] = secretKey
		}

		c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		return nil
	}
}

// WithFilesystemStorageExpiry sets the presigned URL expiry for filesystem storage
func WithFilesystemStorageExpiry(name string, expirySeconds int) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "fs"
		}
		if expirySeconds <= 0 {
			return fmt.Errorf("expiry seconds must be positive, got: %d", expirySeconds)
		}

		// Find existing backend or create new one
		for i := range c.StorageBackends {
			if c.StorageBackends[i].Name == name && c.StorageBackends[i].Type == "fs" {
				c.StorageBackends[i].Config["presign_expires_seconds"] = expirySeconds
				return nil
			}
		}

		// Backend doesn't exist yet, create it with minimal config
		backend := StorageBackendConfig{
			Name: name,
			Type: "fs",
			Config: map[string]interface{}{
				"presign_expires_seconds": expirySeconds,
			},
		}
		c.StorageBackends = append(c.StorageBackends, backend)
		return nil
	}
}

// WithS3Storage adds an S3 storage backend
// If name is empty, defaults to "s3"
func WithS3Storage(name, bucket, region string) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "s3"
		}
		if bucket == "" {
			return fmt.Errorf("S3 bucket cannot be empty")
		}
		if region == "" {
			region = "us-east-1" // Default region
		}

		backend := StorageBackendConfig{
			Name: name,
			Type: "s3",
			Config: map[string]interface{}{
				"bucket": bucket,
				"region": region,
			},
		}

		c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		return nil
	}
}

// WithS3Credentials sets AWS credentials for S3 storage
func WithS3Credentials(name, accessKeyID, secretAccessKey string) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "s3"
		}

		// Find existing S3 backend or create new one
		for i := range c.StorageBackends {
			if c.StorageBackends[i].Name == name && c.StorageBackends[i].Type == "s3" {
				c.StorageBackends[i].Config["access_key_id"] = accessKeyID
				c.StorageBackends[i].Config["secret_access_key"] = secretAccessKey
				return nil
			}
		}

		// Backend doesn't exist yet, create it with minimal config
		backend := StorageBackendConfig{
			Name: name,
			Type: "s3",
			Config: map[string]interface{}{
				"access_key_id":     accessKeyID,
				"secret_access_key": secretAccessKey,
			},
		}
		c.StorageBackends = append(c.StorageBackends, backend)
		return nil
	}
}

// WithS3Endpoint sets a custom S3 endpoint (for MinIO, LocalStack, etc.)
func WithS3Endpoint(name, endpoint string, useSSL, usePathStyle bool) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "s3"
		}

		// Find existing S3 backend or create new one
		for i := range c.StorageBackends {
			if c.StorageBackends[i].Name == name && c.StorageBackends[i].Type == "s3" {
				c.StorageBackends[i].Config["endpoint"] = endpoint
				c.StorageBackends[i].Config["use_ssl"] = useSSL
				c.StorageBackends[i].Config["use_path_style"] = usePathStyle
				return nil
			}
		}

		// Backend doesn't exist yet, create it with minimal config
		backend := StorageBackendConfig{
			Name: name,
			Type: "s3",
			Config: map[string]interface{}{
				"endpoint":        endpoint,
				"use_ssl":         useSSL,
				"use_path_style":  usePathStyle,
			},
		}
		c.StorageBackends = append(c.StorageBackends, backend)
		return nil
	}
}

// WithS3PresignDuration sets the presigned URL duration for S3 (in seconds)
func WithS3PresignDuration(name string, durationSeconds int) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "s3"
		}
		if durationSeconds <= 0 {
			return fmt.Errorf("presign duration must be positive, got: %d", durationSeconds)
		}

		// Find existing S3 backend or create new one
		for i := range c.StorageBackends {
			if c.StorageBackends[i].Name == name && c.StorageBackends[i].Type == "s3" {
				c.StorageBackends[i].Config["presign_duration"] = durationSeconds
				return nil
			}
		}

		// Backend doesn't exist yet, create it with minimal config
		backend := StorageBackendConfig{
			Name: name,
			Type: "s3",
			Config: map[string]interface{}{
				"presign_duration": durationSeconds,
			},
		}
		c.StorageBackends = append(c.StorageBackends, backend)
		return nil
	}
}

// WithMemoryStorage adds a memory storage backend (for testing)
// If name is empty, defaults to "memory"
func WithMemoryStorage(name string) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "memory"
		}

		backend := StorageBackendConfig{
			Name:   name,
			Type:   "memory",
			Config: map[string]interface{}{},
		}

		c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		return nil
	}
}

// WithContentBasedURLs configures content-based URL strategy
func WithContentBasedURLs(apiBaseURL string) Option {
	return func(c *ServerConfig) error {
		if apiBaseURL == "" {
			return fmt.Errorf("API base URL cannot be empty for content-based strategy")
		}
		c.URLStrategy = "content-based"
		c.APIBaseURL = apiBaseURL
		return nil
	}
}

// WithCDNURLs configures CDN URL strategy with hybrid upload support
func WithCDNURLs(cdnBaseURL, uploadBaseURL string) Option {
	return func(c *ServerConfig) error {
		if cdnBaseURL == "" {
			return fmt.Errorf("CDN base URL cannot be empty for CDN strategy")
		}
		c.URLStrategy = "cdn"
		c.CDNBaseURL = cdnBaseURL
		c.UploadBaseURL = uploadBaseURL
		return nil
	}
}

// WithStorageDelegatedURLs configures storage-delegated URL strategy
// This delegates URL generation to the storage backends (e.g., presigned S3/FS URLs)
func WithStorageDelegatedURLs() Option {
	return func(c *ServerConfig) error {
		c.URLStrategy = "storage-delegated"
		return nil
	}
}

// WithObjectKeyGenerator sets the object key generation strategy
// Valid values: "git-like", "tenant-aware", "high-performance", "legacy"
func WithObjectKeyGenerator(generator string) Option {
	return func(c *ServerConfig) error {
		validGenerators := map[string]bool{
			"git-like":         true,
			"tenant-aware":     true,
			"high-performance": true,
			"legacy":           true,
			"default":          true,
		}
		if !validGenerators[generator] {
			return fmt.Errorf("invalid object key generator: %s (valid: git-like, tenant-aware, high-performance, legacy)", generator)
		}
		c.ObjectKeyGenerator = generator
		return nil
	}
}

// WithEventLogging enables or disables event logging
func WithEventLogging(enabled bool) Option {
	return func(c *ServerConfig) error {
		c.EnableEventLogging = enabled
		return nil
	}
}

// WithPreviews enables or disables preview generation
func WithPreviews(enabled bool) Option {
	return func(c *ServerConfig) error {
		c.EnablePreviews = enabled
		return nil
	}
}

// WithAdminAPI enables or disables the admin API endpoints
func WithAdminAPI(enabled bool) Option {
	return func(c *ServerConfig) error {
		c.EnableAdminAPI = enabled
		return nil
	}
}

// WithDefaults is a convenience option that applies sensible defaults
// This is useful as a base before applying more specific options
func WithDefaults() Option {
	return func(c *ServerConfig) error {
		// Apply defaults (same as defaults() function but as an option)
		defaults := defaults()
		*c = defaults
		return nil
	}
}

// Helper functions for chaining multiple options

// WithFilesystemStorageFull provides all filesystem storage configuration in one call
func WithFilesystemStorageFull(name, baseDir, urlPrefix, secretKey string, expirySeconds int) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "fs"
		}
		if baseDir == "" {
			return fmt.Errorf("filesystem base directory cannot be empty")
		}

		backend := StorageBackendConfig{
			Name: name,
			Type: "fs",
			Config: map[string]interface{}{
				"base_dir": baseDir,
			},
		}

		if urlPrefix != "" {
			backend.Config["url_prefix"] = urlPrefix
		}
		if secretKey != "" {
			backend.Config["signature_secret_key"] = secretKey
		}
		if expirySeconds > 0 {
			backend.Config["presign_expires_seconds"] = expirySeconds
		}

		c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		return nil
	}
}

// WithS3StorageFull provides all S3 storage configuration in one call
func WithS3StorageFull(name, bucket, region, accessKey, secretKey, endpoint string, useSSL, usePathStyle bool) Option {
	return func(c *ServerConfig) error {
		if name == "" {
			name = "s3"
		}
		if bucket == "" {
			return fmt.Errorf("S3 bucket cannot be empty")
		}
		if region == "" {
			region = "us-east-1"
		}

		backend := StorageBackendConfig{
			Name: name,
			Type: "s3",
			Config: map[string]interface{}{
				"bucket": bucket,
				"region": region,
			},
		}

		if accessKey != "" {
			backend.Config["access_key_id"] = accessKey
		}
		if secretKey != "" {
			backend.Config["secret_access_key"] = secretKey
		}
		if endpoint != "" {
			backend.Config["endpoint"] = endpoint
			backend.Config["use_ssl"] = useSSL
			backend.Config["use_path_style"] = usePathStyle
		}

		c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		return nil
	}
}
