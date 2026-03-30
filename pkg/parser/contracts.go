// Package parser provides HCL parsing functionality for Terraform files.
package parser

import (
	"context"

	"github.com/zclconf/go-cty/cty"
)

// ModuleParser is the interface for parsing Terraform modules.
type ModuleParser interface {
	ParseModule(ctx context.Context, modulePath string) (*ParsedModule, error)
	ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error)
}

// Parser handles parsing of Terraform HCL files.
type Parser struct {
	segments []string
}

// NewParser creates a new HCL parser with the given pattern segments.
func NewParser(segments []string) *Parser {
	if len(segments) == 0 {
		segments = []string{"service", "environment", "region", "module"}
	}
	return &Parser{segments: append([]string{}, segments...)}
}

// Segments returns the parser's configured pattern segments.
func (p *Parser) Segments() []string { return append([]string(nil), p.segments...) }
