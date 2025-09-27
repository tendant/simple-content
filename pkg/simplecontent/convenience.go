package simplecontent

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Convenience functions for common derived content operations.
// These functions provide simplified interfaces for common use cases
// while keeping the core Service interface clean.

// GetThumbnailsBySize retrieves thumbnails of specific sizes for a parent content.
// This is a convenience function that uses the service's ListDerivedContent method.
func GetThumbnailsBySize(ctx context.Context, svc Service, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error) {
	variants := make([]string, len(sizes))
	for i, size := range sizes {
		variants[i] = fmt.Sprintf("thumbnail_%s", size)
	}

	return svc.ListDerivedContent(ctx,
		WithParentID(parentID),
		WithDerivationType("thumbnail"),
		WithVariants(variants...),
		WithURLs(),
	)
}

// GetRecentDerived retrieves derived content created after a specific time.
// This is a convenience function that uses the service's ListDerivedContent method.
func GetRecentDerived(ctx context.Context, svc Service, parentID uuid.UUID, since time.Time) ([]*DerivedContent, error) {
	return svc.ListDerivedContent(ctx,
		WithParentID(parentID),
		WithCreatedAfter(since),
		WithSortBy("created_at_desc"),
	)
}

// ListDerivedByTypeAndVariant retrieves derived content by specific type and variant.
// This is a convenience function that uses the service's ListDerivedContent method.
func ListDerivedByTypeAndVariant(ctx context.Context, svc Service, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error) {
	return svc.ListDerivedContent(ctx,
		WithParentID(parentID),
		WithDerivationType(derivationType),
		WithVariant(variant),
	)
}

// ListDerivedByVariants retrieves derived content by specific variants.
// This is a convenience function that uses the service's ListDerivedContent method.
func ListDerivedByVariants(ctx context.Context, svc Service, parentID uuid.UUID, variants []string) ([]*DerivedContent, error) {
	return svc.ListDerivedContent(ctx,
		WithParentID(parentID),
		WithVariants(variants...),
	)
}

// ListDerivedContentWithURLs retrieves derived content with URLs populated.
// This is a convenience function that uses the service's ListDerivedContent method.
// Deprecated: Use svc.ListDerivedContent with WithURLs() option instead.
func ListDerivedContentWithURLs(ctx context.Context, svc Service, params ListDerivedContentParams) ([]*DerivedContent, error) {
	// Convert params to options and add URLs
	options := []ListDerivedContentOption{WithURLs()}

	if params.ParentID != nil {
		options = append(options, WithParentID(*params.ParentID))
	}
	if params.DerivationType != nil {
		options = append(options, WithDerivationType(*params.DerivationType))
	}
	if params.Variant != nil {
		options = append(options, WithVariant(*params.Variant))
	}
	if len(params.Variants) > 0 {
		options = append(options, WithVariants(params.Variants...))
	}
	if params.Limit != nil {
		options = append(options, WithLimit(*params.Limit))
	}
	if params.Offset != nil {
		options = append(options, WithOffset(*params.Offset))
	}

	return svc.ListDerivedContent(ctx, options...)
}

// GetDerivedContentWithURLs retrieves a single derived content with URLs populated.
// This is a convenience function that uses the service's GetDerivedRelationship method.
func GetDerivedContentWithURLs(ctx context.Context, svc Service, contentID uuid.UUID) (*DerivedContent, error) {
	// Get the derived content relationship
	derived, err := svc.GetDerivedRelationship(ctx, contentID)
	if err != nil {
		return nil, err
	}

	// Enhance with URLs using the ListDerivedContent method
	results, err := svc.ListDerivedContent(ctx,
		WithParentID(derived.ParentID),
		WithDerivationType(derived.DerivationType),
		WithVariant(derived.Variant),
		WithURLs(),
	)
	if err != nil {
		return nil, err
	}

	// Find the matching derived content
	for _, result := range results {
		if result.ContentID == contentID {
			return result, nil
		}
	}

	// If not found in enhanced results, return the original (without URLs)
	return derived, nil
}

// CountDerivedContent counts derived content matching the given parameters.
// This is a convenience function that uses the service's ListDerivedContent method.
// Deprecated: Use svc.ListDerivedContent with appropriate options instead.
func CountDerivedContent(ctx context.Context, svc Service, params ListDerivedContentParams) (int64, error) {
	// Convert params to options (without limits for counting)
	var options []ListDerivedContentOption

	if params.ParentID != nil {
		options = append(options, WithParentID(*params.ParentID))
	}
	if params.DerivationType != nil {
		options = append(options, WithDerivationType(*params.DerivationType))
	}
	if params.Variant != nil {
		options = append(options, WithVariant(*params.Variant))
	}
	if len(params.Variants) > 0 {
		options = append(options, WithVariants(params.Variants...))
	}
	// Note: Deliberately not including URLs, objects, metadata for counting

	results, err := svc.ListDerivedContent(ctx, options...)
	if err != nil {
		return 0, err
	}

	return int64(len(results)), nil
}

// Upload convenience functions for backward compatibility

// UploadObjectSimple uploads an object without metadata (backward compatibility).
// Deprecated: Use the unified UploadContent or UploadDerivedContent instead.
// This function now requires a StorageService interface for object operations.
func UploadObjectSimple(ctx context.Context, svc StorageService, objectID uuid.UUID, reader io.Reader) error {
	req := UploadObjectRequest{
		ObjectID: objectID,
		Reader:   reader,
		// MimeType is empty for simple upload
	}
	return svc.UploadObject(ctx, req)
}

// UploadObjectWithMimeType uploads an object with a specific MIME type.
// Deprecated: Use the unified UploadContent or UploadDerivedContent instead.
// This function now requires a StorageService interface for object operations.
func UploadObjectWithMimeType(ctx context.Context, svc StorageService, objectID uuid.UUID, reader io.Reader, mimeType string) error {
	req := UploadObjectRequest{
		ObjectID: objectID,
		Reader:   reader,
		MimeType: mimeType,
	}
	return svc.UploadObject(ctx, req)
}

// Helper functions

// stringPtr returns a pointer to a string value.
func stringPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int {
	return &i
}

// GetContentDetails returns all details for content in the simplest possible interface.
// This is the recommended way for clients to get all details for any content.
func GetContentDetails(ctx context.Context, svc Service, contentID uuid.UUID) (*ContentDetails, error) {
	return svc.GetContentDetails(ctx, contentID)
}

// Note: With the introduction of the option pattern in ListDerivedContent, most specific convenience
// functions are no longer needed since the option pattern provides a cleaner, more flexible API.
// Users can simply call:
//   svc.ListDerivedContent(ctx, simplecontent.WithParentID(id), simplecontent.WithDerivationType("thumbnail"))
// instead of creating separate wrapper functions.