package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

func main() {
	// Load .env file if it exists (silently ignore if not found)
	_ = godotenv.Load()

	// Load server configuration
	serverConfig, err := config.Load(config.WithEnv(""))
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Build service
	svc, err := serverConfig.BuildService()
	if err != nil {
		log.Fatalf("Failed to build service: %v", err)
	}

	// Build admin service
	adminSvc, err := buildAdminService(serverConfig)
	if err != nil {
		log.Fatalf("Failed to build admin service: %v", err)
	}

	// Start admin shell
	shell := NewAdminShell(svc, adminSvc)
	shell.Run()
}

func buildAdminService(serverConfig *config.ServerConfig) (admin.AdminService, error) {
	repo, err := serverConfig.BuildRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to build repository: %w", err)
	}
	return admin.New(repo), nil
}

// AdminShell provides an interactive admin interface
type AdminShell struct {
	service  simplecontent.Service
	adminSvc admin.AdminService
}

// NewAdminShell creates a new admin shell
func NewAdminShell(service simplecontent.Service, adminSvc admin.AdminService) *AdminShell {
	return &AdminShell{
		service:  service,
		adminSvc: adminSvc,
	}
}

// Run starts the interactive admin shell
func (s *AdminShell) Run() {
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	fmt.Println("=== Simple Content Admin Shell ===")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		fmt.Print("admin> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "help", "h":
			s.showHelp()
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return
		case "list", "ls":
			s.handleList(ctx, parts[1:])
		case "count":
			s.handleCount(ctx, parts[1:])
		case "stats":
			s.handleStats(ctx, parts[1:])
		case "get":
			s.handleGet(ctx, parts[1:])
		default:
			fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", command)
		}
	}
}

func (s *AdminShell) showHelp() {
	help := `
Available Commands:

  list, ls              List all contents
  list <tenant-id>      List contents for specific tenant

  count                 Count all contents
  count <tenant-id>     Count contents for specific tenant

  stats                 Show overall statistics
  stats <tenant-id>     Show statistics for specific tenant

  get <content-id>      Get details for specific content

  help, h               Show this help message
  exit, quit, q         Exit admin shell

Examples:
  list
  list 550e8400-e29b-41d4-a716-446655440000
  count
  stats
  get abcd1234-5678-90ef-ghij-klmnopqrstuv
`
	fmt.Println(help)
}

func (s *AdminShell) handleList(ctx context.Context, args []string) {
	filters := admin.ContentFilters{}
	limit := 20
	filters.Limit = &limit

	if len(args) > 0 {
		// First arg is tenant ID
		if tenantID, err := uuid.Parse(args[0]); err == nil {
			filters.TenantID = &tenantID
		} else {
			fmt.Printf("Invalid tenant ID: %s\n", args[0])
			return
		}
	}

	resp, err := s.adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: filters,
	})
	if err != nil {
		fmt.Printf("Error listing contents: %v\n", err)
		return
	}

	if len(resp.Contents) == 0 {
		fmt.Println("No contents found")
		return
	}

	fmt.Printf("%-36s  %-20s  %-10s  %-15s\n", "ID", "Name", "Status", "Type")
	fmt.Println(strings.Repeat("-", 90))
	for _, content := range resp.Contents {
		name := content.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		docType := content.DocumentType
		if len(docType) > 15 {
			docType = docType[:12] + "..."
		}
		fmt.Printf("%-36s  %-20s  %-10s  %-15s\n",
			content.ID.String(),
			name,
			content.Status,
			docType,
		)
	}
	fmt.Printf("\nTotal: %d", len(resp.Contents))
	if resp.HasMore {
		fmt.Printf(" (showing first %d, use HTTP API for pagination)", limit)
	}
	fmt.Println()
}

func (s *AdminShell) handleCount(ctx context.Context, args []string) {
	filters := admin.ContentFilters{}

	if len(args) > 0 {
		if tenantID, err := uuid.Parse(args[0]); err == nil {
			filters.TenantID = &tenantID
		} else {
			fmt.Printf("Invalid tenant ID: %s\n", args[0])
			return
		}
	}

	resp, err := s.adminSvc.CountContents(ctx, admin.CountRequest{
		Filters: filters,
	})
	if err != nil {
		fmt.Printf("Error counting contents: %v\n", err)
		return
	}

	fmt.Printf("Total count: %d\n", resp.Count)
}

func (s *AdminShell) handleStats(ctx context.Context, args []string) {
	filters := admin.ContentFilters{}

	if len(args) > 0 {
		if tenantID, err := uuid.Parse(args[0]); err == nil {
			filters.TenantID = &tenantID
		} else {
			fmt.Printf("Invalid tenant ID: %s\n", args[0])
			return
		}
	}

	resp, err := s.adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
		Filters: filters,
		Options: admin.DefaultStatisticsOptions(),
	})
	if err != nil {
		fmt.Printf("Error getting statistics: %v\n", err)
		return
	}

	stats := resp.Statistics
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
			fmt.Printf("  %s: %d\n", tenant, count)
		}
	}

	if len(stats.ByDerivationType) > 0 {
		fmt.Println("\nBy Derivation Type:")
		for dtype, count := range stats.ByDerivationType {
			fmt.Printf("  %-15s: %d\n", dtype, count)
		}
	}
	fmt.Println()
}

func (s *AdminShell) handleGet(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: get <content-id>")
		return
	}

	contentID, err := uuid.Parse(args[0])
	if err != nil {
		fmt.Printf("Invalid content ID: %s\n", args[0])
		return
	}

	content, err := s.service.GetContent(ctx, contentID)
	if err != nil {
		fmt.Printf("Error getting content: %v\n", err)
		return
	}

	// Pretty print as JSON
	data, _ := json.MarshalIndent(content, "", "  ")
	fmt.Println(string(data))
}
