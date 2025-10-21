package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/tendant/simple-content/pkg/simplecontent"
	memoryrep "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
	memorystore "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	"github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
	"github.com/tendant/simple-content/pkg/simplecontent/urlstrategy"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Read configuration from environment
	databaseURL := getEnvOrDefault("DATABASE_URL", "memory")
	dbSchema := getEnvOrDefault("DB_SCHEMA", "content")
	storageURL := getEnvOrDefault("STORAGE_URL", "memory://")
	storageName := getEnvOrDefault("STORAGE_BACKEND_NAME", "default")
	urlStrategyType := getEnvOrDefault("URL_STRATEGY", "content-based")

	// Log configuration
	log.Println("Starting AG-UI Server")
	log.Printf("Database: %s", maskPassword(databaseURL))
	log.Printf("Storage: %s", storageURL)
	log.Printf("Storage Backend: %s", storageName)
	log.Printf("URL Strategy: %s", urlStrategyType)

	// Initialize repository (database)
	var repo simplecontent.Repository
	var err error

	if strings.HasPrefix(databaseURL, "postgresql://") || strings.HasPrefix(databaseURL, "postgres://") {
		log.Printf("Connecting to PostgreSQL (schema: %s)", dbSchema)

		// Create connection pool
		poolConfig, err := pgxpool.ParseConfig(databaseURL)
		if err != nil {
			log.Fatalf("Failed to parse database URL: %v", err)
		}

		// Set search path for schema
		poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s", dbSchema))
			return err
		}

		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}

		repo = postgres.NewWithPool(pool)
		log.Println("Connected to PostgreSQL")
	} else {
		log.Println("Using in-memory repository")
		repo = memoryrep.New()
	}

	// Initialize storage backend
	blobStores := make(map[string]simplecontent.BlobStore)

	if strings.HasPrefix(storageURL, "s3://") {
		bucketName := strings.TrimPrefix(storageURL, "s3://")
		region := getEnvOrDefault("AWS_REGION", "us-east-1")
		endpoint := os.Getenv("S3_ENDPOINT")
		externalEndpoint := os.Getenv("S3_EXTERNAL_ENDPOINT")

		log.Printf("Initializing S3 storage (bucket: %s, region: %s)", bucketName, region)
		if endpoint != "" {
			log.Printf("S3 Endpoint: %s", endpoint)
		}
		if externalEndpoint != "" {
			log.Printf("S3 External Endpoint: %s", externalEndpoint)
		}

		s3Store, err := s3.New(s3.Config{
			Bucket:          bucketName,
			Region:          region,
			Endpoint:        endpoint,
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			UsePathStyle:    getEnvOrDefault("S3_FORCE_PATH_STYLE", "false") == "true",
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage: %v", err)
		}
		blobStores[storageName] = s3Store
		log.Println("S3 storage initialized")
	} else {
		log.Println("Using in-memory storage")
		blobStores[storageName] = memorystore.New()
	}

	// Initialize service with options
	var opts []simplecontent.Option
	opts = append(opts, simplecontent.WithRepository(repo))

	for name, store := range blobStores {
		opts = append(opts, simplecontent.WithBlobStore(name, store))
	}

	switch urlStrategyType {
	case "storage-delegated":
		log.Println("Using storage-delegated URL strategy")
		// Convert to urlstrategy.BlobStore map
		urlBlobStores := make(map[string]urlstrategy.BlobStore)
		for name, store := range blobStores {
			urlBlobStores[name] = store
		}
		urlStrategy := urlstrategy.NewStorageDelegatedStrategy(urlBlobStores)
		opts = append(opts, simplecontent.WithURLStrategy(urlStrategy))
	default:
		log.Println("Using content-based URL strategy")
		baseURL := getEnvOrDefault("API_BASE_URL", "/api/v5")
		urlStrategy := urlstrategy.NewContentBasedStrategy(baseURL)
		opts = append(opts, simplecontent.WithURLStrategy(urlStrategy))
	}

	svc, err := simplecontent.New(opts...)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	log.Println("Service initialized successfully")

	// Create handler
	handler := NewHandler(svc)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// AG-UI Protocol endpoints
	r.Post("/api/v5/content/upload", handler.UploadContent)
	r.Post("/api/v5/content/upload/done", handler.UploadContentDone)
	r.Get("/api/v5/content/contents/{contentId}", handler.GetContent)
	r.Get("/api/v5/content/contents", handler.ListContents)
	r.Delete("/api/v5/content/contents/{contentId}", handler.DeleteContent)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting AG-UI server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Handler handles AG-UI protocol requests
type Handler struct {
	service simplecontent.Service
}

func NewHandler(service simplecontent.Service) *Handler {
	return &Handler{service: service}
}

// UploadContent returns a presigned upload URL
// POST /api/v5/content/upload
func (h *Handler) UploadContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		MimeType  string `json:"mime_type"`
		Filename  string `json:"filename"`
		FileSize  int64  `json:"file_size"`
		OwnerID   string `json:"owner_id"`
		TenantID  string `json:"tenant_id"`
		OwnerType string `json:"owner_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Filename == "" || req.MimeType == "" {
		respondError(w, http.StatusBadRequest, "filename and mime_type are required")
		return
	}

	var ownerID uuid.UUID
	var tenantID uuid.UUID
	var err error
	if req.OwnerID != "" {
		ownerID, err = uuid.Parse(req.OwnerID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid owner ID")
			return
		}
	}
	if req.TenantID != "" {
		tenantID, err = uuid.Parse(req.TenantID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid tenant ID")
			return
		}
	}

	// 1. Create content
	content, err := h.service.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:      ownerID,
		TenantID:     tenantID,
		OwnerType:    req.OwnerType,
		Name:         req.Filename,
		DocumentType: req.MimeType,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create content: %v", err))
		return
	}

	// 2. Set content metadata
	if err := h.service.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
		ContentID:   content.ID,
		FileName:    req.Filename,
		ContentType: req.MimeType,
		FileSize:    req.FileSize,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to set content metadata: %v", err))
		return
	}

	// 3. Create object for the content
	objectID := uuid.New()
	object, err := h.service.(simplecontent.StorageService).CreateObject(ctx, simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "s3-default", // TODO: Make configurable
		Version:            1,
		ObjectKey:          fmt.Sprintf("%s/%s", content.ID.String(), objectID.String()),
		FileName:           req.Filename,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create object: %v", err))
		return
	}

	// 4. Set object metadata
	if err := h.service.(simplecontent.StorageService).SetObjectMetadata(ctx, object.ID, map[string]interface{}{
		"filename":  req.Filename,
		"mime_type": req.MimeType,
		"size":      req.FileSize,
	}); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to set object metadata: %v", err))
		return
	}

	// 5. Get upload URL
	uploadURL, err := h.service.(simplecontent.StorageService).GetUploadURL(ctx, object.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get upload URL: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"content_id": content.ID.String(),
		"upload_url": uploadURL,
	})
}

// UploadContentDone marks a content as uploaded
// POST /api/v5/content/upload/done
func (h *Handler) UploadContentDone(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		ContentID string `json:"content_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	contentID, err := uuid.Parse(req.ContentID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	// Update content status to processed
	if err := h.service.UpdateContentStatus(ctx, contentID, simplecontent.ContentStatusUploaded); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update status: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"content_id": contentID.String(),
		"status":     simplecontent.ContentStatusUploaded,
	})
}

// AnalyzeContent handles multimodal content analysis
// POST /api/v5/contents/analysis
func (h *Handler) AnalyzeContent(w http.ResponseWriter, r *http.Request) {

	var req ContentAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Create analysis job
	analysisID := uuid.New()

	// TODO: Store analysis in database
	// TODO: Queue for processing

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         analysisID.String(),
		"status":     "pending",
		"created_at": time.Now().UTC(),
	})
}

// GetAnalysisStatus returns the status of an analysis job
// GET /api/v5/contents/analysis/{analysisId}
func (h *Handler) GetAnalysisStatus(w http.ResponseWriter, r *http.Request) {
	analysisID := chi.URLParam(r, "analysisId")

	// TODO: Get from database
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":     analysisID,
		"status": "pending",
	})
}

// ListAnalyses lists all analyses
// GET /api/v5/contents/analysis
func (h *Handler) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	// TODO: Get from database with filters
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"analyses": []interface{}{},
		"total":    0,
	})
}

// GetContent returns metadata for a content
// GET /api/v5/contents/{contentId}/metadata
func (h *Handler) GetContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contentIDStr := chi.URLParam(r, "contentId")

	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	// Get content details for download URL
	details, err := h.service.GetContentDetails(ctx, contentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get content details")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"content_id":   contentID.String(),
		"file_name":    details.FileName,
		"file_size":    details.FileSize,
		"download_url": details.Download,
	})
}

// ListContents lists all contents
// GET /api/v5/contents
func (h *Handler) ListContents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Add pagination and filters
	// For now, list ALL contents by using zero UUIDs (which the service interprets as "list all")
	contents, err := h.service.ListContent(ctx, simplecontent.ListContentRequest{
		OwnerID:  uuid.Nil, // List all - no owner filter
		TenantID: uuid.Nil, // List all - no tenant filter
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list contents")
		return
	}

	var response []map[string]interface{}
	for _, content := range contents {
		details, err := h.service.GetContentDetails(ctx, content.ID)
		if err != nil {
			// Skip contents we can't get details for
			log.Printf("Failed to get details for content %s: %v", content.ID, err)
			continue
		}
		response = append(response, map[string]interface{}{
			"content_id":   content.ID.String(),
			"file_name":    details.FileName,
			"file_size":    details.FileSize,
			"download_url": details.Download,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"contents": response,
		"total":    len(response),
	})
}

// DeleteContent deletes a content
// DELETE /api/v5/contents/{contentId}
func (h *Handler) DeleteContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contentIDStr := chi.URLParam(r, "contentId")

	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid content ID")
		return
	}

	if err := h.service.DeleteContent(ctx, contentID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete content")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Content deleted successfully",
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error": message,
	})
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskPassword(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			credPart := parts[0]
			if strings.Contains(credPart, "://") {
				schemeAndCred := strings.Split(credPart, "://")
				if len(schemeAndCred) == 2 {
					scheme := schemeAndCred[0]
					creds := schemeAndCred[1]
					if strings.Contains(creds, ":") {
						userPass := strings.Split(creds, ":")
						return fmt.Sprintf("%s://%s:***@%s", scheme, userPass[0], parts[1])
					}
				}
			}
		}
	}
	return url
}
