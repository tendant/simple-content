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
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/api"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
	// Initialize repository
	repo := memoryrepo.New()

	// Initialize storage backends
	memBackend := memorystorage.New()

	// Initialize file system backend
	fsConfig := fsstorage.Config{
		BaseDir:   "./data/storage", // Default base directory
		URLPrefix: "",               // No URL prefix by default (direct access)
	}
	fsBackend, err := fsstorage.New(fsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize file system storage: %v", err)
	}

	// Create unified service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
		simplecontent.WithBlobStore("fs", fsBackend),
		simplecontent.WithBlobStore("fs-test", fsBackend),
	)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create storage service for advanced operations
	storageSvc, err := simplecontent.NewStorageService(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
		simplecontent.WithBlobStore("fs", fsBackend),
		simplecontent.WithBlobStore("fs-test", fsBackend),
	)
	if err != nil {
		log.Fatalf("Failed to create storage service: %v", err)
	}

	// Initialize API handlers
	contentHandler := api.NewContentHandler(svc, storageSvc)
	filesHandler := api.NewFilesHandler(svc, storageSvc)

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
	r.Mount("/files", filesHandler.Routes())

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
