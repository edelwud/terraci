package parser

import "github.com/hashicorp/hcl/v2"

func extractBackendConfig(ctx *extractContext) {
	for _, terraformBlock := range ctx.index.terraformBlockViews() {
		backends, diags := terraformBlock.BackendBlocks()
		ctx.addDiags(diags)
		for _, backend := range backends {
			cfg := extractBackendAttributes(ctx, backend, ctx.buildEvalContext())

			ctx.parsed.Backend = &BackendConfig{
				Type:   backend.Type(),
				Config: cfg,
			}
			return
		}
	}
}

func extractBackendAttributes(ctx *extractContext, block backendBlockView, evalCtx *hcl.EvalContext) map[string]string {
	cfg := make(map[string]string)
	attrs, diags := block.Attributes()
	ctx.addDiags(diags)
	for name, attr := range attrs {
		if val, ok := evalStringExpr(attr.Expr, evalCtx); ok {
			cfg[name] = val
		}
	}
	return cfg
}
