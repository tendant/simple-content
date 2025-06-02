package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/model"
	"github.com/tendant/simple-content/pkg/service"
)

// FilesHandler handles file upload and management API endpoints
type FilesHandler struct {
	contentService *service.ContentService
	objectService  *service.ObjectService
}

// NewFilesHandler creates a new files handler
func NewFilesHandler(contentService *service.ContentService, objectService *service.ObjectService) *FilesHandler {
	return &FilesHandler{
		contentService: contentService,
		objectService:  objectService,
	}
}

// Routes returns the router for files endpoints
func (h *FilesHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.CreateFile)
	r.Post("/{content_id}/complete", h.CompleteUpload)
	r.Patch("/{content_id}", h.UpdateMetadata)
	r.Get("/{content_id}", h.GetFileInfo)
	return r
}

// CreateFileRequest represents the request to create a new file
type CreateFileRequest struct {
	OwnerID  string `json:"owner_id"`
	TenantID string `json:"tenant_id"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

// CreateFileResponse represents the response after creating a file
type CreateFileResponse struct {
	ContentID string    `json:"content_id"`
	ObjectID  string    `json:"object_id"`
	UploadURL string    `json:"upload_url"`
	CreatedAt time.Time `json:"created_at"`
}

// CompleteUploadRequest represents the request to mark upload as complete
type CompleteUploadRequest struct {
	ObjectID string `json:"object_id"`
}

// UpdateMetadataRequest represents the request to update file metadata
type UpdateMetadataRequest struct {
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FileInfoResponse represents file information including URLs
type FileInfoResponse struct {
	ContentID   string                 `json:"content_id"`
	FileName    string                 `json:"file_name"`
	PreviewURL  string                 `json:"preview_url"`
	DownloadURL string                 `json:"download_url"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Status      string                 `json:"status"`
}

// CreateFile creates a new content and returns upload URL
func (h *FilesHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	var req CreateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse UUIDs
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

	// Create content
	content, err := h.contentService.CreateContent(r.Context(), ownerID, tenantID)
	if err != nil {
		slog.Error("Failed to create content", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("Content created", "content_id", content.ID.String())

	// Create object with default storage backend
	object, err := h.objectService.CreateObject(r.Context(), content.ID, "s3-default", 1)
	if err != nil {
		slog.Error("Failed to create object", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("Object created", "object_id", object.ID.String())

	// Set initial metadata if provided
	if req.FileName != "" || req.MimeType != "" || req.FileSize > 0 {
		metadata := make(map[string]interface{})
		if req.FileName != "" {
			metadata["file_name"] = req.FileName
		}
		if req.MimeType != "" {
			metadata["mime_type"] = req.MimeType
		}
		if req.FileSize > 0 {
			metadata["file_size"] = req.FileSize
		}

		if err := h.objectService.SetObjectMetadata(r.Context(), object.ID, metadata); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Get upload URL
	uploadURL, err := h.objectService.GetUploadURL(r.Context(), object.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CreateFileResponse{
		ContentID: content.ID.String(),
		ObjectID:  object.ID.String(),
		UploadURL: uploadURL,
		CreatedAt: content.CreatedAt,
	}

	render.JSON(w, r, resp)
}

// CompleteUpload marks a client-side upload as complete
func (h *FilesHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	objectID, err := uuid.Parse(req.ObjectID)
	if err != nil {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	// Verify content exists
	content, err := h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Update object metadata from storage to get actual file info
	// This also updates the object status to uploaded
	if err := h.objectService.UpdateObjectMetaFromStorage(r.Context(), objectID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update content status to uploaded
	content.Status = model.ContentStatusUploaded
	if err := h.contentService.UpdateContent(r.Context(), content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, map[string]string{"status": "completed"})
}

// UpdateMetadata updates file metadata
func (h *FilesHandler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req UpdateMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify content exists
	_, err = h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Prepare metadata for SetContentMetadata
	title := req.Title
	description := req.Description
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	// Set content metadata using the available method
	err = h.contentService.SetContentMetadata(
		r.Context(),
		contentID,
		"", // content_type - leave empty to preserve existing
		title,
		description,
		tags,
		0,  // file_size - leave 0 to preserve existing
		"", // created_by - leave empty to preserve existing
		req.Metadata,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, map[string]string{"status": "updated"})
}

// GetFileInfo returns file information including preview and download URLs
func (h *FilesHandler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Get content
	content, err := h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Get content metadata
	metadata, err := h.contentService.GetContentMetadata(r.Context(), contentID)
	if err != nil {
		// If no metadata found, create empty metadata map
		metadata = &model.ContentMetadata{
			ContentID: contentID,
			Metadata:  make(map[string]interface{}),
		}
	}

	// Get objects for this content
	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(objects) == 0 {
		http.Error(w, "No objects found for content", http.StatusNotFound)
		return
	}

	// Use the first object (assuming one object per content for now)
	object := objects[0]

	// Get preview URL
	previewURL, err := h.objectService.GetPreviewURL(r.Context(), object.ID)
	if err != nil {
		// Preview URL generation failed, but this is not critical
		// Log the error but continue with empty preview URL
		previewURL = ""
	}

	// Get download URL
	downloadURL, err := h.objectService.GetDownloadURL(r.Context(), object.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract filename from metadata or object key
	filename := object.ObjectKey
	if filenameVal, ok := metadata.Metadata["filename"]; ok {
		if filenameStr, ok := filenameVal.(string); ok {
			filename = filenameStr
		}
	}

	resp := FileInfoResponse{
		ContentID:   content.ID.String(),
		FileName:    filename,
		PreviewURL:  previewURL,
		DownloadURL: downloadURL,
		Metadata:    metadata.Metadata,
		CreatedAt:   content.CreatedAt,
		UpdatedAt:   content.UpdatedAt,
		Status:      string(content.Status),
	}

	render.JSON(w, r, resp)
}
