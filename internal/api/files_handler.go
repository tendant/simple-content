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
	OwnerID      string `json:"owner_id"`
	OwnerType    string `json:"owner_type"`
	TenantID     string `json:"tenant_id"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	DocumentType string `json:"document_type,omitempty"`
}

// CreateFileResponse represents the response after creating a file
type CreateFileResponse struct {
	ContentID string    `json:"content_id"`
	ObjectID  string    `json:"object_id"`
	UploadURL string    `json:"upload_url"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
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

	if req.OwnerType == "" {
		http.Error(w, "Owner type is required", http.StatusBadRequest)
		return
	}
	if req.DocumentType == "" {
		http.Error(w, "Document type is required", http.StatusBadRequest)
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

	// Update content for missing fields
	content.OwnerType = req.OwnerType
	content.Name = req.FileName
	content.DocumentType = req.DocumentType
	err = h.contentService.UpdateContent(r.Context(), content)
	if err != nil {
		slog.Warn("Failed to update content", "err", err)
	}

	// Set content metadata
	slog.Info("Setting content metadata...")
	err = h.contentService.SetContentMetadata(
		r.Context(),
		content.ID,
		req.MimeType,
		"title",
		"description",
		nil,
		req.FileSize, // File size will be updated later
		ownerID.String(),
		// add not included fields to custom metadata
		map[string]interface{}{
			"file_name":     req.FileName,
			"mime_type":     req.MimeType,
			"document_type": req.DocumentType,
		},
	)
	if err != nil {
		slog.Warn("Failed to set content metadata", "err", err)
	}

	// Create object with default storage backend
	object, err := h.objectService.CreateObject(r.Context(), content.ID, "s3-default", 1)
	if err != nil {
		slog.Error("Failed to create object", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("Object created", "object_id", object.ID.String())

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
	slog.Info("Object metadata", "object_meta", object_metadata)
	err = h.objectService.SetObjectMetadata(r.Context(), object.ID, object_metadata)
	if err != nil {
		slog.Warn("Failed to update object metadata", "err", err)
	}

	// Get upload URL
	uploadURL, err := h.objectService.GetUploadURL(r.Context(), object.ID)
	if err != nil {
		slog.Warn("Failed to get upload URL", "err", err)
	}

	resp := CreateFileResponse{
		ContentID: content.ID.String(),
		ObjectID:  object.ID.String(),
		UploadURL: uploadURL,
		CreatedAt: content.CreatedAt,
		Status:    content.Status,
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

	// Verify content exists
	content, err := h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Get object by content id
	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil || len(objects) == 0 {
		http.Error(w, "Object not found", http.StatusNotFound)
		return
	}
	object := objects[0]

	// Update object metadata from storage to get actual file info
	// This also updates the object status to uploaded
	object_meta, err := h.objectService.UpdateObjectMetaFromStorage(r.Context(), object.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update content status to uploaded
	content.Status = model.ContentStatusUploaded
	if err := h.contentService.UpdateContent(r.Context(), content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get and update content metadata
	content_meta, err := h.contentService.GetContentMetadata(r.Context(), contentID)
	if err != nil {
		slog.Error("GetContentMetadata", "contentID", contentID.String(), "error", err)
		http.Error(w, "Content metadata not found", http.StatusNotFound)
		return
	}
	content_meta.MimeType = object_meta.MimeType
	content_meta.FileSize = object_meta.SizeBytes
	if err := h.contentService.SetContentMetadata(r.Context(), contentID, object_meta.MimeType, "", "", content_meta.Tags, object_meta.SizeBytes, "", content_meta.Metadata); err != nil {
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
