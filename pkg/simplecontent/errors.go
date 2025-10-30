package simplecontent

import (
	"errors"
	"fmt"
	"net/http"

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

	// ErrMaxDerivationDepth indicates maximum derivation depth has been exceeded
	ErrMaxDerivationDepth = errors.New("maximum derivation depth exceeded")

	// ErrNoStorageBackend indicates no storage backend is available
	ErrNoStorageBackend = errors.New("no storage backend available")

	// ErrNoObjectsFound indicates no objects were found for the content
	ErrNoObjectsFound = errors.New("no objects found for content")

	// ErrNoUploadedObjects indicates no uploaded objects were found
	ErrNoUploadedObjects = errors.New("no uploaded objects found")
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

// ErrorMessage returns a caller-friendly error message with technical details
func (e *ContentError) ErrorMessage() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// HTTPStatus returns the appropriate HTTP status code for this error
func (e *ContentError) HTTPStatus() int {
	switch {
	case errors.Is(e.Err, ErrContentNotFound):
		return http.StatusNotFound
	case errors.Is(e.Err, ErrInvalidContentStatus):
		return http.StatusBadRequest
	case errors.Is(e.Err, ErrContentNotReady):
		return http.StatusConflict
	case errors.Is(e.Err, ErrParentNotReady):
		return http.StatusConflict
	case errors.Is(e.Err, ErrContentBeingProcessed):
		return http.StatusConflict
	case errors.Is(e.Err, ErrInvalidUploadState):
		return http.StatusConflict
	case errors.Is(e.Err, ErrMaxDerivationDepth):
		return http.StatusBadRequest
	case errors.Is(e.Err, ErrNoObjectsFound):
		return http.StatusNotFound
	case errors.Is(e.Err, ErrNoUploadedObjects):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
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

// ErrorMessage returns a caller-friendly error message with technical details
func (e *ObjectError) ErrorMessage() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Op
}

// HTTPStatus returns the appropriate HTTP status code for this error
func (e *ObjectError) HTTPStatus() int {
	switch {
	case errors.Is(e.Err, ErrObjectNotFound):
		return http.StatusNotFound
	case errors.Is(e.Err, ErrInvalidObjectStatus):
		return http.StatusBadRequest
	case errors.Is(e.Err, ErrObjectNotReady):
		return http.StatusConflict
	case errors.Is(e.Err, ErrUploadFailed):
		return http.StatusInternalServerError
	case errors.Is(e.Err, ErrDownloadFailed):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
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

// ErrorMessage returns a caller-friendly error message with technical details
func (e *StorageError) ErrorMessage() string {
	if e.Err != nil {
		return fmt.Sprintf("%s on backend %s (key: %s): %v", e.Op, e.Backend, e.Key, e.Err)
	}
	return fmt.Sprintf("%s on backend %s (key: %s)", e.Op, e.Backend, e.Key)
}

// HTTPStatus returns the appropriate HTTP status code for this error
func (e *StorageError) HTTPStatus() int {
	switch {
	case errors.Is(e.Err, ErrStorageBackendNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// ToErrorMessage converts an error to a caller-friendly message with technical details
func ToErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's one of our custom error types
	var contentErr *ContentError
	var objectErr *ObjectError
	var storageErr *StorageError

	switch {
	case errors.As(err, &contentErr):
		return contentErr.ErrorMessage()
	case errors.As(err, &objectErr):
		return objectErr.ErrorMessage()
	case errors.As(err, &storageErr):
		return storageErr.ErrorMessage()
	default:
		return err.Error()
	}
}
