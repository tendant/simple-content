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

	params := ListDerivedContentParams{
		ParentID:       &parentID,
		DerivationType: stringPtr("thumbnail"),
		Variants:       variants,
		IncludeURLs:    true, // Always include URLs for thumbnails
	}
	return svc.ListDerivedContent(ctx, params)
}

// GetRecentDerived retrieves derived content created after a specific time.
// This is a convenience function that uses the service's ListDerivedContent method.
func GetRecentDerived(ctx context.Context, svc Service, parentID uuid.UUID, since time.Time) ([]*DerivedContent, error) {
	params := ListDerivedContentParams{
		ParentID:     &parentID,
		CreatedAfter: &since,
		SortBy:       stringPtr("created_at_desc"),
	}
	return svc.ListDerivedContent(ctx, params)
}

// ListDerivedByTypeAndVariant retrieves derived content by specific type and variant.
// This is a convenience function that uses the service's ListDerivedContent method.
func ListDerivedByTypeAndVariant(ctx context.Context, svc Service, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error) {
	params := ListDerivedContentParams{
		ParentID:       &parentID,
		DerivationType: &derivationType,
		Variant:        &variant,
	}
	return svc.ListDerivedContent(ctx, params)
}

// ListDerivedByVariants retrieves derived content by specific variants.
// This is a convenience function that uses the service's ListDerivedContent method.
func ListDerivedByVariants(ctx context.Context, svc Service, parentID uuid.UUID, variants []string) ([]*DerivedContent, error) {
	params := ListDerivedContentParams{
		ParentID: &parentID,
		Variants: variants,
	}
	return svc.ListDerivedContent(ctx, params)
}

// ListDerivedContentWithURLs retrieves derived content with URLs populated.
// This is a convenience function that uses the service's ListDerivedContent method.
func ListDerivedContentWithURLs(ctx context.Context, svc Service, params ListDerivedContentParams) ([]*DerivedContent, error) {
	params.IncludeURLs = true
	return svc.ListDerivedContent(ctx, params)
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
	params := ListDerivedContentParams{
		ParentID:       &derived.ParentID,
		DerivationType: &derived.DerivationType,
		Variant:        &derived.Variant,
		IncludeURLs:    true,
	}

	results, err := svc.ListDerivedContent(ctx, params)
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
func CountDerivedContent(ctx context.Context, svc Service, params ListDerivedContentParams) (int64, error) {
	// For counting, we temporarily remove limits and get all matching records
	countParams := params
	countParams.Limit = nil
	countParams.Offset = nil
	countParams.IncludeURLs = false    // Don't need URLs for counting
	countParams.IncludeObjects = false // Don't need objects for counting
	countParams.IncludeMetadata = false // Don't need metadata for counting

	results, err := svc.ListDerivedContent(ctx, countParams)
	if err != nil {
		return 0, err
	}

	return int64(len(results)), nil
}

// Upload convenience functions for backward compatibility

// UploadObjectSimple uploads an object without metadata (backward compatibility).
// This is a convenience function that uses the service's UploadObject method.
func UploadObjectSimple(ctx context.Context, svc Service, objectID uuid.UUID, reader io.Reader) error {
	req := UploadObjectRequest{
		ObjectID: objectID,
		Reader:   reader,
		// MimeType is empty for simple upload
	}
	return svc.UploadObject(ctx, req)
}

// UploadObjectWithMimeType uploads an object with a specific MIME type.
// This is a convenience function that uses the service's UploadObject method.
func UploadObjectWithMimeType(ctx context.Context, svc Service, objectID uuid.UUID, reader io.Reader, mimeType string) error {
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

// Convenience functions using the new option pattern

// GetThumbnailsBySizeWithOptions retrieves thumbnails of specific sizes using the option pattern.
func GetThumbnailsBySizeWithOptions(ctx context.Context, svc Service, parentID uuid.UUID, sizes []string) ([]*DerivedContent, error) {
	variants := make([]string, len(sizes))
	for i, size := range sizes {
		variants[i] = fmt.Sprintf("thumbnail_%s", size)
	}

	return svc.ListDerivedContentWithOptions(ctx,
		WithParentID(parentID),
		WithDerivationType("thumbnail"),
		WithVariants(variants...),
		WithURLs(),
	)
}

// GetRecentDerivedWithOptions retrieves derived content created after a specific time using the option pattern.
func GetRecentDerivedWithOptions(ctx context.Context, svc Service, parentID uuid.UUID, since time.Time) ([]*DerivedContent, error) {
	return svc.ListDerivedContentWithOptions(ctx,
		WithParentID(parentID),
		WithCreatedAfter(since),
		WithSortBy("created_at_desc"),
	)
}

// ListDerivedByTypeAndVariantWithOptions retrieves derived content by specific type and variant using the option pattern.
func ListDerivedByTypeAndVariantWithOptions(ctx context.Context, svc Service, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error) {
	return svc.ListDerivedContentWithOptions(ctx,
		WithParentID(parentID),
		WithDerivationType(derivationType),
		WithVariant(variant),
	)
}

// ListDerivedByVariantsWithOptions retrieves derived content by specific variants using the option pattern.
func ListDerivedByVariantsWithOptions(ctx context.Context, svc Service, parentID uuid.UUID, variants []string) ([]*DerivedContent, error) {
	return svc.ListDerivedContentWithOptions(ctx,
		WithParentID(parentID),
		WithVariants(variants...),
	)
}

// GetDerivedContentWithURLsUsingOptions retrieves derived content with URLs populated using the option pattern.
func GetDerivedContentWithURLsUsingOptions(ctx context.Context, svc Service, parentID uuid.UUID, derivationType, variant string) ([]*DerivedContent, error) {
	return svc.ListDerivedContentWithOptions(ctx,
		WithParentID(parentID),
		WithDerivationType(derivationType),
		WithVariant(variant),
		WithURLs(),
	)
}