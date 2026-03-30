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

	p.extractLocals(index, parsed)
	p.extractTfvars(index, parsed)
	p.extractBackendConfig(index, parsed)
	p.extractRequiredProviders(index, parsed)
	p.extractLockFile(index, parsed)
	p.extractRemoteStates(index, parsed)
	p.extractModuleCalls(index, parsed)

	return parsed, nil
}
