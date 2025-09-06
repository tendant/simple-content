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
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

func main() {
	// Load configuration from environment
	serverConfig, err := config.LoadServerConfig()
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Build service from configuration
	svc, err := serverConfig.BuildService()
	if err != nil {
		log.Fatalf("Failed to build service: %v", err)
	}

	// Create HTTP server
	server := NewHTTPServer(svc, serverConfig)

	// Create HTTP server instance
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverConfig.Port),
		Handler: server.Routes(),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Simple Content Server starting on port %s (env: %s)", serverConfig.Port, serverConfig.Environment)
		log.Printf("Default storage backend: %s", serverConfig.DefaultStorageBackend)
		log.Printf("Configured storage backends: %d", len(serverConfig.StorageBackends))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	// Shutdown server gracefully
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

// HTTPServer wraps the simple-content service for HTTP access
type HTTPServer struct {
	service simplecontent.Service
	config  *config.ServerConfig
}

// NewHTTPServer creates a new HTTP server wrapper
func NewHTTPServer(service simplecontent.Service, serverConfig *config.ServerConfig) *HTTPServer {
	return &HTTPServer{
		service: service,
		config:  serverConfig,
	}
}

// Routes sets up the HTTP routes
func (s *HTTPServer) Routes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS for development
	if s.config.Environment == "development" {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}

				next.ServeHTTP(w, r)
			})
		})
	}

	// Health check
	r.Get("/health", s.handleHealth)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Content management
		r.Post("/contents", s.handleCreateContent)
		r.Get("/contents/{contentID}", s.handleGetContent)
		r.Put("/contents/{contentID}", s.handleUpdateContent)
		r.Delete("/contents/{contentID}", s.handleDeleteContent)
		r.Get("/contents", s.handleListContents)

		// Content metadata
		r.Post("/contents/{contentID}/metadata", s.handleSetContentMetadata)
		r.Get("/contents/{contentID}/metadata", s.handleGetContentMetadata)

		// Object management
		r.Post("/contents/{contentID}/objects", s.handleCreateObject)
		r.Get("/objects/{objectID}", s.handleGetObject)
		r.Delete("/objects/{objectID}", s.handleDeleteObject)
		r.Get("/contents/{contentID}/objects", s.handleListObjects)

		// Object upload/download
		r.Post("/objects/{objectID}/upload", s.handleUploadObject)
		r.Get("/objects/{objectID}/download", s.handleDownloadObject)
		r.Get("/objects/{objectID}/upload-url", s.handleGetUploadURL)
		r.Get("/objects/{objectID}/download-url", s.handleGetDownloadURL)
		r.Get("/objects/{objectID}/preview-url", s.handleGetPreviewURL)

		// Demo endpoint
		r.Get("/demo", s.handleDemo)

		// Configuration info
		r.Get("/config", s.handleGetConfig)
	})

	return r
}

// Health check endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"status": "healthy", 
		"environment": "%s",
		"default_storage": "%s"
	}`, s.config.Environment, s.config.DefaultStorageBackend)
}

// Demo endpoint showing basic functionality
func (s *HTTPServer) handleDemo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create a content
	content, err := s.service.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Demo Content",
		Description:  "This is a demo content created via API",
		DocumentType: "text/plain",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create content: %v", err), http.StatusInternalServerError)
		return
	}

	// Create an object for the content
	object, err := s.service.CreateObject(ctx, simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: s.config.DefaultStorageBackend,
		Version:            1,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create object: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"message": "Demo completed successfully",
		"content_id": "%s",
		"object_id": "%s",
		"storage_backend": "%s",
		"environment": "%s"
	}`, content.ID, object.ID, s.config.DefaultStorageBackend, s.config.Environment)
}

// Configuration info endpoint
func (s *HTTPServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	backends := make([]string, len(s.config.StorageBackends))
	for i, backend := range s.config.StorageBackends {
		backends[i] = fmt.Sprintf("%s (%s)", backend.Name, backend.Type)
	}

	fmt.Fprintf(w, `{
		"environment": "%s",
		"database_type": "%s",
		"default_storage_backend": "%s",
		"available_storage_backends": %q,
		"enable_event_logging": %t,
		"enable_previews": %t
	}`,
		s.config.Environment,
		s.config.DatabaseType,
		s.config.DefaultStorageBackend,
		backends,
		s.config.EnableEventLogging,
		s.config.EnablePreviews)
}

// Placeholder handlers - in a real implementation these would be fully implemented

func (s *HTTPServer) handleCreateContent(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleGetContent(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleUpdateContent(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleDeleteContent(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleListContents(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleSetContentMetadata(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleGetContentMetadata(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleCreateObject(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleGetObject(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleDeleteObject(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleListObjects(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleUploadObject(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleDownloadObject(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleGetDownloadURL(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (s *HTTPServer) handleGetPreviewURL(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}
