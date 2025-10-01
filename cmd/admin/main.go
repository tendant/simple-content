package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	repopg "github.com/tendant/simple-content/pkg/simplecontent/repo/postgres"
)

const usage = `Simple Content Admin CLI

A lightweight admin tool for content management that only requires database access.

USAGE:
  admin <command> [options]

COMMANDS:
  list      List contents with optional filtering
  count     Count contents with optional filtering
  stats     Get aggregated statistics

ENVIRONMENT VARIABLES:
  DATABASE_URL      PostgreSQL connection string (required for postgres)
  DATABASE_TYPE     Database type: postgres or memory (default: memory)
  DB_SCHEMA         PostgreSQL schema name (default: content)

  Configuration can be loaded from a .env file in the current directory.
  Command line environment variables override .env file values.

EXAMPLES:
  # List all contents
  admin list

  # List contents for a specific tenant
  admin list --tenant-id=550e8400-e29b-41d4-a716-446655440000

  # List with pagination
  admin list --limit=10 --offset=0

  # List by status
  admin list --status=uploaded

  # Count all contents
  admin count

  # Count by tenant
  admin count --tenant-id=550e8400-e29b-41d4-a716-446655440000

  # Get statistics
  admin stats

  # Get statistics for a specific tenant
  admin stats --tenant-id=550e8400-e29b-41d4-a716-446655440000

  # Output as JSON
  admin list --json
  admin stats --json

OPTIONS (for list/count/stats):
  --tenant-id=<uuid>           Filter by tenant ID
  --owner-id=<uuid>            Filter by owner ID
  --status=<status>            Filter by status (created, uploaded, deleted)
  --derivation-type=<type>     Filter by derivation type
  --document-type=<type>       Filter by document type
  --limit=<n>                  Maximum results (list only, default: 100)
  --offset=<n>                 Pagination offset (list only, default: 0)
  --include-deleted            Include deleted content
  --json                       Output as JSON
`

func main() {
	// Load .env file if it exists (silently ignore if not found)
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	command := os.Args[1]

	// Check for help
	if command == "help" || command == "--help" || command == "-h" {
		fmt.Println(usage)
		os.Exit(0)
	}

	// Create admin service
	adminSvc, err := createAdminService()
	if err != nil {
		log.Fatalf("Failed to create admin service: %v", err)
	}

	ctx := context.Background()

	// Parse common flags
	filters, useJSON := parseFilters(os.Args[2:])

	// Execute command
	switch command {
	case "list":
		handleList(ctx, adminSvc, filters, useJSON)
	case "count":
		handleCount(ctx, adminSvc, filters, useJSON)
	case "stats":
		handleStats(ctx, adminSvc, filters, useJSON)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		fmt.Println(usage)
		os.Exit(1)
	}
}

func createAdminService() (admin.AdminService, error) {
	dbType := getEnv("DATABASE_TYPE", "memory")

	switch dbType {
	case "postgres":
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return nil, fmt.Errorf("DATABASE_URL environment variable is required for postgres")
		}

		dbSchema := getEnv("DB_SCHEMA", "content")

		// Connect to postgres
		poolConfig, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse database URL: %w", err)
		}

		// Set search_path
		poolConfig.ConnConfig.RuntimeParams["search_path"] = dbSchema

		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		// Test connection
		if err := pool.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}

		repo := repopg.NewWithPool(pool)
		return admin.New(repo), nil

	case "memory":
		repo := memory.New()
		return admin.New(repo), nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s (use 'postgres' or 'memory')", dbType)
	}
}

func parseFilters(args []string) (admin.ContentFilters, bool) {
	filters := admin.ContentFilters{}
	useJSON := false

	// Default pagination
	defaultLimit := 100
	defaultOffset := 0
	filters.Limit = &defaultLimit
	filters.Offset = &defaultOffset

	for _, arg := range args {
		if arg == "--json" {
			useJSON = true
			continue
		}

		// Parse key=value flags
		key, value := parseFlag(arg)

		switch key {
		case "tenant-id":
			if id, err := uuid.Parse(value); err == nil {
				filters.TenantID = &id
			}
		case "owner-id":
			if id, err := uuid.Parse(value); err == nil {
				filters.OwnerID = &id
			}
		case "status":
			filters.Status = &value
		case "derivation-type":
			filters.DerivationType = &value
		case "document-type":
			filters.DocumentType = &value
		case "limit":
			if n, err := strconv.Atoi(value); err == nil {
				filters.Limit = &n
			}
		case "offset":
			if n, err := strconv.Atoi(value); err == nil {
				filters.Offset = &n
			}
		case "include-deleted":
			filters.IncludeDeleted = true
		}
	}

	return filters, useJSON
}

func parseFlag(arg string) (string, string) {
	if len(arg) > 2 && arg[:2] == "--" {
		arg = arg[2:]
		for i, c := range arg {
			if c == '=' {
				return arg[:i], arg[i+1:]
			}
		}
		return arg, "true"
	}
	return "", ""
}

func handleList(ctx context.Context, adminSvc admin.AdminService, filters admin.ContentFilters, useJSON bool) {
	resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: filters,
	})
	if err != nil {
		log.Fatalf("Failed to list contents: %v", err)
	}

	if useJSON {
		data, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tTENANT\tOWNER\tSTATUS\tTYPE\tCREATED\n")
	fmt.Fprintf(w, "──────────────────────────────────────\t────────────────\t────────────────────────────────────────\t────────────────────────────────────────\t────────\t────────────────\t──────────────────────\n")

	for _, content := range resp.Contents {
		createdAt := content.CreatedAt.Format("2006-01-02 15:04:05")
		docType := content.DocumentType
		if docType == "" {
			docType = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			content.ID.String()[:8]+"...",
			truncate(content.Name, 15),
			content.TenantID.String()[:8]+"...",
			content.OwnerID.String()[:8]+"...",
			content.Status,
			truncate(docType, 15),
			createdAt,
		)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d", len(resp.Contents))
	if resp.HasMore {
		fmt.Printf(" (has more, use --offset=%d to continue)", *filters.Offset + *filters.Limit)
	}
	fmt.Println()
}

func handleCount(ctx context.Context, adminSvc admin.AdminService, filters admin.ContentFilters, useJSON bool) {
	resp, err := adminSvc.CountContents(ctx, admin.CountRequest{
		Filters: filters,
	})
	if err != nil {
		log.Fatalf("Failed to count contents: %v", err)
	}

	if useJSON {
		data, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Total count: %d\n", resp.Count)
}

func handleStats(ctx context.Context, adminSvc admin.AdminService, filters admin.ContentFilters, useJSON bool) {
	resp, err := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
		Filters: filters,
		Options: admin.DefaultStatisticsOptions(),
	})
	if err != nil {
		log.Fatalf("Failed to get statistics: %v", err)
	}

	if useJSON {
		data, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(data))
		return
	}

	stats := resp.Statistics

	fmt.Println("=== Content Statistics ===")
	fmt.Printf("\nTotal Count: %d\n", stats.TotalCount)

	if len(stats.ByStatus) > 0 {
		fmt.Println("\nBy Status:")
		for status, count := range stats.ByStatus {
			fmt.Printf("  %-15s: %d\n", status, count)
		}
	}

	if len(stats.ByTenant) > 0 {
		fmt.Println("\nBy Tenant:")
		for tenant, count := range stats.ByTenant {
			fmt.Printf("  %s: %d\n", tenant[:8]+"...", count)
		}
	}

	if len(stats.ByDerivationType) > 0 {
		fmt.Println("\nBy Derivation Type:")
		for dtype, count := range stats.ByDerivationType {
			fmt.Printf("  %-15s: %d\n", dtype, count)
		}
	}

	if len(stats.ByDocumentType) > 0 {
		fmt.Println("\nBy Document Type:")
		for docType, count := range stats.ByDocumentType {
			fmt.Printf("  %-30s: %d\n", truncate(docType, 30), count)
		}
	}

	if stats.OldestContent != nil && stats.NewestContent != nil {
		fmt.Println("\nTime Range:")
		fmt.Printf("  Oldest: %s\n", stats.OldestContent.Format(time.RFC3339))
		fmt.Printf("  Newest: %s\n", stats.NewestContent.Format(time.RFC3339))
	}

	fmt.Printf("\nComputed at: %s\n", resp.ComputedAt.Format(time.RFC3339))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
