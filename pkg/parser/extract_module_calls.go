package parser

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

func extractModuleCalls(ctx *extractContext) {
	for _, block := range ctx.index.moduleBlocks() {
		if len(block.Labels) < 1 {
			continue
		}
		call := &ModuleCall{Name: block.Labels[0]}
		parseModuleBlock(ctx, call, block.Body)
		ctx.parsed.ModuleCalls = append(ctx.parsed.ModuleCalls, call)
	}
}

func parseModuleBlock(ctx *extractContext, call *ModuleCall, body hcl.Body) {
	content, _, diags := body.PartialContent(moduleCallSchema())
	ctx.addDiags(diags)
	if content == nil {
		return
	}

	if src, ok := evalContentStringAttr(content, "source"); ok {
		call.Source = src
		if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
			call.IsLocal = true
			call.ResolvedPath = filepath.Clean(filepath.Join(ctx.parsed.Path, src))
		}
	}

	if ver, ok := evalContentStringAttr(content, "version"); ok {
		call.Version = ver
	}
}
