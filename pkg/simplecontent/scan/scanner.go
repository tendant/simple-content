package scan

import (
	"context"
	"fmt"

	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/admin"
)

// Scanner queries contents and processes them with the provided processor.
type Scanner struct {
	adminSvc admin.AdminService
}

// New creates a new Scanner instance.
func New(adminSvc admin.AdminService) *Scanner {
	return &Scanner{adminSvc: adminSvc}
}

// ScanOptions configures the scan operation.
type ScanOptions struct {
	// Filters specifies which contents to process
	Filters admin.ContentFilters

	// Processor defines the processing logic (required unless DryRun is true)
	Processor ContentProcessor

	// BatchSize controls how many contents to query at once (default: 100)
	BatchSize int

	// DryRun if true, doesn't process contents, just reports what would be processed
	DryRun bool

	// OnProgress is called after each batch is processed (optional)
	OnProgress func(processed, total int64)
}

// ScanResult contains statistics about the scan operation.
type ScanResult struct {
	// TotalFound is the total number of contents found matching the filters
	TotalFound int64

	// TotalProcessed is the number of contents successfully processed
	TotalProcessed int64

	// TotalFailed is the number of contents that failed processing
	TotalFailed int64

	// TotalSkipped is the number of contents skipped (currently unused, reserved for future)
	TotalSkipped int64

	// FailedIDs contains the IDs of contents that failed processing
	FailedIDs []string
}

// Scan queries contents matching the filters and processes each one with the provided processor.
// Processing happens in batches for efficiency. If a content fails processing, the error is
// recorded but scanning continues with the next content.
func (s *Scanner) Scan(ctx context.Context, opts ScanOptions) (*ScanResult, error) {
	result := &ScanResult{}

	// Validate options
	if !opts.DryRun && opts.Processor == nil {
		return result, fmt.Errorf("processor is required when DryRun is false")
	}

	// Set defaults
	if opts.BatchSize == 0 {
		opts.BatchSize = 100
	}

	offset := 0
	for {
		// Query batch of contents
		opts.Filters.Limit = &opts.BatchSize
		opts.Filters.Offset = &offset

		resp, err := s.adminSvc.ListAllContents(ctx, admin.ListContentsRequest{
			Filters: opts.Filters,
		})
		if err != nil {
			return result, fmt.Errorf("failed to list contents: %w", err)
		}

		if len(resp.Contents) == 0 {
			break
		}

		result.TotalFound += int64(len(resp.Contents))

		// Process each content in the batch
		for _, content := range resp.Contents {
			if opts.DryRun {
				fmt.Printf("[DRY-RUN] Would process: %s (tenant=%s, type=%s, status=%s)\n",
					content.ID, content.TenantID, content.DocumentType, content.Status)
				result.TotalProcessed++
				continue
			}

			// Call external processor
			if err := opts.Processor.Process(ctx, content); err != nil {
				result.TotalFailed++
				result.FailedIDs = append(result.FailedIDs, content.ID.String())
				fmt.Printf("[ERROR] Failed to process %s: %v\n", content.ID, err)
				continue
			}

			result.TotalProcessed++
		}

		// Report progress if callback provided
		if opts.OnProgress != nil {
			opts.OnProgress(result.TotalProcessed+result.TotalFailed, result.TotalFound)
		}

		// Check if there are more contents
		if !resp.HasMore {
			break
		}

		offset += opts.BatchSize
	}

	return result, nil
}

// ForEach is a convenience method that processes each content with a callback function.
// This is useful for simple inline processing without creating a separate processor type.
//
// Example:
//
//	scanner.ForEach(ctx, filters, func(ctx context.Context, content *simplecontent.Content) error {
//	    fmt.Printf("Processing %s\n", content.ID)
//	    return doSomething(content)
//	})
func (s *Scanner) ForEach(ctx context.Context, filters admin.ContentFilters, fn func(context.Context, *simplecontent.Content) error) (*ScanResult, error) {
	processor := &funcProcessor{fn: fn}
	return s.Scan(ctx, ScanOptions{
		Filters:   filters,
		Processor: processor,
	})
}

// funcProcessor adapts a function to the ContentProcessor interface.
type funcProcessor struct {
	fn func(context.Context, *simplecontent.Content) error
}

func (p *funcProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
	return p.fn(ctx, content)
}
