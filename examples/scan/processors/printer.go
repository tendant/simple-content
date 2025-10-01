package processors

import (
	"context"
	"fmt"

	"github.com/tendant/simple-content/pkg/simplecontent"
)

// PrinterProcessor prints content information to stdout.
// Useful for testing and debugging scan operations.
type PrinterProcessor struct {
	Verbose bool // If true, print detailed information
}

func (p *PrinterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
	if p.Verbose {
		fmt.Printf("Content ID: %s\n", content.ID)
		fmt.Printf("  Tenant ID: %s\n", content.TenantID)
		fmt.Printf("  Owner ID: %s\n", content.OwnerID)
		fmt.Printf("  Name: %s\n", content.Name)
		fmt.Printf("  Document Type: %s\n", content.DocumentType)
		fmt.Printf("  Status: %s\n", content.Status)
		if content.DerivationType != "" {
			fmt.Printf("  Derivation Type: %s\n", content.DerivationType)
		}
		fmt.Printf("  Created: %s\n", content.CreatedAt)
		fmt.Printf("  Updated: %s\n", content.UpdatedAt)
		fmt.Println()
	} else {
		fmt.Printf("%s: %s (%s, %s)\n", content.ID, content.Name, content.DocumentType, content.Status)
	}
	return nil
}
