package parser

import (
	"path/filepath"
	"strings"
)

func extractModuleCalls(ctx *extractContext) {
	for _, module := range ctx.index.moduleBlockViews() {
		call := &ModuleCall{Name: module.Name()}
		parseModuleBlock(ctx, module, call)
		ctx.parsed.ModuleCalls = append(ctx.parsed.ModuleCalls, call)
	}
}

func parseModuleBlock(ctx *extractContext, view moduleBlockView, call *ModuleCall) {
	content, diags := view.Content()
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
