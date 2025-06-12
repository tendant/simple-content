package mcp

import (
	"context"
	"fmt"

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
