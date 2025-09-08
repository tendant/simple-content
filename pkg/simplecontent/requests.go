package simplecontent

import "github.com/google/uuid"

// Request/Response DTOs

// CreateContentRequest contains parameters for creating new content
type CreateContentRequest struct {
	OwnerID        uuid.UUID
	TenantID       uuid.UUID
	Name           string
	Description    string
	DocumentType   string
	DerivationType string
}

// CreateDerivedContentRequest contains parameters for creating derived content.
//
// Category is the user-facing, coarse-grained type for the derived content
// (e.g., "thumbnail", "preview"). This value is stored on the derived
// Content's DerivationType field for easy filtering.
//
// DerivationType is the specific derivation within the category (e.g.,
// "thumbnail_256", "thumbnail_720"). This value is stored in the derived
// content relationship record to precisely identify the derivation variant.
type CreateDerivedContentRequest struct {
    ParentID       uuid.UUID
    OwnerID        uuid.UUID
    TenantID       uuid.UUID
    Category       string
    DerivationType string
    Metadata       map[string]interface{}
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
}

// UploadObjectWithMetadataRequest contains parameters for uploading object with metadata
type UploadObjectWithMetadataRequest struct {
	ObjectID uuid.UUID
	MimeType string
}
