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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

// DirectUploadService wraps the simple-content service for direct upload workflows
type DirectUploadService struct {
	svc         simplecontent.Service
	storageSvc  simplecontent.StorageService  // For object operations
}

// NewDirectUploadService creates a service configured for direct uploads
func NewDirectUploadService() (*DirectUploadService, error) {
	// For this example, we'll use MinIO (S3-compatible) running locally
	// You can start MinIO with: docker run -p 9000:9000 -p 9001:9001 minio/minio server /data --console-address ":9001"

	// Create config with S3 storage backend
	cfg := &config.ServerConfig{
		DatabaseType:          "memory",
		DefaultStorageBackend: "s3",
		StorageBackends: []config.StorageBackendConfig{
			{
				Name: "s3",
				Type: "s3",
				Config: map[string]interface{}{
					"region":                     "us-east-1",
					"bucket":                     "direct-upload-demo",
					"access_key_id":              getEnv("MINIO_ACCESS_KEY", "minioadmin"),
					"secret_access_key":          getEnv("MINIO_SECRET_KEY", "minioadmin"),
					"endpoint":                   getEnv("MINIO_ENDPOINT", "http://localhost:9000"),
					"use_ssl":                    false,
					"use_path_style":             true, // Required for MinIO
					"presign_duration":           1800, // 30 minutes
					"create_bucket_if_not_exist": true,
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	svc, err := cfg.BuildService()
	if err != nil {
		return nil, fmt.Errorf("failed to build service: %w", err)
	}

	// Cast the same service instance to StorageService interface
	// since our service implementation supports both interfaces
	storageSvc, ok := svc.(simplecontent.StorageService)
	if !ok {
		return nil, fmt.Errorf("service does not implement StorageService interface")
	}

	return &DirectUploadService{
		svc:        svc,
		storageSvc: storageSvc,
	}, nil
}

// PrepareUploadRequest contains parameters for preparing a direct upload
type PrepareUploadRequest struct {
	OwnerID     string   `json:"owner_id"`
	TenantID    string   `json:"tenant_id"`
	FileName    string   `json:"file_name"`
	ContentType string   `json:"content_type"`
	FileSize    int64    `json:"file_size"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// PrepareUploadResponse contains the prepared upload information
type PrepareUploadResponse struct {
	ContentID    string            `json:"content_id"`
	ObjectID     string            `json:"object_id"`
	UploadURL    string            `json:"upload_url"`
	ExpiresIn    int               `json:"expires_in"`
	UploadMethod string            `json:"upload_method"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// ConfirmUploadRequest contains parameters for confirming upload completion
type ConfirmUploadRequest struct {
	ObjectID string `json:"object_id"`
}

// PrepareDirectUpload prepares everything needed for a direct client upload
func (dus *DirectUploadService) PrepareDirectUpload(ctx context.Context, req PrepareUploadRequest) (*PrepareUploadResponse, error) {
	// Parse UUIDs
	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner_id: %w", err)
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	// Validate file size (max 100MB for demo)
	if req.FileSize > 100*1024*1024 {
		return nil, fmt.Errorf("file too large: maximum 100MB allowed")
	}

	// 1. Create the content entity
	content, err := dus.svc.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:      ownerID,
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		DocumentType: req.ContentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create content: %w", err)
	}

	// 2. Note: Content metadata is now handled automatically
	// Tags and custom metadata can be set on object level if needed

	// 3. Create object for storage
	object, err := dus.storageSvc.CreateObject(ctx, simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "s3",
		Version:            1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create object: %w", err)
	}

	// 4. Get presigned upload URL
	uploadURL, err := dus.storageSvc.GetUploadURL(ctx, object.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upload URL: %w", err)
	}

	return &PrepareUploadResponse{
		ContentID:    content.ID.String(),
		ObjectID:     object.ID.String(),
		UploadURL:    uploadURL,
		ExpiresIn:    1800, // 30 minutes
		UploadMethod: "PUT",
		Headers: map[string]string{
			"Content-Type": req.ContentType,
		},
	}, nil
}

// ConfirmUpload marks an upload as completed and updates object status
func (dus *DirectUploadService) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) error {
	objectID, err := uuid.Parse(req.ObjectID)
	if err != nil {
		return fmt.Errorf("invalid object_id: %w", err)
	}

	// Get the object to update its status
	object, err := dus.storageSvc.GetObject(ctx, objectID)
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	// Update object status to indicate upload completion
	object.Status = string(simplecontent.ObjectStatusUploaded)
	err = dus.storageSvc.UpdateObject(ctx, object)
	if err != nil {
		return fmt.Errorf("failed to update object status: %w", err)
	}

	// Sync metadata from storage backend
	_, err = dus.storageSvc.UpdateObjectMetaFromStorage(ctx, objectID)
	if err != nil {
		log.Printf("Warning: failed to sync object metadata from storage: %v", err)
		// Don't fail the confirmation for metadata sync issues
	}

	log.Printf("Upload confirmed for object %s", objectID)
	return nil
}

// GetUploadStatus retrieves the current status of an upload
func (dus *DirectUploadService) GetUploadStatus(ctx context.Context, objectID string) (map[string]interface{}, error) {
	id, err := uuid.Parse(objectID)
	if err != nil {
		return nil, fmt.Errorf("invalid object_id: %w", err)
	}

	object, err := dus.storageSvc.GetObject(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	content, err := dus.svc.GetContent(ctx, object.ContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	details, err := dus.svc.GetContentDetails(ctx, content.ID)
	if err != nil {
		log.Printf("Warning: failed to get content details: %v", err)
		details = nil
	}

	status := map[string]interface{}{
		"object_id":    object.ID.String(),
		"content_id":   content.ID.String(),
		"status":       object.Status,
		"version":      object.Version,
		"created_at":   object.CreatedAt,
		"updated_at":   object.UpdatedAt,
		"content_name": content.Name,
		"content_type": content.DocumentType,
	}

	if details != nil {
		status["file_name"] = details.FileName
		status["file_size"] = details.FileSize
		status["tags"] = details.Tags
	}

	return status, nil
}

// HTTP Handlers

func (dus *DirectUploadService) handlePrepareUpload(w http.ResponseWriter, r *http.Request) {
	var req PrepareUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if req.OwnerID == "" || req.TenantID == "" || req.FileName == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_required_fields", "owner_id, tenant_id, file_name, and name are required")
		return
	}

	response, err := dus.PrepareDirectUpload(r.Context(), req)
	if err != nil {
		log.Printf("Failed to prepare upload: %v", err)
		writeError(w, http.StatusBadRequest, "prepare_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (dus *DirectUploadService) handleConfirmUpload(w http.ResponseWriter, r *http.Request) {
	var req ConfirmUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
		return
	}

	if req.ObjectID == "" {
		writeError(w, http.StatusBadRequest, "missing_object_id", "object_id is required")
		return
	}

	err := dus.ConfirmUpload(r.Context(), req)
	if err != nil {
		log.Printf("Failed to confirm upload: %v", err)
		writeError(w, http.StatusBadRequest, "confirm_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "confirmed",
		"object_id": req.ObjectID,
	})
}

func (dus *DirectUploadService) handleGetUploadStatus(w http.ResponseWriter, r *http.Request) {
	objectID := chi.URLParam(r, "objectID")
	if objectID == "" {
		writeError(w, http.StatusBadRequest, "missing_object_id", "objectID parameter is required")
		return
	}

	status, err := dus.GetUploadStatus(r.Context(), objectID)
	if err != nil {
		log.Printf("Failed to get upload status: %v", err)
		writeError(w, http.StatusNotFound, "status_not_found", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (dus *DirectUploadService) handleListContent(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	tenantIDStr := r.URL.Query().Get("tenant_id")

	if ownerIDStr == "" || tenantIDStr == "" {
		writeError(w, http.StatusBadRequest, "missing_parameters", "owner_id and tenant_id query parameters are required")
		return
	}

	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_owner_id", "Invalid owner_id format")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant_id format")
		return
	}

	contents, err := dus.svc.ListContent(r.Context(), simplecontent.ListContentRequest{
		OwnerID:  ownerID,
		TenantID: tenantID,
	})
	if err != nil {
		log.Printf("Failed to list content: %v", err)
		writeError(w, http.StatusInternalServerError, "list_failed", "Failed to list content")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"contents": contents,
		"count":    len(contents),
	})
}

// Utility functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Routes sets up the HTTP routes
func (dus *DirectUploadService) Routes() http.Handler {
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
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "healthy",
			"service": "direct-upload-demo",
		})
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Direct upload workflow
		r.Post("/uploads/prepare", dus.handlePrepareUpload)
		r.Post("/uploads/confirm", dus.handleConfirmUpload)
		r.Get("/uploads/status/{objectID}", dus.handleGetUploadStatus)

		// Content listing
		r.Get("/contents", dus.handleListContent)
	})

	// Serve static demo page
	r.Get("/", dus.serveDemoPage)

	return r
}

func (dus *DirectUploadService) serveDemoPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Direct Upload Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, textarea, select { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #0056b3; }
        button:disabled { background: #ccc; cursor: not-allowed; }
        .progress { width: 100%; height: 20px; background: #f0f0f0; border-radius: 10px; margin: 10px 0; }
        .progress-bar { height: 100%; background: #007bff; border-radius: 10px; transition: width 0.3s; }
        .result { margin-top: 20px; padding: 10px; border-radius: 4px; }
        .success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
        #contentList { margin-top: 30px; }
        .content-item { border: 1px solid #ddd; padding: 10px; margin: 10px 0; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>Direct Upload Demo</h1>
    <p>This demo shows how to upload files directly to storage using presigned URLs.</p>

    <form id="uploadForm">
        <div class="form-group">
            <label for="file">Select File:</label>
            <input type="file" id="file" required>
        </div>

        <div class="form-group">
            <label for="name">Content Name:</label>
            <input type="text" id="name" required placeholder="My Document">
        </div>

        <div class="form-group">
            <label for="description">Description:</label>
            <textarea id="description" placeholder="Optional description"></textarea>
        </div>

        <div class="form-group">
            <label for="tags">Tags (comma-separated):</label>
            <input type="text" id="tags" placeholder="demo, upload, test">
        </div>

        <button type="submit" id="uploadBtn">Upload File</button>
    </form>

    <div class="progress" id="progressContainer" style="display: none;">
        <div class="progress-bar" id="progressBar"></div>
    </div>

    <div id="result"></div>

    <div id="contentList">
        <h2>Uploaded Content</h2>
        <button onclick="loadContent()">Refresh Content List</button>
        <div id="contentItems"></div>
    </div>

    <script>
        // Generate demo UUIDs
        const OWNER_ID = '550e8400-e29b-41d4-a716-446655440000';
        const TENANT_ID = '550e8400-e29b-41d4-a716-446655440001';

        class DirectUploadClient {
            constructor(baseURL) {
                this.baseURL = baseURL;
            }

            async uploadFile(file, metadata = {}) {
                try {
                    this.showProgress(0);

                    // Step 1: Prepare the upload
                    this.showResult('Preparing upload...', 'info');
                    const prepareResponse = await this.prepareUpload(file, metadata);

                    // Step 2: Upload directly to storage
                    this.showResult('Uploading to storage...', 'info');
                    await this.performDirectUpload(file, prepareResponse);

                    // Step 3: Confirm upload completion
                    this.showResult('Confirming upload...', 'info');
                    await this.confirmUpload(prepareResponse.object_id);

                    this.showResult('Upload completed successfully!', 'success');
                    this.hideProgress();

                    // Show upload details
                    this.showUploadDetails(prepareResponse);

                    // Refresh content list
                    loadContent();

                    return prepareResponse;
                } catch (error) {
                    console.error('Upload failed:', error);
                    this.showResult('Upload failed: ' + error.message, 'error');
                    this.hideProgress();
                    throw error;
                }
            }

            async prepareUpload(file, metadata) {
                const response = await fetch('/api/v1/uploads/prepare', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        owner_id: OWNER_ID,
                        tenant_id: TENANT_ID,
                        file_name: file.name,
                        content_type: file.type || 'application/octet-stream',
                        file_size: file.size,
                        name: metadata.name || file.name,
                        description: metadata.description || '',
                        tags: metadata.tags || [],
                    }),
                });

                if (!response.ok) {
                    const error = await response.json();
                    throw new Error(error.error?.message || 'Failed to prepare upload');
                }

                return await response.json();
            }

            async performDirectUpload(file, prepareResponse) {
                return new Promise((resolve, reject) => {
                    const xhr = new XMLHttpRequest();

                    // Track upload progress
                    xhr.upload.onprogress = (event) => {
                        if (event.lengthComputable) {
                            const percent = Math.round((event.loaded / event.total) * 100);
                            this.showProgress(percent);
                        }
                    };

                    xhr.onload = () => {
                        if (xhr.status >= 200 && xhr.status < 300) {
                            resolve();
                        } else {
                            reject(new Error('Upload failed: ' + xhr.statusText));
                        }
                    };

                    xhr.onerror = () => reject(new Error('Upload failed'));

                    xhr.open(prepareResponse.upload_method, prepareResponse.upload_url);

                    // Set headers
                    Object.entries(prepareResponse.headers || {}).forEach(([key, value]) => {
                        xhr.setRequestHeader(key, value);
                    });

                    xhr.send(file);
                });
            }

            async confirmUpload(objectId) {
                const response = await fetch('/api/v1/uploads/confirm', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        object_id: objectId,
                    }),
                });

                if (!response.ok) {
                    const error = await response.json();
                    throw new Error(error.error?.message || 'Failed to confirm upload');
                }

                return await response.json();
            }

            showProgress(percent) {
                const container = document.getElementById('progressContainer');
                const bar = document.getElementById('progressBar');
                container.style.display = 'block';
                bar.style.width = percent + '%';
            }

            hideProgress() {
                document.getElementById('progressContainer').style.display = 'none';
            }

            showResult(message, type) {
                const result = document.getElementById('result');
                result.innerHTML = '<div class="result ' + type + '">' + message + '</div>';
            }

            showUploadDetails(details) {
                const html = '<div class="result info">' +
                    '<h3>Upload Details</h3>' +
                    '<p><strong>Content ID:</strong> ' + details.content_id + '</p>' +
                    '<p><strong>Object ID:</strong> ' + details.object_id + '</p>' +
                    '</div>';
                document.getElementById('result').innerHTML += html;
            }
        }

        // Initialize uploader
        const uploader = new DirectUploadClient('');

        // Handle form submission
        document.getElementById('uploadForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const fileInput = document.getElementById('file');
            const file = fileInput.files[0];
            if (!file) return;

            const name = document.getElementById('name').value;
            const description = document.getElementById('description').value;
            const tags = document.getElementById('tags').value
                .split(',')
                .map(tag => tag.trim())
                .filter(tag => tag.length > 0);

            const uploadBtn = document.getElementById('uploadBtn');
            uploadBtn.disabled = true;
            uploadBtn.textContent = 'Uploading...';

            try {
                await uploader.uploadFile(file, { name, description, tags });
            } finally {
                uploadBtn.disabled = false;
                uploadBtn.textContent = 'Upload File';
            }
        });

        // Load content list
        async function loadContent() {
            try {
                const response = await fetch('/api/v1/contents?owner_id=' + OWNER_ID + '&tenant_id=' + TENANT_ID);
                const data = await response.json();

                const container = document.getElementById('contentItems');
                if (data.contents && data.contents.length > 0) {
                    container.innerHTML = data.contents.map(content =>
                        '<div class="content-item">' +
                        '<strong>' + content.name + '</strong><br>' +
                        'Type: ' + content.document_type + '<br>' +
                        'Status: ' + content.status + '<br>' +
                        'ID: ' + content.id +
                        '</div>'
                    ).join('');
                } else {
                    container.innerHTML = '<p>No content uploaded yet.</p>';
                }
            } catch (error) {
                console.error('Failed to load content:', error);
                document.getElementById('contentItems').innerHTML = '<p>Failed to load content.</p>';
            }
        }

        // Load content on page load
        loadContent();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Direct Upload Demo Server ===")
	fmt.Println()
	fmt.Println("This demo shows how to implement direct client uploads using presigned URLs.")
	fmt.Println("Prerequisites:")
	fmt.Println("  1. MinIO server running on http://localhost:9000")
	fmt.Println("     Start with: docker run -p 9000:9000 -p 9001:9001 minio/minio server /data --console-address \":9001\"")
	fmt.Println("  2. Access credentials: minioadmin/minioadmin (defaults)")
	fmt.Println()

	// Create service
	service, err := NewDirectUploadService()
	if err != nil {
		log.Fatal("Failed to create service:", err)
	}

	// Test connection by attempting to list contents (this will also create the bucket)
	fmt.Println("Testing storage connection...")
	testOwnerID := uuid.New()
	testTenantID := uuid.New()

	// Try to list content to verify connection
	_, err = service.svc.ListContent(ctx, simplecontent.ListContentRequest{
		OwnerID:  testOwnerID,
		TenantID: testTenantID,
	})
	if err != nil {
		log.Printf("Warning: Storage connection test failed: %v", err)
		fmt.Println("  Storage might not be available. Check MinIO is running.")
	} else {
		fmt.Println("  Storage connection: OK")
	}

	// Start HTTP server
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: service.Routes(),
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	fmt.Println("Open your browser to see the demo!")
	fmt.Println()
	fmt.Println("API Endpoints:")
	fmt.Println("  POST /api/v1/uploads/prepare   - Prepare direct upload")
	fmt.Println("  POST /api/v1/uploads/confirm   - Confirm upload completion")
	fmt.Println("  GET  /api/v1/uploads/status/{id} - Check upload status")
	fmt.Println("  GET  /api/v1/contents          - List uploaded content")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}