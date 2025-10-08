package urlstrategy

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ContentBasedStrategy generates URLs based on content ID
// This routes requests through the application server for access control and flexibility
type ContentBasedStrategy struct {
	APIBaseURL string // e.g., "https://api.example.com" or "/api/v1"
}

// NewContentBasedStrategy creates a new content-based URL strategy
func NewContentBasedStrategy(apiBaseURL string) *ContentBasedStrategy {
	// Ensure apiBaseURL doesn't have trailing slash
	apiBaseURL = strings.TrimSuffix(apiBaseURL, "/")
	return &ContentBasedStrategy{
		APIBaseURL: apiBaseURL,
	}
}

// GenerateDownloadURL creates a content-based download URL
func (s *ContentBasedStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	if s.APIBaseURL == "" {
		return "", fmt.Errorf("API base URL not configured")
	}

	// Content-based URL that routes through the application
	return fmt.Sprintf("%s/contents/%s/download", s.APIBaseURL, contentID.String()), nil
}

// GeneratePreviewURL creates a content-based preview URL
func (s *ContentBasedStrategy) GeneratePreviewURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	if s.APIBaseURL == "" {
		return "", fmt.Errorf("API base URL not configured")
	}

	// Content-based preview URL
	return fmt.Sprintf("%s/contents/%s/preview", s.APIBaseURL, contentID.String()), nil
}

// GenerateUploadURL creates a content-based upload URL
func (s *ContentBasedStrategy) GenerateUploadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	if s.APIBaseURL == "" {
		return "", fmt.Errorf("API base URL not configured")
	}

	// Content-based upload URL
	return fmt.Sprintf("%s/contents/%s/upload", s.APIBaseURL, contentID.String()), nil
}

// Enhanced methods with metadata
func (s *ContentBasedStrategy) GenerateDownloadURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	baseURL, err := s.GenerateDownloadURL(ctx, contentID, objectKey, storageBackend, nil)
	if err != nil {
		return "", err
	}

	// Can add query parameters for additional metadata
	var params []string
	if metadata != nil {
		if metadata.FileName != "" {
			params = append(params, fmt.Sprintf("filename=%s", metadata.FileName))
		}
		if metadata.Version > 0 {
			params = append(params, fmt.Sprintf("version=%d", metadata.Version))
		}
	}

	if len(params) > 0 {
		return fmt.Sprintf("%s?%s", baseURL, strings.Join(params, "&")), nil
	}

	return baseURL, nil
}

// GeneratePreviewURLWithMetadata creates preview URLs with metadata
func (s *ContentBasedStrategy) GeneratePreviewURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	baseURL, err := s.GeneratePreviewURL(ctx, contentID, objectKey, storageBackend)
	if err != nil {
		return "", err
	}

	// Add preview-specific parameters
	var params []string
	if metadata != nil {
		if metadata.ContentType != "" {
			params = append(params, fmt.Sprintf("type=%s", metadata.ContentType))
		}
		// Preview might want inline disposition
		params = append(params, "disposition=inline")
	}

	if len(params) > 0 {
		return fmt.Sprintf("%s?%s", baseURL, strings.Join(params, "&")), nil
	}

	return baseURL, nil
}