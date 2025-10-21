package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// ServiceClient wraps the simplecontent.Service to provide a client interface
type ServiceClient struct {
	service simplecontent.Service
	verbose bool
}

// NewServiceClient creates a new service-based client
func NewServiceClient(service simplecontent.Service, verbose bool) *ServiceClient {
	return &ServiceClient{
		service: service,
		verbose: verbose,
	}
}

// UploadFile uploads a file using the service
func (c *ServiceClient) UploadFile(filePath string, analysisType string, metadata map[string]interface{}) (*ContentUploadResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if c.verbose {
		fmt.Printf("Uploading file: %s\n", filePath)
	}

	// Use UploadContent from service
	ctx := context.Background()
	content, err := c.service.UploadContent(ctx, simplecontent.UploadContentRequest{
		Reader:             file,
		Name:               fileInfo.Name(),
		FileName:           fileInfo.Name(),
		FileSize:           fileInfo.Size(),
		DocumentType:       detectMimeType(filePath),
		CustomMetadata:     metadata,
		StorageBackendName: "", // Use default backend
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload content: %w", err)
	}

	// Get content details to generate URL
	details, err := c.service.GetContentDetails(ctx, content.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get content details: %w", err)
	}

	return &ContentUploadResponse{
		ID:  content.ID.String(),
		URL: details.Download,
	}, nil
}

// UploadJSON uploads content using JSON request (base64, URL, or metadata-only)
func (c *ServiceClient) UploadJSON(req ContentUploadRequest) (*ContentUploadResponse, error) {
	ctx := context.Background()

	// Handle request URL mode (metadata-only for pre-signed upload)
	if req.Size != nil && req.Data == "" && req.URL == "" {
		// Create content first
		content, err := c.service.CreateContent(ctx, simplecontent.CreateContentRequest{
			Name:         req.Filename,
			DocumentType: req.MimeType,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create content: %w", err)
		}

		// Create object for the content
		storageService, ok := c.service.(simplecontent.StorageService)
		if !ok {
			return nil, fmt.Errorf("service does not support storage operations")
		}

		object, err := storageService.CreateObject(ctx, simplecontent.CreateObjectRequest{
			ContentID:          content.ID,
			StorageBackendName: "", // Use default
			FileName:           req.Filename,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create object: %w", err)
		}

		// Get upload URL
		uploadURL, err := storageService.GetUploadURL(ctx, object.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get upload URL: %w", err)
		}

		// Get content details
		details, err := c.service.GetContentDetails(ctx, content.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get content details: %w", err)
		}

		return &ContentUploadResponse{
			ID:        content.ID.String(),
			URL:       details.Download,
			UploadURL: uploadURL,
		}, nil
	}

	// Handle URL reference mode
	if req.URL != "" {
		// For URL reference, we'd need to implement URL fetching
		// For now, return an error as this requires additional implementation
		return nil, fmt.Errorf("URL reference upload not yet implemented in service client")
	}

	// Handle base64 mode
	if req.Data != "" {
		// For base64, we'd need to decode and upload
		// For now, return an error as this requires additional implementation
		return nil, fmt.Errorf("base64 upload not yet implemented in service client")
	}

	return nil, fmt.Errorf("invalid upload request")
}

// DownloadContent downloads content by ID
func (c *ServiceClient) DownloadContent(contentIDs []string, outputPath string) error {
	if len(contentIDs) == 0 {
		return fmt.Errorf("no content IDs provided")
	}

	ctx := context.Background()
	contentID, err := uuid.Parse(contentIDs[0])
	if err != nil {
		return fmt.Errorf("invalid content ID: %w", err)
	}

	if c.verbose {
		fmt.Printf("Downloading content: %s\n", contentID)
	}

	reader, err := c.service.DownloadContent(ctx, contentID)
	if err != nil {
		return fmt.Errorf("failed to download content: %w", err)
	}
	defer reader.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, reader); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	if c.verbose {
		fmt.Printf("Downloaded to: %s\n", outputPath)
	}

	return nil
}

// ListContents lists all uploaded contents
func (c *ServiceClient) ListContents(limit, offset int) (*ContentListResponse, error) {
	ctx := context.Background()

	// List all content (note: the service doesn't have pagination yet)
	contents, err := c.service.ListContent(ctx, simplecontent.ListContentRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list contents: %w", err)
	}

	// Convert to response format
	var responses []ContentUploadResponse
	for _, content := range contents {
		// Get content details for URL
		details, err := c.service.GetContentDetails(ctx, content.ID)
		if err != nil {
			// Skip if we can't get details
			continue
		}

		responses = append(responses, ContentUploadResponse{
			ID:  content.ID.String(),
			URL: details.Download,
		})
	}

	// Apply pagination manually
	start := offset
	end := offset + limit
	if limit <= 0 || end > len(responses) {
		end = len(responses)
	}
	if start > len(responses) {
		start = len(responses)
	}

	return &ContentListResponse{
		Contents: responses[start:end],
		Total:    len(responses),
	}, nil
}

// DeleteContent deletes a content
func (c *ServiceClient) DeleteContent(contentID string) error {
	ctx := context.Background()
	id, err := uuid.Parse(contentID)
	if err != nil {
		return fmt.Errorf("invalid content ID: %w", err)
	}

	if c.verbose {
		fmt.Printf("Deleting content: %s\n", contentID)
	}

	return c.service.DeleteContent(ctx, id)
}

// GetContentMetadata gets metadata for a content
func (c *ServiceClient) GetContentMetadata(contentID string) (*ContentDownloadMetadata, error) {
	ctx := context.Background()
	id, err := uuid.Parse(contentID)
	if err != nil {
		return nil, fmt.Errorf("invalid content ID: %w", err)
	}

	metadata, err := c.service.GetContentMetadata(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get content metadata: %w", err)
	}

	content, err := c.service.GetContent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return &ContentDownloadMetadata{
		ID:        contentID,
		Filename:  metadata.FileName,
		MimeType:  metadata.MimeType,
		Size:      metadata.FileSize,
		CreatedAt: content.CreatedAt,
	}, nil
}

// AnalysisFiles submits files for analysis
func (c *ServiceClient) AnalysisFiles(req ContentAnalysisRequest) (*ContentAnalysisResponse, error) {
	// Analysis is not yet implemented in the service
	return nil, fmt.Errorf("analysis not yet implemented in service client")
}

// GetAnalysisStatus gets the status of an analysis job
func (c *ServiceClient) GetAnalysisStatus(analysisID string) (*AnalysisStatusResponse, error) {
	// Analysis is not yet implemented in the service
	return nil, fmt.Errorf("analysis not yet implemented in service client")
}

// ListAnalyses lists all analyses
func (c *ServiceClient) ListAnalyses(status string, limit, offset int) (*ContentListResponse, error) {
	// Analysis is not yet implemented in the service
	return nil, fmt.Errorf("analysis not yet implemented in service client")
}

// detectMimeType detects the MIME type based on file extension
func detectMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".csv":
		return "text/csv"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
