package admin

import (
	"time"

	"github.com/google/uuid"
)

// ContentStatistics provides aggregated statistics about content
type ContentStatistics struct {
	TotalCount         int64                  `json:"total_count"`
	ByStatus           map[string]int64       `json:"by_status,omitempty"`
	ByTenant           map[string]int64       `json:"by_tenant,omitempty"`
	ByDerivationType   map[string]int64       `json:"by_derivation_type,omitempty"`
	ByDocumentType     map[string]int64       `json:"by_document_type,omitempty"`
	OldestContent      *time.Time             `json:"oldest_content,omitempty"`
	NewestContent      *time.Time             `json:"newest_content,omitempty"`
}

// ContentFilters defines flexible filtering options for admin operations
type ContentFilters struct {
	// Identity filters
	TenantID  *uuid.UUID   `json:"tenant_id,omitempty"`
	TenantIDs []uuid.UUID  `json:"tenant_ids,omitempty"`
	OwnerID   *uuid.UUID   `json:"owner_id,omitempty"`
	OwnerIDs  []uuid.UUID  `json:"owner_ids,omitempty"`

	// Status filters
	Status    *string      `json:"status,omitempty"`
	Statuses  []string     `json:"statuses,omitempty"`

	// Type filters
	DerivationType  *string  `json:"derivation_type,omitempty"`
	DerivationTypes []string `json:"derivation_types,omitempty"`
	DocumentType    *string  `json:"document_type,omitempty"`
	DocumentTypes   []string `json:"document_types,omitempty"`

	// Time range filters
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`

	// Pagination
	Limit  *int `json:"limit,omitempty"`
	Offset *int `json:"offset,omitempty"`

	// Sorting
	SortBy    *string `json:"sort_by,omitempty"`    // created_at, updated_at, name
	SortOrder *string `json:"sort_order,omitempty"` // asc, desc

	// Special flags
	IncludeDeleted bool `json:"include_deleted,omitempty"`
}

// StatisticsOptions defines what statistics to compute
type StatisticsOptions struct {
	IncludeStatusBreakdown       bool `json:"include_status_breakdown"`
	IncludeTenantBreakdown       bool `json:"include_tenant_breakdown"`
	IncludeDerivationBreakdown   bool `json:"include_derivation_breakdown"`
	IncludeDocumentTypeBreakdown bool `json:"include_document_type_breakdown"`
	IncludeTimeRange             bool `json:"include_time_range"`
}

// DefaultStatisticsOptions returns statistics options with all breakdowns enabled
func DefaultStatisticsOptions() StatisticsOptions {
	return StatisticsOptions{
		IncludeStatusBreakdown:       true,
		IncludeTenantBreakdown:       true,
		IncludeDerivationBreakdown:   true,
		IncludeDocumentTypeBreakdown: true,
		IncludeTimeRange:             true,
	}
}
