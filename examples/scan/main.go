package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/examples/scan/processors"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	"github.com/tendant/simple-content/pkg/simplecontent/scan"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Simple Content Library - Scanner Example ===\n")

	// Create in-memory repository
	repo := memory.New()

	// Create service for setting up test data
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
	)
	if err != nil {
		log.Fatal("Failed to create service:", err)
	}

	// Create some test content
	tenantID := uuid.New()
	ownerID := uuid.New()

	fmt.Println("Creating test content...")
	testContents := []struct {
		name     string
		docType  string
		status   string
		derived  bool
		derType  string
	}{
		{"Document 1", "application/pdf", "uploaded", false, ""},
		{"Image 1", "image/jpeg", "uploaded", false, ""},
		{"Image 2", "image/png", "created", false, ""},
		{"Video 1", "video/mp4", "uploaded", false, ""},
		{"Thumbnail 1", "image/jpeg", "uploaded", true, "thumbnail"},
		{"Thumbnail 2", "image/jpeg", "uploaded", true, "thumbnail"},
	}

	for _, tc := range testContents {
		content, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
			OwnerID:      ownerID,
			TenantID:     tenantID,
			Name:         tc.name,
			DocumentType: tc.docType,
		})
		if err != nil {
			log.Printf("Failed to create content %s: %v", tc.name, err)
			continue
		}

		// Update status
		if tc.status != "created" {
			content.Status = tc.status
			_ = svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{
				Content: content,
			})
		}

		// Set derivation type if derived
		if tc.derived {
			content.DerivationType = tc.derType
			_ = svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{
				Content: content,
			})
		}
	}

	fmt.Printf("Created %d test contents\n\n", len(testContents))

	// Create admin service for scanning
	adminSvc := admin.New(repo)

	// Create scanner
	scanner := scan.New(adminSvc)

	// Example 1: Simple print processor
	fmt.Println("=== Example 1: Print all contents ===")
	result1, err := scanner.Scan(ctx, scan.ScanOptions{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
		Processor: &processors.PrinterProcessor{},
		BatchSize: 10,
	})
	if err != nil {
		log.Fatal("Scan failed:", err)
	}
	fmt.Printf("\nScanned %d contents, processed %d\n\n", result1.TotalFound, result1.TotalProcessed)

	// Example 2: Count statistics
	fmt.Println("=== Example 2: Count contents by attributes ===")
	counter := processors.NewCounterProcessor()
	_, err = scanner.Scan(ctx, scan.ScanOptions{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
		Processor: counter,
	})
	if err != nil {
		log.Fatal("Scan failed:", err)
	}
	counter.PrintSummary()
	fmt.Println()

	// Example 3: Conditional processing (only images)
	fmt.Println("=== Example 3: Process only images ===")
	result3, err := scanner.Scan(ctx, scan.ScanOptions{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
		Processor: processors.NewConditionalProcessor(
			processors.OnlyImages,
			&processors.PrinterProcessor{},
		),
	})
	if err != nil {
		log.Fatal("Scan failed:", err)
	}
	fmt.Printf("\nFound %d contents, processed %d images\n\n", result3.TotalFound, result3.TotalProcessed)

	// Example 4: Chain multiple processors
	fmt.Println("=== Example 4: Chain processors (count + print) ===")
	counter2 := processors.NewCounterProcessor()
	_, err = scanner.Scan(ctx, scan.ScanOptions{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
			Status:   stringPtr("uploaded"),
		},
		Processor: processors.NewChainProcessor(
			counter2,
			&processors.PrinterProcessor{},
		),
	})
	if err != nil {
		log.Fatal("Scan failed:", err)
	}
	fmt.Println()
	counter2.PrintSummary()
	fmt.Println()

	// Example 5: ForEach convenience method
	fmt.Println("=== Example 5: ForEach with inline function ===")
	result5, err := scanner.ForEach(ctx, admin.ContentFilters{
		TenantID:       &tenantID,
		DerivationType: stringPtr("thumbnail"),
	}, func(ctx context.Context, content *simplecontent.Content) error {
		fmt.Printf("Processing thumbnail: %s\n", content.ID)
		return nil
	})
	if err != nil {
		log.Fatal("ForEach failed:", err)
	}
	fmt.Printf("\nProcessed %d thumbnails\n\n", result5.TotalProcessed)

	// Example 6: Dry-run mode
	fmt.Println("=== Example 6: Dry-run mode ===")
	result6, err := scanner.Scan(ctx, scan.ScanOptions{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
		DryRun:    true,
		BatchSize: 10,
	})
	if err != nil {
		log.Fatal("Scan failed:", err)
	}
	fmt.Printf("\nDry-run complete: would process %d contents\n\n", result6.TotalProcessed)

	// Example 7: Progress callback
	fmt.Println("=== Example 7: Progress tracking ===")
	result7, err := scanner.Scan(ctx, scan.ScanOptions{
		Filters: admin.ContentFilters{
			TenantID: &tenantID,
		},
		Processor: &processors.PrinterProcessor{Verbose: false},
		BatchSize: 2,
		OnProgress: func(processed, total int64) {
			fmt.Printf("[PROGRESS] Processed %d/%d contents\n", processed, total)
		},
	})
	if err != nil {
		log.Fatal("Scan failed:", err)
	}
	fmt.Printf("\nCompleted: %d processed, %d failed\n", result7.TotalProcessed, result7.TotalFailed)

	fmt.Println("\n=== Example completed successfully! ===")
}

func stringPtr(s string) *string {
	return &s
}
