package simplecontent

import (
	"context"

	"github.com/google/uuid"
)

// NoopEventSink is a no-operation implementation of EventSink
// Useful for production when you don't need event handling or for testing
type NoopEventSink struct{}

// NewNoopEventSink creates a new no-operation event sink
func NewNoopEventSink() EventSink {
	return &NoopEventSink{}
}

// ContentCreated does nothing and returns nil
func (n *NoopEventSink) ContentCreated(ctx context.Context, content *Content) error {
	return nil
}

// ContentUpdated does nothing and returns nil
func (n *NoopEventSink) ContentUpdated(ctx context.Context, content *Content) error {
	return nil
}

// ContentDeleted does nothing and returns nil
func (n *NoopEventSink) ContentDeleted(ctx context.Context, contentID uuid.UUID) error {
	return nil
}

// ObjectCreated does nothing and returns nil
func (n *NoopEventSink) ObjectCreated(ctx context.Context, object *Object) error {
	return nil
}

// ObjectUploaded does nothing and returns nil
func (n *NoopEventSink) ObjectUploaded(ctx context.Context, object *Object) error {
	return nil
}

// ObjectDeleted does nothing and returns nil
func (n *NoopEventSink) ObjectDeleted(ctx context.Context, objectID uuid.UUID) error {
	return nil
}

// NoopPreviewer is a no-operation implementation of Previewer
// Always returns nil (no preview generated) and supports no content types
type NoopPreviewer struct{}

// NewNoopPreviewer creates a new no-operation previewer
func NewNoopPreviewer() Previewer {
	return &NoopPreviewer{}
}

// GeneratePreview always returns nil (no preview generated)
func (n *NoopPreviewer) GeneratePreview(ctx context.Context, object *Object, blobStore BlobStore) (*ObjectPreview, error) {
	return nil, nil
}

// SupportsContent always returns false (supports no content types)
func (n *NoopPreviewer) SupportsContent(mimeType string) bool {
	return false
}

// LoggingEventSink is an event sink that logs events but takes no other action
// Useful for development and debugging
type LoggingEventSink struct {
	logger Logger
}

// Logger interface for logging events
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewLoggingEventSink creates a new logging event sink
func NewLoggingEventSink(logger Logger) EventSink {
	return &LoggingEventSink{logger: logger}
}

// ContentCreated logs the content creation event
func (l *LoggingEventSink) ContentCreated(ctx context.Context, content *Content) error {
	l.logger.Infof("Content created: ID=%s, Name=%s, Type=%s", content.ID, content.Name, content.DocumentType)
	return nil
}

// ContentUpdated logs the content update event
func (l *LoggingEventSink) ContentUpdated(ctx context.Context, content *Content) error {
	l.logger.Infof("Content updated: ID=%s, Name=%s", content.ID, content.Name)
	return nil
}

// ContentDeleted logs the content deletion event
func (l *LoggingEventSink) ContentDeleted(ctx context.Context, contentID uuid.UUID) error {
	l.logger.Infof("Content deleted: ID=%s", contentID)
	return nil
}

// ObjectCreated logs the object creation event
func (l *LoggingEventSink) ObjectCreated(ctx context.Context, object *Object) error {
	l.logger.Infof("Object created: ID=%s, ContentID=%s, Backend=%s", object.ID, object.ContentID, object.StorageBackendName)
	return nil
}

// ObjectUploaded logs the object upload event
func (l *LoggingEventSink) ObjectUploaded(ctx context.Context, object *Object) error {
	l.logger.Infof("Object uploaded: ID=%s, ContentID=%s, Status=%s", object.ID, object.ContentID, object.Status)
	return nil
}

// ObjectDeleted logs the object deletion event
func (l *LoggingEventSink) ObjectDeleted(ctx context.Context, objectID uuid.UUID) error {
	l.logger.Infof("Object deleted: ID=%s", objectID)
	return nil
}

// BasicImagePreviewer is a simple previewer that generates preview URLs for common image types
type BasicImagePreviewer struct {
	supportedTypes map[string]bool
}

// NewBasicImagePreviewer creates a new basic image previewer
func NewBasicImagePreviewer() Previewer {
	supportedTypes := map[string]bool{
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
	}

	return &BasicImagePreviewer{
		supportedTypes: supportedTypes,
	}
}

// GeneratePreview generates a preview for supported image types
func (b *BasicImagePreviewer) GeneratePreview(ctx context.Context, object *Object, blobStore BlobStore) (*ObjectPreview, error) {
	if !b.SupportsContent(object.ObjectType) {
		return nil, nil // No preview for unsupported types
	}

	// For images, the preview URL is the same as the object URL
	previewURL, err := blobStore.GetPreviewURL(ctx, object.ObjectKey)
	if err != nil {
		return nil, err
	}

	preview := &ObjectPreview{
		ID:          uuid.New(),
		ObjectID:    object.ID,
		PreviewType: "image",
		PreviewURL:  previewURL,
		Status:      "completed",
		CreatedAt:   object.UpdatedAt,
	}

	return preview, nil
}

// SupportsContent returns true if the MIME type is a supported image type
func (b *BasicImagePreviewer) SupportsContent(mimeType string) bool {
	return b.supportedTypes[mimeType]
}