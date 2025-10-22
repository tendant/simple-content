package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

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

// CreateObjectRequest is the request body for creating an object
type CreateObjectRequest struct {
	StorageBackendName string `json:"storage_backend_name"`
	Version            int    `json:"version"`
	ObjectKey          string `json:"object_key"`
	MimeType           string `json:"mime_type"`
	FileSize           int64  `json:"file_size"`
	FileName           string `json:"file_name"`
}

// ContentHandler handles HTTP requests for content using pkg/simplecontent
type ContentHandler struct {
	service simplecontent.Service
	storage simplecontent.StorageService
}

// NewContentHandler creates a new content handler
func NewContentHandler(service simplecontent.Service, storageService simplecontent.StorageService) *ContentHandler {
	return &ContentHandler{
		service: service,
		storage: storageService,
	}
}

// Routes returns the routes for content
func (h *ContentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateContent)
	r.Get("/{id}", h.GetContent)
	r.Delete("/{id}", h.DeleteContent)
	r.Get("/bulk", h.GetContentsByIDs)

	r.Post("/{id}/objects", h.CreateObject)
	r.Get("/{id}/objects", h.ListObjects)

	// Routes for metadata
	r.Put("/{id}/metadata", h.SetContentMetadata)
	r.Get("/{id}/metadata", h.GetContentMetadataHandler)

	// Routes for derived content
	r.Post("/{id}/derived", h.CreateDerivedContent)
	r.Get("/{id}/derived", h.GetDerivedContent)
	r.Get("/{id}/derived-tree", h.GetDerivedContentTree)

	return r
}

// CreateContentRequest is the request body for creating a content
type CreateContentRequest struct {
	OwnerID      string `json:"owner_id"`
	TenantID     string `json:"tenant_id"`
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
	OwnerType    string `json:"owner_type"`
	MimeType     string `json:"mime_type"`
	FileSize     int64  `json:"file_size"`
	Status       string `json:"status,omitempty"` // Optional initial status (defaults to "created")
}

// ContentResponse is the response body for a content
type ContentResponse struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	OwnerID         string    `json:"owner_id"`
	TenantID        string    `json:"tenant_id"`
	Status          string    `json:"status"`
	MimeType        string    `json:"mime_type"`
	FileSize        int64     `json:"file_size"`
	FileName        string    `json:"file_name"`
	DocumentType    string    `json:"document_type"`
	DerivationType  string    `json:"derivation_type"`
	DerivationLevel int       `json:"derivation_level"`
	ParentID        string    `json:"parent_id,omitempty"`
}

const maxContentsPerRequest = 50

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

	// Create content using the simplified API
	createReq := simplecontent.CreateContentRequest{
		TenantID:     tenantID,
		OwnerID:      ownerID,
		OwnerType:    req.OwnerType,
		Name:         req.FileName,
		DocumentType: req.DocumentType,
	}

	content, err := h.service.CreateContent(r.Context(), createReq)
	if err != nil {
		slog.Error("Failed to create content", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update status if provided in request
	if req.Status != "" {
		statusEnum := simplecontent.ContentStatus(req.Status)
		if !statusEnum.IsValid() {
			slog.Error("Invalid status", "status", req.Status)
			http.Error(w, "Invalid status", http.StatusBadRequest)
			return
		}
		if err := h.service.UpdateContentStatus(r.Context(), content.ID, statusEnum); err != nil {
			slog.Error("Failed to update content status", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Update the content object with the new status
		content.Status = req.Status
	}

	// Set content metadata
	if err := h.service.SetContentMetadata(r.Context(), simplecontent.SetContentMetadataRequest{
		ContentID:   content.ID,
		ContentType: req.MimeType,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		CreatedBy:   ownerID.String(),
	}); err != nil {
		slog.Error("Failed to set content metadata", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set derivation type to "original" for newly created content
	derivationType := content.DerivationType
	if derivationType == "" {
		derivationType = "original"
	}

	// Get derivation level (0 for original content)
	derivationLevel := 0

	resp := ContentResponse{
		ID:              content.ID.String(),
		CreatedAt:       content.CreatedAt,
		UpdatedAt:       content.UpdatedAt,
		OwnerID:         content.OwnerID.String(),
		TenantID:        content.TenantID.String(),
		Status:          content.Status,
		DocumentType:    content.DocumentType,
		FileName:        req.FileName,
		MimeType:        req.MimeType,
		DerivationType:  derivationType,
		DerivationLevel: derivationLevel,
	}

	slog.Info("Content created", "content_id", content.ID.String())
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
	content, err := h.service.GetContent(r.Context(), id)
	if err != nil {
		slog.Error("Failed to get content", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := ContentResponse{
		ID:           content.ID.String(),
		CreatedAt:    content.CreatedAt,
		UpdatedAt:    content.UpdatedAt,
		OwnerID:      content.OwnerID.String(),
		TenantID:     content.TenantID.String(),
		Status:       content.Status,
		DocumentType: content.DocumentType,
	}

	// Get content metadata
	contentMeta, err := h.service.GetContentMetadata(r.Context(), id)
	if err != nil {
		slog.Warn("Fail to get content metadata", "id", idStr)
	} else {
		resp.MimeType = contentMeta.MimeType
		resp.FileSize = contentMeta.FileSize
		resp.FileName = contentMeta.FileName
	}

	slog.Info("Content retrieved", "content_id", idStr)
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
	if len(idStrings) > maxContentsPerRequest {
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
		content, err := h.service.GetContent(r.Context(), id)
		if err != nil {
			slog.Warn("Failed to get content", "id", idStr)
			continue
		}

		// Create response for this content
		resp := ContentResponse{
			ID:           content.ID.String(),
			CreatedAt:    content.CreatedAt,
			UpdatedAt:    content.UpdatedAt,
			OwnerID:      content.OwnerID.String(),
			TenantID:     content.TenantID.String(),
			Status:       content.Status,
			DocumentType: content.DocumentType,
		}

		// Get content metadata
		contentMeta, err := h.service.GetContentMetadata(r.Context(), id)
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

	if err := h.service.DeleteContent(r.Context(), id); err != nil {
		slog.Error("Failed to delete content", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Content deleted", "content_id", idStr)
	w.WriteHeader(http.StatusNoContent)
}

// GetContentDetails retrieves detailed content information with URLs
func (h *ContentHandler) GetContentDetails(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	details, err := h.service.GetContentDetails(r.Context(), id)

	if err != nil {
		slog.Error("Failed to get content details", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	slog.Info("Content details retrieved", "content_id", idStr)
	render.JSON(w, r, details)
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

	storageBackend := req.StorageBackendName
	if storageBackend == "" {
		storageBackend = DEFAULT_STORAGE_BACKEND
	}

	// Create object
	object, err := h.storage.CreateObject(r.Context(), simplecontent.CreateObjectRequest{
		ContentID:          contentID,
		StorageBackendName: storageBackend,
		Version:            1,
		FileName:           req.FileName,
	})
	if err != nil {
		slog.Error("Fail to create object", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set object metadata
	err = h.storage.SetObjectMetadata(r.Context(), object.ID, map[string]interface{}{
		"mime_type":  req.MimeType,
		"size_bytes": req.FileSize,
		"file_name":  req.FileName,
	})
	if err != nil {
		slog.Warn("Failed to update object metadata", "err", err)
	}

	// Get upload URL
	uploadURL, err := h.storage.GetUploadURL(r.Context(), object.ID)
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

	objects, err := h.service.GetObjectsByContentID(r.Context(), contentID)
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
		objects = []*simplecontent.Object{GetLatestVersionObject(objects)}
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

// GetLatestVersionObject returns the object with the highest version number from a slice of objects.
// If there are multiple objects with the same highest version, it returns the first one found.
func GetLatestVersionObject(objects []*simplecontent.Object) *simplecontent.Object {
	if len(objects) == 0 {
		return nil
	}

	latestObject := objects[0]
	for _, obj := range objects[1:] {
		if obj.Version > latestObject.Version {
			latestObject = obj
		}
	}

	return latestObject
}

// CreateDerivedContentRequest is the request body for creating derived content
type CreateDerivedContentRequest struct {
	DerivedContentID   uuid.UUID              `json:"derived_content_id"`
	DerivationType     string                 `json:"derivation_type"`
	DerivationParams   map[string]interface{} `json:"derivation_params"`
	ProcessingMetadata map[string]interface{} `json:"processing_metadata"`
	OwnerID            string                 `json:"owner_id"`
	TenantID           string                 `json:"tenant_id"`
	Status             string                 `json:"status,omitempty"` // Optional initial status (defaults to "created")
}

// CreateDerivedContentResponse is the response body for derived content creation
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

	var req CreateDerivedContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var ownerID uuid.UUID
	var tenantID uuid.UUID
	if req.OwnerID != "" {
		// Parse owner and tenant IDs
		ownerID, err = uuid.Parse(req.OwnerID)
		if err != nil {
			slog.Error("Invalid owner ID", "owner_id", req.OwnerID, "error", err)
			http.Error(w, "Invalid owner ID", http.StatusBadRequest)
			return
		}
	}
	if req.TenantID != "" {
		tenantID, err = uuid.Parse(req.TenantID)
		if err != nil {
			slog.Error("Invalid tenant ID", "tenant_id", req.TenantID, "error", err)
			http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
			return
		}
	}

	// Merge derivation params and processing metadata into a single metadata map
	metadata := make(map[string]interface{})
	for k, v := range req.DerivationParams {
		metadata[k] = v
	}
	for k, v := range req.ProcessingMetadata {
		metadata[k] = v
	}

	// Parse initial status if provided
	var initialStatus simplecontent.ContentStatus
	if req.Status != "" {
		initialStatus = simplecontent.ContentStatus(req.Status)
		if !initialStatus.IsValid() {
			slog.Error("Invalid status", "status", req.Status)
			http.Error(w, "Invalid status", http.StatusBadRequest)
			return
		}
	}

	// Create derived content using the new API
	derivedContent, err := h.service.CreateDerivedContent(r.Context(), simplecontent.CreateDerivedContentRequest{
		ParentID:       parentID,
		OwnerID:        ownerID,
		TenantID:       tenantID,
		DerivationType: req.DerivationType,
		Metadata:       metadata,
		InitialStatus:  initialStatus,
	})
	if err != nil {
		slog.Error("Failed to create derived content", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Compute derivation level by getting parent's level + 1
	parentDerivationLevel := 0
	if parentDerived, err := h.service.GetDerivedRelationship(r.Context(), parentID); err == nil {
		// Parent is also derived, compute its level recursively
		parentDerivationLevel = h.computeDerivationLevel(r.Context(), parentDerived.ParentID) + 1
	}
	derivationLevel := parentDerivationLevel + 1

	// Create ContentResponse instead of CreateDerivedContentResponse for consistency
	contentResp := ContentResponse{
		ID:              derivedContent.ID.String(),
		ParentID:        parentIDStr,
		CreatedAt:       derivedContent.CreatedAt,
		UpdatedAt:       derivedContent.UpdatedAt,
		OwnerID:         derivedContent.OwnerID.String(),
		TenantID:        derivedContent.TenantID.String(),
		Status:          derivedContent.Status,
		DerivationType:  "derived",
		DerivationLevel: derivationLevel,
		DocumentType:    derivedContent.DocumentType,
	}

	slog.Info("Derived content created", "parent_id", parentIDStr, "derived_id", derivedContent.ID.String())
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, contentResp)
}

// SetContentMetadata sets metadata for a content
func (h *ContentHandler) SetContentMetadata(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	var metadataReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&metadataReq); err != nil {
		slog.Error("Invalid request body", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build SetContentMetadataRequest
	req := simplecontent.SetContentMetadataRequest{
		ContentID: id,
	}

	// Extract known fields
	if contentType, ok := metadataReq["content_type"].(string); ok {
		req.ContentType = contentType
	}
	if title, ok := metadataReq["title"].(string); ok {
		req.Title = title
	}
	if description, ok := metadataReq["description"].(string); ok {
		req.Description = description
	}
	if fileName, ok := metadataReq["file_name"].(string); ok {
		req.FileName = fileName
	}
	if fileSize, ok := metadataReq["file_size"].(float64); ok {
		req.FileSize = int64(fileSize)
	}
	if createdBy, ok := metadataReq["created_by"].(string); ok {
		req.CreatedBy = createdBy
	}
	if tags, ok := metadataReq["tags"].([]interface{}); ok {
		tagStrings := make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				tagStrings[i] = tagStr
			}
		}
		req.Tags = tagStrings
	}

	// Store custom metadata in a separate map
	customMetadata := make(map[string]interface{})
	knownFields := map[string]bool{
		"content_type": true, "title": true, "description": true,
		"tags": true, "file_name": true, "file_size": true, "created_by": true,
		"metadata": true, // Handle metadata separately
	}
	for key, value := range metadataReq {
		if !knownFields[key] {
			customMetadata[key] = value
		}
	}

	// If there's a nested "metadata" field, merge its contents into customMetadata
	if nestedMetadata, ok := metadataReq["metadata"].(map[string]interface{}); ok {
		for k, v := range nestedMetadata {
			customMetadata[k] = v
		}
	}

	req.CustomMetadata = customMetadata

	// Set content metadata
	if err := h.service.SetContentMetadata(r.Context(), req); err != nil {
		slog.Error("Failed to set content metadata", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("Content metadata set", "content_id", idStr)
	w.WriteHeader(http.StatusNoContent)
}

// GetContentMetadataHandler retrieves metadata for a content
func (h *ContentHandler) GetContentMetadataHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("Invalid content ID", "content_id", idStr, "error", err)
		http.Error(w, "Invalid content ID", http.StatusBadRequest)
		return
	}

	metadata, err := h.service.GetContentMetadata(r.Context(), id)
	if err != nil {
		slog.Error("Failed to get content metadata", "content_id", idStr, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Convert to response format matching ContentMetadataResponse
	resp := map[string]interface{}{
		"content_id":   metadata.ContentID.String(),
		"content_type": metadata.MimeType,
	}
	if metadata.FileName != "" {
		resp["file_name"] = metadata.FileName
	}
	if metadata.FileSize > 0 {
		resp["file_size"] = metadata.FileSize
	}
	if len(metadata.Tags) > 0 {
		resp["tags"] = metadata.Tags
	}

	// Extract known fields from metadata map
	if metadata.Metadata != nil {
		if title, ok := metadata.Metadata["title"].(string); ok {
			resp["title"] = title
		}
		if description, ok := metadata.Metadata["description"].(string); ok {
			resp["description"] = description
		}
		if createdBy, ok := metadata.Metadata["created_by"].(string); ok {
			resp["created_by"] = createdBy
		}
		// Include the remaining custom metadata
		resp["metadata"] = metadata.Metadata
	}

	slog.Info("Content metadata retrieved", "content_id", idStr)
	render.JSON(w, r, resp)
}

// GetDerivedContent retrieves direct derived content for a parent
func (h *ContentHandler) GetDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentIDStr := chi.URLParam(r, "id")
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		slog.Error("Invalid parent content ID", "parent_id", parentIDStr, "error", err)
		http.Error(w, "Invalid parent content ID", http.StatusBadRequest)
		return
	}

	// Get direct derived content (only immediate children)
	derivedList, err := h.service.ListDerivedContent(r.Context(), simplecontent.WithParentID(parentID))
	if err != nil {
		slog.Error("Failed to get derived content", "parent_id", parentIDStr, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to ContentResponse
	var resp []ContentResponse
	for _, derived := range derivedList {
		// Get the actual content to get all fields
		content, err := h.service.GetContent(r.Context(), derived.ContentID)
		if err != nil {
			slog.Warn("Failed to get content for derived", "content_id", derived.ContentID.String())
			continue
		}

		// Compute derivation level
		derivationLevel := 1 + h.computeDerivationLevel(r.Context(), derived.ParentID)

		contentResp := ContentResponse{
			ID:              content.ID.String(),
			ParentID:        derived.ParentID.String(),
			CreatedAt:       content.CreatedAt,
			UpdatedAt:       content.UpdatedAt,
			OwnerID:         content.OwnerID.String(),
			TenantID:        content.TenantID.String(),
			Status:          content.Status,
			DerivationType:  "derived",
			DerivationLevel: derivationLevel,
			DocumentType:    content.DocumentType,
		}

		resp = append(resp, contentResp)
	}

	slog.Info("Derived content retrieved", "parent_id", parentIDStr, "count", len(resp))
	render.JSON(w, r, resp)
}

// GetDerivedContentTree retrieves the entire derived content tree (recursive)
func (h *ContentHandler) GetDerivedContentTree(w http.ResponseWriter, r *http.Request) {
	rootIDStr := chi.URLParam(r, "id")
	rootID, err := uuid.Parse(rootIDStr)
	if err != nil {
		slog.Error("Invalid root content ID", "root_id", rootIDStr, "error", err)
		http.Error(w, "Invalid root content ID", http.StatusBadRequest)
		return
	}

	// Get the root content
	rootContent, err := h.service.GetContent(r.Context(), rootID)
	if err != nil {
		slog.Error("Failed to get root content", "root_id", rootIDStr, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Build the tree recursively
	var resp []ContentResponse

	// Add root content
	rootDerivationLevel := h.computeDerivationLevel(r.Context(), rootID)
	rootResp := ContentResponse{
		ID:              rootContent.ID.String(),
		CreatedAt:       rootContent.CreatedAt,
		UpdatedAt:       rootContent.UpdatedAt,
		OwnerID:         rootContent.OwnerID.String(),
		TenantID:        rootContent.TenantID.String(),
		Status:          rootContent.Status,
		DerivationType:  "original",
		DerivationLevel: rootDerivationLevel,
		DocumentType:    rootContent.DocumentType,
	}
	if rootDerivationLevel > 0 {
		rootResp.DerivationType = "derived"
		if derived, err := h.service.GetDerivedRelationship(r.Context(), rootID); err == nil {
			rootResp.ParentID = derived.ParentID.String()
		}
	}
	resp = append(resp, rootResp)

	// Recursively add all descendants
	h.addDescendants(r.Context(), rootID, &resp)

	slog.Info("Derived content tree retrieved", "root_id", rootIDStr, "count", len(resp))
	render.JSON(w, r, resp)
}

// addDescendants recursively adds all descendant content to the response
func (h *ContentHandler) addDescendants(ctx context.Context, parentID uuid.UUID, resp *[]ContentResponse) {
	derivedList, err := h.service.ListDerivedContent(ctx, simplecontent.WithParentID(parentID))
	if err != nil {
		return
	}

	for _, derived := range derivedList {
		content, err := h.service.GetContent(ctx, derived.ContentID)
		if err != nil {
			continue
		}

		derivationLevel := 1 + h.computeDerivationLevel(ctx, derived.ParentID)

		contentResp := ContentResponse{
			ID:              content.ID.String(),
			ParentID:        derived.ParentID.String(),
			CreatedAt:       content.CreatedAt,
			UpdatedAt:       content.UpdatedAt,
			OwnerID:         content.OwnerID.String(),
			TenantID:        content.TenantID.String(),
			Status:          content.Status,
			DerivationType:  "derived",
			DerivationLevel: derivationLevel,
			DocumentType:    content.DocumentType,
		}

		*resp = append(*resp, contentResp)

		// Recursively add descendants of this content
		h.addDescendants(ctx, content.ID, resp)
	}
}

// computeDerivationLevel computes the derivation level by recursively traversing parent chain
// Maximum depth is capped at 100 to prevent infinite loops
func (h *ContentHandler) computeDerivationLevel(ctx context.Context, contentID uuid.UUID) int {
	return h.computeDerivationLevelWithDepth(ctx, contentID, 0)
}

func (h *ContentHandler) computeDerivationLevelWithDepth(ctx context.Context, contentID uuid.UUID, currentDepth int) int {
	// Hard limit to prevent infinite loops (should never reach this with max depth of 5)
	const maxSafetyDepth = 100
	if currentDepth >= maxSafetyDepth {
		return maxSafetyDepth
	}

	derived, err := h.service.GetDerivedRelationship(ctx, contentID)
	if err != nil {
		// This is an original content (not derived), so level is 0
		return 0
	}
	// Recursively compute parent's level and add 1
	return 1 + h.computeDerivationLevelWithDepth(ctx, derived.ParentID, currentDepth+1)
}
