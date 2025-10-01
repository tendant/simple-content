# Using Admin Tools in Your Project

Guide for integrating Simple Content admin functionality into your own applications.

## Quick Start

The admin package is a **library** that you can import and use programmatically in your own projects.

```go
import "github.com/tendant/simple-content/pkg/simplecontent/admin"
```

---

## Option 1: Use the Admin Package as a Library

### Installation

```bash
go get github.com/tendant/simple-content
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
)

func main() {
    // 1. Create your repository
    cfg, err := pgxpool.ParseConfig("postgres://user:pass@localhost/dbname")
    if err != nil {
        log.Fatal(err)
    }

    // Set schema
    cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
        _, err := conn.Exec(ctx, "SET search_path TO content")
        return err
    }

    pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
    if err != nil {
        log.Fatal(err)
    }

    repo := repopg.NewWithPool(pool)

    // 2. Create admin service
    adminSvc := admin.New(repo)

    // 3. Use admin operations
    ctx := context.Background()

    // List all contents
    resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
        Filters: admin.ContentFilters{},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Total contents: %d\n", len(resp.Contents))

    // Count contents
    countResp, err := adminSvc.CountContents(ctx, admin.CountRequest{})
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Count: %d\n", countResp.Count)

    // Get statistics
    statsResp, err := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
        Options: admin.DefaultStatisticsOptions(),
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("By status: %+v\n", statsResp.Statistics.ByStatus)
}
```

### With Functional Options

```go
import "github.com/google/uuid"

func listTenantContent(adminSvc admin.AdminService, tenantID uuid.UUID) {
    filters := admin.ContentFilters{}

    // Use functional options for cleaner code
    filters.TenantID = &tenantID
    limit := 100
    filters.Limit = &limit

    resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
        Filters: filters,
    })

    // Process results...
}
```

---

## Option 2: Build Your Own CLI Tool

Copy and customize the admin CLI for your project:

### 1. Copy the Template

```bash
# Copy the admin CLI to your project
cp -r cmd/admin /path/to/your/project/cmd/content-admin

cd /path/to/your/project/cmd/content-admin
```

### 2. Customize Imports

```go
// Update main.go imports
import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
    // ... your other imports
)
```

### 3. Add Custom Commands

```go
// In your customized admin CLI
func main() {
    // ... existing code ...

    switch command {
    case "list":
        handleList(ctx, adminSvc, filters, useJSON)
    case "count":
        handleCount(ctx, adminSvc, filters, useJSON)
    case "stats":
        handleStats(ctx, adminSvc, filters, useJSON)

    // Add your custom commands
    case "export":
        handleCustomExport(ctx, adminSvc)
    case "validate":
        handleCustomValidation(ctx, adminSvc)
    default:
        fmt.Printf("Unknown command: %s\n", command)
    }
}

func handleCustomExport(ctx context.Context, adminSvc admin.AdminService) {
    // Your custom logic
    resp, _ := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{})

    // Export to CSV, Excel, etc.
    exportToCSV(resp.Contents)
}
```

### 4. Build Your Custom Tool

```bash
go build -o mycontent-admin ./cmd/content-admin
./mycontent-admin list
```

---

## Option 3: Add HTTP Admin Endpoints to Your Server

### Using Chi Router (Recommended)

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
)

func main() {
    // Create admin service
    adminSvc := admin.New(repo)

    // Setup router
    r := chi.NewRouter()

    // Add admin routes
    r.Route("/admin", func(r chi.Router) {
        // Add authentication middleware
        r.Use(requireAdminAuth)

        r.Get("/contents", handleAdminList(adminSvc))
        r.Get("/contents/count", handleAdminCount(adminSvc))
        r.Get("/contents/stats", handleAdminStats(adminSvc))
    })

    http.ListenAndServe(":8080", r)
}

func handleAdminList(adminSvc admin.AdminService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Parse query parameters
        filters := parseFiltersFromQuery(r)

        resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
            Filters: filters,
        })
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(resp)
    }
}

func handleAdminCount(adminSvc admin.AdminService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        filters := parseFiltersFromQuery(r)

        resp, err := adminSvc.CountContents(ctx, admin.CountRequest{
            Filters: filters,
        })
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(resp)
    }
}

func handleAdminStats(adminSvc admin.AdminService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        filters := parseFiltersFromQuery(r)

        resp, err := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
            Filters: filters,
            Options: admin.DefaultStatisticsOptions(),
        })
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(resp)
    }
}

func parseFiltersFromQuery(r *http.Request) admin.ContentFilters {
    filters := admin.ContentFilters{}

    if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
        if id, err := uuid.Parse(tenantID); err == nil {
            filters.TenantID = &id
        }
    }

    if status := r.URL.Query().Get("status"); status != "" {
        filters.Status = &status
    }

    // Add more filter parsing...

    return filters
}

func requireAdminAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Your authentication logic
        token := r.Header.Get("Authorization")
        if !isValidAdminToken(token) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Using Standard Library

```go
package main

import (
    "encoding/json"
    "net/http"

    "github.com/tendant/simple-content/pkg/simplecontent/admin"
)

func main() {
    adminSvc := admin.New(repo)

    http.HandleFunc("/admin/contents", adminAuthMiddleware(adminListHandler(adminSvc)))
    http.HandleFunc("/admin/contents/count", adminAuthMiddleware(adminCountHandler(adminSvc)))
    http.HandleFunc("/admin/contents/stats", adminAuthMiddleware(adminStatsHandler(adminSvc)))

    http.ListenAndServe(":8080", nil)
}

func adminListHandler(adminSvc admin.AdminService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation similar to above
    }
}

func adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Authentication check
        if !isAdmin(r) {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        next(w, r)
    }
}
```

---

## Option 4: Create Custom Admin Dashboard

### Backend API

```go
package admin

import (
    "context"
    "encoding/json"
    "net/http"

    scadmin "github.com/tendant/simple-content/pkg/simplecontent/admin"
)

type DashboardAPI struct {
    adminSvc scadmin.AdminService
}

func NewDashboardAPI(adminSvc scadmin.AdminService) *DashboardAPI {
    return &DashboardAPI{adminSvc: adminSvc}
}

func (api *DashboardAPI) GetDashboardData(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Get overall statistics
    stats, err := api.adminSvc.GetStatistics(ctx, scadmin.StatisticsRequest{
        Options: scadmin.DefaultStatisticsOptions(),
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Get recent contents
    limit := 10
    recent, err := api.adminSvc.ListAllContents(ctx, scadmin.ListContentsRequest{
        Filters: scadmin.ContentFilters{
            Limit: &limit,
        },
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Combine data for dashboard
    dashboard := map[string]interface{}{
        "statistics": stats.Statistics,
        "recent":     recent.Contents,
        "timestamp":  stats.ComputedAt,
    }

    json.NewEncoder(w).Encode(dashboard)
}

func (api *DashboardAPI) GetTenantReport(w http.ResponseWriter, r *http.Request) {
    tenantID, _ := uuid.Parse(r.URL.Query().Get("tenant_id"))
    ctx := r.Context()

    // Get tenant-specific stats
    stats, _ := api.adminSvc.GetStatistics(ctx, scadmin.StatisticsRequest{
        Filters: scadmin.ContentFilters{
            TenantID: &tenantID,
        },
        Options: scadmin.DefaultStatisticsOptions(),
    })

    // Get tenant content count
    count, _ := api.adminSvc.CountContents(ctx, scadmin.CountRequest{
        Filters: scadmin.ContentFilters{
            TenantID: &tenantID,
        },
    })

    report := map[string]interface{}{
        "tenant_id":  tenantID,
        "count":      count.Count,
        "statistics": stats.Statistics,
    }

    json.NewEncoder(w).Encode(report)
}
```

### Frontend (React Example)

```jsx
// AdminDashboard.jsx
import React, { useEffect, useState } from 'react';

function AdminDashboard() {
    const [stats, setStats] = useState(null);

    useEffect(() => {
        fetch('/admin/contents/stats')
            .then(res => res.json())
            .then(data => setStats(data.statistics));
    }, []);

    if (!stats) return <div>Loading...</div>;

    return (
        <div className="dashboard">
            <h1>Content Dashboard</h1>

            <div className="stat-card">
                <h2>Total Contents</h2>
                <p className="big-number">{stats.total_count}</p>
            </div>

            <div className="stat-card">
                <h2>By Status</h2>
                {Object.entries(stats.by_status || {}).map(([status, count]) => (
                    <div key={status}>
                        <span>{status}:</span>
                        <span>{count}</span>
                    </div>
                ))}
            </div>

            <div className="stat-card">
                <h2>By Tenant</h2>
                {Object.entries(stats.by_tenant || {}).map(([tenant, count]) => (
                    <div key={tenant}>
                        <span>{tenant}:</span>
                        <span>{count}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

export default AdminDashboard;
```

---

## Option 5: Use Existing CLI Tools Directly

### Connect to Your Database

If your project uses PostgreSQL with the same schema:

```bash
# Install the CLI tool
go install github.com/tendant/simple-content/cmd/admin@latest

# Or build from source
git clone https://github.com/tendant/simple-content
cd simple-content
go build -o admin ./cmd/admin

# Option 1: Configure with .env file (recommended)
cat > .env << EOF
DATABASE_TYPE=postgres
DATABASE_URL=postgres://user:pass@yourdb.example.com/yourdb
DB_SCHEMA=content
EOF

./admin list
./admin count
./admin stats

# Option 2: Configure with environment variables
export DATABASE_URL="postgres://user:pass@yourdb.example.com/yourdb"
export DATABASE_TYPE=postgres
export DB_SCHEMA=content

./admin list
./admin count
./admin stats
```

---

## Integration Patterns

### Pattern 1: Monitoring Service

```go
package monitoring

import (
    "context"
    "time"

    "github.com/tendant/simple-content/pkg/simplecontent/admin"
)

type Monitor struct {
    adminSvc admin.AdminService
}

func (m *Monitor) CollectMetrics() map[string]interface{} {
    ctx := context.Background()

    stats, _ := m.adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
        Options: admin.DefaultStatisticsOptions(),
    })

    return map[string]interface{}{
        "content.total":          stats.Statistics.TotalCount,
        "content.by_status":      stats.Statistics.ByStatus,
        "content.by_tenant":      stats.Statistics.ByTenant,
        "content.collected_at":   time.Now(),
    }
}

// Export to Prometheus, Datadog, etc.
func (m *Monitor) ExportToPrometheus() {
    metrics := m.CollectMetrics()
    // Push to Prometheus
}
```

### Pattern 2: Scheduled Reports

```go
package reports

import (
    "context"
    "fmt"
    "time"

    "github.com/tendant/simple-content/pkg/simplecontent/admin"
)

type Reporter struct {
    adminSvc admin.AdminService
}

func (r *Reporter) GenerateDailyReport() string {
    ctx := context.Background()

    stats, _ := r.adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
        Options: admin.DefaultStatisticsOptions(),
    })

    report := fmt.Sprintf(`
Daily Content Report - %s
========================================

Total Contents: %d

By Status:
%+v

By Tenant:
%+v
`,
        time.Now().Format("2006-01-02"),
        stats.Statistics.TotalCount,
        stats.Statistics.ByStatus,
        stats.Statistics.ByTenant,
    )

    return report
}

// Send report via email, Slack, etc.
func (r *Reporter) SendReport() {
    report := r.GenerateDailyReport()
    sendEmail("admin@example.com", "Daily Report", report)
}
```

### Pattern 3: Data Validation

```go
package validation

import (
    "context"
    "fmt"

    "github.com/tendant/simple-content/pkg/simplecontent/admin"
)

type Validator struct {
    adminSvc admin.AdminService
}

func (v *Validator) ValidateDataIntegrity() []string {
    ctx := context.Background()
    var errors []string

    // Check for orphaned content
    limit := 1000
    resp, _ := v.adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
        Filters: admin.ContentFilters{
            Limit: &limit,
        },
    })

    for _, content := range resp.Contents {
        // Your validation logic
        if content.Status == "created" && isOlderThan(content.CreatedAt, 24*time.Hour) {
            errors = append(errors,
                fmt.Sprintf("Content %s stuck in created state", content.ID))
        }
    }

    return errors
}
```

---

## Environment-Specific Setup

### Development

```go
// dev_admin.go
// +build dev

package main

import (
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func createAdminService() admin.AdminService {
    repo := memory.New()
    return admin.New(repo)
}
```

### Production

```go
// prod_admin.go
// +build !dev

package main

import (
    "context"
    "os"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
)

func createAdminService() admin.AdminService {
    cfg, _ := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
    cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
        _, err := conn.Exec(ctx, "SET search_path TO content")
        return err
    }
    pool, _ := pgxpool.NewWithConfig(context.Background(), cfg)
    repo := repopg.NewWithPool(pool)
    return admin.New(repo)
}
```

---

## Security Best Practices

### 1. Use Read-Only Database User

```sql
CREATE USER admin_readonly WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE content_db TO admin_readonly;
GRANT USAGE ON SCHEMA content TO admin_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA content TO admin_readonly;
```

### 2. Add Authentication to HTTP Endpoints

```go
func requireAdmin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check JWT token
        token := extractToken(r)
        claims, err := validateToken(token)
        if err != nil || !claims.IsAdmin {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Log admin action
        logAdminAccess(claims.UserID, r.Method, r.URL.Path)

        next.ServeHTTP(w, r)
    })
}
```

### 3. Audit Logging

```go
type AuditLogger struct {
    adminSvc admin.AdminService
    logger   *log.Logger
}

func (a *AuditLogger) ListAllContents(ctx context.Context, req admin.ListContentsRequest) (*admin.ListContentsResponse, error) {
    userID := getUserFromContext(ctx)
    a.logger.Printf("User %s listing contents with filters: %+v", userID, req.Filters)

    resp, err := a.adminSvc.ListAllContents(ctx, req)

    a.logger.Printf("User %s listed %d contents", userID, len(resp.Contents))
    return resp, err
}
```

---

## Complete Example Project

See full working example at:
```
examples/your-project-admin/
├── main.go              # Custom CLI
├── server.go            # HTTP API
├── dashboard/           # Web UI
└── monitoring/          # Metrics collection
```

---

## Troubleshooting

**Import errors:**
```bash
go get -u github.com/tendant/simple-content
go mod tidy
```

**Schema mismatch:**
- Ensure your database schema matches Simple Content's schema
- Run migrations: `goose -dir ./migrations/postgres postgres "$DATABASE_URL" up`

**Connection errors:**
- Verify DATABASE_URL is correct
- Check database user permissions
- Test with: `psql "$DATABASE_URL" -c "SELECT 1"`

---

## Further Reading

- [Admin Package API](../pkg/simplecontent/admin/README.md)
- [Admin Tools Comparison](./ADMIN_TOOLS.md)
- [CLI Tool Guide](../cmd/admin/README.md)
- [Interactive Shell Guide](../cmd/admin-shell/README.md)
