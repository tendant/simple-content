package processors

import (
	"context"
	"sync"

	"github.com/tendant/simple-content/pkg/simplecontent"
)

// CounterProcessor counts contents by various attributes.
// Useful for generating statistics during a scan.
type CounterProcessor struct {
	mu sync.Mutex

	// ByStatus counts contents by status
	ByStatus map[string]int64

	// ByDocumentType counts contents by document type
	ByDocumentType map[string]int64

	// ByTenant counts contents by tenant ID
	ByTenant map[string]int64

	// ByDerivationType counts derived contents by derivation type
	ByDerivationType map[string]int64

	// Total is the total count of all contents processed
	Total int64
}

func NewCounterProcessor() *CounterProcessor {
	return &CounterProcessor{
		ByStatus:         make(map[string]int64),
		ByDocumentType:   make(map[string]int64),
		ByTenant:         make(map[string]int64),
		ByDerivationType: make(map[string]int64),
	}
}

func (p *CounterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Total++
	p.ByStatus[content.Status]++
	p.ByDocumentType[content.DocumentType]++
	p.ByTenant[content.TenantID.String()]++

	if content.DerivationType != "" {
		p.ByDerivationType[content.DerivationType]++
	}

	return nil
}

// PrintSummary prints a summary of the counts
func (p *CounterProcessor) PrintSummary() {
	p.mu.Lock()
	defer p.mu.Unlock()

	println("=== Scan Summary ===")
	println("Total:", p.Total)

	println("\nBy Status:")
	for status, count := range p.ByStatus {
		println(" ", status, ":", count)
	}

	println("\nBy Document Type:")
	for docType, count := range p.ByDocumentType {
		println(" ", docType, ":", count)
	}

	if len(p.ByDerivationType) > 0 {
		println("\nBy Derivation Type:")
		for derType, count := range p.ByDerivationType {
			println(" ", derType, ":", count)
		}
	}

	println("\nBy Tenant:")
	for tenant, count := range p.ByTenant {
		println(" ", tenant[:8]+"...", ":", count)
	}
}
