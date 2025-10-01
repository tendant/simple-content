package scan

import (
	"context"

	"github.com/tendant/simple-content/pkg/simplecontent"
)

// ContentProcessor processes individual content items.
// External apps implement this to define custom processing logic.
//
// Example implementations:
//   - Event emitter (sends content events to message queue)
//   - Job creator (creates simple-process jobs for backfill)
//   - Status updater (updates content status in bulk)
//   - Validator (validates content integrity)
//   - Reporter (generates reports/exports)
type ContentProcessor interface {
	// Process is called for each content found during scan.
	// Return error to mark this content as failed (scan continues with next content).
	Process(ctx context.Context, content *simplecontent.Content) error
}
