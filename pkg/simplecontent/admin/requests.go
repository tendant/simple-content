package admin

import (
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// ListContentsRequest contains parameters for admin content listing
type ListContentsRequest struct {
	Filters ContentFilters `json:"filters"`
}

// ListContentsResponse contains the paginated list of contents
type ListContentsResponse struct {
	Contents   []*simplecontent.Content `json:"contents"`
	TotalCount *int64                   `json:"total_count,omitempty"` // Optional, set if count was requested
	Limit      int                      `json:"limit"`
	Offset     int                      `json:"offset"`
	HasMore    bool                     `json:"has_more"`
}

// CountRequest contains parameters for counting contents
type CountRequest struct {
	Filters ContentFilters `json:"filters"`
}

// CountResponse contains the count result
type CountResponse struct {
	Count int64 `json:"count"`
}

// StatisticsRequest contains parameters for retrieving content statistics
type StatisticsRequest struct {
	Filters ContentFilters    `json:"filters"`
	Options StatisticsOptions `json:"options"`
}

// StatisticsResponse contains the statistics result
type StatisticsResponse struct {
	Statistics ContentStatistics `json:"statistics"`
	ComputedAt time.Time         `json:"computed_at"`
}

// ListContentsOption provides functional options for listing contents
type ListContentsOption func(*ContentFilters)

// WithTenantID filters by tenant ID
func WithTenantID(tenantID uuid.UUID) ListContentsOption {
	return func(f *ContentFilters) {
		f.TenantID = &tenantID
	}
}

// WithTenantIDs filters by multiple tenant IDs
func WithTenantIDs(tenantIDs ...uuid.UUID) ListContentsOption {
	return func(f *ContentFilters) {
		f.TenantIDs = tenantIDs
	}
}

// WithOwnerID filters by owner ID
func WithOwnerID(ownerID uuid.UUID) ListContentsOption {
	return func(f *ContentFilters) {
		f.OwnerID = &ownerID
	}
}

// WithOwnerIDs filters by multiple owner IDs
func WithOwnerIDs(ownerIDs ...uuid.UUID) ListContentsOption {
	return func(f *ContentFilters) {
		f.OwnerIDs = ownerIDs
	}
}

// WithStatus filters by status
func WithStatus(status string) ListContentsOption {
	return func(f *ContentFilters) {
		f.Status = &status
	}
}

// WithStatuses filters by multiple statuses
func WithStatuses(statuses ...string) ListContentsOption {
	return func(f *ContentFilters) {
		f.Statuses = statuses
	}
}

// WithDerivationType filters by derivation type
func WithDerivationType(derivationType string) ListContentsOption {
	return func(f *ContentFilters) {
		f.DerivationType = &derivationType
	}
}

// WithDerivationTypes filters by multiple derivation types
func WithDerivationTypes(derivationTypes ...string) ListContentsOption {
	return func(f *ContentFilters) {
		f.DerivationTypes = derivationTypes
	}
}

// WithDocumentType filters by document type
func WithDocumentType(documentType string) ListContentsOption {
	return func(f *ContentFilters) {
		f.DocumentType = &documentType
	}
}

// WithDocumentTypes filters by multiple document types
func WithDocumentTypes(documentTypes ...string) ListContentsOption {
	return func(f *ContentFilters) {
		f.DocumentTypes = documentTypes
	}
}

// WithCreatedAfter filters by created after time
func WithCreatedAfter(t time.Time) ListContentsOption {
	return func(f *ContentFilters) {
		f.CreatedAfter = &t
	}
}

// WithCreatedBefore filters by created before time
func WithCreatedBefore(t time.Time) ListContentsOption {
	return func(f *ContentFilters) {
		f.CreatedBefore = &t
	}
}

// WithUpdatedAfter filters by updated after time
func WithUpdatedAfter(t time.Time) ListContentsOption {
	return func(f *ContentFilters) {
		f.UpdatedAfter = &t
	}
}

// WithUpdatedBefore filters by updated before time
func WithUpdatedBefore(t time.Time) ListContentsOption {
	return func(f *ContentFilters) {
		f.UpdatedBefore = &t
	}
}

// WithLimit sets the pagination limit
func WithLimit(limit int) ListContentsOption {
	return func(f *ContentFilters) {
		f.Limit = &limit
	}
}

// WithOffset sets the pagination offset
func WithOffset(offset int) ListContentsOption {
	return func(f *ContentFilters) {
		f.Offset = &offset
	}
}

// WithPagination sets both limit and offset
func WithPagination(limit, offset int) ListContentsOption {
	return func(f *ContentFilters) {
		f.Limit = &limit
		f.Offset = &offset
	}
}

// WithSortBy sets the sort field
func WithSortBy(sortBy string) ListContentsOption {
	return func(f *ContentFilters) {
		f.SortBy = &sortBy
	}
}

// WithSortOrder sets the sort order
func WithSortOrder(sortOrder string) ListContentsOption {
	return func(f *ContentFilters) {
		f.SortOrder = &sortOrder
	}
}

// WithIncludeDeleted includes deleted contents in results
func WithIncludeDeleted() ListContentsOption {
	return func(f *ContentFilters) {
		f.IncludeDeleted = true
	}
}
