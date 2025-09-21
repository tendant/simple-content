package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

// loadServerConfigFromEnv constructs a ServerConfig by reading process environment variables.
// This keeps environment-specific logic within the executable instead of the library.
func loadServerConfigFromEnv() (*config.ServerConfig, error) {
	cfg := &config.ServerConfig{
		Port:                  getEnv("PORT", "8080"),
		Environment:           getEnv("ENVIRONMENT", "development"),
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		DatabaseType:          getEnv("DATABASE_TYPE", "memory"),
		DBSchema:              getEnv("CONTENT_DB_SCHEMA", "content"),
		DefaultStorageBackend: getEnv("DEFAULT_STORAGE_BACKEND", "memory"),
		EnableEventLogging:    getBoolEnv("ENABLE_EVENT_LOGGING", true),
		EnablePreviews:        getBoolEnv("ENABLE_PREVIEWS", true),
	}

	backendConfigs, err := loadStorageBackendConfigsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load storage backend configs: %w", err)
	}
	cfg.StorageBackends = backendConfigs

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func loadStorageBackendConfigsFromEnv() ([]config.StorageBackendConfig, error) {
	var configs []config.StorageBackendConfig

	configs = append(configs, config.StorageBackendConfig{
		Name:   "memory",
		Type:   "memory",
		Config: map[string]interface{}{},
	})

	fsBaseDir := os.Getenv("FS_BASE_DIR")
	if fsBaseDir != "" {
		configs = append(configs, config.StorageBackendConfig{
			Name: "fs",
			Type: "fs",
			Config: map[string]interface{}{
				"base_dir":   fsBaseDir,
				"url_prefix": os.Getenv("FS_URL_PREFIX"),
			},
		})
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket != "" {
		configs = append(configs, config.StorageBackendConfig{
			Name: "s3",
			Type: "s3",
			Config: map[string]interface{}{
				"region":                     os.Getenv("S3_REGION"),
				"bucket":                     s3Bucket,
				"access_key_id":              os.Getenv("S3_ACCESS_KEY_ID"),
				"secret_access_key":          os.Getenv("S3_SECRET_ACCESS_KEY"),
				"endpoint":                   os.Getenv("S3_ENDPOINT"),
				"use_ssl":                    getBoolEnv("S3_USE_SSL", true),
				"use_path_style":             getBoolEnv("S3_USE_PATH_STYLE", false),
				"presign_duration":           getIntEnv("S3_PRESIGN_DURATION", 3600),
				"enable_sse":                 getBoolEnv("S3_ENABLE_SSE", false),
				"sse_algorithm":              os.Getenv("S3_SSE_ALGORITHM"),
				"sse_kms_key_id":             os.Getenv("S3_SSE_KMS_KEY_ID"),
				"create_bucket_if_not_exist": getBoolEnv("S3_CREATE_BUCKET_IF_NOT_EXIST", false),
			},
		})
	}

	return configs, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
