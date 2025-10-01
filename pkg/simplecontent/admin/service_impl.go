package admin

import (
	"context"
	"time"

	"github.com/tendant/simple-content/pkg/simplecontent"
)

// adminService implements the AdminService interface
type adminService struct {
	repo simplecontent.Repository
}

// Ensure adminService implements AdminService
var _ AdminService = (*adminService)(nil)

// ListAllContents returns a paginated list of contents with optional filtering
func (s *adminService) ListAllContents(ctx context.Context, req ListContentsRequest) (*ListContentsResponse, error) {
	// Convert admin filters to repository filters
	repoFilters := s.convertToRepoListFilters(req.Filters)

	// Get contents from repository
	contents, err := s.repo.ListContentWithFilters(ctx, repoFilters)
	if err != nil {
		return nil, err
	}

	// Determine pagination details
	limit := 100 // default
	if repoFilters.Limit != nil {
		limit = *repoFilters.Limit
	}
	offset := 0
	if repoFilters.Offset != nil {
		offset = *repoFilters.Offset
	}

	// Check if there are more results
	hasMore := len(contents) == limit

	response := &ListContentsResponse{
		Contents: contents,
		Limit:    limit,
		Offset:   offset,
		HasMore:  hasMore,
	}

	return response, nil
}

// CountContents returns the count of contents matching the given filters
func (s *adminService) CountContents(ctx context.Context, req CountRequest) (*CountResponse, error) {
	// Convert admin filters to repository filters
	repoFilters := s.convertToRepoCountFilters(req.Filters)

	// Get count from repository
	count, err := s.repo.CountContentWithFilters(ctx, repoFilters)
	if err != nil {
		return nil, err
	}

	response := &CountResponse{
		Count: count,
	}

	return response, nil
}

// GetStatistics returns aggregated statistics about contents
func (s *adminService) GetStatistics(ctx context.Context, req StatisticsRequest) (*StatisticsResponse, error) {
	// Convert admin filters to repository filters
	repoFilters := s.convertToRepoCountFilters(req.Filters)

	// Convert admin options to repository options
	repoOptions := simplecontent.ContentStatisticsOptions{
		IncludeStatusBreakdown:       req.Options.IncludeStatusBreakdown,
		IncludeTenantBreakdown:       req.Options.IncludeTenantBreakdown,
		IncludeDerivationBreakdown:   req.Options.IncludeDerivationBreakdown,
		IncludeDocumentTypeBreakdown: req.Options.IncludeDocumentTypeBreakdown,
		IncludeTimeRange:             req.Options.IncludeTimeRange,
	}

	// Get statistics from repository
	repoStats, err := s.repo.GetContentStatistics(ctx, repoFilters, repoOptions)
	if err != nil {
		return nil, err
	}

	// Convert repository statistics to admin statistics
	stats := ContentStatistics{
		TotalCount:       repoStats.TotalCount,
		ByStatus:         repoStats.ByStatus,
		ByTenant:         repoStats.ByTenant,
		ByDerivationType: repoStats.ByDerivationType,
		ByDocumentType:   repoStats.ByDocumentType,
		OldestContent:    repoStats.OldestContent,
		NewestContent:    repoStats.NewestContent,
	}

	response := &StatisticsResponse{
		Statistics: stats,
		ComputedAt: time.Now(),
	}

	return response, nil
}

// convertToRepoListFilters converts admin ContentFilters to repository ContentListFilters
func (s *adminService) convertToRepoListFilters(filters ContentFilters) simplecontent.ContentListFilters {
	return simplecontent.ContentListFilters{
		TenantID:        filters.TenantID,
		TenantIDs:       filters.TenantIDs,
		OwnerID:         filters.OwnerID,
		OwnerIDs:        filters.OwnerIDs,
		Status:          filters.Status,
		Statuses:        filters.Statuses,
		DerivationType:  filters.DerivationType,
		DerivationTypes: filters.DerivationTypes,
		DocumentType:    filters.DocumentType,
		DocumentTypes:   filters.DocumentTypes,
		CreatedAfter:    filters.CreatedAfter,
		CreatedBefore:   filters.CreatedBefore,
		UpdatedAfter:    filters.UpdatedAfter,
		UpdatedBefore:   filters.UpdatedBefore,
		Limit:           filters.Limit,
		Offset:          filters.Offset,
		SortBy:          filters.SortBy,
		SortOrder:       filters.SortOrder,
		IncludeDeleted:  filters.IncludeDeleted,
	}
}

// convertToRepoCountFilters converts admin ContentFilters to repository ContentCountFilters
func (s *adminService) convertToRepoCountFilters(filters ContentFilters) simplecontent.ContentCountFilters {
	return simplecontent.ContentCountFilters{
		TenantID:        filters.TenantID,
		TenantIDs:       filters.TenantIDs,
		OwnerID:         filters.OwnerID,
		OwnerIDs:        filters.OwnerIDs,
		Status:          filters.Status,
		Statuses:        filters.Statuses,
		DerivationType:  filters.DerivationType,
		DerivationTypes: filters.DerivationTypes,
		DocumentType:    filters.DocumentType,
		DocumentTypes:   filters.DocumentTypes,
		CreatedAfter:    filters.CreatedAfter,
		CreatedBefore:   filters.CreatedBefore,
		UpdatedAfter:    filters.UpdatedAfter,
		UpdatedBefore:   filters.UpdatedBefore,
		IncludeDeleted:  filters.IncludeDeleted,
	}
}
