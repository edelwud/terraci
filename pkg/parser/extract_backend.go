package parser

import "github.com/hashicorp/hcl/v2"

func (p *Parser) extractBackendConfig(index *moduleIndex, pm *ParsedModule) {
	for _, block := range index.terraformBlocks() {
		content, _, diags := block.Body.PartialContent(backendSchema())
		pm.addDiags(diags)
		if content == nil {
			continue
		}

		for _, backendBlock := range content.Blocks {
			if len(backendBlock.Labels) < 1 {
				continue
			}

			evalCtx := p.evalContextBuilder().build(pm.Path, pm.Locals, pm.Variables)
			cfg := extractBackendAttributes(backendBlock, evalCtx, pm)

			pm.Backend = &BackendConfig{
				Type:   backendBlock.Labels[0],
				Config: cfg,
			}
			return
		}
	}
}

func extractBackendAttributes(block *hcl.Block, evalCtx *hcl.EvalContext, pm *ParsedModule) map[string]string {
	cfg := make(map[string]string)
	attrs, diags := block.Body.JustAttributes()
	pm.addDiags(diags)
	for name, attr := range attrs {
		if val, ok := evalStringExpr(attr.Expr, evalCtx); ok {
			cfg[name] = val
		}
	}
	return cfg
}
