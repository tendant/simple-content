package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tendant/simple-content/internal/api"
	"github.com/tendant/simple-content/pkg/repository/memory"
	"github.com/tendant/simple-content/pkg/service"
	fsStorage "github.com/tendant/simple-content/pkg/storage/fs"
	memoryStorage "github.com/tendant/simple-content/pkg/storage/memory"
)

func main() {
	// Initialize in-memory repositories
	contentRepo := memory.NewContentRepository()
	contentMetadataRepo := memory.NewContentMetadataRepository()
	objectRepo := memory.NewObjectRepository()
	objectMetadataRepo := memory.NewObjectMetadataRepository()
	storageBackendRepo := memory.NewStorageBackendRepository()

	// Initialize storage backends
	memBackend := memoryStorage.NewMemoryBackend()

	// Initialize file system backend
	fsConfig := fsStorage.Config{
		BaseDir:   "./data/storage", // Default base directory
		URLPrefix: "",               // No URL prefix by default (direct access)
	}
	fsBackend, err := fsStorage.NewFSBackend(fsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize file system storage: %v", err)
	}

	// Initialize services
	contentService := service.NewContentService(contentRepo, contentMetadataRepo)
	objectService := service.NewObjectService(objectRepo, objectMetadataRepo, contentRepo)
	storageBackendService := service.NewStorageBackendService(storageBackendRepo)

	// Register the storage backends with the object service
	objectService.RegisterBackend("memory", memBackend)
	objectService.RegisterBackend("fs", fsBackend)
	objectService.RegisterBackend("fs-test", fsBackend)

	// Create a default file system storage backend
	ctx := context.Background()
	_, err = storageBackendService.CreateStorageBackend(
		ctx,
		"fs-default",
		"fs",
		map[string]interface{}{
			"base_dir": fsConfig.BaseDir,
		},
	)
	if err != nil {
		log.Printf("Warning: Failed to create default file system storage backend: %v", err)
	}

	// Initialize API handlers
	contentHandler := api.NewContentHandler(contentService, objectService)
	objectHandler := api.NewObjectHandler(objectService)
	storageBackendHandler := api.NewStorageBackendHandler(storageBackendService)

	// Set up router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Mount routes
	r.Mount("/content", contentHandler.Routes())
	r.Mount("/object", objectHandler.Routes())
	r.Mount("/storage-backend", storageBackendHandler.Routes())

	// Add a simple health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
