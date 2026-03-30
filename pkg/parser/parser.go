package parser

import (
	"context"
	"fmt"

	moduleparse "github.com/edelwud/terraci/pkg/parser/internal/moduleparse"
)

// ParseModule parses all Terraform files in a module directory.
func (p *Parser) ParseModule(ctx context.Context, modulePath string) (*ParsedModule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	parsed, err := moduleparse.Run(ctx, modulePath, p.segments)
	if err != nil {
		return nil, fmt.Errorf("load module: %w", err)
	}

	return fromInternalParsedModule(parsed), nil
}
