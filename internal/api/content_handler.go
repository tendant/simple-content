// Deprecated: This service is deprecated and will be removed in a future version.
// Please use the new module instead.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/pkg/service"
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
	r.Get("/bulk", h.GetContentsByIDs)

	r.Put("/{id}/metadata", h.UpdateMetadata)
	r.Get("/{id}/metadata", h.GetMetadata)

	r.Post("/{id}/objects", h.CreateObject)
	r.Get("/{id}/objects", h.ListObjects)
	r.Get("/{id}/download", h.GetDownload)

	// Routes for derived content
	r.Post("/{id}/derived", h.CreateDerivedContent)
	r.Get("/{id}/derived", h.GetDerivedContent)
	r.Get("/{id}/derived-tree", h.GetDerivedContentTree)

	return r
}

// CreateContentRequest is the request body for creating a content
type CreateContentRequest struct {
	OwnerID        string `json:"owner_id"`
	TenantID       string `json:"tenant_id"`
	DocumentType   string `json:"document_type"`
	DerivationType string `json:"derivation_type"`
	FileName       string `json:"file_name"`
	OwnerType      string `json:"owner_type"`
	MimeType       string `json:"mime_type"`
	FileSize       int64  `json:"file_size"`
}

const MAX_CONTENTS_PER_REQUEST = 50

// ContentResponse is the response body for a content
type ContentResponse struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	OwnerID        string    `json:"owner_id"`
	TenantID       string    `json:"tenant_id"`
	Status         string    `json:"status"`
	DerivationType string    `json:"derivation_type"`
	MimeType       string    `json:"mime_type"`
	FileSize       int64     `json:"file_size"`
	FileName       string    `json:"file_name"`
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
		slog.Error("Invalid owner ID", "owner_id", req.OwnerID, "error", err)
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		slog.Error("Invalid tenant ID", "tenant_id", req.TenantID, "error", err)
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	// Create content
	createParams := service.CreateContentParams{
		OwnerID:        ownerID,
		TenantID:       tenantID,
		Title:          req.FileName,
		DocumentType:   req.DocumentType,
		DerivationType: req.DerivationType,
		Description:    "",
	}
	if req.DerivationType == "" {
		createParams.DerivationType = "original"
	}
	content, err := h.contentService.CreateContent(r.Context(), createParams)
	if err != nil {
		slog.Error("Fail to create content", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update content for missing fields
	content.OwnerType = req.OwnerType
	updateParams := service.UpdateContentParams{
		Content: content,
	}
	err = h.contentService.UpdateContent(r.Context(), updateParams)
	if err != nil {
		slog.Warn("Failed to update content", "err", err)
	}

	// Set content metadata
	slog.Info("Setting content metadata...")
	metadataParams := service.SetContentMetadataParams{
		ContentID:   content.ID,
		ContentType: req.MimeType,
		FileName:    req.FileName,
		Title:       req.FileName,
		Description: "description",
		Tags:        nil,
		FileSize:    req.FileSize,
		CreatedBy:   ownerID.String(),
	}
	err = h.contentService.SetContentMetadata(r.Context(), metadataParams)
	if err != nil {
		slog.Warn("Failed to set content metadata", "err", err)
	}

	resp := ContentResponse{
		ID:             content.ID.String(),
		CreatedAt:      content.CreatedAt,
		UpdatedAt:      content.UpdatedAt,
		OwnerID:        content.OwnerID.String(),
		TenantID:       content.TenantID.String(),
		Status:         content.Status,
		DerivationType: content.DerivationType,
	}

	slog.Info("Content created", "content", content)
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// GetContent retrieves a content by ID
func (h *ContentHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Get Content
	content, err := h.contentService.GetContent(r.Context(), id)
	if err != nil {
		slog.Error("Fail to get content", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	resp := ContentResponse{
		ID:             content.ID.String(),
		CreatedAt:      content.CreatedAt,
		UpdatedAt:      content.UpdatedAt,
		OwnerID:        content.OwnerID.String(),
		TenantID:       content.TenantID.String(),
		Status:         content.Status,
		DerivationType: content.DerivationType,
	}

	// Get Content Metadata
	contentMeta, err := h.contentService.GetContentMetadata(r.Context(), id)
	if err != nil {
		slog.Warn("Fail to get content metadata", "id", idStr)

	} else {
		resp.MimeType = contentMeta.MimeType
		resp.FileSize = contentMeta.FileSize
		resp.FileName = contentMeta.FileName
	}

	slog.Info("Content retrieved", "content", resp)
	render.JSON(w, r, resp)
}

// GetContentsByIDs retrieves multiple contents by their IDs
func (h *ContentHandler) GetContentsByIDs(w http.ResponseWriter, r *http.Request) {
	// Get the id parameters from the query string
	idStrings := r.URL.Query()["id"]
	if len(idStrings) == 0 {
		http.Error(w, "Missing required 'id' parameter", http.StatusBadRequest)
		return
	}
	if len(idStrings) > MAX_CONTENTS_PER_REQUEST {
		http.Error(w, "Too many IDs requested", http.StatusBadRequest)
		return
	}

	// Create a slice to hold the response
	var contents []ContentResponse

	// Process each ID
	for _, idStr := range idStrings {
		// Parse the UUID
		id, err := uuid.Parse(idStr)
		if err != nil {
			slog.Warn("Invalid content ID", "id", idStr)
			continue
		}

		// Get the content
		content, err := h.contentService.GetContent(r.Context(), id)
		if err != nil {
			slog.Warn("Fail to get content", "id", idStr)
			continue
		}

		// Create response for this content
		resp := ContentResponse{
			ID:             content.ID.String(),
			CreatedAt:      content.CreatedAt,
			UpdatedAt:      content.UpdatedAt,
			OwnerID:        content.OwnerID.String(),
			TenantID:       content.TenantID.String(),
			Status:         content.Status,
			DerivationType: content.DerivationType,
		}

		// Get Content Metadata
		contentMeta, err := h.contentService.GetContentMetadata(r.Context(), id)
		if err != nil {
			slog.Warn("Fail to get content metadata", "id", idStr)
		} else {
			resp.MimeType = contentMeta.MimeType
			resp.FileSize = contentMeta.FileSize
			resp.FileName = contentMeta.FileName
		}

		contents = append(contents, resp)
	}

	// Return all found contents
	render.JSON(w, r, contents)
}

// DeleteContent deletes a content by ID
func (h *ContentHandler) DeleteContent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	deleteParams := service.DeleteContentParams{
		ID: id,
	}
	if err := h.contentService.DeleteContent(r.Context(), deleteParams); err != nil {
		slog.Error("Fail to delete content", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Content deleted", "content_id", idStr)
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
			slog.Error("Invalid owner ID", "owner_id", ownerIDStr, "error", err)
			http.Error(w, "Invalid owner ID", http.StatusBadRequest)
			return
		}
	}

	if tenantIDStr != "" {
		tenantID, err = uuid.Parse(tenantIDStr)
		if err != nil {
			slog.Error("Invalid tenant ID", "tenant_id", tenantIDStr, "error", err)
			http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
			return
		}
	}

	listParams := service.ListContentParams{
		OwnerID:  ownerID,
		TenantID: tenantID,
	}
	contents, err := h.contentService.ListContent(r.Context(), listParams)
	if err != nil {
		slog.Error("Fail to list contents", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resp []ContentResponse
	for _, content := range contents {
		resp = append(resp, ContentResponse{
			ID:             content.ID.String(),
			CreatedAt:      content.CreatedAt,
			UpdatedAt:      content.UpdatedAt,
			OwnerID:        content.OwnerID.String(),
			TenantID:       content.TenantID.String(),
			Status:         content.Status,
			DerivationType: content.DerivationType,
		})
	}

	render.JSON(w, r, resp)
}

// CreateDerivedContentRequest is the request body for creating derived content
type CreateDerivedContentRequest struct {
	ParentContentID    uuid.UUID              `json:"parent_content_id"`
	DerivedContentID   uuid.UUID              `json:"derived_content_id"`
	DerivationType     string                 `json:"derivation_type"`
	DerivationParams   map[string]interface{} `json:"derivation_params"`
	ProcessingMetadata map[string]interface{} `json:"processing_metadata"`
}

type CreateDerivedContentResponse struct {
	ParentContentID  string `json:"parent_content_id"`
	DerivedContentID string `json:"derived_content_id"`
	DerivationType   string `json:"derivation_type"`
}

// CreateDerivedContent creates a new derived content from a parent content
func (h *ContentHandler) CreateDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentIDStr := chi.URLParam(r, "id")
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		http.Error(w, "Invalid parent content ID", http.StatusBadRequest)
		return
	}

	slog.Info("Creating derived content", "parent_id", parentIDStr)
	var req CreateDerivedContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("Creating derived content", "derived_content_id", req)
	deriveParams := service.CreateDerivedRelationshipParams{
		ParentID:           parentID,
		DerivedContentID:   req.DerivedContentID,
		DerivationType:     req.DerivationType,
		DerivationParams:   req.DerivationParams,
		ProcessingMetadata: req.ProcessingMetadata,
	}
	if err := h.contentService.CreateDerivedContentRelationship(r.Context(), deriveParams); err != nil {
		slog.Error("Failed to create derived content relationship", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CreateDerivedContentResponse{
		ParentContentID:  parentIDStr,
		DerivedContentID: req.DerivedContentID.String(),
		DerivationType:   req.DerivationType,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// GetDerivedContent retrieves all content directly derived from a specific parent
func (h *ContentHandler) GetDerivedContent(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, nil)
}

// GetDerivedContentTree retrieves the entire tree of derived content
func (h *ContentHandler) GetDerivedContentTree(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, nil)
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
	ContentID         string                 `json:"content_id"`
	MimeType          string                 `json:"mime_type"`
	FileName          string                 `json:"file_name,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	FileSize          int64                  `json:"file_size,omitempty"`
	Checksum          string                 `json:"checksum,omitempty"`
	ChecksumAlgorithm string                 `json:"checksum_algorithm,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMetadata updates metadata for a content
func (h *ContentHandler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req ContentMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid request body", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metadataParams := service.SetContentMetadataParams{
		ContentID:      id,
		ContentType:    req.ContentType,
		Title:          req.Title,
		Description:    req.Description,
		Tags:           req.Tags,
		FileSize:       req.FileSize,
		CreatedBy:      req.CreatedBy,
		CustomMetadata: req.Metadata,
	}
	if err := h.contentService.SetContentMetadata(r.Context(), metadataParams); err != nil {
		slog.Error("Fail to update content metadata", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Content metadata updated", "content_id", idStr)
	w.WriteHeader(http.StatusNoContent)
}

// GetMetadata retrieves metadata for a content
func (h *ContentHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	metadata, err := h.contentService.GetContentMetadata(r.Context(), id)
	if err != nil {
		slog.Error("Fail to get content metadata", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ContentMetadataResponse{
		ContentID:         metadata.ContentID.String(),
		MimeType:          metadata.MimeType,
		FileName:          metadata.FileName,
		Tags:              metadata.Tags,
		FileSize:          metadata.FileSize,
		Checksum:          metadata.Checksum,
		ChecksumAlgorithm: metadata.ChecksumAlgorithm,
		Metadata:          metadata.Metadata,
	}

	slog.Info("Content metadata retrieved", "content", resp)

	render.JSON(w, r, resp)
}

// CreateObjectRequest is the request body for creating an object
type CreateObjectRequest struct {
	StorageBackendName string `json:"storage_backend_name"`
	Version            int    `json:"version"`
	ObjectKey          string `json:"object_key"`
	MimeType           string `json:"mime_type"`
	FileSize           int64  `json:"file_size"`
	FileName           string `json:"file_name"`
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
	UploadURL          string    `json:"upload_url"`
}

// CreateObject creates a new object for a content
func (h *ContentHandler) CreateObject(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req CreateObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid request body", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create object
	createObjectParams := service.CreateObjectParams{
		ContentID:          contentID,
		StorageBackendName: "s3-default",
		Version:            1,
		ObjectKey:          req.ObjectKey,
	}
	object, err := h.objectService.CreateObject(r.Context(), createObjectParams)
	if err != nil {
		slog.Error("Fail to create object", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update object for missing fields
	object.FileName = req.FileName
	object.ObjectType = req.MimeType
	err = h.objectService.UpdateObject(r.Context(), object)
	if err != nil {
		slog.Warn("Failed to update object", "err", err)
	}

	// Update object metadata for missing fields
	object_metadata := make(map[string]interface{})
	object_metadata["mime_type"] = req.MimeType
	object_metadata["size_bytes"] = req.FileSize
	object_metadata["file_name"] = req.FileName
	err = h.objectService.SetObjectMetadata(r.Context(), object.ID, object_metadata)
	if err != nil {
		slog.Warn("Failed to update object metadata", "err", err)
	}

	// Get upload URL
	uploadURL, err := h.objectService.GetUploadURL(r.Context(), object.ID)
	if err != nil {
		slog.Error("Failed to get upload URL", "err", err)
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
		UploadURL:          uploadURL,
	}

	slog.Info("Object created", "object_id", resp.ID)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// ListObjects lists objects for a content
// Query parameters:
//   - latest=true: Only return the latest version object (default: true)
func (h *ContentHandler) ListObjects(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		slog.Error("Fail to get objects by content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(objects) == 0 {
		slog.Warn("No objects found for content", "content_id", contentIDStr)
		http.Error(w, "No objects found for content "+contentIDStr, http.StatusNotFound)
		return
	}

	// Check if we should only return the latest version
	latestOnly := true // Default to true for backward compatibility
	latestParam := r.URL.Query().Get("latest")
	if latestParam == "false" {
		latestOnly = false
	}

	// Filter objects based on the latest parameter
	if latestOnly {
		latestObject := service.GetLatestVersionObject(objects)
		objects = []*domain.Object{latestObject}
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
		slog.Error("Invalid content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Get the latest object for this content
	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		slog.Error("Fail to get objects by content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(objects) == 0 {
		slog.Error("No objects found for this content", "content_id", contentIDStr)
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
		slog.Error("No active objects found for this content", "content_id", contentIDStr)
		http.Error(w, "No active objects found for this content", http.StatusNotFound)
		return
	}

	// For in-memory backend, we can't provide a direct URL, so we'll return the object ID
	// In a real implementation, this would return a signed URL or redirect to a download endpoint
	response := map[string]string{
		"object_id": latestObject.ID,
		"message":   "For in-memory backend, use the object_id to download the content directly",
	}

	slog.Info("Download URL retrieved", "object_id", latestObject.ID)

	render.JSON(w, r, response)
}
