package config

import (
	"fmt"
	"os"
	"strconv"
)

// WithEnv applies environment variable overrides using the provided prefix.
func WithEnv(prefix string) Option {
	return func(c *ServerConfig) error {
		if v, ok := lookupEnv(prefix, "PORT"); ok && v != "" {
			c.Port = v
		}
		if v, ok := lookupEnv(prefix, "ENVIRONMENT"); ok && v != "" {
			c.Environment = v
		}
		if v, ok := lookupEnv(prefix, "DATABASE_URL"); ok {
			c.DatabaseURL = v
		}
		if v, ok := lookupEnv(prefix, "DATABASE_TYPE"); ok && v != "" {
			c.DatabaseType = v
		}
		if v, ok := lookupEnv(prefix, "DATABASE_SCHEMA"); ok && v != "" {
			c.DBSchema = v
		}
		if v, ok := lookupEnv(prefix, "DEFAULT_STORAGE_BACKEND"); ok && v != "" {
			c.DefaultStorageBackend = v
		}
		if err := applyBoolEnv(prefix, "ENABLE_EVENT_LOGGING", func(b bool) {
			c.EnableEventLogging = b
		}); err != nil {
			return err
		}
		if err := applyBoolEnv(prefix, "ENABLE_PREVIEWS", func(b bool) {
			c.EnablePreviews = b
		}); err != nil {
			return err
		}

		fsBaseDir, _ := lookupEnv(prefix, "FS_BASE_DIR")
		if fsBaseDir != "" {
			backend := StorageBackendConfig{
				Name: "fs",
				Type: "fs",
				Config: map[string]interface{}{
					"base_dir": fsBaseDir,
				},
			}
			if v, ok := lookupEnv(prefix, "FS_URL_PREFIX"); ok && v != "" {
				backend.Config["url_prefix"] = v
			}
			c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		}

		if s3Bucket, _ := lookupEnv(prefix, "S3_BUCKET"); s3Bucket != "" {
			backend := StorageBackendConfig{
				Name: "s3",
				Type: "s3",
				Config: map[string]interface{}{
					"bucket": s3Bucket,
				},
			}
			if v, ok := lookupEnv(prefix, "S3_REGION"); ok && v != "" {
				backend.Config["region"] = v
			}
			if v, ok := lookupEnv(prefix, "S3_ACCESS_KEY_ID"); ok && v != "" {
				backend.Config["access_key_id"] = v
			}
			if v, ok := lookupEnv(prefix, "S3_SECRET_ACCESS_KEY"); ok && v != "" {
				backend.Config["secret_access_key"] = v
			}
			if v, ok := lookupEnv(prefix, "S3_ENDPOINT"); ok && v != "" {
				backend.Config["endpoint"] = v
			}
			if v, ok, err := parseBoolEnv(prefix, "S3_USE_SSL"); err != nil {
				return err
			} else if ok {
				backend.Config["use_ssl"] = v
			}
			if v, ok, err := parseBoolEnv(prefix, "S3_USE_PATH_STYLE"); err != nil {
				return err
			} else if ok {
				backend.Config["use_path_style"] = v
			}
			if v, ok, err := parseIntEnv(prefix, "S3_PRESIGN_DURATION"); err != nil {
				return err
			} else if ok {
				backend.Config["presign_duration"] = v
			}
			if v, ok, err := parseBoolEnv(prefix, "S3_ENABLE_SSE"); err != nil {
				return err
			} else if ok {
				backend.Config["enable_sse"] = v
			}
			if v, ok := lookupEnv(prefix, "S3_SSE_ALGORITHM"); ok && v != "" {
				backend.Config["sse_algorithm"] = v
			}
			if v, ok := lookupEnv(prefix, "S3_SSE_KMS_KEY_ID"); ok && v != "" {
				backend.Config["sse_kms_key_id"] = v
			}
			if v, ok, err := parseBoolEnv(prefix, "S3_CREATE_BUCKET_IF_NOT_EXIST"); err != nil {
				return err
			} else if ok {
				backend.Config["create_bucket_if_not_exist"] = v
			}
			c.StorageBackends = upsertStorageBackend(c.StorageBackends, backend)
		}

		return nil
	}
}

func lookupEnv(prefix, key string) (string, bool) {
	return os.LookupEnv(prefix + key)
}

func applyBoolEnv(prefix, key string, setter func(bool)) error {
	if setter == nil {
		return nil
	}
	if v, ok, err := parseBoolEnv(prefix, key); err != nil {
		return err
	} else if ok {
		setter(v)
	}
	return nil
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
