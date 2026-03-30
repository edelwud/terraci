package parser

import (
	"context"
	"fmt"
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

	for _, extractor := range p.extractors() {
		extractor(extractCtx)
	}

	return parsed, nil
}

func (p *Parser) extractors() []moduleExtractor {
	return []moduleExtractor{
		extractLocals,
		extractTfvars,
		extractBackendConfig,
		extractRequiredProviders,
		extractLockFile,
		extractRemoteStates,
		extractModuleCalls,
	}
}
