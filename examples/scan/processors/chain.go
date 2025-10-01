package processors

import (
	"context"

	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/scan"
)

// ChainProcessor chains multiple processors together.
// Each processor is called in sequence. If any processor returns an error,
// the chain stops and the error is returned.
type ChainProcessor struct {
	Processors []scan.ContentProcessor
}

func (p *ChainProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
	for _, processor := range p.Processors {
		if err := processor.Process(ctx, content); err != nil {
			return err
		}
	}
	return nil
}

// NewChainProcessor creates a new ChainProcessor with the given processors.
func NewChainProcessor(processors ...scan.ContentProcessor) *ChainProcessor {
	return &ChainProcessor{Processors: processors}
}
