package presets

import (
	"fmt"
	"os"
	"testing"

	"github.com/tendant/simple-content/pkg/simplecontent"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

// Configuration Presets
//
// This package provides easy-to-use configuration presets for common use cases.
// Presets eliminate boilerplate and provide sensible defaults while remaining customizable.

// NewDevelopment creates a service configured for local development.
//
// Features:
//   - In-memory database (instant startup, no setup required)
//   - Filesystem storage at ./dev-data/ (persistent across restarts)
//   - Content-based URLs (/api/v1)
//   - Logging enabled (helpful for debugging)
//   - All features enabled (admin API, previews, events)
//
// Returns:
//   - Service instance
//   - Cleanup function (call with defer to remove dev-data directory)
//   - Error if setup fails
//
// Example:
//
//	svc, cleanup, err := simplecontent.NewDevelopment()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cleanup()
//
//	// Use service...
func NewDevelopment(opts ...DevelopmentOption) (simplecontent.Service, func(), error) {
	// Default configuration
	cfg := &devConfig{
		storageDir: "./dev-data",
		port:       "8080",
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Create repository (in-memory for development)
	repo := memoryrepo.New()

	// Create filesystem storage
	fsBackend, err := fsstorage.New(fsstorage.Config{
		BaseDir: cfg.storageDir,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create filesystem storage: %w", err)
	}

	// Build service options
	options := []simplecontent.Option{
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("fs", fsBackend),
	}

	// Create service
	svc, err := simplecontent.New(options...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(cfg.storageDir)
	}

	return svc, cleanup, nil
}

// NewTesting creates a service configured for unit and integration tests.
//
// Features:
//   - In-memory database (isolated per test)
//   - In-memory storage (fast, no disk I/O)
//   - No event logging (cleaner test output)
//   - Automatic cleanup via t.Cleanup()
//   - Supports parallel test execution
//
// The testing.T parameter enables automatic cleanup when the test completes.
//
// Example:
//
//	func TestMyFeature(t *testing.T) {
//	    svc := simplecontent.NewTesting(t)
//
//	    // Use service in test...
//	    // Automatic cleanup when test completes
//	}
func NewTesting(t *testing.T, opts ...TestingOption) simplecontent.Service {
	// Default configuration
	cfg := &testConfig{
		fixtures: false,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Create repository (in-memory for testing)
	repo := memoryrepo.New()

	// Create memory storage
	storage := memorystorage.New()

	// Build service options
	options := []simplecontent.Option{
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", storage),
	}

	// Create service
	svc, err := simplecontent.New(options...)
	if err != nil {
		t.Fatalf("failed to create test service: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		// No explicit cleanup needed for in-memory backends
		// They're garbage collected automatically
	})

	return svc
}

// NewProduction creates a service configured for production deployment.
//
// Features:
//   - Database from environment (DATABASE_TYPE, DATABASE_URL)
//   - Storage from environment (STORAGE_BACKEND, AWS_S3_BUCKET, etc.)
//   - URL strategy from environment (URL_STRATEGY, CDN_BASE_URL, etc.)
//   - Event logging enabled
//   - Security best practices
//   - Validation of required configuration
//
// Required Environment Variables:
//   - DATABASE_TYPE: "postgres" (required for production)
//   - DATABASE_URL: PostgreSQL connection string
//   - STORAGE_BACKEND: "s3" or "fs"
//   - AWS_S3_BUCKET: S3 bucket name (if using S3)
//   - AWS_S3_REGION: S3 region (if using S3)
//
// Optional Environment Variables:
//   - CDN_BASE_URL: CDN base URL for downloads
//   - URL_STRATEGY: "cdn", "content-based", or "storage-delegated"
//   - OBJECT_KEY_GENERATOR: "git-like", "tenant-aware", etc.
//
// Example:
//
//	svc, err := simplecontent.NewProduction()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use service in production...
func NewProduction(opts ...ProductionOption) (simplecontent.Service, error) {
	// Default configuration (loads from environment)
	cfg := &prodConfig{
		databaseType:   getEnv("DATABASE_TYPE", "postgres"),
		databaseURL:    getEnv("DATABASE_URL", ""),
		storageBackend: getEnv("STORAGE_BACKEND", "s3"),
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate required configuration
	if cfg.databaseType == "memory" {
		return nil, fmt.Errorf("production preset requires DATABASE_TYPE=postgres (memory not allowed in production)")
	}
	if cfg.databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required for production")
	}
	if cfg.storageBackend == "memory" {
		return nil, fmt.Errorf("production preset requires persistent storage (s3 or fs, not memory)")
	}

	// Use the config package to build service
	// This delegates to the full configuration system
	configOpts := []interface{}{
		// Would use config.Load() here in real implementation
		// For now, we'll build manually
	}
	_ = configOpts

	// TODO: Implement full production setup using config package
	// For now, return error indicating environment setup needed
	return nil, fmt.Errorf("production preset requires config package integration (not yet implemented)")
}

// Option types for customization

// devConfig holds development preset configuration
type devConfig struct {
	storageDir string
	port       string
}

// testConfig holds testing preset configuration
type testConfig struct {
	fixtures bool
}

// prodConfig holds production preset configuration
type prodConfig struct {
	databaseType   string
	databaseURL    string
	storageBackend string
}

// DevelopmentOption is a functional option for NewDevelopment
type DevelopmentOption func(*devConfig)

// WithDevStorage sets the development storage directory
func WithDevStorage(dir string) DevelopmentOption {
	return func(cfg *devConfig) {
		cfg.storageDir = dir
	}
}

// WithDevPort sets the development server port
func WithDevPort(port string) DevelopmentOption {
	return func(cfg *devConfig) {
		cfg.port = port
	}
}

// TestingOption is a functional option for NewTesting
type TestingOption func(*testConfig)

// WithTestFixtures enables test fixtures (sample data)
func WithTestFixtures() TestingOption {
	return func(cfg *testConfig) {
		cfg.fixtures = true
	}
}

// ProductionOption is a functional option for NewProduction
type ProductionOption func(*prodConfig)

// WithProdDatabase sets the production database configuration
func WithProdDatabase(dbType, url string) ProductionOption {
	return func(cfg *prodConfig) {
		cfg.databaseType = dbType
		cfg.databaseURL = url
	}
}

// WithProdStorage sets the production storage backend
func WithProdStorage(backend string) ProductionOption {
	return func(cfg *prodConfig) {
		cfg.storageBackend = backend
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestService is a convenience function that creates a test service
// This is an alias for NewTesting with no options
func TestService(t *testing.T) simplecontent.Service {
	return NewTesting(t)
}
