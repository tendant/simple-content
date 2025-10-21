package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/api"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func TestServerSetup(t *testing.T) {
	// Initialize repository and storage backend
	repo := memoryrepo.New()
	memBackend := memorystorage.New()

	// Initialize unified service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create storage service for advanced operations
	storageSvc, err := simplecontent.NewStorageService(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Initialize API handlers
	contentHandler := api.NewContentHandler(svc, storageSvc)

	// Routes
	r.Mount("/contents", contentHandler.Routes())

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected 'OK', got %s", w.Body.String())
	}
}

func TestContentRoutes(t *testing.T) {
	// Initialize repository and storage backend
	repo := memoryrepo.New()
	memBackend := memorystorage.New()

	// Initialize unified service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create storage service for advanced operations
	storageSvc, err := simplecontent.NewStorageService(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize API handlers
	contentHandler := api.NewContentHandler(svc, storageSvc)

	// Initialize router
	r := chi.NewRouter()
	r.Mount("/contents", contentHandler.Routes())

	// Test that routes are properly mounted
	testCases := []struct {
		method       string
		path         string
		expectRouted bool // true if route should be handled (not return generic 404)
		description  string
	}{
		{"GET", "/contents/list", true, "list contents endpoint"},
		{"GET", "/contents/123e4567-e89b-12d3-a456-426614174000", true, "get content by ID endpoint"},
		{"POST", "/contents/", true, "create content endpoint"},
		{"GET", "/contents/invalid-id", true, "invalid ID should be handled by content handler"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if tc.expectRouted {
			// For routed endpoints, we expect either success or a business logic error (not a generic 404)
			// A business logic 404 will have a specific error message, while routing 404 is generic
			if w.Code == http.StatusNotFound && w.Body.String() == "404 page not found\n" {
				t.Errorf("Route %s %s returned generic 404, routes may not be properly mounted", tc.method, tc.path)
			}
		} else {
			// For non-existent routes, we expect a generic 404
			if w.Code != http.StatusNotFound {
				t.Errorf("Route %s %s should return 404 but returned %d", tc.method, tc.path, w.Code)
			}
		}
	}
}
