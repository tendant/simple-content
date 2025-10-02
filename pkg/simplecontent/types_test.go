package simplecontent

import (
	"errors"
	"testing"
)

// TestContentStatusIsValid tests the IsValid method for ContentStatus
func TestContentStatusIsValid(t *testing.T) {
	tests := []struct {
		name   string
		status ContentStatus
		want   bool
	}{
		{
			name:   "valid status: created",
			status: ContentStatusCreated,
			want:   true,
		},
		{
			name:   "valid status: uploaded",
			status: ContentStatusUploaded,
			want:   true,
		},
		{
			name:   "valid status: deleted",
			status: ContentStatusDeleted,
			want:   true,
		},
		{
			name:   "invalid status: empty string",
			status: ContentStatus(""),
			want:   false,
		},
		{
			name:   "invalid status: unknown",
			status: ContentStatus("unknown"),
			want:   false,
		},
		{
			name:   "invalid status: active (undefined)",
			status: ContentStatus("active"),
			want:   false,
		},
		{
			name:   "invalid status: processing",
			status: ContentStatus("processing"),
			want:   false,
		},
		{
			name:   "invalid status: typo - uploadd",
			status: ContentStatus("uploadd"),
			want:   false,
		},
		{
			name:   "invalid status: uppercase CREATED",
			status: ContentStatus("CREATED"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("ContentStatus.IsValid() = %v, want %v for status %q", got, tt.want, tt.status)
			}
		})
	}
}

// TestParseContentStatus tests the ParseContentStatus function
func TestParseContentStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ContentStatus
		wantErr bool
		errType error
	}{
		{
			name:    "valid: created",
			input:   "created",
			want:    ContentStatusCreated,
			wantErr: false,
		},
		{
			name:    "valid: uploaded",
			input:   "uploaded",
			want:    ContentStatusUploaded,
			wantErr: false,
		},
		{
			name:    "valid: deleted",
			input:   "deleted",
			want:    ContentStatusDeleted,
			wantErr: false,
		},
		{
			name:    "invalid: empty string",
			input:   "",
			want:    "",
			wantErr: true,
			errType: ErrInvalidContentStatus,
		},
		{
			name:    "invalid: unknown",
			input:   "unknown",
			want:    "",
			wantErr: true,
			errType: ErrInvalidContentStatus,
		},
		{
			name:    "invalid: active",
			input:   "active",
			want:    "",
			wantErr: true,
			errType: ErrInvalidContentStatus,
		},
		{
			name:    "invalid: processing",
			input:   "processing",
			want:    "",
			wantErr: true,
			errType: ErrInvalidContentStatus,
		},
		{
			name:    "invalid: typo",
			input:   "creaded",
			want:    "",
			wantErr: true,
			errType: ErrInvalidContentStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContentStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseContentStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ParseContentStatus() error = %v, want error type %v", err, tt.errType)
				}
			}
			if got != tt.want {
				t.Errorf("ParseContentStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestObjectStatusIsValid tests the IsValid method for ObjectStatus
func TestObjectStatusIsValid(t *testing.T) {
	tests := []struct {
		name   string
		status ObjectStatus
		want   bool
	}{
		{
			name:   "valid status: created",
			status: ObjectStatusCreated,
			want:   true,
		},
		{
			name:   "valid status: uploading",
			status: ObjectStatusUploading,
			want:   true,
		},
		{
			name:   "valid status: uploaded",
			status: ObjectStatusUploaded,
			want:   true,
		},
		{
			name:   "valid status: processing",
			status: ObjectStatusProcessing,
			want:   true,
		},
		{
			name:   "valid status: processed",
			status: ObjectStatusProcessed,
			want:   true,
		},
		{
			name:   "valid status: failed",
			status: ObjectStatusFailed,
			want:   true,
		},
		{
			name:   "valid status: deleted",
			status: ObjectStatusDeleted,
			want:   true,
		},
		{
			name:   "invalid status: empty string",
			status: ObjectStatus(""),
			want:   false,
		},
		{
			name:   "invalid status: unknown",
			status: ObjectStatus("unknown"),
			want:   false,
		},
		{
			name:   "invalid status: active",
			status: ObjectStatus("active"),
			want:   false,
		},
		{
			name:   "invalid status: pending",
			status: ObjectStatus("pending"),
			want:   false,
		},
		{
			name:   "invalid status: typo - uploadedd",
			status: ObjectStatus("uploadedd"),
			want:   false,
		},
		{
			name:   "invalid status: uppercase CREATED",
			status: ObjectStatus("CREATED"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("ObjectStatus.IsValid() = %v, want %v for status %q", got, tt.want, tt.status)
			}
		})
	}
}

// TestParseObjectStatus tests the ParseObjectStatus function
func TestParseObjectStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ObjectStatus
		wantErr bool
		errType error
	}{
		{
			name:    "valid: created",
			input:   "created",
			want:    ObjectStatusCreated,
			wantErr: false,
		},
		{
			name:    "valid: uploading",
			input:   "uploading",
			want:    ObjectStatusUploading,
			wantErr: false,
		},
		{
			name:    "valid: uploaded",
			input:   "uploaded",
			want:    ObjectStatusUploaded,
			wantErr: false,
		},
		{
			name:    "valid: processing",
			input:   "processing",
			want:    ObjectStatusProcessing,
			wantErr: false,
		},
		{
			name:    "valid: processed",
			input:   "processed",
			want:    ObjectStatusProcessed,
			wantErr: false,
		},
		{
			name:    "valid: failed",
			input:   "failed",
			want:    ObjectStatusFailed,
			wantErr: false,
		},
		{
			name:    "valid: deleted",
			input:   "deleted",
			want:    ObjectStatusDeleted,
			wantErr: false,
		},
		{
			name:    "invalid: empty string",
			input:   "",
			want:    "",
			wantErr: true,
			errType: ErrInvalidObjectStatus,
		},
		{
			name:    "invalid: unknown",
			input:   "unknown",
			want:    "",
			wantErr: true,
			errType: ErrInvalidObjectStatus,
		},
		{
			name:    "invalid: active",
			input:   "active",
			want:    "",
			wantErr: true,
			errType: ErrInvalidObjectStatus,
		},
		{
			name:    "invalid: pending",
			input:   "pending",
			want:    "",
			wantErr: true,
			errType: ErrInvalidObjectStatus,
		},
		{
			name:    "invalid: typo",
			input:   "proceessing",
			want:    "",
			wantErr: true,
			errType: ErrInvalidObjectStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseObjectStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseObjectStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("ParseObjectStatus() error = %v, want error type %v", err, tt.errType)
				}
			}
			if got != tt.want {
				t.Errorf("ParseObjectStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
