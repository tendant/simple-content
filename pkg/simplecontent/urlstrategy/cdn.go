package urlstrategy

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// CDNStrategy generates URLs that point directly to a CDN
// This provides maximum performance with zero database lookups during file access
type CDNStrategy struct {
	BaseURL string // e.g., "https://cdn.example.com"
}

// NewCDNStrategy creates a new CDN URL strategy
func NewCDNStrategy(baseURL string) *CDNStrategy {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &CDNStrategy{
		BaseURL: baseURL,
	}
}

// GenerateDownloadURL creates a direct CDN URL for downloading content
func (s *CDNStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	if s.BaseURL == "" {
		return "", fmt.Errorf("CDN base URL not configured")
	}

	// Direct CDN URL pointing to the object key
	return fmt.Sprintf("%s/%s", s.BaseURL, objectKey), nil
}

// GeneratePreviewURL creates a direct CDN URL for previewing content
func (s *CDNStrategy) GeneratePreviewURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	// For CDN strategy, preview and download URLs are the same
	// The browser will handle the file based on content type
	return s.GenerateDownloadURL(ctx, contentID, objectKey, storageBackend)
}

// GenerateUploadURL returns empty for CDN strategy (uploads typically go through API)
func (s *CDNStrategy) GenerateUploadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	// CDN strategy doesn't support direct uploads
	return "", nil
}

// Enhanced methods with metadata
func (s *CDNStrategy) GenerateDownloadURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	// CDN strategy can optionally append filename for better SEO/UX
	baseURL, err := s.GenerateDownloadURL(ctx, contentID, objectKey, storageBackend)
	if err != nil {
		return "", err
	}

	// If we have a filename, we could potentially append it for SEO
	// e.g., https://cdn.example.com/path/to/file?filename=document.pdf
	if metadata != nil && metadata.FileName != "" {
		return fmt.Sprintf("%s?filename=%s", baseURL, metadata.FileName), nil
	}

	return baseURL, nil
}

// GeneratePreviewURLWithMetadata creates preview URLs with metadata
func (s *CDNStrategy) GeneratePreviewURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	// For preview, we might want different handling based on content type
	baseURL, err := s.GeneratePreviewURL(ctx, contentID, objectKey, storageBackend)
	if err != nil {
		return "", err
	}

	// Could add preview-specific parameters
	if metadata != nil && metadata.ContentType != "" {
		// For example, add content type hint for better browser handling
		return fmt.Sprintf("%s?type=%s", baseURL, metadata.ContentType), nil
	}

	return baseURL, nil
}