package admin

import (
	"context"

	"github.com/tendant/simple-content/pkg/simplecontent"
)

// AdminService defines the interface for administrative content operations.
// These operations bypass normal owner_id/tenant_id restrictions and are
// intended for operational, monitoring, and bulk processing use cases.
//
// IMPORTANT: Endpoints using this service should be protected with appropriate
// authentication and authorization middleware to ensure only authorized
// administrators can access these operations.
type AdminService interface {
	// ListAllContents returns a paginated list of contents with optional filtering.
	// Unlike the regular ListContent operation, this does not require owner_id or tenant_id.
	ListAllContents(ctx context.Context, req ListContentsRequest) (*ListContentsResponse, error)

	// CountContents returns the count of contents matching the given filters.
	// This is useful for pagination and monitoring purposes.
	CountContents(ctx context.Context, req CountRequest) (*CountResponse, error)

	// GetStatistics returns aggregated statistics about contents.
	// This provides breakdown by status, tenant, derivation type, etc.
	GetStatistics(ctx context.Context, req StatisticsRequest) (*StatisticsResponse, error)
}

// New creates a new AdminService instance that uses the provided repository.
func New(repo simplecontent.Repository) AdminService {
	return &adminService{
		repo: repo,
	}
}
