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
	r.Get("/bulk", h.GetFilesByContentIDs)
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
	ContentID      string                 `json:"content_id"`
	FileName       string                 `json:"file_name"`
	PreviewURL     string                 `json:"preview_url"`
	DownloadURL    string                 `json:"download_url"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Status         string                 `json:"status"`
	MimeType       string                 `json:"mime_type"`
	FileSize       int64                  `json:"file_size"`
	DerivationType string                 `json:"derivation_type"`
	OwnerID        string                 `json:"owner_id"`
	OwnerType      string                 `json:"owner_type"`
	TenantID       string                 `json:"tenant_id"`
}

// CreateFile creates a new content and returns upload URL
func (h *FilesHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	var req CreateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Fail to decode request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse UUIDs
	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		slog.Error("Invalid owner ID", "owner_id", req.OwnerID, "error", err)
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	if req.OwnerType == "" {
		slog.Error("Owner type is required", "owner_type", req.OwnerType)
		http.Error(w, "Owner type is required", http.StatusBadRequest)
		return
	}
	if req.DocumentType == "" {
		slog.Error("Document type is required", "document_type", req.DocumentType)
		http.Error(w, "Document type is required", http.StatusBadRequest)
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
		DerivationType: domain.ContentDerivationTypeOriginal,
	}
	content, err := h.contentService.CreateContent(r.Context(), createParams)
	if err != nil {
		slog.Error("Failed to create content", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("Content created", "content", content)

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
		FileSize:    req.FileSize, // File size will be updated later
		CreatedBy:   ownerID.String(),
	}
	err = h.contentService.SetContentMetadata(r.Context(), metadataParams)
	if err != nil {
		slog.Warn("Failed to set content metadata", "err", err)
	}

	// Create object with default storage backend
	createObjectParams := service.CreateObjectParams{
		ContentID:          content.ID,
		StorageBackendName: "s3-default",
		Version:            1,
	}
	object, err := h.objectService.CreateObject(r.Context(), createObjectParams)
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

	resp := CreateFileResponse{
		ContentID: content.ID.String(),
		ObjectID:  object.ID.String(),
		UploadURL: uploadURL,
		CreatedAt: content.CreatedAt,
		Status:    content.Status,
	}
	slog.Info("File created", "response", resp)

	render.JSON(w, r, resp)
}

// CompleteUpload marks a client-side upload as complete
func (h *FilesHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Verify content exists
	content, err := h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		slog.Error("Content not found", "content_id", contentID.String(), "error", err)
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Get object by content id
	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil || len(objects) == 0 {
		slog.Error("Object not found", "content_id", contentID.String(), "error", err)
		http.Error(w, "Object not found", http.StatusNotFound)
		return
	}

	// Get the latest version of the object
	object := service.GetLatestVersionObject(objects)

	// Update object metadata from storage to get actual file info
	// This also updates the object status to uploaded
	object_meta, err := h.objectService.UpdateObjectMetaFromStorage(r.Context(), object.ID)
	if err != nil {
		slog.Error("Fail to update object metadata", "object_id", object.ID.String(), "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update content status to uploaded
	content.Status = model.ContentStatusUploaded
	updateParams := service.UpdateContentParams{
		Content: content,
	}
	if err := h.contentService.UpdateContent(r.Context(), updateParams); err != nil {
		slog.Error("Fail to update content", "content_id", contentID.String(), "error", err)
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
	metadataParams := service.SetContentMetadataParams{
		ContentID:      contentID,
		ContentType:    object_meta.MimeType,
		Title:          content.Name,
		Description:    content.Description,
		Tags:           content_meta.Tags,
		FileSize:       object_meta.SizeBytes,
		CreatedBy:      "",
		CustomMetadata: content_meta.Metadata,
		FileName:       content_meta.FileName,
	}
	if err := h.contentService.SetContentMetadata(r.Context(), metadataParams); err != nil {
		slog.Error("Fail to update content metadata", "content_id", contentID.String(), "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Info("Content metadata updated", "content_id", contentID.String(), "content status", content.Status)
	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, map[string]string{"status": "completed"})
}

// UpdateMetadata updates file metadata
func (h *FilesHandler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var req UpdateMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Fail to decode request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify content exists
	content, err := h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		slog.Error("Content not found", "content_id", contentID.String(), "error", err)
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	title := content.Name
	description := content.Description
	// Update content
	if req.Title != "" || req.Description != "" {
		// Update title and description if provided
		if req.Title != "" {
			title = req.Title
		}
		if req.Description != "" {
			description = req.Description
		}
		err := h.contentService.UpdateContent(r.Context(), service.UpdateContentParams{
			Content: &model.Content{
				ID:             contentID,
				Name:           title,
				Description:    description,
				UpdatedAt:      time.Now(),
				TenantID:       content.TenantID,
				OwnerID:        content.OwnerID,
				OwnerType:      content.OwnerType,
				DocumentType:   content.DocumentType,
				Status:         content.Status,
				DerivationType: content.DerivationType,
				CreatedAt:      content.CreatedAt,
			},
		})
		if err != nil {
			slog.Error("Fail to update content", "content_id", contentID.String(), "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Get and update content metadata using the available method
	tags := req.Tags
	if req.Metadata != nil || tags != nil {
		// Update tags if provided
		if tags != nil {
			req.Metadata["tags"] = tags
		}
		// Get content metadata
		content_meta, err := h.contentService.GetContentMetadata(r.Context(), contentID)
		if err != nil {
			slog.Error("GetContentMetadata", "contentID", contentID.String(), "error", err)
			// Create default metadata if it doesn't exist
			content_meta = &model.ContentMetadata{
				ContentID: contentID,
				FileName:  "",
				MimeType:  "",
				FileSize:  0,
			}
		}
		// Update content metadata
		metadataParams := service.SetContentMetadataParams{
			ContentID:      contentID,
			FileName:       content_meta.FileName,
			ContentType:    content_meta.MimeType,
			Title:          title,
			Description:    description,
			Tags:           tags,
			FileSize:       content_meta.FileSize,
			CreatedBy:      "",
			CustomMetadata: req.Metadata,
		}
		err = h.contentService.SetContentMetadata(r.Context(), metadataParams)
		if err != nil {
			slog.Error("Fail to update content metadata", "content_id", contentID.String(), "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	slog.Info("Content updated", "content_id", contentID.String())
	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, map[string]string{"status": "updated"})
}

// GetFileInfo returns file information including preview and download URLs
func (h *FilesHandler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "content_id")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", contentIDStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	// Get content
	content, err := h.contentService.GetContent(r.Context(), contentID)
	if err != nil {
		slog.Error("Fail to get content", "content_id", contentID.String(), "error", err)
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	// Get content metadata
	metadata, err := h.contentService.GetContentMetadata(r.Context(), contentID)
	if err != nil {
		slog.Warn("Content metadata not found", "content_id", contentID.String(), "error", err)
		// If no metadata found, create empty metadata map
		metadata = &model.ContentMetadata{
			ContentID: contentID,
			Metadata:  make(map[string]interface{}),
		}
	}

	// Get objects for this content
	objects, err := h.objectService.GetObjectsByContentID(r.Context(), contentID)
	if err != nil {
		slog.Error("Fail to get objects", "content_id", contentID.String(), "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(objects) == 0 {
		slog.Error("No objects found for content", "content_id", contentID.String())
		http.Error(w, "No objects found for content", http.StatusNotFound)
		return
	}

	// Get the latest version of the object
	object := service.GetLatestVersionObject(objects)

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
		slog.Error("Fail to get download URL", "object_id", object.ID.String(), "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract filename from metadata or object key
	filename := object.ObjectKey
	if filenameVal, ok := metadata.Metadata["file_name"]; ok {
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
		MimeType:    metadata.MimeType,
		FileSize:    metadata.FileSize,
		OwnerID:     content.OwnerID.String(),
		TenantID:    content.TenantID.String(),
		OwnerType:   string(content.OwnerType),
	}

	slog.Info("GetFileInfo", "content", resp)

	render.JSON(w, r, resp)
}

// GetBulkFiles retrieves multiple files by their IDs
func (h *FilesHandler) GetFilesByContentIDs(w http.ResponseWriter, r *http.Request) {
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
	var files []FileInfoResponse

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
		resp := FileInfoResponse{
			ContentID:      content.ID.String(),
			CreatedAt:      content.CreatedAt,
			UpdatedAt:      content.UpdatedAt,
			Status:         string(content.Status),
			DerivationType: content.DerivationType,
			OwnerID:        content.OwnerID.String(),
			TenantID:       content.TenantID.String(),
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

		// Get objects for this content
		objects, err := h.objectService.GetObjectsByContentID(r.Context(), id)
		if err != nil {
			slog.Warn("Fail to get objects", "id", idStr)
		}

		if len(objects) == 0 {
			slog.Warn("No objects found for content", "id", idStr)
			continue
		}

		// Get the latest version of the object
		object := service.GetLatestVersionObject(objects)

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

		resp.DownloadURL = downloadURL
		resp.PreviewURL = previewURL

		files = append(files, resp)
	}

	render.JSON(w, r, files)
}
