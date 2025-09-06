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
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func main() {
	// Initialize repository (in-memory for example)
	repo := memoryrepo.New()

	// Initialize storage backends
	memoryStore := memorystorage.New()

	// Create the simple-content service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memoryStore),
		simplecontent.WithBlobStore("default", memoryStore),
	)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create HTTP server using the service
	server := NewHTTPServer(svc)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: server.Routes(),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
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
}

// NewHTTPServer creates a new HTTP server wrapper
func NewHTTPServer(service simplecontent.Service) *HTTPServer {
	return &HTTPServer{service: service}
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

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Basic API routes - these would be expanded based on your needs
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/demo", s.handleDemo)
	})

	return r
}

// handleDemo shows how to use the service
func (s *HTTPServer) handleDemo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create a content
	content, err := s.service.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:     generateUUID(),
		TenantID:    generateUUID(),
		Name:        "Demo Content",
		Description: "This is a demo content",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create an object for the content
	object, err := s.service.CreateObject(ctx, simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "memory",
		Version:            1,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"content_id": "%s", "object_id": "%s", "message": "Demo completed successfully"}`, content.ID, object.ID)
}

func generateUUID() uuid.UUID {
	return uuid.New()
}