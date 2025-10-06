package config

import (
	"testing"
)

func TestEnvDatabaseURL(t *testing.T) {
	tests := []struct {
		name           string
		dbURL          string
		wantType       string
		wantURL        string
		wantError      bool
	}{
		{"empty defaults to memory", "", "memory", "", false},
		{"memory keyword", "memory", "memory", "", false},
		{"postgresql URL", "postgresql://user:pass@localhost/db", "postgres", "postgresql://user:pass@localhost/db", false},
		{"postgres URL", "postgres://user:pass@localhost/db", "postgres", "postgres://user:pass@localhost/db", false},
		{"invalid URL", "mysql://localhost/db", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbURL != "" {
				t.Setenv("DATABASE_URL", tt.dbURL)
			}

			cfg, err := Load(WithEnv(""))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.DatabaseType != tt.wantType {
				t.Errorf("expected database type %q, got %q", tt.wantType, cfg.DatabaseType)
			}
			if cfg.DatabaseURL != tt.wantURL {
				t.Errorf("expected database URL %q, got %q", tt.wantURL, cfg.DatabaseURL)
			}
		})
	}
}

func TestEnvStorageURL(t *testing.T) {
	tests := []struct {
		name            string
		storageURL      string
		wantBackendType string
		wantBackendName string
		wantError       bool
	}{
		{"empty defaults to memory", "", "memory", "memory", false},
		{"memory keyword", "memory", "memory", "memory", false},
		{"memory URL", "memory://", "memory", "memory", false},
		{"filesystem URL", "file:///var/data", "fs", "fs", false},
		{"S3 URL", "s3://my-bucket", "s3", "s3", false},
		{"invalid URL", "ftp://example.com", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.storageURL != "" {
				t.Setenv("STORAGE_URL", tt.storageURL)
			}

			cfg, err := Load(WithEnv(""))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.DefaultStorageBackend != tt.wantBackendName {
				t.Errorf("expected default backend %q, got %q", tt.wantBackendName, cfg.DefaultStorageBackend)
			}

			if len(cfg.StorageBackends) == 0 {
				t.Fatal("expected at least one storage backend")
			}

			backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
			if backend.Type != tt.wantBackendType {
				t.Errorf("expected backend type %q, got %q", tt.wantBackendType, backend.Type)
			}
			if backend.Name != tt.wantBackendName {
				t.Errorf("expected backend name %q, got %q", tt.wantBackendName, backend.Name)
			}
		})
	}
}

func TestEnvFilesystemStorage(t *testing.T) {
	t.Setenv("STORAGE_URL", "file:///var/data/storage")

	cfg, err := Load(WithEnv(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultStorageBackend != "fs" {
		t.Errorf("expected default backend 'fs', got %q", cfg.DefaultStorageBackend)
	}

	if len(cfg.StorageBackends) == 0 {
		t.Fatal("expected at least one storage backend")
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Type != "fs" {
		t.Errorf("expected backend type 'fs', got %q", backend.Type)
	}

	baseDir, ok := backend.Config["base_dir"].(string)
	if !ok {
		t.Fatal("base_dir not found or not a string")
	}
	if baseDir != "/var/data/storage" {
		t.Errorf("expected base_dir '/var/data/storage', got %q", baseDir)
	}
}

func TestEnvS3Storage(t *testing.T) {
	t.Setenv("STORAGE_URL", "s3://my-test-bucket")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI")

	cfg, err := Load(WithEnv(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultStorageBackend != "s3" {
		t.Errorf("expected default backend 's3', got %q", cfg.DefaultStorageBackend)
	}

	if len(cfg.StorageBackends) == 0 {
		t.Fatal("expected at least one storage backend")
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Type != "s3" {
		t.Errorf("expected backend type 's3', got %q", backend.Type)
	}

	bucket, ok := backend.Config["bucket"].(string)
	if !ok {
		t.Fatal("bucket not found or not a string")
	}
	if bucket != "my-test-bucket" {
		t.Errorf("expected bucket 'my-test-bucket', got %q", bucket)
	}

	region, ok := backend.Config["region"].(string)
	if !ok {
		t.Fatal("region not found or not a string")
	}
	if region != "eu-west-1" {
		t.Errorf("expected region 'eu-west-1', got %q", region)
	}

	accessKey, ok := backend.Config["access_key_id"].(string)
	if !ok {
		t.Fatal("access_key_id not found or not a string")
	}
	if accessKey != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected access_key_id 'AKIAIOSFODNN7EXAMPLE', got %q", accessKey)
	}
}

func TestEnvServerConfig(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("ENVIRONMENT", "production")

	cfg, err := Load(WithEnv(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected port '9090', got %q", cfg.Port)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected environment 'production', got %q", cfg.Environment)
	}
}

func TestEnvCompleteConfig(t *testing.T) {
	// Test a complete configuration from environment
	t.Setenv("PORT", "8888")
	t.Setenv("ENVIRONMENT", "staging")
	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost/testdb")
	t.Setenv("STORAGE_URL", "file:///data/storage")

	cfg, err := Load(WithEnv(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify server config
	if cfg.Port != "8888" {
		t.Errorf("expected port '8888', got %q", cfg.Port)
	}
	if cfg.Environment != "staging" {
		t.Errorf("expected environment 'staging', got %q", cfg.Environment)
	}

	// Verify database config
	if cfg.DatabaseType != "postgres" {
		t.Errorf("expected database type 'postgres', got %q", cfg.DatabaseType)
	}
	if cfg.DatabaseURL != "postgresql://user:pass@localhost/testdb" {
		t.Errorf("expected database URL 'postgresql://user:pass@localhost/testdb', got %q", cfg.DatabaseURL)
	}

	// Verify storage config
	if cfg.DefaultStorageBackend != "fs" {
		t.Errorf("expected default storage 'fs', got %q", cfg.DefaultStorageBackend)
	}
}
