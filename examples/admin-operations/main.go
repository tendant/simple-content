package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
	repomemory "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
	fmt.Println("=== Admin Operations Example ===")

	// Setup: Create repository and service
	repo := repomemory.New()
	storage := memorystorage.New()

	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", storage),
	)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create admin service
	adminSvc := admin.New(repo)

	ctx := context.Background()

	// Create some sample data for testing
	fmt.Println("Creating sample content...")
	createSampleData(ctx, svc)

	// Example 1: List all contents without filters
	fmt.Println("\n--- Example 1: List ALL Contents (no filters) ---")
	listAllContents(ctx, adminSvc)

	// Example 2: List contents with tenant filtering
	fmt.Println("\n--- Example 2: List Contents by Tenant ---")
	listContentsByTenant(ctx, adminSvc)

	// Example 3: List contents with status filtering
	fmt.Println("\n--- Example 3: List Contents by Status ---")
	listContentsByStatus(ctx, adminSvc)

	// Example 4: Count contents
	fmt.Println("\n--- Example 4: Count Contents ---")
	countContents(ctx, adminSvc)

	// Example 5: Get statistics
	fmt.Println("\n--- Example 5: Get Content Statistics ---")
	getStatistics(ctx, adminSvc)

	// Example 6: Pagination
	fmt.Println("\n--- Example 6: Paginated Listing ---")
	paginatedListing(ctx, adminSvc)

	fmt.Println("\n=== Example Complete ===")
}

// createSampleData creates sample content for demonstration
func createSampleData(ctx context.Context, svc simplecontent.Service) {
	tenant1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenant2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	owner1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	owner2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	// Create content for tenant1
	for i := 0; i < 5; i++ {
		_, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
			OwnerID:      owner1,
			TenantID:     tenant1,
			Name:         fmt.Sprintf("Document %d", i+1),
			DocumentType: "application/pdf",
		})
		if err != nil {
			log.Printf("Failed to create content: %v", err)
		}
	}

	// Create content for tenant2
	for i := 0; i < 3; i++ {
		_, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
			OwnerID:      owner2,
			TenantID:     tenant2,
			Name:         fmt.Sprintf("Image %d", i+1),
			DocumentType: "image/jpeg",
		})
		if err != nil {
			log.Printf("Failed to create content: %v", err)
		}
	}

	fmt.Printf("Created 8 sample contents (5 for tenant1, 3 for tenant2)\n")
}

// listAllContents lists all contents without any filters
func listAllContents(ctx context.Context, adminSvc admin.AdminService) {
	resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: admin.ContentFilters{},
	})
	if err != nil {
		log.Fatalf("Failed to list all contents: %v", err)
	}

	fmt.Printf("Total contents found: %d\n", len(resp.Contents))
	for i, content := range resp.Contents {
		fmt.Printf("  %d. %s (Tenant: %s, Status: %s)\n",
			i+1, content.Name, content.TenantID, content.Status)
	}
}

// listContentsByTenant lists contents for a specific tenant
func listContentsByTenant(ctx context.Context, adminSvc admin.AdminService) {
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
	})
	if err != nil {
		log.Fatalf("Failed to list contents by tenant: %v", err)
	}

	fmt.Printf("Contents for tenant %s: %d\n", tenantID, len(resp.Contents))
	for i, content := range resp.Contents {
		fmt.Printf("  %d. %s (Owner: %s)\n",
			i+1, content.Name, content.OwnerID)
	}
}

// listContentsByStatus lists contents with a specific status
func listContentsByStatus(ctx context.Context, adminSvc admin.AdminService) {
	status := string(simplecontent.ContentStatusCreated)

	resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: admin.ContentFilters{
			Status: &status,
		},
	})
	if err != nil {
		log.Fatalf("Failed to list contents by status: %v", err)
	}

	fmt.Printf("Contents with status '%s': %d\n", status, len(resp.Contents))
	for i, content := range resp.Contents {
		fmt.Printf("  %d. %s (Tenant: %s)\n",
			i+1, content.Name, content.TenantID)
	}
}

// countContents demonstrates counting with filters
func countContents(ctx context.Context, adminSvc admin.AdminService) {
	// Count all contents
	allResp, err := adminSvc.CountContents(ctx, admin.CountRequest{
		Filters: admin.ContentFilters{},
	})
	if err != nil {
		log.Fatalf("Failed to count all contents: %v", err)
	}
	fmt.Printf("Total contents: %d\n", allResp.Count)

	// Count by tenant
	tenantID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tenantResp, err := adminSvc.CountContents(ctx, admin.CountRequest{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
	})
	if err != nil {
		log.Fatalf("Failed to count contents by tenant: %v", err)
	}
	fmt.Printf("Contents for tenant %s: %d\n", tenantID, tenantResp.Count)
}

// getStatistics demonstrates retrieving aggregated statistics
func getStatistics(ctx context.Context, adminSvc admin.AdminService) {
	resp, err := adminSvc.GetStatistics(ctx, admin.StatisticsRequest{
		Filters: admin.ContentFilters{},
		Options: admin.DefaultStatisticsOptions(),
	})
	if err != nil {
		log.Fatalf("Failed to get statistics: %v", err)
	}

	stats := resp.Statistics
	fmt.Printf("Total Count: %d\n", stats.TotalCount)

	if len(stats.ByStatus) > 0 {
		fmt.Println("\nBy Status:")
		for status, count := range stats.ByStatus {
			fmt.Printf("  %s: %d\n", status, count)
		}
	}

	if len(stats.ByTenant) > 0 {
		fmt.Println("\nBy Tenant:")
		for tenant, count := range stats.ByTenant {
			fmt.Printf("  %s: %d\n", tenant, count)
		}
	}

	if len(stats.ByDocumentType) > 0 {
		fmt.Println("\nBy Document Type:")
		for docType, count := range stats.ByDocumentType {
			fmt.Printf("  %s: %d\n", docType, count)
		}
	}

	if stats.OldestContent != nil && stats.NewestContent != nil {
		fmt.Printf("\nTime Range:\n")
		fmt.Printf("  Oldest: %s\n", stats.OldestContent.Format(time.RFC3339))
		fmt.Printf("  Newest: %s\n", stats.NewestContent.Format(time.RFC3339))
	}

	fmt.Printf("\nStatistics computed at: %s\n", resp.ComputedAt.Format(time.RFC3339))
}

// paginatedListing demonstrates pagination
func paginatedListing(ctx context.Context, adminSvc admin.AdminService) {
	limit := 3
	offset := 0

	fmt.Println("Fetching page 1 (limit=3, offset=0):")
	resp, err := adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
		Filters: admin.ContentFilters{
			Limit:  &limit,
			Offset: &offset,
		},
	})
	if err != nil {
		log.Fatalf("Failed to get page 1: %v", err)
	}

	for i, content := range resp.Contents {
		fmt.Printf("  %d. %s\n", i+1, content.Name)
	}
	fmt.Printf("Has more: %v\n", resp.HasMore)

	if resp.HasMore {
		offset = 3
		fmt.Println("\nFetching page 2 (limit=3, offset=3):")
		resp, err = adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
			Filters: admin.ContentFilters{
				Limit:  &limit,
				Offset: &offset,
			},
		})
		if err != nil {
			log.Fatalf("Failed to get page 2: %v", err)
		}

		for i, content := range resp.Contents {
			fmt.Printf("  %d. %s\n", i+1, content.Name)
		}
		fmt.Printf("Has more: %v\n", resp.HasMore)
	}
}
