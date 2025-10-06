package presigned

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// SignatureValidator is an interface for storage backends that support signature validation
// This allows presigned handlers to validate signatures without importing specific storage implementations
type SignatureValidator interface {
	IsSignedURLEnabled() bool
	ValidateUploadSignature(objectKey, signature string, expiresAt int64) error
	ValidateDownloadSignature(objectKey, signature string, expiresAt int64, filename string) error
	ValidatePreviewSignature(objectKey, signature string, expiresAt int64) error
}

// Handlers provides HTTP handlers for presigned upload/download URLs
// These handlers work with storage backends that support HMAC signature validation
type Handlers struct {
	blobStores     map[string]simplecontent.BlobStore
	defaultBackend string
}

// NewHandlers creates a new set of presigned URL handlers
// blobStores: map of storage backend name to BlobStore implementation
// defaultBackend: name of the default storage backend to use (typically "fs")
func NewHandlers(blobStores map[string]simplecontent.BlobStore, defaultBackend string) *Handlers {
	return &Handlers{
		blobStores:     blobStores,
		defaultBackend: defaultBackend,
	}
}

// HandleUpload handles PUT requests to presigned upload URLs
// This endpoint mimics S3 presigned URL behavior for filesystem storage
// URL format: PUT /upload/{objectKey...}?signature={hmac}&expires={timestamp}
// The objectKey can contain slashes (e.g., "originals/objects/ab/cd1234_file.pdf")
//
// Authentication:
// - If FS_SIGNATURE_SECRET_KEY is configured, validates HMAC signature and expiration
// - If not configured, allows all uploads (backward compatibility, not recommended for production)
func (h *Handlers) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// Extract object key from URL path
	// chi.URLParam(r, "*") gives us everything after /upload/
	objectKey := chi.URLParam(r, "*")
	if objectKey == "" {
		writeError(w, http.StatusBadRequest, "missing_object_key", "object key is required in URL path", nil)
		return
	}

	// Get the default storage backend (assumes filesystem)
	blobStore, ok := h.blobStores[h.defaultBackend]
	if !ok {
		writeError(w, http.StatusInternalServerError, "storage_backend_not_found",
			fmt.Sprintf("storage backend %s not found", h.defaultBackend), nil)
		return
	}

	// If the blob store supports signature validation and has it enabled, validate the signature
	if validator, ok := blobStore.(SignatureValidator); ok && validator.IsSignedURLEnabled() {
		// Extract signature and expiration from query parameters
		signature := r.URL.Query().Get("signature")
		expiresStr := r.URL.Query().Get("expires")

		if signature == "" {
			writeError(w, http.StatusUnauthorized, "missing_signature", "signature parameter is required", nil)
			return
		}
		if expiresStr == "" {
			writeError(w, http.StatusUnauthorized, "missing_expires", "expires parameter is required", nil)
			return
		}

		expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_expires", "expires parameter must be a valid timestamp", nil)
			return
		}

		// Validate signature
		if err := validator.ValidateUploadSignature(objectKey, signature, expiresAt); err != nil {
			log.Printf("Presigned upload signature validation failed for objectKey %s: %v", objectKey, err)
			writeError(w, http.StatusForbidden, "invalid_signature", err.Error(), nil)
			return
		}

		log.Printf("Presigned upload signature validated for objectKey: %s", objectKey)
	}

	// Upload the file to storage
	err := blobStore.Upload(r.Context(), objectKey, r.Body)
	if err != nil {
		log.Printf("Presigned upload failed for objectKey %s: %v", objectKey, err)
		writeError(w, http.StatusInternalServerError, "upload_failed",
			fmt.Sprintf("failed to upload file: %v", err), nil)
		return
	}

	log.Printf("Presigned upload succeeded for objectKey: %s", objectKey)

	// Return success (mimic S3 presigned URL response - typically 200 OK with empty body)
	w.WriteHeader(http.StatusOK)
}

// HandleDownload handles GET requests to presigned download URLs
// This endpoint mimics S3 presigned URL behavior for filesystem storage
// URL format: GET /download/{objectKey...}?signature={hmac}&expires={timestamp}&filename={name}
// The objectKey can contain slashes (e.g., "originals/objects/ab/cd1234_file.pdf")
//
// Authentication:
// - If FS_SIGNATURE_SECRET_KEY is configured, validates HMAC signature and expiration
// - If not configured, allows all downloads (backward compatibility, not recommended for production)
func (h *Handlers) HandleDownload(w http.ResponseWriter, r *http.Request) {
	// Extract object key from URL path
	objectKey := chi.URLParam(r, "*")
	if objectKey == "" {
		writeError(w, http.StatusBadRequest, "missing_object_key", "object key is required in URL path", nil)
		return
	}

	filename := r.URL.Query().Get("filename")

	// Get the default storage backend (assumes filesystem)
	blobStore, ok := h.blobStores[h.defaultBackend]
	if !ok {
		writeError(w, http.StatusInternalServerError, "storage_backend_not_found",
			fmt.Sprintf("storage backend %s not found", h.defaultBackend), nil)
		return
	}

	// If the blob store supports signature validation and has it enabled, validate the signature
	if validator, ok := blobStore.(SignatureValidator); ok && validator.IsSignedURLEnabled() {
		// Extract signature and expiration from query parameters
		signature := r.URL.Query().Get("signature")
		expiresStr := r.URL.Query().Get("expires")

		if signature == "" {
			writeError(w, http.StatusUnauthorized, "missing_signature", "signature parameter is required", nil)
			return
		}
		if expiresStr == "" {
			writeError(w, http.StatusUnauthorized, "missing_expires", "expires parameter is required", nil)
			return
		}

		expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_expires", "expires parameter must be a valid timestamp", nil)
			return
		}

		// Validate signature
		if err := validator.ValidateDownloadSignature(objectKey, signature, expiresAt, filename); err != nil {
			log.Printf("Presigned download signature validation failed for objectKey %s: %v", objectKey, err)
			writeError(w, http.StatusForbidden, "invalid_signature", err.Error(), nil)
			return
		}

		log.Printf("Presigned download signature validated for objectKey: %s", objectKey)
	}

	// Download and serve the file
	rc, err := blobStore.Download(r.Context(), objectKey)
	if err != nil {
		log.Printf("Presigned download failed for objectKey %s: %v", objectKey, err)
		writeError(w, http.StatusNotFound, "download_failed", "object not found", nil)
		return
	}
	defer rc.Close()

	// Set content headers
	if meta, err := blobStore.GetObjectMeta(r.Context(), objectKey); err == nil {
		w.Header().Set("Content-Type", meta.ContentType)
	}

	if filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	// Stream file to response
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("Presigned download copy error: %v", err)
	}
}

// HandlePreview handles GET requests to presigned preview URLs
// This endpoint mimics S3 presigned URL behavior for filesystem storage
// URL format: GET /preview/{objectKey...}?signature={hmac}&expires={timestamp}
// The objectKey can contain slashes (e.g., "originals/objects/ab/cd1234_file.pdf")
//
// Authentication:
// - If FS_SIGNATURE_SECRET_KEY is configured, validates HMAC signature and expiration
// - If not configured, allows all previews (backward compatibility, not recommended for production)
func (h *Handlers) HandlePreview(w http.ResponseWriter, r *http.Request) {
	// Extract object key from URL path
	objectKey := chi.URLParam(r, "*")
	if objectKey == "" {
		writeError(w, http.StatusBadRequest, "missing_object_key", "object key is required in URL path", nil)
		return
	}

	// Get the default storage backend (assumes filesystem)
	blobStore, ok := h.blobStores[h.defaultBackend]
	if !ok {
		writeError(w, http.StatusInternalServerError, "storage_backend_not_found",
			fmt.Sprintf("storage backend %s not found", h.defaultBackend), nil)
		return
	}

	// If the blob store supports signature validation and has it enabled, validate the signature
	if validator, ok := blobStore.(SignatureValidator); ok && validator.IsSignedURLEnabled() {
		// Extract signature and expiration from query parameters
		signature := r.URL.Query().Get("signature")
		expiresStr := r.URL.Query().Get("expires")

		if signature == "" {
			writeError(w, http.StatusUnauthorized, "missing_signature", "signature parameter is required", nil)
			return
		}
		if expiresStr == "" {
			writeError(w, http.StatusUnauthorized, "missing_expires", "expires parameter is required", nil)
			return
		}

		expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_expires", "expires parameter must be a valid timestamp", nil)
			return
		}

		// Validate signature
		if err := validator.ValidatePreviewSignature(objectKey, signature, expiresAt); err != nil {
			log.Printf("Presigned preview signature validation failed for objectKey %s: %v", objectKey, err)
			writeError(w, http.StatusForbidden, "invalid_signature", err.Error(), nil)
			return
		}

		log.Printf("Presigned preview signature validated for objectKey: %s", objectKey)
	}

	// Download and serve the file (same as download but without Content-Disposition header)
	rc, err := blobStore.Download(r.Context(), objectKey)
	if err != nil {
		log.Printf("Presigned preview failed for objectKey %s: %v", objectKey, err)
		writeError(w, http.StatusNotFound, "preview_failed", "object not found", nil)
		return
	}
	defer rc.Close()

	// Set content type for inline preview
	if meta, err := blobStore.GetObjectMeta(r.Context(), objectKey); err == nil {
		w.Header().Set("Content-Type", meta.ContentType)
	}

	// Stream file to response
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("Presigned preview copy error: %v", err)
	}
}

// Mount mounts the presigned handlers on a chi router
// This is a convenience method for chi users
func (h *Handlers) Mount(r chi.Router) {
	r.Put("/upload/*", h.HandleUpload)
	r.Get("/download/*", h.HandleDownload)
	r.Get("/preview/*", h.HandlePreview)
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if details != nil {
		errorResponse["error"].(map[string]interface{})["details"] = details
	}

	// Simple JSON encoding
	fmt.Fprintf(w, `{"error":{"code":"%s","message":"%s"}}`, code, message)
}
