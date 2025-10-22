package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/api"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
	fmt.Println("🔧 Simple Content - Middleware Example")
	fmt.Println("========================================\n")

	// Create service
	svc, err := setupService()
	if err != nil {
		log.Fatal(err)
	}

	// Create server with middleware
	server := setupServer(svc)

	// Start server
	port := "8080"
	fmt.Printf("🚀 Server starting on http://localhost:%s\n", port)
	fmt.Println("\nEndpoints:")
	fmt.Println("  GET  /health              - Health check (no auth)")
	fmt.Println("  GET  /api/v1/contents     - List contents (requires auth)")
	fmt.Println("  POST /api/v1/contents     - Create content (requires auth)")
	fmt.Println("\nExample requests:")
	fmt.Println("  # Health check (no auth)")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("\n  # List contents (requires auth header)")
	fmt.Println("  curl -H 'Authorization: Bearer demo-token' http://localhost:8080/api/v1/contents")
	fmt.Println("\n  # Create content")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/contents \\")
	fmt.Println("    -H 'Authorization: Bearer demo-token' \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"name\": \"test.txt\", \"document_type\": \"text\"}'")
	fmt.Println("\nPress Ctrl+C to stop\n")

	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatal(err)
	}
}

func setupService() (simplecontent.Service, error) {
	// Create in-memory backend
	repo := memoryrepo.New()
	storage := memorystorage.New()

	// Create service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", storage),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return svc, nil
}

func setupServer(svc simplecontent.Service) http.Handler {
	// Create router
	r := chi.NewRouter()

	// Create logger
	logger := log.New(os.Stdout, "[API] ", log.LstdFlags)

	// Create metrics collector (simple in-memory)
	metrics := &SimpleMetrics{}

	// Create rate limiter
	limiter := api.NewRateLimiter(60) // 60 requests per minute

	// Public middleware chain (no auth)
	publicChain := api.NewMiddlewareChain(
		api.RequestIDMiddleware,
		api.RecoveryMiddleware,
		api.LoggingMiddleware(logger),
		api.CORSMiddleware([]string{"*"}, nil, nil),
	)

	// Protected middleware chain (with auth)
	protectedChain := api.NewMiddlewareChain(
		api.RequestIDMiddleware,
		api.RecoveryMiddleware,
		api.LoggingMiddleware(logger),
		limiter.Middleware,
		api.RequestSizeLimitMiddleware(10*1024*1024), // 10MB
		api.CORSMiddleware([]string{"*"}, nil, nil),
		api.AuthenticationMiddleware(demoAuthFunc),
		api.MetricsMiddleware(metrics),
		api.TimeoutMiddleware(30*time.Second),
	)

	// Public routes (no auth required)
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return publicChain.Wrap(next)
		})

		r.Get("/health", healthHandler)
		r.Get("/metrics", metrics.Handler)
	})

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return protectedChain.Wrap(next)
		})

		r.Route("/api/v1", func(r chi.Router) {
			setupContentRoutes(r, svc)
		})
	})

	return r
}

func setupContentRoutes(r chi.Router, svc simplecontent.Service) {
	r.Route("/contents", func(r chi.Router) {
		r.Get("/", listContentsHandler(svc))
		r.Post("/", createContentHandler(svc))
		r.Get("/{id}", getContentHandler(svc))
		r.Delete("/{id}", deleteContentHandler(svc))
	})
}

// Demo authentication function
func demoAuthFunc(r *http.Request) (userID, tenantID uuid.UUID, err error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("missing authorization header")
	}

	// Simple demo: accept any token starting with "Bearer "
	if len(token) < 7 || token[:7] != "Bearer " {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid token format")
	}

	// In production, validate JWT or API key here
	// For demo, just return fixed UUIDs
	userID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

	return userID, tenantID, nil
}

// Handlers

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func listContentsHandler(svc simplecontent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user/tenant from context (set by auth middleware)
		userID, _ := r.Context().Value(api.UserIDKey).(uuid.UUID)
		tenantID, _ := r.Context().Value(api.TenantIDKey).(uuid.UUID)

		contents, err := svc.ListContent(context.Background(), simplecontent.ListContentRequest{
			OwnerID:  userID,
			TenantID: tenantID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"contents": contents,
			"count":    len(contents),
		})
	}
}

func createContentHandler(svc simplecontent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user/tenant from context
		userID, _ := r.Context().Value(api.UserIDKey).(uuid.UUID)
		tenantID, _ := r.Context().Value(api.TenantIDKey).(uuid.UUID)

		// Parse request
		var req struct {
			Name         string `json:"name"`
			DocumentType string `json:"document_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create content
		content, err := svc.CreateContent(context.Background(), simplecontent.CreateContentRequest{
			OwnerID:      userID,
			TenantID:     tenantID,
			Name:         req.Name,
			DocumentType: req.DocumentType,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(content)
	}
}

func getContentHandler(svc simplecontent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid content ID", http.StatusBadRequest)
			return
		}

		content, err := svc.GetContent(context.Background(), contentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(content)
	}
}

func deleteContentHandler(svc simplecontent.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid content ID", http.StatusBadRequest)
			return
		}

		if err := svc.DeleteContent(context.Background(), contentID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Simple metrics collector

type SimpleMetrics struct {
	requests []RequestMetric
}

type RequestMetric struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	Size       int64
	Timestamp  time.Time
}

func (m *SimpleMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration, size int64) {
	m.requests = append(m.requests, RequestMetric{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   duration,
		Size:       size,
		Timestamp:  time.Now(),
	})

	// Keep only last 100 requests
	if len(m.requests) > 100 {
		m.requests = m.requests[1:]
	}
}

func (m *SimpleMetrics) Handler(w http.ResponseWriter, r *http.Request) {
	// Calculate stats
	total := len(m.requests)
	var totalDuration time.Duration
	statusCounts := make(map[int]int)

	for _, req := range m.requests {
		totalDuration += req.Duration
		statusCounts[req.StatusCode]++
	}

	avgDuration := time.Duration(0)
	if total > 0 {
		avgDuration = totalDuration / time.Duration(total)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_requests":      total,
		"average_duration_ms": avgDuration.Milliseconds(),
		"status_counts":       statusCounts,
		"recent_requests":     m.requests[max(0, len(m.requests)-10):], // Last 10
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
