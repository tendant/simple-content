package simplecontent

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Error types
var (
	// ErrContentNotFound indicates a content was not found
	ErrContentNotFound = errors.New("content not found")
	
	// ErrObjectNotFound indicates an object was not found
	ErrObjectNotFound = errors.New("object not found")
	
	// ErrStorageBackendNotFound indicates a storage backend was not found
	ErrStorageBackendNotFound = errors.New("storage backend not found")
	
	// ErrInvalidContentStatus indicates an invalid content status
	ErrInvalidContentStatus = errors.New("invalid content status")
	
	// ErrInvalidObjectStatus indicates an invalid object status
	ErrInvalidObjectStatus = errors.New("invalid object status")
	
	// ErrUploadFailed indicates an upload operation failed
	ErrUploadFailed = errors.New("upload failed")
	
	// ErrDownloadFailed indicates a download operation failed
	ErrDownloadFailed = errors.New("download failed")

	// ErrContentNotReady indicates content is not in a state ready for download
	ErrContentNotReady = errors.New("content not ready for download")

	// ErrObjectNotReady indicates object is not in a state ready for download
	ErrObjectNotReady = errors.New("object not ready for download")

	// ErrInvalidUploadState indicates content/object cannot be uploaded in its current state
	ErrInvalidUploadState = errors.New("invalid state for upload operation")

	// ErrParentNotReady indicates parent content is not ready for creating derived content
	ErrParentNotReady = errors.New("parent content not ready for derivation")

	// ErrContentBeingProcessed indicates operation cannot proceed while content is being processed
	ErrContentBeingProcessed = errors.New("content is being processed")
)

// ContentError represents an error related to content operations
type ContentError struct {
	ContentID uuid.UUID
	Op        string
	Err       error
}

func (e *ContentError) Error() string {
	return fmt.Sprintf("content operation %s failed for content %s: %v", e.Op, e.ContentID, e.Err)
}

func (e *ContentError) Unwrap() error {
	return e.Err
}

// ObjectError represents an error related to object operations
type ObjectError struct {
	ObjectID uuid.UUID
	Op       string
	Err      error
}

func (e *ObjectError) Error() string {
	return fmt.Sprintf("object operation %s failed for object %s: %v", e.Op, e.ObjectID, e.Err)
}

func (e *ObjectError) Unwrap() error {
	return e.Err
}

// StorageError represents an error related to storage operations
type StorageError struct {
	Backend string
	Key     string
	Op      string
	Err     error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage operation %s failed for key %s on backend %s: %v", e.Op, e.Key, e.Backend, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}