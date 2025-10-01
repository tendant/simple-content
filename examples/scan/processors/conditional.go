package processors

import (
	"context"

	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/scan"
)

// ConditionalProcessor conditionally processes contents based on a predicate function.
// If the condition returns true, the wrapped processor is called.
// If the condition returns false, the content is skipped (returns nil).
type ConditionalProcessor struct {
	Condition func(*simplecontent.Content) bool
	Processor scan.ContentProcessor
}

func (p *ConditionalProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
	if p.Condition(content) {
		return p.Processor.Process(ctx, content)
	}
	return nil // Skip
}

// NewConditionalProcessor creates a new ConditionalProcessor.
func NewConditionalProcessor(condition func(*simplecontent.Content) bool, processor scan.ContentProcessor) *ConditionalProcessor {
	return &ConditionalProcessor{
		Condition: condition,
		Processor: processor,
	}
}

// Example condition functions:

// OnlyImages returns true for image content types.
func OnlyImages(content *simplecontent.Content) bool {
	return len(content.DocumentType) >= 6 && content.DocumentType[:6] == "image/"
}

// OnlyVideos returns true for video content types.
func OnlyVideos(content *simplecontent.Content) bool {
	return len(content.DocumentType) >= 6 && content.DocumentType[:6] == "video/"
}

// OnlyStatus returns a condition function that matches a specific status.
func OnlyStatus(status string) func(*simplecontent.Content) bool {
	return func(content *simplecontent.Content) bool {
		return content.Status == status
	}
}

// OnlyOriginals returns true for non-derived content.
func OnlyOriginals(content *simplecontent.Content) bool {
	return content.DerivationType == ""
}

// OnlyDerived returns true for derived content.
func OnlyDerived(content *simplecontent.Content) bool {
	return content.DerivationType != ""
}
