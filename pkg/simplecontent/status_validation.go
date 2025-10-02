package simplecontent

import "fmt"

// canDownloadContent checks if content can be downloaded based on its status.
// Returns true if download is allowed, false with an error otherwise.
func canDownloadContent(status ContentStatus) (bool, error) {
	switch status {
	case ContentStatusUploaded, ContentStatusProcessed, ContentStatusArchived:
		return true, nil
	case ContentStatusCreated:
		return false, fmt.Errorf("%w: content has not been uploaded yet (status: %s)", ErrContentNotReady, status)
	case ContentStatusUploading:
		return false, fmt.Errorf("%w: content upload is still in progress (status: %s)", ErrContentNotReady, status)
	case ContentStatusProcessing:
		return false, fmt.Errorf("%w: content is being processed (status: %s)", ErrContentNotReady, status)
	case ContentStatusFailed:
		return false, fmt.Errorf("%w: content upload or processing failed (status: %s)", ErrContentNotReady, status)
	default:
		return false, fmt.Errorf("%w: unknown status %s", ErrInvalidContentStatus, status)
	}
}

// canDownloadObject checks if an object can be downloaded based on its status.
// Returns true if download is allowed, false with an error otherwise.
func canDownloadObject(status ObjectStatus) (bool, error) {
	switch status {
	case ObjectStatusUploaded, ObjectStatusProcessed:
		return true, nil
	case ObjectStatusCreated:
		return false, fmt.Errorf("%w: object has not been uploaded yet (status: %s)", ErrObjectNotReady, status)
	case ObjectStatusUploading:
		return false, fmt.Errorf("%w: object upload is still in progress (status: %s)", ErrObjectNotReady, status)
	case ObjectStatusProcessing:
		return false, fmt.Errorf("%w: object is being processed (status: %s)", ErrObjectNotReady, status)
	case ObjectStatusFailed:
		return false, fmt.Errorf("%w: object upload or processing failed (status: %s)", ErrObjectNotReady, status)
	default:
		return false, fmt.Errorf("%w: unknown status %s", ErrInvalidObjectStatus, status)
	}
}

// canUploadContent checks if content can be uploaded based on its current status.
// Returns true if upload is allowed, false with an error otherwise.
func canUploadContent(status ContentStatus) (bool, error) {
	switch status {
	case ContentStatusCreated, ContentStatusFailed:
		return true, nil
	case ContentStatusUploading:
		return false, fmt.Errorf("%w: content upload is already in progress (status: %s)", ErrInvalidUploadState, status)
	case ContentStatusUploaded:
		return false, fmt.Errorf("%w: content has already been uploaded (status: %s)", ErrInvalidUploadState, status)
	case ContentStatusProcessing:
		return false, fmt.Errorf("%w: content is being processed (status: %s)", ErrInvalidUploadState, status)
	case ContentStatusProcessed:
		return false, fmt.Errorf("%w: content has already been processed (status: %s)", ErrInvalidUploadState, status)
	case ContentStatusArchived:
		return false, fmt.Errorf("%w: content has been archived (status: %s)", ErrInvalidUploadState, status)
	default:
		return false, fmt.Errorf("%w: unknown status %s", ErrInvalidContentStatus, status)
	}
}

// canUploadObject checks if an object can be uploaded based on its current status.
// Returns true if upload is allowed, false with an error otherwise.
func canUploadObject(status ObjectStatus) (bool, error) {
	switch status {
	case ObjectStatusCreated, ObjectStatusFailed:
		return true, nil
	case ObjectStatusUploading:
		return false, fmt.Errorf("%w: object upload is already in progress (status: %s)", ErrInvalidUploadState, status)
	case ObjectStatusUploaded:
		return false, fmt.Errorf("%w: object has already been uploaded (status: %s)", ErrInvalidUploadState, status)
	case ObjectStatusProcessing:
		return false, fmt.Errorf("%w: object is being processed (status: %s)", ErrInvalidUploadState, status)
	case ObjectStatusProcessed:
		return false, fmt.Errorf("%w: object has already been processed (status: %s)", ErrInvalidUploadState, status)
	default:
		return false, fmt.Errorf("%w: unknown status %s", ErrInvalidObjectStatus, status)
	}
}

// canDeleteContent checks if content can be deleted based on its current status.
// The force parameter allows deletion even during processing (with warning).
// Returns true if deletion is allowed, false with an error otherwise.
func canDeleteContent(status ContentStatus, force bool) (bool, error) {
	switch status {
	case ContentStatusProcessing:
		if !force {
			return false, fmt.Errorf("%w: use force=true to delete content being processed (status: %s)", ErrContentBeingProcessed, status)
		}
		// Allow deletion with force flag, caller should log a warning
		return true, nil
	case ContentStatusCreated, ContentStatusUploading, ContentStatusUploaded,
		ContentStatusProcessed, ContentStatusFailed, ContentStatusArchived:
		return true, nil
	default:
		return false, fmt.Errorf("%w: unknown status %s", ErrInvalidContentStatus, status)
	}
}

// canCreateDerived checks if derived content can be created from a parent content
// based on the parent's current status.
// Returns true if derivation is allowed, false with an error otherwise.
//
// Note: Original content uses "uploaded" status (terminal state).
// Derived content uses "processed" status (terminal state).
// Both are allowed as parent status to support derived-from-derived scenarios.
func canCreateDerived(parentStatus ContentStatus) (bool, error) {
	switch parentStatus {
	case ContentStatusUploaded:  // Original content (primary use case)
		return true, nil
	case ContentStatusProcessed:  // Derived content (for derived-from-derived)
		return true, nil
	case ContentStatusCreated:
		return false, fmt.Errorf("%w: parent content has not been uploaded yet (status: %s)", ErrParentNotReady, parentStatus)
	case ContentStatusUploading:
		return false, fmt.Errorf("%w: parent content upload is still in progress (status: %s)", ErrParentNotReady, parentStatus)
	case ContentStatusProcessing:
		return false, fmt.Errorf("%w: parent content is being processed (status: %s)", ErrParentNotReady, parentStatus)
	case ContentStatusFailed:
		return false, fmt.Errorf("%w: parent content upload or processing failed (status: %s)", ErrParentNotReady, parentStatus)
	case ContentStatusArchived:
		return false, fmt.Errorf("%w: parent content has been archived (status: %s)", ErrParentNotReady, parentStatus)
	default:
		return false, fmt.Errorf("%w: unknown status %s", ErrInvalidContentStatus, parentStatus)
	}
}
