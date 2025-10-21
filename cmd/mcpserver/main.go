package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tendant/simple-content/internal/mcp"
	"github.com/tendant/simple-content/pkg/simplecontent"
	repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
	s3store "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)

type Config struct {
	Host    string `env:"HOST" env-default:"localhost"`
	Port    uint16 `env:"PORT" env-default:"8000"`
	BaseUrl string `env:"BASE_URL" env-default:"http://localhost:8000"`
	DB      DbConfig
	S3      S3Config
}

type DbConfig struct {
	Port     uint16 `env:"CONTENT_PG_PORT" env-default:"5432"`
	Host     string `env:"CONTENT_PG_HOST" env-default:"localhost"`
	Name     string `env:"CONTENT_PG_NAME" env-default:"powercard_db"`
	User     string `env:"CONTENT_PG_USER" env-default:"content"`
	Password string `env:"CONTENT_PG_PASSWORD" env-default:"pwd"`
}

type S3Config struct {
	Endpoint        string `env:"AWS_S3_ENDPOINT"`
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID" env-default:"minioadmin"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY" env-default:"minioadmin"`
	BucketName      string `env:"AWS_S3_BUCKET" env-default:"mymusic"`
	Region          string `env:"AWS_S3_REGION" env-default:"us-east-1"`
	UseSSL          bool   `env:"AWS_S3_USE_SSL" env-default:"false"`
}

func (c DbConfig) toDatabaseUrl() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Name,
	}
	return u.String()
}

func NewDbPool(ctx context.Context, dbConfig DbConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dbConfig.toDatabaseUrl())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

const S3_URL_DURATION = 3600 * 6 // 6 hours

func initializeS3Backend(config S3Config) (simplecontent.BlobStore, error) {
	s3Config := s3store.Config{
		Endpoint:               config.Endpoint,
		AccessKeyID:            config.AccessKeyID,
		SecretAccessKey:        config.SecretAccessKey,
		Bucket:                 config.BucketName,
		Region:                 config.Region,
		UseSSL:                 config.UseSSL,
		CreateBucketIfNotExist: false,
		PresignDuration:        S3_URL_DURATION, // 6 hours
	}

	backend, err := s3store.New(s3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 backend: %w", err)
	}

	return backend, nil
}

func main() {
	// Define command line flags for database configs and server mode

	// Server mode flags
	var mode = flag.String("mode", "stdio", "Server mode: 'stdio', 'sse', or 'http'")

	// Parse command line flags
	flag.Parse()

	// Load environment variables from .env file
	if err := godotenv.Load(".env"); err != nil {
		// It's okay if .env doesn't exist, we'll use default values
		slog.Info("No .env file found or error loading it, using default values", "err", err)
	}

	var cfg Config
	cleanenv.ReadEnv(&cfg)

	// Create MCP server with appropriate options based on mode
	s := server.NewMCPServer(
		"Content Server Mcp",
		"1.0.0",
		server.WithResourceCapabilities(true, true), // Enable SSE and JSON-RPC
	)

	// Initialize database connection
	ctx := context.Background()
	dbPool, err := NewDbPool(ctx, cfg.DB)
	if err != nil {
		slog.Error("Failed to connect to database", "err", err)
		os.Exit(1)
	}

	// Initialize repository
	repo := repopg.New(dbPool)

	// Initialize S3 storage backend
	s3Backend, err := initializeS3Backend(cfg.S3)
	if err != nil {
		slog.Error("Failed to initialize S3 backend", "err", err)
		os.Exit(1)
	}

	// Initialize unified service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("s3-default", s3Backend),
	)
	if err != nil {
		slog.Error("Failed to create service", "err", err)
		os.Exit(1)
	}

	// Create storage service for advanced object operations
	storageSvc, err := simplecontent.NewStorageService(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("s3-default", s3Backend),
	)
	if err != nil {
		slog.Error("Failed to create storage service", "err", err)
		os.Exit(1)
	}

	// Register hello content tools
	handler := mcp.NewHandler(svc, storageSvc)
	handler.RegisterTools(s)

	// Start the server based on the selected mode
	switch *mode {
	case "sse":
		// Construct base URL from host and port
		sseServer := server.NewSSEServer(s, server.WithBaseURL(cfg.BaseUrl))
		slog.Info("Starting SSE server", "base url", cfg.BaseUrl)
		if err := sseServer.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			slog.Error("Failed to start SSE server", "err", err)
			os.Exit(-1)
		}
	case "http":
		httpServer := server.NewStreamableHTTPServer(s)
		slog.Info("HTTP server listening", "port", cfg.Port)
		if err := httpServer.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			slog.Error("Server error", "err", err)
			os.Exit(-1)
		}
	default:
		// Default to stdio mode
		slog.Info("Starting in stdio mode")
		if err := server.ServeStdio(s); err != nil {
			slog.Error("Failed to start stdio server", "err", err)
			os.Exit(-1)
		}
	}
}
