package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DirectUploadClient demonstrates how to implement a Go client for direct uploads
type DirectUploadClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDirectUploadClient creates a new client for direct uploads
func NewDirectUploadClient(baseURL string) *DirectUploadClient {
	return &DirectUploadClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadRequest contains the metadata for a file upload
type UploadRequest struct {
	OwnerID     string   `json:"owner_id"`
	TenantID    string   `json:"tenant_id"`
	FilePath    string   // Local file path (not sent in request)
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// PrepareResponse contains the response from the prepare upload endpoint
type PrepareResponse struct {
	ContentID    string            `json:"content_id"`
	ObjectID     string            `json:"object_id"`
	UploadURL    string            `json:"upload_url"`
	ExpiresIn    int               `json:"expires_in"`
	UploadMethod string            `json:"upload_method"`
	Headers      map[string]string `json:"headers"`
}

// UploadFile performs the complete direct upload workflow
func (c *DirectUploadClient) UploadFile(ctx context.Context, req UploadRequest) (*PrepareResponse, error) {
	// Step 1: Open and validate the file
	file, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Step 2: Prepare the upload
	fmt.Printf("Preparing upload for file: %s (%d bytes)\n", req.FilePath, fileInfo.Size())
	prepareResp, err := c.prepareUpload(ctx, req, fileInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upload: %w", err)
	}

	fmt.Printf("Upload prepared - Object ID: %s\n", prepareResp.ObjectID)
	fmt.Printf("Upload URL expires in: %d seconds\n", prepareResp.ExpiresIn)

	// Step 3: Perform direct upload to storage
	fmt.Println("Uploading file directly to storage...")
	err = c.performDirectUpload(ctx, file, prepareResp)
	if err != nil {
		return nil, fmt.Errorf("direct upload failed: %w", err)
	}

	fmt.Println("Direct upload completed")

	// Step 4: Confirm the upload
	fmt.Println("Confirming upload completion...")
	err = c.confirmUpload(ctx, prepareResp.ObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm upload: %w", err)
	}

	fmt.Printf("Upload confirmed! Content ID: %s\n", prepareResp.ContentID)
	return prepareResp, nil
}

// prepareUpload sends the prepare request to the server
func (c *DirectUploadClient) prepareUpload(ctx context.Context, req UploadRequest, fileInfo os.FileInfo) (*PrepareResponse, error) {
	// Detect content type based on file extension
	contentType := detectContentType(req.FilePath)

	prepareReq := map[string]interface{}{
		"owner_id":     req.OwnerID,
		"tenant_id":    req.TenantID,
		"file_name":    filepath.Base(req.FilePath),
		"content_type": contentType,
		"file_size":    fileInfo.Size(),
		"name":         req.Name,
		"description":  req.Description,
		"tags":         req.Tags,
	}

	var prepareResp PrepareResponse
	err := c.makeJSONRequest(ctx, "POST", "/api/v1/uploads/prepare", prepareReq, &prepareResp)
	if err != nil {
		return nil, err
	}

	return &prepareResp, nil
}

// performDirectUpload uploads the file directly to the storage backend
func (c *DirectUploadClient) performDirectUpload(ctx context.Context, file *os.File, prepareResp *PrepareResponse) error {
	// Reset file pointer to beginning
	_, err := file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Create request to storage backend
	req, err := http.NewRequestWithContext(ctx, prepareResp.UploadMethod, prepareResp.UploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set required headers
	for key, value := range prepareResp.Headers {
		req.Header.Set(key, value)
	}

	// Perform the upload
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// confirmUpload notifies the service that the upload is complete
func (c *DirectUploadClient) confirmUpload(ctx context.Context, objectID string) error {
	confirmReq := map[string]interface{}{
		"object_id": objectID,
	}

	var confirmResp map[string]interface{}
	err := c.makeJSONRequest(ctx, "POST", "/api/v1/uploads/confirm", confirmReq, &confirmResp)
	if err != nil {
		return err
	}

	return nil
}

// GetUploadStatus retrieves the current status of an upload
func (c *DirectUploadClient) GetUploadStatus(ctx context.Context, objectID string) (map[string]interface{}, error) {
	var status map[string]interface{}
	err := c.makeJSONRequest(ctx, "GET", fmt.Sprintf("/api/v1/uploads/status/%s", objectID), nil, &status)
	if err != nil {
		return nil, err
	}

	return status, nil
}

// ListContent retrieves content for a given owner and tenant
func (c *DirectUploadClient) ListContent(ctx context.Context, ownerID, tenantID string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/api/v1/contents?owner_id=%s&tenant_id=%s", ownerID, tenantID)

	var response map[string]interface{}
	err := c.makeJSONRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, err
	}

	if contents, ok := response["contents"].([]interface{}); ok {
		result := make([]map[string]interface{}, len(contents))
		for i, content := range contents {
			if contentMap, ok := content.(map[string]interface{}); ok {
				result[i] = contentMap
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("unexpected response format")
}

// makeJSONRequest is a helper for making HTTP requests with JSON bodies
func (c *DirectUploadClient) makeJSONRequest(ctx context.Context, method, endpoint string, body interface{}, response interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		if errorResp != nil && errorResp["error"] != nil {
			if errMap, ok := errorResp["error"].(map[string]interface{}); ok {
				if message, ok := errMap["message"].(string); ok {
					return fmt.Errorf("API error: %s", message)
				}
			}
		}
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if response != nil {
		err = json.NewDecoder(resp.Body).Decode(response)
		if err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// detectContentType attempts to detect the MIME type based on file extension
func detectContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

// Example usage function
func exampleUsage() {
	// Create client
	client := NewDirectUploadClient("http://localhost:8080")
	ctx := context.Background()

	// Demo UUIDs (same as in the web interface)
	ownerID := "550e8400-e29b-41d4-a716-446655440000"
	tenantID := "550e8400-e29b-41d4-a716-446655440001"

	// Upload a file
	uploadReq := UploadRequest{
		OwnerID:     ownerID,
		TenantID:    tenantID,
		FilePath:    "./sample-file.txt", // Make sure this file exists
		Name:        "Sample Document",
		Description: "Uploaded via Go client",
		Tags:        []string{"demo", "go-client", "direct-upload"},
	}

	result, err := client.UploadFile(ctx, uploadReq)
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}

	fmt.Printf("Upload successful!\n")
	fmt.Printf("Content ID: %s\n", result.ContentID)
	fmt.Printf("Object ID: %s\n", result.ObjectID)

	// Check upload status
	status, err := client.GetUploadStatus(ctx, result.ObjectID)
	if err != nil {
		fmt.Printf("Failed to get status: %v\n", err)
		return
	}

	fmt.Printf("Upload Status: %+v\n", status)

	// List all content
	contents, err := client.ListContent(ctx, ownerID, tenantID)
	if err != nil {
		fmt.Printf("Failed to list content: %v\n", err)
		return
	}

	fmt.Printf("Total content items: %d\n", len(contents))
	for i, content := range contents {
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, content["name"], content["id"])
	}
}

// Utility function to create a sample file for testing
func createSampleFile(path string) error {
	content := fmt.Sprintf("This is a sample file created at %s\nUsed for testing direct upload functionality.\n",
		time.Now().Format(time.RFC3339))

	return os.WriteFile(path, []byte(content), 0644)
}

// Command-line interface for the client
func clientMain() {
	if len(os.Args) < 2 {
		fmt.Println("Direct Upload Client - Usage Examples:")
		fmt.Println()
		fmt.Println("  go run client.go upload <file-path> <name>")
		fmt.Println("  go run client.go status <object-id>")
		fmt.Println("  go run client.go list")
		fmt.Println("  go run client.go demo")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run client.go upload ./document.pdf 'My Document'")
		fmt.Println("  go run client.go status 550e8400-e29b-41d4-a716-446655440000")
		fmt.Println("  go run client.go list")
		fmt.Println("  go run client.go demo  # Creates sample file and uploads it")
		return
	}

	client := NewDirectUploadClient("http://localhost:8080")
	ctx := context.Background()

	// Demo UUIDs
	ownerID := "550e8400-e29b-41d4-a716-446655440000"
	tenantID := "550e8400-e29b-41d4-a716-446655440001"

	command := os.Args[1]

	switch command {
	case "upload":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run client.go upload <file-path> <name>")
			return
		}

		filePath := os.Args[2]
		name := os.Args[3]
		description := ""
		if len(os.Args) > 4 {
			description = os.Args[4]
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("File not found: %s\n", filePath)
			return
		}

		uploadReq := UploadRequest{
			OwnerID:     ownerID,
			TenantID:    tenantID,
			FilePath:    filePath,
			Name:        name,
			Description: description,
			Tags:        []string{"cli-upload", "demo"},
		}

		result, err := client.UploadFile(ctx, uploadReq)
		if err != nil {
			fmt.Printf("Upload failed: %v\n", err)
			return
		}

		fmt.Printf("Upload successful!\n")
		fmt.Printf("Content ID: %s\n", result.ContentID)
		fmt.Printf("Object ID: %s\n", result.ObjectID)

	case "status":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run client.go status <object-id>")
			return
		}

		objectID := os.Args[2]
		status, err := client.GetUploadStatus(ctx, objectID)
		if err != nil {
			fmt.Printf("Failed to get status: %v\n", err)
			return
		}

		fmt.Printf("Upload Status:\n")
		for key, value := range status {
			fmt.Printf("  %s: %v\n", key, value)
		}

	case "list":
		contents, err := client.ListContent(ctx, ownerID, tenantID)
		if err != nil {
			fmt.Printf("Failed to list content: %v\n", err)
			return
		}

		if len(contents) == 0 {
			fmt.Println("No content found.")
			return
		}

		fmt.Printf("Content (%d items):\n", len(contents))
		for i, content := range contents {
			fmt.Printf("  %d. %s\n", i+1, content["name"])
			fmt.Printf("     ID: %s\n", content["id"])
			fmt.Printf("     Type: %s\n", content["document_type"])
			fmt.Printf("     Status: %s\n", content["status"])
			fmt.Println()
		}

	case "demo":
		// Create a sample file and upload it
		sampleFile := "./sample-upload-demo.txt"
		fmt.Printf("Creating sample file: %s\n", sampleFile)

		err := createSampleFile(sampleFile)
		if err != nil {
			fmt.Printf("Failed to create sample file: %v\n", err)
			return
		}

		defer os.Remove(sampleFile) // Clean up after demo

		uploadReq := UploadRequest{
			OwnerID:     ownerID,
			TenantID:    tenantID,
			FilePath:    sampleFile,
			Name:        "Demo Upload",
			Description: "Sample file created and uploaded via Go client demo",
			Tags:        []string{"demo", "sample", "go-client"},
		}

		result, err := client.UploadFile(ctx, uploadReq)
		if err != nil {
			fmt.Printf("Demo upload failed: %v\n", err)
			return
		}

		fmt.Printf("Demo completed successfully!\n")
		fmt.Printf("Content ID: %s\n", result.ContentID)
		fmt.Printf("Object ID: %s\n", result.ObjectID)

		// Show status
		fmt.Println("\nChecking upload status...")
		status, err := client.GetUploadStatus(ctx, result.ObjectID)
		if err != nil {
			fmt.Printf("Failed to get status: %v\n", err)
		} else {
			fmt.Printf("Status: %s\n", status["status"])
			fmt.Printf("File Size: %v bytes\n", status["file_size"])
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: upload, status, list, demo")
	}
}

// This allows the file to be run standalone or imported as a package
func init() {
	if len(os.Args) > 0 && filepath.Base(os.Args[0]) == "client" {
		clientMain()
		os.Exit(0)
	}
}