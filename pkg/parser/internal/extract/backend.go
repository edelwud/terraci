package extract

import "github.com/edelwud/terraci/pkg/parser/internal/source"

func extractBackendConfig(ctx *Context) {
	for _, terraformBlock := range ctx.Index.TerraformBlockViews() {
		backends, diags := terraformBlock.BackendBlocks()
		ctx.Sink.AddDiags(diags)
		for _, backend := range backends {
			cfg := extractBackendAttributes(ctx, backend)
			ctx.Sink.SetBackend(Backend{
				Type:   backend.Type(),
				Config: cfg,
			})
			return
		}
	}
}

func extractBackendAttributes(ctx *Context, block source.BackendBlockView) map[string]string {
	cfg := make(map[string]string)
	attrs, diags := block.Attributes()
	ctx.Sink.AddDiags(diags)

	evalCtx := ctx.buildEvalContext()
	for name := range attrs {
		if val, ok := evalStringAttr(attrs, name, evalCtx); ok {
			cfg[name] = val
		}
	}

	return cfg
}
