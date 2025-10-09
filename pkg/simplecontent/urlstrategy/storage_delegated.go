package urlstrategy

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BlobStore interface for URL generation (to avoid circular imports)
type BlobStore interface {
	GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error)
	GetPreviewURL(ctx context.Context, objectKey string) (string, error)
	GetUploadURL(ctx context.Context, objectKey string) (string, error)
}

// StorageDelegatedStrategy delegates URL generation to the storage backends
// This maintains backward compatibility with existing storage backend URL generation
type StorageDelegatedStrategy struct {
	BlobStores map[string]BlobStore
}

// NewStorageDelegatedStrategy creates a new storage-delegated URL strategy
func NewStorageDelegatedStrategy(blobStores map[string]BlobStore) *StorageDelegatedStrategy {
	return &StorageDelegatedStrategy{
		BlobStores: blobStores,
	}
}

// GenerateDownloadURL delegates to the storage backend's GetDownloadURL method
func (s *StorageDelegatedStrategy) GenerateDownloadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Use filename from metadata if available, otherwise generate one
	filename := ""
	if metadata != nil && metadata.FileName != "" {
		// Remove extension from provided filename and use ContentType for extension
		baseFilename := strings.TrimSuffix(metadata.FileName, filepath.Ext(metadata.FileName))
		ext := getExtensionFromContentType(metadata.ContentType)
		if ext != "" {
			filename = baseFilename + ext
		} else {
			filename = metadata.FileName // Keep original if no content type mapping
		}
	} else {
		// No filename provided, generate one with content type extension
		contentType := ""
		if metadata != nil {
			contentType = metadata.ContentType
		}
		filename = generateFilename(contentID, contentType)
	}

	// Delegate to storage backend (current behavior)
	return backend.GetDownloadURL(ctx, objectKey, filename)
}

// GeneratePreviewURL delegates to the storage backend's GetPreviewURL method
func (s *StorageDelegatedStrategy) GeneratePreviewURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Delegate to storage backend (current behavior)
	return backend.GetPreviewURL(ctx, objectKey)
}

// GenerateUploadURL delegates to the storage backend's GetUploadURL method
func (s *StorageDelegatedStrategy) GenerateUploadURL(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Delegate to storage backend (current behavior)
	return backend.GetUploadURL(ctx, objectKey)
}

// Enhanced methods with metadata
func (s *StorageDelegatedStrategy) GenerateDownloadURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	backend, exists := s.BlobStores[storageBackend]
	if !exists {
		return "", fmt.Errorf("storage backend %s not found", storageBackend)
	}

	// Use filename from metadata if available
	filename := ""
	if metadata != nil {
		filename = metadata.FileName
	}

	// Delegate to storage backend with filename
	return backend.GetDownloadURL(ctx, objectKey, filename)
}

// GeneratePreviewURLWithMetadata creates preview URLs with metadata
func (s *StorageDelegatedStrategy) GeneratePreviewURLWithMetadata(ctx context.Context, contentID uuid.UUID, objectKey string, storageBackend string, metadata *URLMetadata) (string, error) {
	// Preview URLs typically don't need metadata in current storage backend interface
	return s.GeneratePreviewURL(ctx, contentID, objectKey, storageBackend)
}

// generateFilename creates a fallback filename when none is provided
// It uses the contentType to determine the appropriate file extension
// and includes a timestamp to ensure uniqueness
func generateFilename(contentID uuid.UUID, contentType string) string {
	timestamp := time.Now().Format("20060102-150405")
	ext := getExtensionFromContentType(contentType)
	if ext != "" {
		return fmt.Sprintf("%s_%s%s", contentID.String(), timestamp, ext)
	}
	return fmt.Sprintf("%s_%s", contentID.String(), timestamp)
}

// getExtensionFromContentType maps MIME types to file extensions
func getExtensionFromContentType(contentType string) string {
	// Remove any parameters from content type (e.g., "text/plain; charset=utf-8" -> "text/plain")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	// Common MIME type to extension mappings
	mimeMap := map[string]string{
		// Images
		"image/jpeg":    ".jpg",
		"image/jpg":     ".jpg",
		"image/png":     ".png",
		"image/gif":     ".gif",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"image/bmp":     ".bmp",
		"image/tiff":    ".tiff",
		// Documents
		"application/pdf":    ".pdf",
		"application/msword": ".doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.ms-excel": ".xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
		"application/vnd.ms-powerpoint":                                             ".ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
		// Text
		"text/plain":      ".txt",
		"text/html":       ".html",
		"text/css":        ".css",
		"text/javascript": ".js",
		"text/csv":        ".csv",
		"text/xml":        ".xml",
		// Video
		"video/mp4":       ".mp4",
		"video/mpeg":      ".mpeg",
		"video/quicktime": ".mov",
		"video/x-msvideo": ".avi",
		"video/webm":      ".webm",
		// Audio
		"audio/mpeg": ".mp3",
		"audio/wav":  ".wav",
		"audio/ogg":  ".ogg",
		"audio/webm": ".weba",
		// Archives
		"application/zip":             ".zip",
		"application/x-tar":           ".tar",
		"application/gzip":            ".gz",
		"application/x-7z-compressed": ".7z",
		// Other
		"application/json": ".json",
		"application/xml":  ".xml",
	}

	if ext, ok := mimeMap[contentType]; ok {
		return ext
	}
	return ""
}
