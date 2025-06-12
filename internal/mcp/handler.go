package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
	)
	s.AddTool(uploadTool, h.handleUploadContent)

}

// handleUploadContent handles the upload_content tool call
func (h *Handler) handleUploadContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the base64 data parameter
	dataVal, ok := request.GetArguments()["data"]
	if !ok || dataVal == nil {
		return nil, fmt.Errorf("data parameter is required")
	}

	dataStr, ok := dataVal.(string)
	if !ok {
		return nil, fmt.Errorf("data parameter must be a string")
	}

	// Validate base64 format (placeholder validation)
	_, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %v", err)
	}

	// Generate a UUID for the content ID
	contentID := uuid.New().String()

	// Generate a placeholder download URL
	downloadURL := fmt.Sprintf("https://content.example.com/download/%s", contentID)

	return mcp.NewToolResultText(fmt.Sprintf("Content uploaded successfully.\nContent ID: %s\nDownload URL: %s", contentID, downloadURL)), nil
}
