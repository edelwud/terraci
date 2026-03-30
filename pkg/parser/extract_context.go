package parser

import "github.com/hashicorp/hcl/v2"

type moduleExtractor func(*extractContext)

type extractContext struct {
	index       *moduleIndex
	parsed      *ParsedModule
	evalBuilder evalContextBuilder
}

func newExtractContext(index *moduleIndex, parsed *ParsedModule, evalBuilder evalContextBuilder) *extractContext {
	return &extractContext{
		index:       index,
		parsed:      parsed,
		evalBuilder: evalBuilder,
	}
}

func (c *extractContext) addDiags(diags hcl.Diagnostics) {
	c.parsed.addDiags(diags)
}

func (c *extractContext) buildEvalContext() *hcl.EvalContext {
	return c.evalBuilder.build(c.parsed.Path, c.parsed.Locals, c.parsed.Variables)
}
