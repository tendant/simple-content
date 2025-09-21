package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tendant/simple-content/pkg/model"
	"github.com/tendant/simple-content/pkg/service"
)

// Handler implements a simple hello content MCP tool
type Handler struct {
	objectService  *service.ObjectService
	contentService *service.ContentService
}

// NewHandler creates a new instance of HelloContentHandler
func NewHandler(
	contentService *service.ContentService,
	objectService *service.ObjectService,
) *Handler {
	return &Handler{
		contentService: contentService,
		objectService:  objectService,
	}
}

// RegisterTools registers the hello content tools with the MCP server
func (h *Handler) RegisterTools(s *server.MCPServer) {
	// Register the hello_content tool

	// Register the upload_content tool
	uploadTool := mcp.NewTool("upload_content",
		mcp.WithDescription("Upload content from base64 encoded data and return content ID and download URL"),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description("Base64 encoded content data"),
		),
		mcp.WithString("owner_id",
			mcp.Required(),
			mcp.Description("Content owner id"),
		),
		mcp.WithString("owner_type",
			mcp.Required(),
			mcp.Description("Content owner type"),
		),
		mcp.WithString("tenant_id",
			mcp.Description("Content tenant id"),
		),
	)
	s.AddTool(uploadTool, h.handleUploadContent)
}

// handleUploadContent handles the upload_content tool call
func (h *Handler) handleUploadContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	// Get the owner id parameter
	ownerIDStr, ok := request.GetArguments()["owner_id"].(string)
	if !ok || ownerIDStr == "" {
		slog.Error("owner_id parameter is required")
		return nil, fmt.Errorf("owner_id parameter is required")
	}
	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		slog.Error("invalid owner id")
		return nil, fmt.Errorf("invalid owner id")
	}

	// Get the owner type parameter
	ownerTypeStr, ok := request.GetArguments()["owner_type"].(string)
	if !ok || ownerTypeStr == "" {
		slog.Error("owner_type parameter is required")
		return nil, fmt.Errorf("owner_type parameter is required")
	}

	// Get the tenant id parameter
	tenantId := uuid.Nil
	tenantIDStr, ok := request.GetArguments()["tenant_id"].(string)
	if ok {
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			slog.Error("invalid tenant id")
			return nil, fmt.Errorf("invalid tenant id")
		}
		tenantId = tenantID
	}

	// Get the base64 data parameter
	dataVal, ok := request.GetArguments()["data"]
	if !ok || dataVal == nil {
		return nil, fmt.Errorf("data parameter is required")
	}

	dataStr, ok := dataVal.(string)
	if !ok {
		return nil, fmt.Errorf("data parameter must be a string")
	}

	// Validate base64 format and decode the data
	decodedData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %v", err)
	}

	// Write the decoded data to a text file
	tempDir, err := os.MkdirTemp("/tmp", "mcp-content")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempFileName := "generated_content.txt"
	outputFilePath := filepath.Join(tempDir, tempFileName)
	err = os.WriteFile(outputFilePath, decodedData, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to file: %v", err)
	}

	slog.Info("Successfully wrote decoded data", slog.String("path", outputFilePath))

	// Create content
	contentParams := service.CreateContentParams{
		OwnerID:      ownerID,
		TenantID:     tenantId,
		Title:        tempFileName,
		Description:  "Content uploaded via MCP tool",
		DocumentType: "text/plain",
	}

	content, err := h.contentService.CreateContent(ctx, contentParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create content: %v", err)
	}

	// Update Content for missing fields
	content.OwnerType = ownerTypeStr
	updateParams := service.UpdateContentParams{
		Content: content,
	}
	err = h.contentService.UpdateContent(ctx, updateParams)
	if err != nil {
		slog.Warn("Failed to update content", "err", err)
	}

	// Set content metadata
	metadataParams := service.SetContentMetadataParams{
		ContentID:   content.ID,
		ContentType: "text/plain",
		Title:       tempFileName,
		Description: "Content uploaded via MCP tool",
		Tags:        []string{"upload", "mcp"},
		FileName:    tempFileName,
		FileSize:    int64(len(decodedData)),
		CreatedBy:   "mcp-tool",
		CustomMetadata: map[string]interface{}{
			"source": "mcp-upload",
			"format": "text",
		},
	}

	err = h.contentService.SetContentMetadata(ctx, metadataParams)
	if err != nil {
		return nil, fmt.Errorf("failed to set content metadata: %v", err)
	}

	// Create a new object for the content
	objectParams := service.CreateObjectParams{
		ContentID:          content.ID,
		StorageBackendName: "s3-default", // Assuming "default" is a registered backend
		Version:            1,
	}

	object, err := h.objectService.CreateObject(ctx, objectParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create object: %v", err)
	}

	// Upload the object to storage
	reader := bytes.NewReader(decodedData)
	err = h.objectService.UploadObject(ctx, object.ID, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %v", err)
	}

	// Get object metadata from storage
	_, err = h.objectService.UpdateObjectMetaFromStorage(ctx, object.ID)
	if err != nil {
		slog.Warn("Failed to update object metadata from storage", "err", err)
	}

	// Update content status to uploaded
	content.Status = model.ContentStatusUploaded
	updateParams = service.UpdateContentParams{
		Content: content,
	}

	err = h.contentService.UpdateContent(ctx, updateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update content status: %v", err)
	}

	// Get a download URL for the object
	downloadURL, err := h.objectService.GetDownloadURL(ctx, object.ID)
	if err != nil {
		slog.Error("Failed to get download URL", "err", err)
		return nil, fmt.Errorf("failed to get download URL: %v", err)
	}

	slog.Info("Created content", slog.Any("content_id", content.ID))

	// Delete the temporary file after successful upload
	err = os.Remove(outputFilePath)
	if err != nil {
		slog.Warn("Failed to delete temporary file", "path", outputFilePath, "err", err)
		// Continue despite error, as the upload was successful
	} else {
		slog.Info("Deleted temporary file after successful upload", "path", outputFilePath)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Content uploaded successfully.\nContent ID: %s\nDownload URL: %s", content.ID, downloadURL)), nil
}
