package simplecontent

import (
	"io"

	"github.com/google/uuid"
)

// Request/Response DTOs

// CreateContentRequest contains parameters for creating new content
type CreateContentRequest struct {
	OwnerID        uuid.UUID
	OwnerType      string
	TenantID       uuid.UUID
	Name           string
	Description    string
	DocumentType   string
	DerivationType string
}

// CreateDerivedContentRequest contains parameters for creating derived content.
//
// DerivationType is the user-facing derivation type stored on the derived
// Content (e.g., "thumbnail", "preview", "transcode").
// Variant is the specific derivation (e.g., "thumbnail_256"). If Variant is
// provided and DerivationType is empty, the service infers DerivationType from
// the prefix before the first underscore in Variant.
//
// InitialStatus allows setting a custom initial status for async workflows.
// If not provided, defaults to "created". Common values:
//   - "created" (default): Content placeholder created, waiting for processing
//   - "processing": Immediately mark as being processed (useful for queue consumers)
type CreateDerivedContentRequest struct {
    ParentID       uuid.UUID
    OwnerID        uuid.UUID
    TenantID       uuid.UUID
    DerivationType string
    Variant        string
    Metadata       map[string]interface{}
    InitialStatus  ContentStatus // Optional: defaults to "created"
	OwnerType      string
}

// UpdateContentRequest contains parameters for updating content
type UpdateContentRequest struct {
	Content *Content
}

// ListContentRequest contains parameters for listing content
type ListContentRequest struct {
	OwnerID  uuid.UUID
	TenantID uuid.UUID
}

// SetContentMetadataRequest contains parameters for setting content metadata
type SetContentMetadataRequest struct {
	ContentID      uuid.UUID
	ContentType    string
	Title          string
	Description    string
	Tags           []string
	FileName       string
	FileSize       int64
	CreatedBy      string
	CustomMetadata map[string]interface{}
}

// CreateObjectRequest contains parameters for creating an object
type CreateObjectRequest struct {
	ContentID          uuid.UUID
	StorageBackendName string
	Version            int
	ObjectKey          string
	FileName           string
}

// UploadObjectRequest contains parameters for uploading an object
type UploadObjectRequest struct {
	ObjectID uuid.UUID
	Reader   io.Reader
	MimeType string // Optional - for metadata
}

// UploadObjectWithMetadataRequest is deprecated - use UploadObjectRequest instead
type UploadObjectWithMetadataRequest struct {
	ObjectID uuid.UUID
	MimeType string
}

// UploadContentRequest contains parameters for uploading content with data.
// This replaces the multi-step workflow of CreateContent + CreateObject + UploadObject.
type UploadContentRequest struct {
	OwnerID            uuid.UUID
	TenantID           uuid.UUID
	Name               string
	Description        string
	DocumentType       string
	StorageBackendName string // Optional - uses default if empty
	Reader             io.Reader
	FileName           string // Optional - for metadata
	FileSize           int64  // Optional - for metadata
	Tags               []string // Optional - for metadata
	CustomMetadata     map[string]interface{} // Optional - additional metadata
}

// UploadDerivedContentRequest contains parameters for uploading derived content.
// This replaces the workflow of CreateDerivedContent + CreateObject + UploadObject.
type UploadDerivedContentRequest struct {
	ParentID           uuid.UUID
	OwnerID            uuid.UUID
	TenantID           uuid.UUID
	DerivationType     string
	Variant            string
	StorageBackendName string // Optional - uses default if empty
	Reader             io.Reader
	FileName           string // Optional - for metadata
	FileSize           int64  // Optional - for metadata
	Tags               []string // Optional - for metadata
	Metadata           map[string]interface{} // Derivation metadata
}

// UploadObjectForContentRequest contains parameters for uploading an object to existing content.
// This is used for async workflows where content is created first, then data uploaded later.
//
// Example async workflow:
//   1. CreateDerivedContent() with InitialStatus="processing"
//   2. Worker generates thumbnail
//   3. UploadObjectForContent() with thumbnail data
//   4. UpdateContentStatus() to "processed"
type UploadObjectForContentRequest struct {
	ContentID          uuid.UUID
	StorageBackendName string // Optional - uses default if empty
	Reader             io.Reader
	FileName           string // Optional - for metadata
	MimeType           string // Optional - for metadata
}

// ContentDetailsOption provides configuration for GetContentDetails calls
type ContentDetailsOption func(*ContentDetailsConfig)

// ContentDetailsConfig holds configuration for content details retrieval
type ContentDetailsConfig struct {
	IncludeUploadURL bool
	URLExpiryTime    int // Seconds
}

// WithUploadAccess configures GetContentDetails to include upload URLs for content that needs data
func WithUploadAccess() ContentDetailsOption {
	return func(cfg *ContentDetailsConfig) {
		cfg.IncludeUploadURL = true
		cfg.URLExpiryTime = 1800 // 30 minutes default
	}
}

// WithUploadAccessExpiry configures GetContentDetails to include upload URLs with custom expiry
func WithUploadAccessExpiry(expirySeconds int) ContentDetailsOption {
	return func(cfg *ContentDetailsConfig) {
		cfg.IncludeUploadURL = true
		cfg.URLExpiryTime = expirySeconds
	}
}
