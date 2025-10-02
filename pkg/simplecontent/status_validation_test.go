package simplecontent

import (
	"errors"
	"testing"
)

// TestCanDownloadContent tests the canDownloadContent validation function
func TestCanDownloadContent(t *testing.T) {
	tests := []struct {
		name      string
		status    ContentStatus
		wantOK    bool
		wantError error
	}{
		{
			name:      "allow: uploaded",
			status:    ContentStatusUploaded,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: processed",
			status:    ContentStatusProcessed,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: archived",
			status:    ContentStatusArchived,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "deny: created",
			status:    ContentStatusCreated,
			wantOK:    false,
			wantError: ErrContentNotReady,
		},
		{
			name:      "deny: uploading",
			status:    ContentStatusUploading,
			wantOK:    false,
			wantError: ErrContentNotReady,
		},
		{
			name:      "deny: processing",
			status:    ContentStatusProcessing,
			wantOK:    false,
			wantError: ErrContentNotReady,
		},
		{
			name:      "deny: failed",
			status:    ContentStatusFailed,
			wantOK:    false,
			wantError: ErrContentNotReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := canDownloadContent(tt.status)
			if ok != tt.wantOK {
				t.Errorf("canDownloadContent(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Errorf("canDownloadContent(%q) error = %v, want error wrapping %v", tt.status, err, tt.wantError)
			}
		})
	}
}

// TestCanDownloadObject tests the canDownloadObject validation function
func TestCanDownloadObject(t *testing.T) {
	tests := []struct {
		name      string
		status    ObjectStatus
		wantOK    bool
		wantError error
	}{
		{
			name:      "allow: uploaded",
			status:    ObjectStatusUploaded,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: processed",
			status:    ObjectStatusProcessed,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "deny: created",
			status:    ObjectStatusCreated,
			wantOK:    false,
			wantError: ErrObjectNotReady,
		},
		{
			name:      "deny: uploading",
			status:    ObjectStatusUploading,
			wantOK:    false,
			wantError: ErrObjectNotReady,
		},
		{
			name:      "deny: processing",
			status:    ObjectStatusProcessing,
			wantOK:    false,
			wantError: ErrObjectNotReady,
		},
		{
			name:      "deny: failed",
			status:    ObjectStatusFailed,
			wantOK:    false,
			wantError: ErrObjectNotReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := canDownloadObject(tt.status)
			if ok != tt.wantOK {
				t.Errorf("canDownloadObject(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Errorf("canDownloadObject(%q) error = %v, want error wrapping %v", tt.status, err, tt.wantError)
			}
		})
	}
}

// TestCanUploadContent tests the canUploadContent validation function
func TestCanUploadContent(t *testing.T) {
	tests := []struct {
		name      string
		status    ContentStatus
		wantOK    bool
		wantError error
	}{
		{
			name:      "allow: created",
			status:    ContentStatusCreated,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: failed",
			status:    ContentStatusFailed,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "deny: uploading",
			status:    ContentStatusUploading,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: uploaded",
			status:    ContentStatusUploaded,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: processing",
			status:    ContentStatusProcessing,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: processed",
			status:    ContentStatusProcessed,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: archived",
			status:    ContentStatusArchived,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := canUploadContent(tt.status)
			if ok != tt.wantOK {
				t.Errorf("canUploadContent(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Errorf("canUploadContent(%q) error = %v, want error wrapping %v", tt.status, err, tt.wantError)
			}
		})
	}
}

// TestCanUploadObject tests the canUploadObject validation function
func TestCanUploadObject(t *testing.T) {
	tests := []struct {
		name      string
		status    ObjectStatus
		wantOK    bool
		wantError error
	}{
		{
			name:      "allow: created",
			status:    ObjectStatusCreated,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: failed",
			status:    ObjectStatusFailed,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "deny: uploading",
			status:    ObjectStatusUploading,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: uploaded",
			status:    ObjectStatusUploaded,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: processing",
			status:    ObjectStatusProcessing,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
		{
			name:      "deny: processed",
			status:    ObjectStatusProcessed,
			wantOK:    false,
			wantError: ErrInvalidUploadState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := canUploadObject(tt.status)
			if ok != tt.wantOK {
				t.Errorf("canUploadObject(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Errorf("canUploadObject(%q) error = %v, want error wrapping %v", tt.status, err, tt.wantError)
			}
		})
	}
}

// TestCanDeleteContent tests the canDeleteContent validation function
func TestCanDeleteContent(t *testing.T) {
	tests := []struct {
		name      string
		status    ContentStatus
		force     bool
		wantOK    bool
		wantError error
	}{
		{
			name:      "allow: created",
			status:    ContentStatusCreated,
			force:     false,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: uploading",
			status:    ContentStatusUploading,
			force:     false,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: uploaded",
			status:    ContentStatusUploaded,
			force:     false,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: processed",
			status:    ContentStatusProcessed,
			force:     false,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: failed",
			status:    ContentStatusFailed,
			force:     false,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: archived",
			status:    ContentStatusArchived,
			force:     false,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "deny: processing without force",
			status:    ContentStatusProcessing,
			force:     false,
			wantOK:    false,
			wantError: ErrContentBeingProcessed,
		},
		{
			name:      "allow: processing with force",
			status:    ContentStatusProcessing,
			force:     true,
			wantOK:    true,
			wantError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := canDeleteContent(tt.status, tt.force)
			if ok != tt.wantOK {
				t.Errorf("canDeleteContent(%q, force=%v) ok = %v, want %v", tt.status, tt.force, ok, tt.wantOK)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Errorf("canDeleteContent(%q, force=%v) error = %v, want error wrapping %v", tt.status, tt.force, err, tt.wantError)
			}
		})
	}
}

// TestCanCreateDerived tests the canCreateDerived validation function
func TestCanCreateDerived(t *testing.T) {
	tests := []struct {
		name      string
		status    ContentStatus
		wantOK    bool
		wantError error
	}{
		{
			name:      "allow: uploaded",
			status:    ContentStatusUploaded,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "allow: processed",
			status:    ContentStatusProcessed,
			wantOK:    true,
			wantError: nil,
		},
		{
			name:      "deny: created",
			status:    ContentStatusCreated,
			wantOK:    false,
			wantError: ErrParentNotReady,
		},
		{
			name:      "deny: uploading",
			status:    ContentStatusUploading,
			wantOK:    false,
			wantError: ErrParentNotReady,
		},
		{
			name:      "deny: processing",
			status:    ContentStatusProcessing,
			wantOK:    false,
			wantError: ErrParentNotReady,
		},
		{
			name:      "deny: failed",
			status:    ContentStatusFailed,
			wantOK:    false,
			wantError: ErrParentNotReady,
		},
		{
			name:      "deny: archived",
			status:    ContentStatusArchived,
			wantOK:    false,
			wantError: ErrParentNotReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := canCreateDerived(tt.status)
			if ok != tt.wantOK {
				t.Errorf("canCreateDerived(%q) ok = %v, want %v", tt.status, ok, tt.wantOK)
			}
			if tt.wantError != nil && !errors.Is(err, tt.wantError) {
				t.Errorf("canCreateDerived(%q) error = %v, want error wrapping %v", tt.status, err, tt.wantError)
			}
		})
	}
}
