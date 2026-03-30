package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
)

type evalContextBuilder struct {
	inner evalctx.Builder
}

func newEvalContextBuilder(segments []string) evalContextBuilder {
	return evalContextBuilder{inner: evalctx.NewBuilder(segments)}
}

func (p *Parser) evalContextBuilder() evalContextBuilder {
	return newEvalContextBuilder(p.segments)
}

// build creates an HCL evaluation context with path-derived locals.
func (b evalContextBuilder) build(modulePath string, locals, variables map[string]cty.Value) *hcl.EvalContext {
	return b.inner.Build(modulePath, locals, variables)
}

// extractPathLocals derives locals from the module path based on configured segments.
func (b evalContextBuilder) extractPathLocals(pathParts []string) map[string]cty.Value {
	return b.inner.ExtractPathLocals(pathParts)
}
