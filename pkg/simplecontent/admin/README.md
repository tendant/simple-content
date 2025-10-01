# Admin API

The `admin` package provides administrative operations for the simple-content service that bypass normal owner_id/tenant_id restrictions. These operations are designed for operational, monitoring, and bulk processing use cases.

## ⚠️ Security Warning

**IMPORTANT**: Admin API endpoints expose unrestricted access to content across all tenants and owners. In production deployments:

- Always enable authentication/authorization middleware
- Restrict access to trusted administrators only
- Log all admin operations for audit trails
- Consider deploying admin API on a separate internal-only endpoint

## Features

- **List All Contents**: Paginated listing with flexible filtering
- **Count Contents**: Efficient counting for monitoring and analytics
- **Get Statistics**: Aggregated statistics with breakdowns by status, tenant, type, etc.
- **Flexible Filtering**: Filter by tenant, owner, status, document type, date ranges
- **Pagination Support**: Offset-based pagination with configurable limits

## Usage

### Programmatic Usage

```go
import (
	"context"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
)

// Create admin service
repo := postgres.NewRepository(db)
adminSvc := admin.New(repo)

// List all contents
resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
	Filters: admin.ContentFilters{
		Limit: &limit,
		Offset: &offset,
	},
})

// Count contents by tenant
tenantID := uuid.MustParse("...")
countResp, err := adminSvc.CountContents(ctx, admin.CountRequest{
	Filters: admin.ContentFilters{
		TenantID: &tenantID,
	},
})

// Get statistics
statsResp, err := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
	Filters: admin.ContentFilters{},
	Options: admin.DefaultStatisticsOptions(),
})
```

### HTTP API Usage

Enable admin API in server configuration:

```bash
ENABLE_ADMIN_API=true
```

#### List All Contents

```bash
GET /api/v1/admin/contents?limit=100&offset=0
GET /api/v1/admin/contents?tenant_id=<uuid>
GET /api/v1/admin/contents?status=uploaded
GET /api/v1/admin/contents?derivation_type=thumbnail
GET /api/v1/admin/contents?include_deleted=true
```

Query Parameters:
- `tenant_id` (uuid): Filter by tenant
- `owner_id` (uuid): Filter by owner
- `status` (string): Filter by status (created, uploaded, deleted)
- `derivation_type` (string): Filter by derivation type
- `document_type` (string): Filter by document type
- `limit` (int): Maximum results (default: 100, max: 1000)
- `offset` (int): Pagination offset (default: 0)
- `include_deleted` (boolean): Include deleted content (default: false)

Response:
```json
{
  "contents": [...],
  "total_count": null,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

#### Count Contents

```bash
GET /api/v1/admin/contents/count
GET /api/v1/admin/contents/count?tenant_id=<uuid>
GET /api/v1/admin/contents/count?status=uploaded
```

Query Parameters: Same as list (excluding pagination)

Response:
```json
{
  "count": 12345
}
```

#### Get Statistics

```bash
GET /api/v1/admin/contents/stats
GET /api/v1/admin/contents/stats?tenant_id=<uuid>
GET /api/v1/admin/contents/stats?include_tenant=false
```

Query Parameters:
- Filtering: Same as list/count
- Options:
  - `include_status` (boolean): Include status breakdown (default: true)
  - `include_tenant` (boolean): Include tenant breakdown (default: true)
  - `include_derivation` (boolean): Include derivation type breakdown (default: true)
  - `include_document_type` (boolean): Include document type breakdown (default: true)
  - `include_time_range` (boolean): Include time range (default: true)

Response:
```json
{
  "statistics": {
    "total_count": 12345,
    "by_status": {
      "created": 1000,
      "uploaded": 10000,
      "deleted": 1345
    },
    "by_tenant": {
      "tenant-1-uuid": 8000,
      "tenant-2-uuid": 4345
    },
    "by_derivation_type": {
      "original": 10000,
      "thumbnail": 1500,
      "preview": 845
    },
    "by_document_type": {
      "application/pdf": 5000,
      "image/jpeg": 4000,
      "video/mp4": 3345
    },
    "oldest_content": "2024-01-01T00:00:00Z",
    "newest_content": "2024-12-31T23:59:59Z"
  },
  "computed_at": "2024-12-31T23:59:59Z"
}
```

## Use Cases

### 1. Monitoring Dashboard

```go
// Get overall system statistics
stats, err := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
	Filters: admin.ContentFilters{},
	Options: admin.DefaultStatisticsOptions(),
})

// Display metrics:
// - Total content count
// - Status distribution (created/uploaded/failed)
// - Tenant usage distribution
// - Content type distribution
```

### 2. Tenant Analytics

```go
// Get all content for a specific tenant
tenantID := uuid.MustParse("...")
contents, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
	Filters: admin.ContentFilters{
		TenantID: &tenantID,
	},
})

// Count content by status for reporting
count, err := adminSvc.CountContents(ctx, admin.CountRequest{
	Filters: admin.ContentFilters{
		TenantID: &tenantID,
		Status:   &uploadedStatus,
	},
})
```

### 3. Bulk Processing

```go
// Process all uploaded content in batches
limit := 1000
offset := 0
status := "uploaded"

for {
	resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: admin.ContentFilters{
			Status: &status,
			Limit:  &limit,
			Offset: &offset,
		},
	})

	// Process batch
	for _, content := range resp.Contents {
		processContent(content)
	}

	if !resp.HasMore {
		break
	}
	offset += limit
}
```

### 4. Data Cleanup

```go
// Find and clean up old deleted content
thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
deleted := "deleted"

resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
	Filters: admin.ContentFilters{
		Status:        &deleted,
		UpdatedBefore: &thirtyDaysAgo,
		IncludeDeleted: true,
	},
})

// Permanently delete old soft-deleted content
for _, content := range resp.Contents {
	hardDeleteContent(content.ID)
}
```

## Filtering Options

### ContentFilters

- **Identity Filters**:
  - `TenantID` / `TenantIDs`: Filter by tenant(s)
  - `OwnerID` / `OwnerIDs`: Filter by owner(s)

- **Type Filters**:
  - `Status` / `Statuses`: Filter by status (created, uploaded, deleted)
  - `DerivationType` / `DerivationTypes`: Filter by derivation type
  - `DocumentType` / `DocumentTypes`: Filter by document MIME type

- **Time Range Filters**:
  - `CreatedAfter` / `CreatedBefore`: Filter by creation time
  - `UpdatedAfter` / `UpdatedBefore`: Filter by update time

- **Pagination**:
  - `Limit`: Maximum results per request (default: 100, max: 1000)
  - `Offset`: Pagination offset

- **Sorting**:
  - `SortBy`: Sort field (created_at, updated_at, name, status)
  - `SortOrder`: Sort order (ASC, DESC)

- **Special Flags**:
  - `IncludeDeleted`: Include soft-deleted content

## Functional Options

```go
import "github.com/tendant/simple-content/pkg/simplecontent/admin"

// Using functional options for cleaner code
resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
	Filters: admin.ContentFilters{
		TenantID: admin.WithTenantID(tenantID),
		Status:   admin.WithStatus("uploaded"),
		Limit:    admin.WithPagination(100, 0),
	},
})
```

## Performance Considerations

### Indexes

For optimal performance, ensure the following database indexes exist:

```sql
-- For tenant-based queries
CREATE INDEX idx_content_tenant_id ON content.content(tenant_id);

-- For status queries
CREATE INDEX idx_content_status ON content.content(status);

-- For time-based queries
CREATE INDEX idx_content_created_at ON content.content(created_at);
CREATE INDEX idx_content_updated_at ON content.content(updated_at);

-- Composite index for common admin queries
CREATE INDEX idx_content_tenant_status ON content.content(tenant_id, status);
```

### Query Optimization

- **Use COUNT queries** before large listing operations to determine total size
- **Implement cursor-based pagination** for very large datasets (future enhancement)
- **Limit result sizes**: Default limit is 100, maximum is 1000
- **Filter early**: Apply specific filters to reduce result sets
- **Use statistics API** for aggregations instead of client-side computation

## Security Best Practices

1. **Authentication**: Always require authentication for admin endpoints
2. **Authorization**: Implement role-based access control (RBAC)
3. **Audit Logging**: Log all admin operations with user identity
4. **Rate Limiting**: Prevent abuse with rate limits
5. **IP Whitelisting**: Restrict admin API to internal networks
6. **Separate Deployment**: Consider deploying admin API separately

Example middleware:

```go
func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify admin JWT token
		token := extractToken(r)
		claims, err := verifyAdminToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Log admin operation
		logAdminOperation(claims.UserID, r.Method, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}
```

## Example Application

See `examples/admin-operations/main.go` for a complete working example demonstrating:

- Creating sample data
- Listing all contents
- Filtering by tenant and status
- Counting contents
- Retrieving statistics
- Paginated listing

Run the example:

```bash
go run ./examples/admin-operations/main.go
```

## Future Enhancements

- Cursor-based pagination for better performance with large datasets
- Bulk update operations
- Export to CSV/JSON
- Scheduled reports
- Webhook notifications for admin events
- Advanced filtering with SQL-like expressions
