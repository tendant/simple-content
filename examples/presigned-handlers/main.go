package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/presigned"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

// This example demonstrates how library clients can use the reusable
// presigned URL handlers from pkg/simplecontent/presigned package.
//
// The example shows:
// 1. Creating a filesystem storage backend
// 2. Creating presigned handlers
// 3. Mounting them on a custom HTTP server
//
// Usage:
//   go run ./examples/presigned-handlers
//
// Test with curl:
//   # Upload
//   curl -X PUT 'http://localhost:8080/upload/test.txt?signature=XXX&expires=YYY' -d 'Hello World'
//
//   # Download
//   curl 'http://localhost:8080/download/test.txt?signature=XXX&expires=YYY'
//
// Note: For signature generation, see examples/presigned-upload

func main() {
	port := getEnvOrDefault("PORT", "8080")
	baseDir := getEnvOrDefault("FS_BASE_DIR", "/tmp/example-presigned-handlers")
	secretKey := getEnvOrDefault("FS_SIGNATURE_SECRET_KEY", "test-secret-key")

	log.Printf("Starting presigned handlers example...")
	log.Printf("Storage directory: %s", baseDir)
	log.Printf("Secret key: %s", secretKey)

	// Create filesystem storage backend
	fsBackend, err := fsstorage.New(fsstorage.Config{
		BaseDir:            baseDir,
		URLPrefix:          "/api/v1",
		SignatureSecretKey: secretKey,
		PresignExpires:     1 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create filesystem backend: %v", err)
	}

	// Create blob stores map (in this example, we only have one backend)
	blobStores := map[string]simplecontent.BlobStore{
		"fs": fsBackend,
	}

	// Create presigned handlers
	presignedHandlers := presigned.NewHandlers(blobStores, "fs")

	// Set up HTTP router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Mount presigned handlers
	// This creates the following routes:
	//   PUT /upload/{objectKey...}   - Upload files with signature validation
	//   GET /download/{objectKey...} - Download files with signature validation
	//   GET /preview/{objectKey...}  - Preview files with signature validation
	presignedHandlers.Mount(r)

	// Add a health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})

	// Add an info endpoint showing how to use the API
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, `Presigned URL Handlers Example

Available endpoints:
  PUT /upload/{objectKey}?signature=XXX&expires=YYY
  GET /download/{objectKey}?signature=XXX&expires=YYY
  GET /preview/{objectKey}?signature=XXX&expires=YYY
  GET /health

For signature generation, see examples/presigned-upload

Storage directory: %s
Server listening on: http://localhost:%s
`, baseDir, port)
	})

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server listening on http://localhost:%s", port)
	log.Printf("Presigned upload endpoint: http://localhost:%s/upload/{objectKey}", port)
	log.Printf("Presigned download endpoint: http://localhost:%s/download/{objectKey}", port)
	log.Printf("Presigned preview endpoint: http://localhost:%s/preview/{objectKey}", port)
	log.Printf("\nVisit http://localhost:%s/ for usage instructions", port)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
