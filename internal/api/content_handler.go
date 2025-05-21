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

// ContentHandler handles HTTP requests for content
type ContentHandler struct {
	contentService *service.ContentService
	objectService  *service.ObjectService
}

// NewContentHandler creates a new content handler
func NewContentHandler(
	contentService *service.ContentService,
	objectService *service.ObjectService,
) *ContentHandler {
	return &ContentHandler{
		contentService: contentService,
		objectService:  objectService,
	}
}

// Routes returns the routes for content
func (h *ContentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateContent)
	r.Get("/{id}", h.GetContent)
	r.Delete("/{id}", h.DeleteContent)
	r.Get("/list", h.ListContents)

	r.Put("/{id}/metadata", h.UpdateMetadata)
	r.Get("/{id}/metadata", h.GetMetadata)

	r.Post("/{id}/objects", h.CreateObject)
	r.Get("/{id}/objects", h.ListObjects)
	r.Get("/{id}/download", h.GetDownload)

	// Routes for derived content
	r.Post("/{id}/derive", h.CreateDerivedContent)
	r.Get("/{id}/derived", h.GetDerivedContent)
	r.Get("/{id}/derived-tree", h.GetDerivedContentTree)

	return r
}

// CreateContentRequest is the request body for creating a content
type CreateContentRequest struct {
	OwnerID  string `json:"owner_id"`
	TenantID string `json:"tenant_id"`
}

// ContentResponse is the response body for a content
type ContentResponse struct {
	ID              string    `json:"id"`
	ParentID        string    `json:"parent_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	OwnerID         string    `json:"owner_id"`
	TenantID        string    `json:"tenant_id"`
	Status          string    `json:"status"`
	DerivationType  string    `json:"derivation_type"`
	DerivationLevel int       `json:"derivation_level"`
}

// CreateContent creates a new content
func (h *ContentHandler) CreateContent(w http.ResponseWriter, r *http.Request) {
	var req CreateContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	content, err := h.contentService.CreateContent(r.Context(), ownerID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ContentResponse{
		ID:              content.ID.String(),
		CreatedAt:       content.CreatedAt,
		UpdatedAt:       content.UpdatedAt,
		OwnerID:         content.OwnerID.String(),
		TenantID:        content.TenantID.String(),
		Status:          content.Status,
		DerivationType:  content.DerivationType,
		DerivationLevel: content.DerivationLevel,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// GetContent retrieves a content by ID
func (h *ContentHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	content, err := h.contentService.GetContent(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	parentIDStr := ""
	if content.ParentID != nil {
		parentIDStr = content.ParentID.String()
	}

	resp := ContentResponse{
		ID:              content.ID.String(),
		ParentID:        parentIDStr,
		CreatedAt:       content.CreatedAt,
		UpdatedAt:       content.UpdatedAt,
		OwnerID:         content.OwnerID.String(),
		TenantID:        content.TenantID.String(),
		Status:          content.Status,
		DerivationType:  content.DerivationType,
		DerivationLevel: content.DerivationLevel,
	}

	render.JSON(w, r, resp)
}

// DeleteContent deletes a content by ID
func (h *ContentHandler) DeleteContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	if err := h.contentService.DeleteContent(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListContentsRequest is the query parameters for listing contents
type ListContentsRequest struct {
	OwnerID  string `json:"owner_id"`
	TenantID string `json:"tenant_id"`
}

// ListContents lists contents by owner ID and tenant ID
func (h *ContentHandler) ListContents(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	tenantIDStr := r.URL.Query().Get("tenant_id")

	var ownerID, tenantID uuid.UUID
	var err error

	if ownerIDStr != "" {
		ownerID, err = uuid.Parse(ownerIDStr)
		if err != nil {
			http.Error(w, "Invalid owner ID", http.StatusBadRequest)
			return
		}
	}

	if tenantIDStr != "" {
		tenantID, err = uuid.Parse(tenantIDStr)
		if err != nil {
			http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
			return
		}
	}

	contents, err := h.contentService.ListContents(r.Context(), ownerID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []ContentResponse
	for _, content := range contents {
		parentIDStr := ""
		if content.ParentID != nil {
			parentIDStr = content.ParentID.String()
		}

		resp = append(resp, ContentResponse{
			ID:              content.ID.String(),
			ParentID:        parentIDStr,
			CreatedAt:       content.CreatedAt,
			UpdatedAt:       content.UpdatedAt,
			OwnerID:         content.OwnerID.String(),
			TenantID:        content.TenantID.String(),
			Status:          content.Status,
			DerivationType:  content.DerivationType,
			DerivationLevel: content.DerivationLevel,
		})
	}

	render.JSON(w, r, resp)
}

// CreateDerivedContentRequest is the request body for creating derived content
type CreateDerivedContentRequest struct {
	OwnerID  string `json:"owner_id"`
	TenantID string `json:"tenant_id"`
}

// CreateDerivedContent creates a new content derived from an existing content
func (h *ContentHandler) CreateDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentIDStr := chi.URLParam(r, "id")
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		http.Error(w, "Invalid parent content ID", http.StatusBadRequest)
		return
	}

	var req CreateDerivedContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	content, err := h.contentService.CreateDerivedContent(r.Context(), parentID, ownerID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parentIDStr = ""
	if content.ParentID != nil {
		parentIDStr = content.ParentID.String()
	}

	resp := ContentResponse{
		ID:              content.ID.String(),
		ParentID:        parentIDStr,
		CreatedAt:       content.CreatedAt,
		UpdatedAt:       content.UpdatedAt,
		OwnerID:         content.OwnerID.String(),
		TenantID:        content.TenantID.String(),
		Status:          content.Status,
		DerivationType:  content.DerivationType,
		DerivationLevel: content.DerivationLevel,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// GetDerivedContent retrieves all content directly derived from a specific parent
func (h *ContentHandler) GetDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentIDStr := chi.URLParam(r, "id")
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		http.Error(w, "Invalid parent content ID", http.StatusBadRequest)
		return
	}

	contents, err := h.contentService.GetDerivedContent(r.Context(), parentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []ContentResponse
	for _, content := range contents {
		parentIDStr := ""
		if content.ParentID != nil {
			parentIDStr = content.ParentID.String()
		}

		resp = append(resp, ContentResponse{
			ID:              content.ID.String(),
			ParentID:        parentIDStr,
			CreatedAt:       content.CreatedAt,
			UpdatedAt:       content.UpdatedAt,
			OwnerID:         content.OwnerID.String(),
			TenantID:        content.TenantID.String(),
			Status:          content.Status,
			DerivationType:  content.DerivationType,
			DerivationLevel: content.DerivationLevel,
		})
	}

	render.JSON(w, r, resp)
}

// GetDerivedContentTree retrieves the entire tree of derived content
func (h *ContentHandler) GetDerivedContentTree(w http.ResponseWriter, r *http.Request) {
	rootIDStr := chi.URLParam(r, "id")
	rootID, err := uuid.Parse(rootIDStr)
	if err != nil {
		http.Error(w, "Invalid root content ID", http.StatusBadRequest)
		return
	}

	contents, err := h.contentService.GetDerivedContentTree(r.Context(), rootID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []ContentResponse
	for _, content := range contents {
		parentIDStr := ""
		if content.ParentID != nil {
			parentIDStr = content.ParentID.String()
		}

		resp = append(resp, ContentResponse{
			ID:              content.ID.String(),
			ParentID:        parentIDStr,
			CreatedAt:       content.CreatedAt,
			UpdatedAt:       content.UpdatedAt,
			OwnerID:         content.OwnerID.String(),
			TenantID:        content.TenantID.String(),
			Status:          content.Status,
			DerivationType:  content.DerivationType,
			DerivationLevel: content.DerivationLevel,
		})
	}

	render.JSON(w, r, resp)
}

// ContentMetadataRequest is the request body for updating content metadata
type ContentMetadataRequest struct {
	ContentType string                 `json:"content_type"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	FileSize    int64                  `json:"file_size,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ContentMetadataResponse is the response body for content metadata
type ContentMetadataResponse struct {
	ContentID   string                 `json:"content_id"`
	ContentType string                 `json:"content_type"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	FileSize    int64                  `json:"file_size,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMetadata updates metadata for a content
func (h *ContentHandler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req ContentMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.contentService.SetContentMetadata(
		r.Context(),
		id,
		req.ContentType,
		req.Title,
		req.Description,
		req.Tags,
		req.FileSize,
		req.CreatedBy,
		req.Metadata,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetMetadata retrieves metadata for a content
func (h *ContentHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	metadata, err := h.contentService.GetContentMetadata(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ContentMetadataResponse{
		ContentID:   metadata.ContentID.String(),
		ContentType: metadata.ContentType,
		Title:       metadata.Title,
		Description: metadata.Description,
		Tags:        metadata.Tags,
		FileSize:    metadata.FileSize,
		CreatedBy:   metadata.CreatedBy,
		Metadata:    metadata.Metadata,
	}

	render.JSON(w, r, resp)
}

// CreateObjectRequest is the request body for creating an object
type CreateObjectRequest struct {
	StorageBackendName string `json:"storage_backend_name"`
	Version            int    `json:"version"`
}

// ObjectResponse is the response body for an object
type ObjectResponse struct {
	ID                 string    `json:"id"`
	ContentID          string    `json:"content_id"`
	StorageBackendName string    `json:"storage_backend_name"`
	Version            int       `json:"version"`
	ObjectKey          string    `json:"object_key"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateObject creates a new object for a content
func (h *ContentHandler) CreateObject(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req CreateObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	object, err := h.objectService.CreateObject(r.Context(), contentID, req.StorageBackendName, req.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ObjectResponse{
		ID:                 object.ID.String(),
		ContentID:          object.ContentID.String(),
		StorageBackendName: object.StorageBackendName,
		Version:            object.Version,
		ObjectKey:          object.ObjectKey,
		Status:             object.Status,
		CreatedAt:          object.CreatedAt,
		UpdatedAt:          object.UpdatedAt,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// ListObjects lists objects for a content
func (h *ContentHandler) ListObjects(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []ObjectResponse
	for _, object := range objects {
		resp = append(resp, ObjectResponse{
			ID:                 object.ID.String(),
			ContentID:          object.ContentID.String(),
			StorageBackendName: object.StorageBackendName,
			Version:            object.Version,
			ObjectKey:          object.ObjectKey,
			Status:             object.Status,
			CreatedAt:          object.CreatedAt,
			UpdatedAt:          object.UpdatedAt,
		})
	}

	render.JSON(w, r, resp)
}

// GetDownload gets a download URL for a content
func (h *ContentHandler) GetDownload(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Get the latest object for this content
	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(objects) == 0 {
		http.Error(w, "No objects found for this content", http.StatusNotFound)
		return
	}

	// Find the latest version
	var latestObject *ObjectResponse
	latestVersion := 0
	for _, object := range objects {
		if object.Version > latestVersion {
			latestVersion = object.Version
			resp := ObjectResponse{
				ID:                 object.ID.String(),
				ContentID:          object.ContentID.String(),
				StorageBackendName: object.StorageBackendName,
				Version:            object.Version,
				ObjectKey:          object.ObjectKey,
				Status:             object.Status,
				CreatedAt:          object.CreatedAt,
				UpdatedAt:          object.UpdatedAt,
			}
			latestObject = &resp
		}
	}

	if latestObject == nil {
		http.Error(w, "No active objects found for this content", http.StatusNotFound)
		return
	}

	// For in-memory backend, we can't provide a direct URL, so we'll return the object ID
	// In a real implementation, this would return a signed URL or redirect to a download endpoint
	response := map[string]string{
		"object_id": latestObject.ID,
		"message":   "For in-memory backend, use the object_id to download the content directly",
	}

	render.JSON(w, r, response)
}
