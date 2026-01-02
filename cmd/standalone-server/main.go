package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
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
	"github.com/tendant/simple-content/pkg/simplecontent/api"
	"github.com/tendant/simple-content/pkg/simplecontent/presets"
)

// Standalone simple-content server for quick testing
// Uses in-memory repository + filesystem storage (./dev-data)
// No database setup required

func main() {
	// Command-line flags
	portFlag := flag.String("port", "", "HTTP port (default: 4000)")
	storageDirFlag := flag.String("data-dir", "", "Storage directory (default: ./dev-data)")
	flag.Parse()

	// Configuration priority: CLI args > environment variables > defaults
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "4000"
	}

	storageDir := *storageDirFlag
	if storageDir == "" {
		storageDir = os.Getenv("STORAGE_DIR")
	}
	if storageDir == "" {
		storageDir = "./dev-data"
	}

	log.Println("=== Simple Content Standalone Server ===")
	log.Printf("  Mode: In-memory repository + filesystem storage")
	log.Printf("  Storage directory: %s", storageDir)
	log.Printf("  Port: %s", port)
	log.Println()

	// Initialize service with development preset
	svc, cleanup, err := presets.NewDevelopment(
		presets.WithDevStorage(storageDir),
		presets.WithDevPort(port),
	)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	defer cleanup()

	log.Println("✓ Service initialized")

	// Create HTTP server
	server := NewHTTPServer(svc, port, storageDir)

	// Create HTTP server instance
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: server.Routes(),
	}

	// Start server in goroutine
	go func() {
		log.Printf("✓ Server ready on http://localhost:%s", port)
		log.Println()
		log.Println("Available endpoints:")
		log.Println("  GET  /health                                - Health check")
		log.Println("  POST /api/v1/contents                       - Upload content")
		log.Println("  GET  /api/v1/contents/{id}                  - Get content")
		log.Println("  GET  /api/v1/contents/{id}/download         - Download content")
		log.Println("  POST /api/v1/contents/{id}/derived          - Create derived content")
		log.Println("  GET  /api/v1/contents/{id}/derived          - List derived content")
		log.Println("  GET  /api/v1/test                           - Run end-to-end test")
		log.Println()
		log.Println("Quick test:")
		log.Printf("  curl http://localhost:%s/api/v1/test\n", port)
		log.Println()

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// HTTPServer wraps the simple-content service
type HTTPServer struct {
	service    simplecontent.Service
	port       string
	storageDir string
}

// NewHTTPServer creates a new HTTP server wrapper
func NewHTTPServer(service simplecontent.Service, port, storageDir string) *HTTPServer {
	return &HTTPServer{
		service:    service,
		port:       port,
		storageDir: storageDir,
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

	// Health check
	r.Get("/health", s.handleHealth)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Content handlers using the API package
		contentHandler := api.NewContentHandler(s.service, s.service.(simplecontent.StorageService))
		r.Mount("/contents", contentHandler.Routes())

		// Test endpoint
		r.Get("/test", s.handleTest)
	})

	return r
}

// handleHealth returns health status
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"mode":        "standalone",
		"storage_dir": s.storageDir,
		"port":        s.port,
	})
}

// handleTest runs an end-to-end test
func (s *HTTPServer) handleTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Println("=== Running End-to-End Test ===")

	// Step 1: Upload original content
	log.Println("Step 1: Uploading original content...")

	testData := []byte("This is a test image file for content management")
	content, err := s.service.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		TenantID:     uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Name:         "Test Image",
		DocumentType: "image/jpeg",
		Reader:       bytes.NewReader(testData),
		FileName:     "test-image.jpg",
		Tags:         []string{"test", "image"},
	})
	if err != nil {
		log.Printf("Failed to upload content: %v", err)
		http.Error(w, fmt.Sprintf("Upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Content uploaded: %s (status: %s)", content.ID, content.Status)

	// Step 2: Get content details
	log.Println("Step 2: Getting content details...")

	details, err := s.service.GetContentDetails(ctx, content.ID)
	if err != nil {
		log.Printf("Failed to get content details: %v", err)
		http.Error(w, fmt.Sprintf("Get details failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Content details retrieved")
	log.Printf("  ID: %s", details.ID)
	log.Printf("  File Name: %s", details.FileName)
	log.Printf("  Ready: %t", details.Ready)

	// Step 3: Create derived content (thumbnail)
	log.Println("Step 3: Creating derived content...")

	thumbnailData := []byte("This is a thumbnail")
	derived, err := s.service.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
		ParentID:       content.ID,
		OwnerID:        content.OwnerID,
		TenantID:       content.TenantID,
		DerivationType: "thumbnail",
		Variant:        "thumbnail_256",
		Reader:         bytes.NewReader(thumbnailData),
		FileName:       "thumb_256.jpg",
		Tags:           []string{"thumbnail", "256x256"},
	})
	if err != nil {
		log.Printf("Failed to create derived content: %v", err)
		http.Error(w, fmt.Sprintf("Derived content creation failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Derived content created: %s", derived.ID)
	log.Printf("  Type: %s", derived.DerivationType)
	log.Printf("  Status: %s", derived.Status)

	// Step 4: List derived content
	log.Println("Step 4: Listing derived content...")

	derivedList, err := s.service.ListDerivedContent(ctx, simplecontent.WithParentID(content.ID))
	if err != nil {
		log.Printf("Failed to list derived content: %v", err)
		http.Error(w, fmt.Sprintf("List derived failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Found %d derived content(s)", len(derivedList))
	for _, d := range derivedList {
		log.Printf("  - Type: %s, Variant: %s, Status: %s", d.DerivationType, d.Variant, d.Status)
	}

	// Step 5: Get parent content with derived
	log.Println("Step 5: Getting content with derived...")

	withDerived, err := s.service.GetContentDetails(ctx, content.ID)
	if err != nil {
		log.Printf("Failed to get content with derived: %v", err)
		http.Error(w, fmt.Sprintf("Get with derived failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Content with derived retrieved")
	log.Printf("  Content ID: %s", withDerived.ID)
	log.Printf("  Derived count: %d", len(derivedList))

	log.Println("=== Test Complete ===")

	// Return test results
	response := map[string]interface{}{
		"test_status":   "success",
		"content_id":    content.ID.String(),
		"derived_id":    derived.ID.String(),
		"derived_count": len(derivedList),
		"content": map[string]interface{}{
			"id":     content.ID,
			"name":   content.Name,
			"status": content.Status,
			"ready":  withDerived.Ready,
		},
		"derived_contents": derivedList,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
