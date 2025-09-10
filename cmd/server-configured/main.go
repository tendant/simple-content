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
    "strings"
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
		r.Post("/contents/{parentID}/derived", s.handleCreateDerivedContent)
		r.Get("/contents/{contentID}", s.handleGetContent)
		r.Get("/contents/{contentID}/derived", s.handleListDerivedForParent)
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
    if rel, err := s.service.GetDerivedRelationshipByContentID(r.Context(), id); err == nil && rel != nil {
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
    if rel, err := s.service.GetDerivedRelationshipByContentID(r.Context(), derived.ID); err == nil && rel != nil {
        variant = rel.DerivationType
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
        if rel, err := s.service.GetDerivedRelationshipByContentID(r.Context(), c.ID); err == nil && rel != nil {
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
    rels, err := s.service.ListDerivedByParent(r.Context(), parentID)
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

func (s *HTTPServer) handleSetContentMetadata(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "contentID")
    contentID, err := uuid.Parse(idStr)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
        return
    }
    var req struct {
        ContentType    string                 `json:"content_type"`
        Title          string                 `json:"title"`
        Description    string                 `json:"description"`
        Tags           []string               `json:"tags"`
        FileName       string                 `json:"file_name"`
        FileSize       int64                  `json:"file_size"`
        CreatedBy      string                 `json:"created_by"`
        CustomMetadata map[string]interface{} `json:"custom_metadata"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
        return
    }
    if err := s.service.SetContentMetadata(r.Context(), simplecontent.SetContentMetadataRequest{
        ContentID:      contentID,
        ContentType:    req.ContentType,
        Title:          req.Title,
        Description:    req.Description,
        Tags:           req.Tags,
        FileName:       req.FileName,
        FileSize:       req.FileSize,
        CreatedBy:      req.CreatedBy,
        CustomMetadata: req.CustomMetadata,
    }); err != nil {
        writeServiceError(w, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleGetContentMetadata(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "contentID")
    contentID, err := uuid.Parse(idStr)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID", nil)
        return
    }
    md, err := s.service.GetContentMetadata(r.Context(), contentID)
    if err != nil {
        writeServiceError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, md)
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
    obj, err := s.service.CreateObject(r.Context(), simplecontent.CreateObjectRequest{
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
    obj, err := s.service.GetObject(r.Context(), id)
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
    if err := s.service.DeleteObject(r.Context(), id); err != nil {
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
    objs, err := s.service.GetObjectsByContentID(r.Context(), contentID)
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
    if mimeType != "" && !strings.HasPrefix(mimeType, "multipart/") {
        if err := s.service.UploadObjectWithMetadata(r.Context(), r.Body, simplecontent.UploadObjectWithMetadataRequest{ObjectID: id, MimeType: mimeType}); err != nil {
            writeServiceError(w, err)
            return
        }
    } else {
        if err := s.service.UploadObject(r.Context(), id, r.Body); err != nil {
            writeServiceError(w, err)
            return
        }
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
    rc, err := s.service.DownloadObject(r.Context(), id)
    if err != nil {
        writeServiceError(w, err)
        return
    }
    defer rc.Close()
    if md, mdErr := s.service.GetObjectMetadata(r.Context(), id); mdErr == nil {
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
    url, err := s.service.GetUploadURL(r.Context(), id)
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
    url, err := s.service.GetDownloadURL(r.Context(), id)
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
    url, err := s.service.GetPreviewURL(r.Context(), id)
    if err != nil {
        writeServiceError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

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
