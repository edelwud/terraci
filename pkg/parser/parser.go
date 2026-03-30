package parser

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/parser/internal/extract"
)

// ParseModule parses all Terraform files in a module directory.
func (p *Parser) ParseModule(ctx context.Context, modulePath string) (*ParsedModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	index, err := newModuleLoader().Load(ctx, modulePath)
	if err != nil {
		return nil, fmt.Errorf("load module: %w", err)
	}

	assembler := newModuleAssembler(modulePath, index)
	parsed := assembler.Result()
	extractCtx := newExtractContext(index, parsed, p.evalContextBuilder())
	extract.RunDefault(extractCtx.extractionContext())

	return parsed, nil
}
