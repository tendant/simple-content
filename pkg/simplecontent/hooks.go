package simplecontent

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Hook system allows extending Simple Content behavior without modifying core code.
// Hooks are called at specific points in the content lifecycle.

// Hooks defines all available lifecycle hooks
type Hooks struct {
	// Content lifecycle hooks
	BeforeContentCreate  []BeforeContentCreateHook
	AfterContentCreate   []AfterContentCreateHook
	BeforeContentUpload  []BeforeContentUploadHook
	AfterContentUpload   []AfterContentUploadHook
	BeforeContentDownload []BeforeContentDownloadHook
	AfterContentDownload []AfterContentDownloadHook
	BeforeContentDelete  []BeforeContentDeleteHook
	AfterContentDelete   []AfterContentDeleteHook

	// Derived content hooks
	BeforeDerivedCreate []BeforeDerivedCreateHook
	AfterDerivedCreate  []AfterDerivedCreateHook

	// Metadata hooks
	BeforeMetadataSet []BeforeMetadataSetHook
	AfterMetadataSet  []AfterMetadataSetHook

	// Status change hooks
	OnStatusChange []StatusChangeHook

	// Error hooks
	OnError []ErrorHook
}

// Hook context carries information through the hook chain
type HookContext struct {
	Context   context.Context
	Metadata  map[string]interface{} // Custom metadata passed between hooks
	StopChain bool                   // Set to true to stop processing remaining hooks
}

// NewHookContext creates a new hook context
func NewHookContext(ctx context.Context) *HookContext {
	return &HookContext{
		Context:  ctx,
		Metadata: make(map[string]interface{}),
	}
}

// Content Lifecycle Hooks

// BeforeContentCreateHook is called before creating content
type BeforeContentCreateHook func(hctx *HookContext, req *CreateContentRequest) error

// AfterContentCreateHook is called after content is created
type AfterContentCreateHook func(hctx *HookContext, content *Content) error

// BeforeContentUploadHook is called before uploading content data
type BeforeContentUploadHook func(hctx *HookContext, contentID uuid.UUID, reader io.Reader) (io.Reader, error)

// AfterContentUploadHook is called after content data is uploaded
type AfterContentUploadHook func(hctx *HookContext, contentID uuid.UUID, bytesWritten int64) error

// BeforeContentDownloadHook is called before downloading content
type BeforeContentDownloadHook func(hctx *HookContext, contentID uuid.UUID) error

// AfterContentDownloadHook is called after content is downloaded
type AfterContentDownloadHook func(hctx *HookContext, contentID uuid.UUID, reader io.ReadCloser) (io.ReadCloser, error)

// BeforeContentDeleteHook is called before deleting content
type BeforeContentDeleteHook func(hctx *HookContext, contentID uuid.UUID) error

// AfterContentDeleteHook is called after content is deleted
type AfterContentDeleteHook func(hctx *HookContext, contentID uuid.UUID) error

// Derived Content Hooks

// BeforeDerivedCreateHook is called before creating derived content
type BeforeDerivedCreateHook func(hctx *HookContext, req *CreateDerivedContentRequest) error

// AfterDerivedCreateHook is called after derived content is created
type AfterDerivedCreateHook func(hctx *HookContext, parent *Content, derived *Content) error

// Metadata Hooks

// BeforeMetadataSetHook is called before setting metadata
type BeforeMetadataSetHook func(hctx *HookContext, req *SetContentMetadataRequest) error

// AfterMetadataSetHook is called after metadata is set
type AfterMetadataSetHook func(hctx *HookContext, metadata *ContentMetadata) error

// Status Change Hooks

// StatusChangeHook is called when content status changes
type StatusChangeHook func(hctx *HookContext, contentID uuid.UUID, oldStatus, newStatus ContentStatus) error

// Error Hooks

// ErrorHook is called when an error occurs
type ErrorHook func(hctx *HookContext, operation string, err error)

// Hook execution helpers

// executeBeforeContentCreate runs all BeforeContentCreate hooks
func (h *Hooks) executeBeforeContentCreate(ctx context.Context, req *CreateContentRequest) error {
	if len(h.BeforeContentCreate) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.BeforeContentCreate {
		if err := hook(hctx, req); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeAfterContentCreate runs all AfterContentCreate hooks
func (h *Hooks) executeAfterContentCreate(ctx context.Context, content *Content) error {
	if len(h.AfterContentCreate) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.AfterContentCreate {
		if err := hook(hctx, content); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeBeforeContentUpload runs all BeforeContentUpload hooks
func (h *Hooks) executeBeforeContentUpload(ctx context.Context, contentID uuid.UUID, reader io.Reader) (io.Reader, error) {
	if len(h.BeforeContentUpload) == 0 {
		return reader, nil
	}

	hctx := NewHookContext(ctx)
	currentReader := reader

	for _, hook := range h.BeforeContentUpload {
		modifiedReader, err := hook(hctx, contentID, currentReader)
		if err != nil {
			return nil, err
		}
		if modifiedReader != nil {
			currentReader = modifiedReader
		}
		if hctx.StopChain {
			break
		}
	}
	return currentReader, nil
}

// executeAfterContentUpload runs all AfterContentUpload hooks
func (h *Hooks) executeAfterContentUpload(ctx context.Context, contentID uuid.UUID, bytesWritten int64) error {
	if len(h.AfterContentUpload) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.AfterContentUpload {
		if err := hook(hctx, contentID, bytesWritten); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeBeforeContentDelete runs all BeforeContentDelete hooks
func (h *Hooks) executeBeforeContentDelete(ctx context.Context, contentID uuid.UUID) error {
	if len(h.BeforeContentDelete) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.BeforeContentDelete {
		if err := hook(hctx, contentID); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeAfterContentDelete runs all AfterContentDelete hooks
func (h *Hooks) executeAfterContentDelete(ctx context.Context, contentID uuid.UUID) error {
	if len(h.AfterContentDelete) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.AfterContentDelete {
		if err := hook(hctx, contentID); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeOnStatusChange runs all OnStatusChange hooks
func (h *Hooks) executeOnStatusChange(ctx context.Context, contentID uuid.UUID, oldStatus, newStatus ContentStatus) error {
	if len(h.OnStatusChange) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.OnStatusChange {
		if err := hook(hctx, contentID, oldStatus, newStatus); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeOnError runs all OnError hooks
func (h *Hooks) executeOnError(ctx context.Context, operation string, err error) {
	if len(h.OnError) == 0 {
		return
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.OnError {
		hook(hctx, operation, err)
		if hctx.StopChain {
			break
		}
	}
}

// executeBeforeMetadataSet runs all BeforeMetadataSet hooks
func (h *Hooks) executeBeforeMetadataSet(ctx context.Context, req *SetContentMetadataRequest) error {
	if len(h.BeforeMetadataSet) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.BeforeMetadataSet {
		if err := hook(hctx, req); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// executeAfterMetadataSet runs all AfterMetadataSet hooks
func (h *Hooks) executeAfterMetadataSet(ctx context.Context, metadata *ContentMetadata) error {
	if len(h.AfterMetadataSet) == 0 {
		return nil
	}

	hctx := NewHookContext(ctx)
	for _, hook := range h.AfterMetadataSet {
		if err := hook(hctx, metadata); err != nil {
			return err
		}
		if hctx.StopChain {
			break
		}
	}
	return nil
}

// Common hook implementations (examples)

// LoggingHook logs all operations
func LoggingHook(logger func(format string, args ...interface{})) *Hooks {
	return &Hooks{
		AfterContentCreate: []AfterContentCreateHook{
			func(hctx *HookContext, content *Content) error {
				logger("Content created: %s (owner: %s)", content.ID, content.OwnerID)
				return nil
			},
		},
		AfterContentUpload: []AfterContentUploadHook{
			func(hctx *HookContext, contentID uuid.UUID, bytesWritten int64) error {
				logger("Content uploaded: %s (%d bytes)", contentID, bytesWritten)
				return nil
			},
		},
		AfterContentDelete: []AfterContentDeleteHook{
			func(hctx *HookContext, contentID uuid.UUID) error {
				logger("Content deleted: %s", contentID)
				return nil
			},
		},
		OnError: []ErrorHook{
			func(hctx *HookContext, operation string, err error) {
				logger("Error in %s: %v", operation, err)
			},
		},
	}
}

// ValidationHook adds custom validation
func ValidationHook(validator func(*CreateContentRequest) error) BeforeContentCreateHook {
	return func(hctx *HookContext, req *CreateContentRequest) error {
		return validator(req)
	}
}

// MetricsHook tracks metrics
func MetricsHook(metrics interface {
	IncrementCounter(name string)
	RecordDuration(name string, duration int64)
}) *Hooks {
	return &Hooks{
		AfterContentCreate: []AfterContentCreateHook{
			func(hctx *HookContext, content *Content) error {
				metrics.IncrementCounter("content.created")
				return nil
			},
		},
		AfterContentUpload: []AfterContentUploadHook{
			func(hctx *HookContext, contentID uuid.UUID, bytesWritten int64) error {
				metrics.IncrementCounter("content.uploaded")
				metrics.RecordDuration("upload.bytes", bytesWritten)
				return nil
			},
		},
		AfterContentDelete: []AfterContentDeleteHook{
			func(hctx *HookContext, contentID uuid.UUID) error {
				metrics.IncrementCounter("content.deleted")
				return nil
			},
		},
	}
}
