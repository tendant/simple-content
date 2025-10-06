package config

import (
	"testing"
)

func TestWithPort(t *testing.T) {
	cfg, err := Load(WithPort("9090"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got: %s", cfg.Port)
	}
}

func TestWithPortEmpty(t *testing.T) {
	_, err := Load(WithPort(""))
	if err == nil {
		t.Error("expected error for empty port, got nil")
	}
}

func TestWithEnvironment(t *testing.T) {
	cfg, err := Load(WithEnvironment("production"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected environment production, got: %s", cfg.Environment)
	}
}

func TestWithDatabase(t *testing.T) {
	tests := []struct {
		name      string
		dbType    string
		url       string
		wantError bool
	}{
		{"memory valid", "memory", "", false},
		{"postgres valid", "postgres", "postgresql://localhost/test", false},
		{"postgres missing url", "postgres", "", true},
		{"invalid type", "mysql", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(WithDatabase(tt.dbType, tt.url))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if cfg.DatabaseType != tt.dbType {
				t.Errorf("expected database type %s, got: %s", tt.dbType, cfg.DatabaseType)
			}
			if cfg.DatabaseURL != tt.url {
				t.Errorf("expected database URL %s, got: %s", tt.url, cfg.DatabaseURL)
			}
		})
	}
}

func TestWithFilesystemStorage(t *testing.T) {
	cfg, err := Load(
		WithFilesystemStorage("", "./data", "/api/v1", "secret"),
		WithDefaultStorage("fs"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check backend was added
	if len(cfg.StorageBackends) == 0 {
		t.Fatal("expected storage backend to be added")
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Name != "fs" {
		t.Errorf("expected backend name 'fs', got: %s", backend.Name)
	}
	if backend.Type != "fs" {
		t.Errorf("expected backend type 'fs', got: %s", backend.Type)
	}
	if backend.Config["base_dir"] != "./data" {
		t.Errorf("expected base_dir './data', got: %v", backend.Config["base_dir"])
	}
	if backend.Config["url_prefix"] != "/api/v1" {
		t.Errorf("expected url_prefix '/api/v1', got: %v", backend.Config["url_prefix"])
	}
	if backend.Config["signature_secret_key"] != "secret" {
		t.Errorf("expected signature_secret_key 'secret', got: %v", backend.Config["signature_secret_key"])
	}
}

func TestWithFilesystemStorageMissingBaseDir(t *testing.T) {
	_, err := Load(
		WithFilesystemStorage("", "", "/api/v1", "secret"),
	)
	if err == nil {
		t.Error("expected error for missing base directory, got nil")
	}
}

func TestWithS3Storage(t *testing.T) {
	cfg, err := Load(
		WithS3Storage("", "my-bucket", "us-west-2"),
		WithDefaultStorage("s3"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check backend was added
	if len(cfg.StorageBackends) == 0 {
		t.Fatal("expected storage backend to be added")
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Name != "s3" {
		t.Errorf("expected backend name 's3', got: %s", backend.Name)
	}
	if backend.Type != "s3" {
		t.Errorf("expected backend type 's3', got: %s", backend.Type)
	}
	if backend.Config["bucket"] != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got: %v", backend.Config["bucket"])
	}
	if backend.Config["region"] != "us-west-2" {
		t.Errorf("expected region 'us-west-2', got: %v", backend.Config["region"])
	}
}

func TestWithS3Credentials(t *testing.T) {
	cfg, err := Load(
		WithS3Storage("", "my-bucket", "us-west-2"),
		WithS3Credentials("s3", "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
		WithDefaultStorage("s3"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Config["access_key_id"] != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected access_key_id to be set")
	}
	if backend.Config["secret_access_key"] != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("expected secret_access_key to be set")
	}
}

func TestWithS3Endpoint(t *testing.T) {
	cfg, err := Load(
		WithS3Storage("", "my-bucket", "us-east-1"),
		WithS3Endpoint("s3", "http://localhost:9000", false, true),
		WithDefaultStorage("s3"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Config["endpoint"] != "http://localhost:9000" {
		t.Errorf("expected endpoint to be set")
	}
	if backend.Config["use_ssl"] != false {
		t.Errorf("expected use_ssl to be false")
	}
	if backend.Config["use_path_style"] != true {
		t.Errorf("expected use_path_style to be true")
	}
}

func TestWithMemoryStorage(t *testing.T) {
	cfg, err := Load(
		WithMemoryStorage(""),
		WithDefaultStorage("memory"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check backend was added
	if len(cfg.StorageBackends) != 1 {
		t.Fatalf("expected 1 storage backend, got: %d", len(cfg.StorageBackends))
	}

	backend := cfg.StorageBackends[0]
	if backend.Name != "memory" {
		t.Errorf("expected backend name 'memory', got: %s", backend.Name)
	}
	if backend.Type != "memory" {
		t.Errorf("expected backend type 'memory', got: %s", backend.Type)
	}
}

func TestWithContentBasedURLs(t *testing.T) {
	cfg, err := Load(WithContentBasedURLs("/api/v1"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.URLStrategy != "content-based" {
		t.Errorf("expected URL strategy 'content-based', got: %s", cfg.URLStrategy)
	}
	if cfg.APIBaseURL != "/api/v1" {
		t.Errorf("expected API base URL '/api/v1', got: %s", cfg.APIBaseURL)
	}
}

func TestWithCDNURLs(t *testing.T) {
	cfg, err := Load(WithCDNURLs("https://cdn.example.com", "https://api.example.com"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.URLStrategy != "cdn" {
		t.Errorf("expected URL strategy 'cdn', got: %s", cfg.URLStrategy)
	}
	if cfg.CDNBaseURL != "https://cdn.example.com" {
		t.Errorf("expected CDN base URL 'https://cdn.example.com', got: %s", cfg.CDNBaseURL)
	}
	if cfg.UploadBaseURL != "https://api.example.com" {
		t.Errorf("expected upload base URL 'https://api.example.com', got: %s", cfg.UploadBaseURL)
	}
}

func TestWithStorageDelegatedURLs(t *testing.T) {
	cfg, err := Load(WithStorageDelegatedURLs())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.URLStrategy != "storage-delegated" {
		t.Errorf("expected URL strategy 'storage-delegated', got: %s", cfg.URLStrategy)
	}
}

func TestWithObjectKeyGenerator(t *testing.T) {
	tests := []struct {
		name      string
		generator string
		wantError bool
	}{
		{"git-like valid", "git-like", false},
		{"tenant-aware valid", "tenant-aware", false},
		{"high-performance valid", "high-performance", false},
		{"legacy valid", "legacy", false},
		{"default valid", "default", false},
		{"invalid generator", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(WithObjectKeyGenerator(tt.generator))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if cfg.ObjectKeyGenerator != tt.generator {
				t.Errorf("expected generator %s, got: %s", tt.generator, cfg.ObjectKeyGenerator)
			}
		})
	}
}

func TestWithEventLogging(t *testing.T) {
	cfg, err := Load(WithEventLogging(false))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.EnableEventLogging != false {
		t.Errorf("expected event logging to be false, got: %t", cfg.EnableEventLogging)
	}
}

func TestWithPreviews(t *testing.T) {
	cfg, err := Load(WithPreviews(false))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.EnablePreviews != false {
		t.Errorf("expected previews to be false, got: %t", cfg.EnablePreviews)
	}
}

func TestWithAdminAPI(t *testing.T) {
	cfg, err := Load(WithAdminAPI(true))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.EnableAdminAPI != true {
		t.Errorf("expected admin API to be true, got: %t", cfg.EnableAdminAPI)
	}
}

func TestComposedOptions(t *testing.T) {
	// Test composing multiple options together
	cfg, err := Load(
		WithPort("9090"),
		WithEnvironment("production"),
		WithDatabase("postgres", "postgresql://localhost/test"),
		WithFilesystemStorage("fs", "./data", "/api/v1", "secret"),
		WithDefaultStorage("fs"),
		WithContentBasedURLs("/api/v1"),
		WithObjectKeyGenerator("git-like"),
		WithEventLogging(true),
		WithPreviews(true),
		WithAdminAPI(false),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify all options were applied
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got: %s", cfg.Port)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected environment production, got: %s", cfg.Environment)
	}
	if cfg.DatabaseType != "postgres" {
		t.Errorf("expected database type postgres, got: %s", cfg.DatabaseType)
	}
	if cfg.DefaultStorageBackend != "fs" {
		t.Errorf("expected default storage fs, got: %s", cfg.DefaultStorageBackend)
	}
	if cfg.URLStrategy != "content-based" {
		t.Errorf("expected URL strategy content-based, got: %s", cfg.URLStrategy)
	}
	if cfg.ObjectKeyGenerator != "git-like" {
		t.Errorf("expected object key generator git-like, got: %s", cfg.ObjectKeyGenerator)
	}
	if !cfg.EnableEventLogging {
		t.Error("expected event logging to be enabled")
	}
	if !cfg.EnablePreviews {
		t.Error("expected previews to be enabled")
	}
	if cfg.EnableAdminAPI {
		t.Error("expected admin API to be disabled")
	}
}

func TestOptionsOverrideDefaults(t *testing.T) {
	// Test that options override defaults
	cfg, err := Load(
		WithPort("9090"), // Override default port 8080
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got: %s", cfg.Port)
	}
}

func TestEnvOverridesOptions(t *testing.T) {
	// Test that env vars can override programmatic options
	t.Setenv("PORT", "7070")

	cfg, err := Load(
		WithPort("9090"),
		WithEnv(""), // Env should override
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Port != "7070" {
		t.Errorf("expected env to override port to 7070, got: %s", cfg.Port)
	}
}

func TestWithFilesystemStorageFull(t *testing.T) {
	cfg, err := Load(
		WithFilesystemStorageFull("myfs", "./data", "/api/v1", "secret", 1800),
		WithDefaultStorage("myfs"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Name != "myfs" {
		t.Errorf("expected backend name 'myfs', got: %s", backend.Name)
	}
	if backend.Config["base_dir"] != "./data" {
		t.Errorf("expected base_dir './data', got: %v", backend.Config["base_dir"])
	}
	if backend.Config["presign_expires_seconds"] != 1800 {
		t.Errorf("expected presign_expires_seconds 1800, got: %v", backend.Config["presign_expires_seconds"])
	}
}

func TestWithS3StorageFull(t *testing.T) {
	cfg, err := Load(
		WithS3StorageFull("mys3", "my-bucket", "us-west-2", "key", "secret", "http://localhost:9000", false, true),
		WithDefaultStorage("mys3"),
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	backend := cfg.StorageBackends[len(cfg.StorageBackends)-1]
	if backend.Name != "mys3" {
		t.Errorf("expected backend name 'mys3', got: %s", backend.Name)
	}
	if backend.Config["bucket"] != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got: %v", backend.Config["bucket"])
	}
	if backend.Config["endpoint"] != "http://localhost:9000" {
		t.Errorf("expected endpoint 'http://localhost:9000', got: %v", backend.Config["endpoint"])
	}
	if backend.Config["use_path_style"] != true {
		t.Errorf("expected use_path_style true, got: %v", backend.Config["use_path_style"])
	}
}
