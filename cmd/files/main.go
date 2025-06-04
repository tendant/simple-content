package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tendant/chi-demo/app"
	"github.com/tendant/simple-content/internal/api"
	psqlrepo "github.com/tendant/simple-content/pkg/repository/psql"
	"github.com/tendant/simple-content/pkg/service"
	"github.com/tendant/simple-content/pkg/storage/s3"
)

type Config struct {
	Server ServerConfig `env-prefix:"SERVER_"`
	DB     DbConfig     `env-prefix:"DB_"`
	S3     S3Config     `env-prefix:"S3_"`
}

type ServerConfig struct {
	Port string `env:"PORT" env-default:"8080"`
	Host string `env:"HOST" env-default:"localhost"`
}

type DbConfig struct {
	Port     uint16 `env:"PORT" env-default:"5432"`
	Host     string `env:"HOST" env-default:"localhost"`
	Name     string `env:"NAME" env-default:"powercard_db"`
	User     string `env:"USER" env-default:"content"`
	Password string `env:"PASSWORD" env-default:"pwd"`
}

type S3Config struct {
	Endpoint        string `env:"ENDPOINT" env-default:"http://localhost:9000"`
	AccessKeyID     string `env:"ACCESS_KEY_ID" env-default:"minioadmin"`
	SecretAccessKey string `env:"SECRET_ACCESS_KEY" env-default:"minioadmin"`
	BucketName      string `env:"BUCKET_NAME" env-default:"content-bucket"`
	Region          string `env:"REGION" env-default:"us-east-1"`
	UseSSL          bool   `env:"USE_SSL" env-default:"false"`
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

func initializeS3Backend(config S3Config) (*s3.S3Backend, error) {
	s3Config := s3.Config{
		Endpoint:               config.Endpoint,
		AccessKeyID:            config.AccessKeyID,
		SecretAccessKey:        config.SecretAccessKey,
		Bucket:                 config.BucketName,
		Region:                 config.Region,
		UseSSL:                 config.UseSSL,
		CreateBucketIfNotExist: false,
		PresignDuration:        3600, // 1 hour
	}

	backend, err := s3.NewS3Backend(s3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 backend: %w", err)
	}

	// Type assert to get the concrete S3Backend type
	s3Backend, ok := backend.(*s3.S3Backend)
	if !ok {
		return nil, fmt.Errorf("failed to cast to S3Backend")
	}

	return s3Backend, nil
}

func main() {
	// Load configuration
	var config Config
	if err := cleanenv.ReadEnv(&config); err != nil {
		slog.Error("Failed to read configuration", "err", err)
		os.Exit(1)
	}

	// Initialize database connection
	ctx := context.Background()
	dbPool, err := NewDbPool(ctx, config.DB)
	if err != nil {
		slog.Error("Failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Initialize repositories
	repoFactory := psqlrepo.NewRepositoryFactory(dbPool)
	contentRepo := repoFactory.NewContentRepository()
	contentMetadataRepo := repoFactory.NewContentMetadataRepository()
	objectRepo := repoFactory.NewObjectRepository()
	objectMetadataRepo := repoFactory.NewObjectMetadataRepository()

	// Initialize S3 storage backend
	s3Backend, err := initializeS3Backend(config.S3)
	if err != nil {
		slog.Error("Failed to initialize S3 backend", "err", err)
		os.Exit(1)
	}

	// Initialize services
	contentService := service.NewContentService(
		contentRepo,
		contentMetadataRepo,
	)

	objectService := service.NewObjectService(
		objectRepo,
		objectMetadataRepo,
		contentRepo,
		contentMetadataRepo,
	)

	// Register the S3 backend with the object service
	objectService.RegisterBackend("s3-default", s3Backend)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(60 * time.Second))

	server := app.DefaultApp()

	app.RoutesHealthz(server.R)
	app.RoutesHealthzReady(server.R)

	// Initialize API handlers
	contentHandler := api.NewContentHandler(contentService, objectService)
	filesHandler := api.NewFilesHandler(contentService, objectService)

	// Routes
	server.R.Mount("/contents", contentHandler.Routes())
	server.R.Mount("/files", filesHandler.Routes())

	// Start server
	server.Run()
}

func RoutesHealthz(r *chi.Mux) {
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, http.StatusText(http.StatusOK))
	})
}

func RoutesHealthzReady(r *chi.Mux) {
	r.Get("/healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, http.StatusText(http.StatusOK))
	})
}
