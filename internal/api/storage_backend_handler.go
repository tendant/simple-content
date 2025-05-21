package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/service"
)

// StorageBackendHandler handles HTTP requests for storage backends
type StorageBackendHandler struct {
	storageBackendService *service.StorageBackendService
}

// NewStorageBackendHandler creates a new storage backend handler
func NewStorageBackendHandler(storageBackendService *service.StorageBackendService) *StorageBackendHandler {
	return &StorageBackendHandler{
		storageBackendService: storageBackendService,
	}
}

// Routes returns the routes for storage backends
func (h *StorageBackendHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateStorageBackend)
	r.Get("/{id}", h.GetStorageBackend)
	r.Put("/{id}", h.UpdateStorageBackend)
	r.Delete("/{id}", h.DeleteStorageBackend)
	r.Get("/", h.ListStorageBackends)

	return r
}

// CreateStorageBackendRequest is the request body for creating a storage backend
type CreateStorageBackendRequest struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// StorageBackendResponse is the response body for a storage backend
type StorageBackendResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// CreateStorageBackend creates a new storage backend
func (h *StorageBackendHandler) CreateStorageBackend(w http.ResponseWriter, r *http.Request) {
	var req CreateStorageBackendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	backend, err := h.storageBackendService.CreateStorageBackend(r.Context(), req.Name, req.Type, req.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := StorageBackendResponse{
		ID:        backend.ID.String(),
		Name:      backend.Name,
		Type:      backend.Type,
		Config:    backend.Config,
		IsActive:  backend.IsActive,
		CreatedAt: backend.CreatedAt,
		UpdatedAt: backend.UpdatedAt,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// GetStorageBackend retrieves a storage backend by ID
func (h *StorageBackendHandler) GetStorageBackend(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid storage backend ID", http.StatusBadRequest)
		return
	}

	backend, err := h.storageBackendService.GetStorageBackend(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := StorageBackendResponse{
		ID:        backend.ID.String(),
		Name:      backend.Name,
		Type:      backend.Type,
		Config:    backend.Config,
		IsActive:  backend.IsActive,
		CreatedAt: backend.CreatedAt,
		UpdatedAt: backend.UpdatedAt,
	}

	render.JSON(w, r, resp)
}

// UpdateStorageBackendRequest is the request body for updating a storage backend
type UpdateStorageBackendRequest struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Config   map[string]interface{} `json:"config"`
	IsActive bool                   `json:"is_active"`
}

// UpdateStorageBackend updates a storage backend
func (h *StorageBackendHandler) UpdateStorageBackend(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid storage backend ID", http.StatusBadRequest)
		return
	}

	var req UpdateStorageBackendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	backend, err := h.storageBackendService.GetStorageBackend(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	backend.Name = req.Name
	backend.Type = req.Type
	backend.Config = req.Config
	backend.IsActive = req.IsActive

	if err := h.storageBackendService.UpdateStorageBackend(r.Context(), backend); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := StorageBackendResponse{
		ID:        backend.ID.String(),
		Name:      backend.Name,
		Type:      backend.Type,
		Config:    backend.Config,
		IsActive:  backend.IsActive,
		CreatedAt: backend.CreatedAt,
		UpdatedAt: backend.UpdatedAt,
	}

	render.JSON(w, r, resp)
}

// DeleteStorageBackend deletes a storage backend
func (h *StorageBackendHandler) DeleteStorageBackend(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid storage backend ID", http.StatusBadRequest)
		return
	}

	if err := h.storageBackendService.DeleteStorageBackend(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListStorageBackends lists all storage backends
func (h *StorageBackendHandler) ListStorageBackends(w http.ResponseWriter, r *http.Request) {
	backends, err := h.storageBackendService.ListStorageBackends(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []StorageBackendResponse
	for _, backend := range backends {
		resp = append(resp, StorageBackendResponse{
			ID:        backend.ID.String(),
			Name:      backend.Name,
			Type:      backend.Type,
			Config:    backend.Config,
			IsActive:  backend.IsActive,
			CreatedAt: backend.CreatedAt,
			UpdatedAt: backend.UpdatedAt,
		})
	}

	render.JSON(w, r, resp)
}
