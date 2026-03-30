package parser

import "github.com/hashicorp/hcl/v2"

func extractBackendConfig(ctx *extractContext) {
	for _, block := range ctx.index.terraformBlocks() {
		content, _, diags := block.Body.PartialContent(backendSchema())
		ctx.addDiags(diags)
		if content == nil {
			continue
		}

		for _, backendBlock := range content.Blocks {
			if len(backendBlock.Labels) < 1 {
				continue
			}

			cfg := extractBackendAttributes(ctx, backendBlock, ctx.buildEvalContext())

			ctx.parsed.Backend = &BackendConfig{
				Type:   backendBlock.Labels[0],
				Config: cfg,
			}
			return
		}
	}
}

func extractBackendAttributes(ctx *extractContext, block *hcl.Block, evalCtx *hcl.EvalContext) map[string]string {
	cfg := make(map[string]string)
	attrs, diags := block.Body.JustAttributes()
	ctx.addDiags(diags)
	for name, attr := range attrs {
		if val, ok := evalStringExpr(attr.Expr, evalCtx); ok {
			cfg[name] = val
		}
	}
	return cfg
}
