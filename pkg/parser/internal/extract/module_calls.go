package extract

import (
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

func extractModuleCalls(ctx *Context) {
	for _, module := range ctx.Source.ModuleBlockViews() {
		call := ModuleCall{Name: module.Name()}
		parseModuleBlock(ctx, module, &call)
		ctx.Sink.AppendModuleCall(call)
	}
}

func parseModuleBlock(ctx *Context, view source.ModuleBlockView, call *ModuleCall) {
	content, diags := view.Content()
	ctx.Sink.AddDiags(diags)
	if content == nil {
		return
	}

	if src, ok := evalContentStringAttr(content, "source"); ok {
		call.Source = src
		if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
			call.IsLocal = true
			call.ResolvedPath = filepath.Clean(filepath.Join(ctx.Sink.Path(), src))
		}
	}

	if ver, ok := evalContentStringAttr(content, "version"); ok {
		call.Version = ver
	}
}
