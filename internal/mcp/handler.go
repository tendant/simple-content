package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// HelloContentHandler implements a simple hello content MCP tool
type HelloContentHandler struct{}

// NewHelloContentHandler creates a new instance of HelloContentHandler
func NewHelloContentHandler() *HelloContentHandler {
	return &HelloContentHandler{}
}

// RegisterTools registers the hello content tools with the MCP server
func (h *HelloContentHandler) RegisterTools(s *server.MCPServer) {
	// Register the hello_content tool
	s.AddTool(mcp.Tool{
		Name:        "hello_content",
		Description: "A simple tool that returns a hello message with optional content",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
		},
	}, h.handleHelloContent)

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

// handleHelloContent handles the hello_content tool call
func (h *HelloContentHandler) handleHelloContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get name parameter with default
	name := "World"
	if nameVal, ok := request.GetArguments()["name"]; ok && nameVal != nil {
		if nameStr, ok := nameVal.(string); ok && nameStr != "" {
			name = nameStr
		}
	}

	// Get optional content parameter
	content := ""
	if contentVal, ok := request.GetArguments()["content"]; ok && contentVal != nil {
		if contentStr, ok := contentVal.(string); ok {
			content = contentStr
		}
	}

	// Build the response message
	message := fmt.Sprintf("Hello, %s!", name)
	if content != "" {
		message += fmt.Sprintf("\n\nContent: %s", content)
	}

	// Return the result using the helper function
	return mcp.NewToolResultText(message), nil
}

// handleUploadContent handles the upload_content tool call
func (h *HelloContentHandler) handleUploadContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
