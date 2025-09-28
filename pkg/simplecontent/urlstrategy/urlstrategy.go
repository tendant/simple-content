package urlstrategy

import (
	"context"

	"github.com/google/uuid"
)

// URLStrategy defines the interface for URL generation strategies
type URLStrategy interface {
	// GenerateDownloadURL creates a download URL for content
	GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error)

	// GeneratePreviewURL creates a preview URL for content
	GeneratePreviewURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error)

	// GenerateUploadURL creates an upload URL for content (optional, may return empty)
	GenerateUploadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error)
}

// URLMetadata contains additional information that strategies might need
type URLMetadata struct {
	FileName    string
	ContentType string
	Version     int
}

// EnhancedURLStrategy provides additional metadata for URL generation
type EnhancedURLStrategy interface {
	URLStrategy

	// Enhanced methods with metadata
	GenerateDownloadURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error)
	GeneratePreviewURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error)
}