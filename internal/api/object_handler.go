// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/service"
)

// ObjectHandler handles HTTP requests for objects
type ObjectHandler struct {
	objectService *service.ObjectService
}

// StandaloneObjectResponse is the response body for an object
type StandaloneObjectResponse struct {
	ID                 string    `json:"id"`
	ContentID          string    `json:"content_id"`
	StorageBackendName string    `json:"storage_backend_name"`
	Version            int       `json:"version"`
	ObjectKey          string    `json:"object_key"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// NewObjectHandler creates a new object handler
func NewObjectHandler(objectService *service.ObjectService) *ObjectHandler {
	return &ObjectHandler{
		objectService: objectService,
	}
}

// Routes returns the routes for objects
func (h *ObjectHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateObject)
	r.Get("/{id}", h.GetObject)
	r.Delete("/{id}", h.DeleteObject)
	r.Post("/{id}/upload", h.UploadObject)
	r.Get("/{id}/download", h.DownloadObject)
	r.Put("/{id}/metadata", h.UpdateMetadata)
	r.Get("/{id}/metadata", h.GetMetadata)

	return r
}

// CreateStandaloneObjectRequest represents a request to create an object directly
type CreateStandaloneObjectRequest struct {
	ContentID          string `json:"content_id"`
	StorageBackendName string `json:"storage_backend_name"`
	Version            int    `json:"version"`
}

// CreateObject creates a new object
func (h *ObjectHandler) CreateObject(w http.ResponseWriter, r *http.Request) {
	var req CreateStandaloneObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	contentID, err := uuid.Parse(req.ContentID)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: req.StorageBackendName,
		Version:            req.Version,
	}
	object, err := h.objectService.CreateObject(r.Context(), createObjectParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := StandaloneObjectResponse{
		ID:                 object.ID.String(),
		ContentID:          object.ContentID.String(),
		StorageBackendName: object.StorageBackendName,
		Version:            object.Version,
		ObjectKey:          object.ObjectKey,
		Status:             object.Status,
		CreatedAt:          object.CreatedAt,
		UpdatedAt:          object.UpdatedAt,
	}

	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, resp)
}

// GetObject retrieves an object by ID
func (h *ObjectHandler) GetObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	object, err := h.objectService.GetObject(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := StandaloneObjectResponse{
		ID:                 object.ID.String(),
		ContentID:          object.ContentID.String(),
		StorageBackendName: object.StorageBackendName,
		Version:            object.Version,
		ObjectKey:          object.ObjectKey,
		Status:             object.Status,
		CreatedAt:          object.CreatedAt,
		UpdatedAt:          object.UpdatedAt,
	}

	render.JSON(w, r, resp)
}

// DeleteObject deletes an object by ID
func (h *ObjectHandler) DeleteObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	if err := h.objectService.DeleteObject(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UploadObject uploads content to an object
func (h *ObjectHandler) UploadObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	// Limit the request body size
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB

	if err := h.objectService.UploadObject(r.Context(), id, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DownloadObject downloads content from an object
func (h *ObjectHandler) DownloadObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	reader, err := h.objectService.DownloadObject(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// For in-memory implementation, we'll read the entire content into memory
	// In a real implementation, we would stream the content directly to the response
	data, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get object metadata to determine content type
	metadata, err := h.objectService.GetObjectMetadata(r.Context(), id)
	if err == nil && metadata != nil {
		if contentType, ok := metadata["content_type"].(string); ok {
			w.Header().Set("Content-Type", contentType)
		}
	}

	// Set content disposition header
	w.Header().Set("Content-Disposition", "attachment")

	// Write the content to the response
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
}

// UpdateMetadata updates metadata for an object
func (h *ObjectHandler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.objectService.SetObjectMetadata(r.Context(), id, metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetMetadata retrieves metadata for an object
func (h *ObjectHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	metadata, err := h.objectService.GetObjectMetadata(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, metadata)
}
