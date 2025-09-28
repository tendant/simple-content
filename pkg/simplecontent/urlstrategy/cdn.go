package urlstrategy

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// CDNStrategy generates URLs that point directly to a CDN for downloads
// and uses application endpoints for uploads (hybrid approach)
type CDNStrategy struct {
	CDNBaseURL    string // e.g., "https://cdn.example.com" (for downloads)
	UploadBaseURL string // e.g., "https://api.example.com" or "/api/v1" (for uploads)
}

// NewCDNStrategy creates a new CDN URL strategy with CDN for downloads only
func NewCDNStrategy(cdnBaseURL string) *CDNStrategy {
	// Ensure cdnBaseURL doesn't have trailing slash
	cdnBaseURL = strings.TrimSuffix(cdnBaseURL, "/")
	return &CDNStrategy{
		CDNBaseURL:    cdnBaseURL,
		UploadBaseURL: "/api/v1", // Default to content-based uploads
	}
}

// NewCDNStrategyWithUpload creates a new CDN URL strategy with custom upload URL
func NewCDNStrategyWithUpload(cdnBaseURL, uploadBaseURL string) *CDNStrategy {
	// Ensure URLs don't have trailing slashes
	cdnBaseURL = strings.TrimSuffix(cdnBaseURL, "/")
	uploadBaseURL = strings.TrimSuffix(uploadBaseURL, "/")
	return &CDNStrategy{
		CDNBaseURL:    cdnBaseURL,
		UploadBaseURL: uploadBaseURL,
	}
}

// GenerateDownloadURL creates a direct CDN URL for downloading content
func (s *CDNStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	if s.CDNBaseURL == "" {
		return "", fmt.Errorf("CDN base URL not configured")
	}

	// Direct CDN URL pointing to the object key
	return fmt.Sprintf("%s/%s", s.CDNBaseURL, objectKey), nil
}

// GeneratePreviewURL creates a direct CDN URL for previewing content
func (s *CDNStrategy) GeneratePreviewURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	// For CDN strategy, preview and download URLs are the same
	// The browser will handle the file based on content type
	return s.GenerateDownloadURL(ctx, contentID, objectKey, storageBackend)
}

// GenerateUploadURL creates an upload URL using the configured upload base URL
func (s *CDNStrategy) GenerateUploadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	if s.UploadBaseURL == "" {
		return "", fmt.Errorf("upload base URL not configured")
	}

	// Use content-based upload URL for hybrid approach
	return fmt.Sprintf("%s/contents/%s/upload", s.UploadBaseURL, contentID), nil
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