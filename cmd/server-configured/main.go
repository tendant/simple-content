package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
	"github.com/tendant/simple-content/pkg/simplecontent/presigned"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	s3storage "github.com/tendant/simple-content/pkg/simplecontent/storage/s3"
)

// loadServerConfigFromEnv constructs a ServerConfig by reading process environment variables.
// This keeps environment-specific logic within the executable instead of the library.
func loadServerConfigFromEnv() (*config.ServerConfig, error) {
	cfg, err := config.Load(config.WithEnv(""))
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

func main() {
	// Load configuration from environment
	serverConfig, err := loadServerConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Verify database connectivity (for postgres) before building the service
	if serverConfig.DatabaseType == "postgres" {
		if err := config.PingPostgres(serverConfig.DatabaseURL, serverConfig.DBSchema); err != nil {
			log.Fatalf("Database connectivity check failed: %v", err)
		}
		log.Printf("Database connectivity OK (schema: %s)", serverConfig.DBSchema)
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
		if serverConfig.EnableAdminAPI {
			log.Printf("Admin API: ENABLED (WARNING: Ensure authentication middleware is configured)")
		} else {
			log.Printf("Admin API: disabled")
		}

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
	service        simplecontent.Service
	storageService simplecontent.StorageService // For object operations
	adminService   admin.AdminService           // For admin operations
	repository     simplecontent.Repository     // For direct repository access (presigned uploads)
	blobStores     map[string]simplecontent.BlobStore // For direct blob storage access
	config         *config.ServerConfig
}

// NewHTTPServer creates a new HTTP server wrapper
func NewHTTPServer(service simplecontent.Service, serverConfig *config.ServerConfig) *HTTPServer {
	// Cast to StorageService for object operations
	storageService, ok := service.(simplecontent.StorageService)
	if !ok {
		log.Fatalf("Service does not implement StorageService interface - object operations will not be available")
	}

	// Build repository for direct access (needed for presigned uploads)
	repo := buildRepository(serverConfig)

	// Build blobstores for direct access (needed for presigned uploads)
	blobStores := buildBlobStores(serverConfig)

	// Create admin service if admin API is enabled
	var adminSvc admin.AdminService
	if serverConfig.EnableAdminAPI {
		adminSvc = admin.New(repo)
	}

	return &HTTPServer{
		service:        service,
		storageService: storageService,
		adminService:   adminSvc,
		repository:     repo,
		blobStores:     blobStores,
		config:         serverConfig,
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
		r.Post("/contents/{parentID}/derived", s.handleCreateDerivedContent)
		r.Get("/contents/{contentID}", s.handleGetContent)
		r.Get("/contents/{contentID}/derived", s.handleListDerivedForParent)
		r.Put("/contents/{contentID}", s.handleUpdateContent)
		r.Delete("/contents/{contentID}", s.handleDeleteContent)
		r.Get("/contents", s.handleListContents)

		// Content details (unified interface for clients)
		r.Get("/contents/{contentID}/details", s.handleGetContentDetails)

		// Content data access
		r.Get("/contents/{contentID}/download", s.handleContentDownload)
		r.Get("/contents/{contentID}/preview", s.handleContentPreview)
		r.Post("/contents/{contentID}/upload", s.handleContentUpload)

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

		// Presigned-style endpoints for filesystem storage (mimics S3 presigned URLs)
		// Handles PUT /upload/{objectKey...} and GET /download|preview/{objectKey...}
		presignedHandlers := presigned.NewHandlers(s.blobStores, s.config.DefaultStorageBackend)
		presignedHandlers.Mount(r)

		// Demo endpoint
		r.Get("/demo", s.handleDemo)

		// Configuration info
		r.Get("/config", s.handleGetConfig)

		// Admin API (conditionally enabled)
		if s.config.EnableAdminAPI {
			r.Route("/admin", func(r chi.Router) {
				// TODO: Add authentication middleware here in production
				r.Get("/contents", s.handleAdminListContents)
				r.Get("/contents/count", s.handleAdminCountContents)
				r.Get("/contents/stats", s.handleAdminGetStatistics)
			})
		}
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
	object, err := s.storageService.CreateObject(ctx, simplecontent.CreateObjectRequest{
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
	var req struct {
		OwnerID        string                 `json:"owner_id"`
		TenantID       string                 `json:"tenant_id"`
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		DocumentType   string                 `json:"document_type"`
		DerivationType string                 `json:"derivation_type"`
		Metadata       map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_owner_id", "owner_id must be a UUID", nil)
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tenant_id", "tenant_id must be a UUID", nil)
		return
	}

	content, err := s.service.CreateContent(r.Context(), simplecontent.CreateContentRequest{
		OwnerID:        ownerID,
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		DocumentType:   req.DocumentType,
		DerivationType: req.DerivationType,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, contentResponse(content, ""))
}

func (s *HTTPServer) handleGetContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}
	content, err := s.service.GetContent(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	variant := ""
	if rel, err := s.service.GetDerivedRelationship(r.Context(), id); err == nil && rel != nil {
		variant = rel.DerivationType
	}
	writeJSON(w, http.StatusOK, contentResponse(content, variant))
}

// handleCreateDerivedContent creates a derived Content linked to a parent content.
// Request body: { owner_id, tenant_id, derivation_type, variant, metadata }
func (s *HTTPServer) handleCreateDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentStr := chi.URLParam(r, "parentID")
	parentID, err := uuid.Parse(parentStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parent_id", "parentID must be a UUID", nil)
		return
	}
	var req struct {
		OwnerID        string                 `json:"owner_id"`
		TenantID       string                 `json:"tenant_id"`
		DerivationType string                 `json:"derivation_type"`
		Variant        string                 `json:"variant"`
		Metadata       map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_owner_id", "owner_id must be a UUID", nil)
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tenant_id", "tenant_id must be a UUID", nil)
		return
	}
	derived, err := s.service.CreateDerivedContent(r.Context(), simplecontent.CreateDerivedContentRequest{
		ParentID:       parentID,
		OwnerID:        ownerID,
		TenantID:       tenantID,
		DerivationType: req.DerivationType,
		Variant:        req.Variant,
		Metadata:       req.Metadata,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	variant := ""
	if rel, err := s.service.GetDerivedRelationship(r.Context(), derived.ID); err == nil && rel != nil {
		variant = rel.Variant
	}
	writeJSON(w, http.StatusCreated, contentResponse(derived, variant))
}

func (s *HTTPServer) handleUpdateContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}
	existing, err := s.service.GetContent(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var req struct {
		Name         *string `json:"name"`
		Description  *string `json:"description"`
		DocumentType *string `json:"document_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.DocumentType != nil {
		existing.DocumentType = *req.DocumentType
	}
	if err := s.service.UpdateContent(r.Context(), simplecontent.UpdateContentRequest{Content: existing}); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contentResponse(existing, ""))
}

func (s *HTTPServer) handleDeleteContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}
	if err := s.service.DeleteContent(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleListContents(w http.ResponseWriter, r *http.Request) {
	ownerStr := r.URL.Query().Get("owner_id")
	tenantStr := r.URL.Query().Get("tenant_id")
	if ownerStr == "" || tenantStr == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "owner_id and tenant_id are required", nil)
		return
	}
	ownerID, err := uuid.Parse(ownerStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_owner_id", "owner_id must be a UUID", nil)
		return
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tenant_id", "tenant_id must be a UUID", nil)
		return
	}
	contents, err := s.service.ListContent(r.Context(), simplecontent.ListContentRequest{OwnerID: ownerID, TenantID: tenantID})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]map[string]interface{}, 0, len(contents))
	for _, c := range contents {
		v := ""
		if rel, err := s.service.GetDerivedRelationship(r.Context(), c.ID); err == nil && rel != nil {
			v = rel.DerivationType
		}
		out = append(out, contentResponse(c, v))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleListDerivedForParent lists all derived contents for a given parent content ID.
// Response items include the child content (with derivation_type) and its variant.
func (s *HTTPServer) handleListDerivedForParent(w http.ResponseWriter, r *http.Request) {
	parentStr := chi.URLParam(r, "contentID")
	parentID, err := uuid.Parse(parentStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}

	// List derived relationships filtered by parent via service
	rels, err := s.service.ListDerivedContent(r.Context(), simplecontent.WithParentID(parentID))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// For each relationship, fetch the child content and build the response
	out := make([]map[string]interface{}, 0, len(rels))
	for _, rel := range rels {
		child, err := s.service.GetContent(r.Context(), rel.ContentID)
		if err != nil {
			// Skip if child content not found
			continue
		}
		// rel.DerivationType is the specific variant (COALESCE handled in repo)
		out = append(out, contentResponse(child, rel.DerivationType))
	}

	writeJSON(w, http.StatusOK, out)
}


func (s *HTTPServer) handleGetContentDetails(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}

	// Parse query parameters for options
	var options []simplecontent.ContentDetailsOption

	// Check if upload access is requested
	if uploadAccess := r.URL.Query().Get("upload_access"); uploadAccess == "true" {
		// Check if expiry time is specified
		if expiryStr := r.URL.Query().Get("expiry_seconds"); expiryStr != "" {
			if expirySeconds, err := strconv.Atoi(expiryStr); err == nil && expirySeconds > 0 {
				options = append(options, simplecontent.WithUploadAccessExpiry(expirySeconds))
			} else {
				options = append(options, simplecontent.WithUploadAccess())
			}
		} else {
			options = append(options, simplecontent.WithUploadAccess())
		}
	}

	details, err := s.service.GetContentDetails(r.Context(), contentID, options...)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, details)
}

func (s *HTTPServer) handleCreateObject(w http.ResponseWriter, r *http.Request) {
	// Support both styles: path param contentID or JSON body field
	pathContentID := chi.URLParam(r, "contentID")
	var req struct {
		ContentID          string `json:"content_id"`
		StorageBackendName string `json:"storage_backend_name"`
		Version            int    `json:"version"`
		ObjectKey          string `json:"object_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	cid := req.ContentID
	if cid == "" && pathContentID != "" {
		cid = pathContentID
	}
	contentID, err := uuid.Parse(cid)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "content_id must be a UUID", nil)
		return
	}
	backend := req.StorageBackendName
	if backend == "" {
		backend = s.config.DefaultStorageBackend
	}
	obj, err := s.storageService.CreateObject(r.Context(), simplecontent.CreateObjectRequest{
		ContentID:          contentID,
		StorageBackendName: backend,
		Version:            req.Version,
		ObjectKey:          req.ObjectKey,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, obj)
}

func (s *HTTPServer) handleGetObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	obj, err := s.storageService.GetObject(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *HTTPServer) handleDeleteObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	if err := s.storageService.DeleteObject(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleListObjects(w http.ResponseWriter, r *http.Request) {
	contentStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(contentStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}
	objs, err := s.storageService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, objs)
}

func (s *HTTPServer) handleUploadObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	mimeType := r.Header.Get("Content-Type")
	if strings.HasPrefix(mimeType, "multipart/") {
		mimeType = "" // Don't store multipart MIME type
	}

	req := simplecontent.UploadObjectRequest{
		ObjectID: id,
		Reader:   r.Body,
		MimeType: mimeType,
	}
	if err := s.storageService.UploadObject(r.Context(), req); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleDownloadObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	rc, err := s.storageService.DownloadObject(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	defer rc.Close()
	if md, mdErr := s.storageService.GetObjectMetadata(r.Context(), id); mdErr == nil {
		if mt, ok := md["mime_type"].(string); ok && mt != "" {
			w.Header().Set("Content-Type", mt)
		}
		if fn, ok := md["file_name"].(string); ok && fn != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))
		}
	}
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("download copy error: %v", err)
	}
}

func (s *HTTPServer) handleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	url, err := s.storageService.GetUploadURL(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (s *HTTPServer) handleGetDownloadURL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	url, err := s.storageService.GetDownloadURL(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (s *HTTPServer) handleGetPreviewURL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "objectID")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "objectID must be a UUID", nil)
		return
	}
	url, err := s.storageService.GetPreviewURL(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// Presigned handlers are now provided by pkg/simplecontent/presigned package
// See presigned.NewHandlers() for reusable HTTP handlers

// --- Helpers ---

type errorBody struct {
	Error struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var eb errorBody
	eb.Error.Code = code
	eb.Error.Message = message
	eb.Error.Details = details
	_ = json.NewEncoder(w).Encode(eb)
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	msg := err.Error()

	if errors.Is(err, simplecontent.ErrContentNotFound) || errors.Is(err, simplecontent.ErrObjectNotFound) {
		status = http.StatusNotFound
		code = "not_found"
	}
	if errors.Is(err, simplecontent.ErrInvalidContentStatus) || errors.Is(err, simplecontent.ErrInvalidObjectStatus) {
		status = http.StatusBadRequest
		code = "invalid_status"
	}
	if errors.Is(err, simplecontent.ErrUploadFailed) || errors.Is(err, simplecontent.ErrDownloadFailed) {
		status = http.StatusBadGateway
		code = "storage_error"
	}
	if errors.Is(err, simplecontent.ErrStorageBackendNotFound) {
		status = http.StatusBadRequest
		code = "storage_backend_not_found"
	}

	writeError(w, status, code, msg, nil)
}

// contentResponse augments a Content with explicit variant for clients.
// DerivationType on Content is the user-facing derivation type for derived items.
// Variant is optional and included when available (resolved from relationship).
func contentResponse(c *simplecontent.Content, variant string) map[string]interface{} {
	m := map[string]interface{}{
		"id":              c.ID,
		"tenant_id":       c.TenantID,
		"owner_id":        c.OwnerID,
		"owner_type":      c.OwnerType,
		"name":            c.Name,
		"description":     c.Description,
		"document_type":   c.DocumentType,
		"status":          c.Status,
		"derivation_type": c.DerivationType,
		"created_at":      c.CreatedAt,
		"updated_at":      c.UpdatedAt,
	}
	if variant != "" {
		m["variant"] = variant
	}
	return m
}

// handleContentDownload downloads content directly using content ID
func (s *HTTPServer) handleContentDownload(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}

	// Get the primary object for this content
	objects, err := s.storageService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if len(objects) == 0 {
		writeError(w, http.StatusNotFound, "no_objects", "No objects found for this content", nil)
		return
	}

	// Use the first object as primary
	primaryObject := objects[0]

	// Download the object data
	rc, err := s.storageService.DownloadObject(r.Context(), primaryObject.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	defer rc.Close()

	// Set appropriate headers
	if md, mdErr := s.storageService.GetObjectMetadata(r.Context(), primaryObject.ID); mdErr == nil {
		if mt, ok := md["mime_type"].(string); ok && mt != "" {
			w.Header().Set("Content-Type", mt)
		}
		if fn, ok := md["file_name"].(string); ok && fn != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))
		}
	}

	// Stream the content
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("content download copy error: %v", err)
	}
}

// handleContentPreview provides preview access to content using content ID
func (s *HTTPServer) handleContentPreview(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}

	// Get the primary object for this content
	objects, err := s.storageService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if len(objects) == 0 {
		writeError(w, http.StatusNotFound, "no_objects", "No objects found for this content", nil)
		return
	}

	// Use the first object as primary
	primaryObject := objects[0]

	// Download the object data for preview
	rc, err := s.storageService.DownloadObject(r.Context(), primaryObject.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	defer rc.Close()

	// Set appropriate headers for preview (inline content disposition)
	if md, mdErr := s.storageService.GetObjectMetadata(r.Context(), primaryObject.ID); mdErr == nil {
		if mt, ok := md["mime_type"].(string); ok && mt != "" {
			w.Header().Set("Content-Type", mt)
		}
		if fn, ok := md["file_name"].(string); ok && fn != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fn))
		}
	}

	// Stream the content
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("content preview copy error: %v", err)
	}
}

// handleContentUpload uploads content data directly using content ID
func (s *HTTPServer) handleContentUpload(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
		return
	}

	// Verify the content exists
	_, err = s.service.GetContent(r.Context(), contentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Get or create an object for this content
	objects, err := s.storageService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	var primaryObject *simplecontent.Object
	if len(objects) > 0 {
		// Use existing primary object
		primaryObject = objects[0]
	} else {
		// Create a new object
		obj, err := s.storageService.CreateObject(r.Context(), simplecontent.CreateObjectRequest{
			ContentID:          contentID,
			StorageBackendName: s.config.DefaultStorageBackend,
			Version:            1,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		primaryObject = obj
	}

	// Get content type from request
	mimeType := r.Header.Get("Content-Type")
	if strings.HasPrefix(mimeType, "multipart/") {
		mimeType = "" // Don't store multipart MIME type
	}

	// Upload the content data
	req := simplecontent.UploadObjectRequest{
		ObjectID: primaryObject.ID,
		Reader:   r.Body,
		MimeType: mimeType,
	}
	if err := s.storageService.UploadObject(r.Context(), req); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// buildAdminService builds the admin service from server config
// buildRepository creates a repository instance from server config
func buildRepository(serverConfig *config.ServerConfig) simplecontent.Repository {
	var repo simplecontent.Repository

	switch serverConfig.DatabaseType {
	case "postgres":
		// Connect to postgres
		poolConfig, err := pgxpool.ParseConfig(serverConfig.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to parse postgres URL: %v", err)
		}

		// Set search_path if schema is specified
		if serverConfig.DBSchema != "" {
			poolConfig.ConnConfig.RuntimeParams["search_path"] = serverConfig.DBSchema
		}

		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			log.Fatalf("Failed to connect to postgres: %v", err)
		}

		repo = repopg.NewWithPool(pool)
	case "memory":
		repo = memory.New()
	default:
		log.Fatalf("Unsupported database type: %s", serverConfig.DatabaseType)
	}

	return repo
}

// buildBlobStores creates blob storage backends from server config
// This duplicates the logic from config.buildStorageBackend because that method is private
func buildBlobStores(serverConfig *config.ServerConfig) map[string]simplecontent.BlobStore {
	blobStores := make(map[string]simplecontent.BlobStore)

	for _, backendCfg := range serverConfig.StorageBackends {
		var store simplecontent.BlobStore
		var err error

		switch backendCfg.Type {
		case "memory":
			store = memorystorage.New()

		case "fs":
			baseDir := getStringFromConfig(backendCfg.Config, "base_dir", "./data/storage")
			urlPrefix := getStringFromConfig(backendCfg.Config, "url_prefix", "")
			signatureSecretKey := getStringFromConfig(backendCfg.Config, "signature_secret_key", "")
			presignExpires := getIntFromConfig(backendCfg.Config, "presign_expires_seconds", 3600)
			store, err = fsstorage.New(fsstorage.Config{
				BaseDir:            baseDir,
				URLPrefix:          urlPrefix,
				SignatureSecretKey: signatureSecretKey,
				PresignExpires:     time.Duration(presignExpires) * time.Second,
			})

		case "s3":
			s3Cfg := s3storage.Config{
				Region:                 getStringFromConfig(backendCfg.Config, "region", "us-east-1"),
				Bucket:                 getStringFromConfig(backendCfg.Config, "bucket", ""),
				AccessKeyID:            getStringFromConfig(backendCfg.Config, "access_key_id", ""),
				SecretAccessKey:        getStringFromConfig(backendCfg.Config, "secret_access_key", ""),
				Endpoint:               getStringFromConfig(backendCfg.Config, "endpoint", ""),
				UseSSL:                 getBoolFromConfig(backendCfg.Config, "use_ssl", true),
				UsePathStyle:           getBoolFromConfig(backendCfg.Config, "use_path_style", false),
				PresignDuration:        getIntFromConfig(backendCfg.Config, "presign_duration", 3600),
				EnableSSE:              getBoolFromConfig(backendCfg.Config, "enable_sse", false),
				SSEAlgorithm:           getStringFromConfig(backendCfg.Config, "sse_algorithm", ""),
				SSEKMSKeyID:            getStringFromConfig(backendCfg.Config, "sse_kms_key_id", ""),
				CreateBucketIfNotExist: getBoolFromConfig(backendCfg.Config, "create_bucket_if_not_exist", false),
			}
			store, err = s3storage.New(s3Cfg)

		default:
			log.Fatalf("Unknown storage backend type: %s", backendCfg.Type)
		}

		if err != nil {
			log.Fatalf("Failed to build storage backend %s: %v", backendCfg.Name, err)
		}

		blobStores[backendCfg.Name] = store
	}

	return blobStores
}

// Helper functions to extract values from config map
func getStringFromConfig(config map[string]interface{}, key, defaultValue string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getBoolFromConfig(config map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getIntFromConfig(config map[string]interface{}, key string, defaultValue int) int {
	if v, ok := config[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
		// Handle float64 from JSON unmarshaling
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

func buildAdminService(serverConfig *config.ServerConfig) admin.AdminService {
	repo := buildRepository(serverConfig)
	return admin.New(repo)
}

// Admin HTTP Handlers

func (s *HTTPServer) handleAdminListContents(w http.ResponseWriter, r *http.Request) {
	if s.adminService == nil {
		writeError(w, http.StatusForbidden, "admin_disabled", "Admin API is not enabled", nil)
		return
	}

	// Parse query parameters for filters
	filters := admin.ContentFilters{}

	// Tenant filtering
	if tenantIDStr := r.URL.Query().Get("tenant_id"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant_id format", nil)
			return
		}
		filters.TenantID = &tenantID
	}

	// Owner filtering
	if ownerIDStr := r.URL.Query().Get("owner_id"); ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_owner_id", "Invalid owner_id format", nil)
			return
		}
		filters.OwnerID = &ownerID
	}

	// Status filtering
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = &status
	}

	// DerivationType filtering
	if derivationType := r.URL.Query().Get("derivation_type"); derivationType != "" {
		filters.DerivationType = &derivationType
	}

	// DocumentType filtering
	if documentType := r.URL.Query().Get("document_type"); documentType != "" {
		filters.DocumentType = &documentType
	}

	// Pagination
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 1000 {
				l = 1000 // max limit
			}
			limit = l
		}
	}
	filters.Limit = &limit

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters.Offset = &offset

	// Include deleted
	if includeDeletedStr := r.URL.Query().Get("include_deleted"); includeDeletedStr == "true" {
		filters.IncludeDeleted = true
	}

	// Call admin service
	resp, err := s.adminService.ListAllContents(r.Context(), admin.ListContentsRequest{
		Filters: filters,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *HTTPServer) handleAdminCountContents(w http.ResponseWriter, r *http.Request) {
	if s.adminService == nil {
		writeError(w, http.StatusForbidden, "admin_disabled", "Admin API is not enabled", nil)
		return
	}

	// Parse query parameters for filters (same as list)
	filters := admin.ContentFilters{}

	if tenantIDStr := r.URL.Query().Get("tenant_id"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant_id format", nil)
			return
		}
		filters.TenantID = &tenantID
	}

	if ownerIDStr := r.URL.Query().Get("owner_id"); ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_owner_id", "Invalid owner_id format", nil)
			return
		}
		filters.OwnerID = &ownerID
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = &status
	}

	if derivationType := r.URL.Query().Get("derivation_type"); derivationType != "" {
		filters.DerivationType = &derivationType
	}

	if documentType := r.URL.Query().Get("document_type"); documentType != "" {
		filters.DocumentType = &documentType
	}

	if includeDeletedStr := r.URL.Query().Get("include_deleted"); includeDeletedStr == "true" {
		filters.IncludeDeleted = true
	}

	// Call admin service
	resp, err := s.adminService.CountContents(r.Context(), admin.CountRequest{
		Filters: filters,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "count_failed", err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *HTTPServer) handleAdminGetStatistics(w http.ResponseWriter, r *http.Request) {
	if s.adminService == nil {
		writeError(w, http.StatusForbidden, "admin_disabled", "Admin API is not enabled", nil)
		return
	}

	// Parse filters (same as list/count)
	filters := admin.ContentFilters{}

	if tenantIDStr := r.URL.Query().Get("tenant_id"); tenantIDStr != "" {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant_id format", nil)
			return
		}
		filters.TenantID = &tenantID
	}

	// Parse options (what statistics to include)
	options := admin.DefaultStatisticsOptions() // Default: all enabled

	if includeStatus := r.URL.Query().Get("include_status"); includeStatus == "false" {
		options.IncludeStatusBreakdown = false
	}
	if includeTenant := r.URL.Query().Get("include_tenant"); includeTenant == "false" {
		options.IncludeTenantBreakdown = false
	}
	if includeDerivation := r.URL.Query().Get("include_derivation"); includeDerivation == "false" {
		options.IncludeDerivationBreakdown = false
	}
	if includeDocType := r.URL.Query().Get("include_document_type"); includeDocType == "false" {
		options.IncludeDocumentTypeBreakdown = false
	}
	if includeTime := r.URL.Query().Get("include_time_range"); includeTime == "false" {
		options.IncludeTimeRange = false
	}

	// Call admin service
	resp, err := s.adminService.GetStatistics(r.Context(), admin.StatisticsRequest{
		Filters: filters,
		Options: options,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "stats_failed", err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
