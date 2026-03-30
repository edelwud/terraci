package parser

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

func (p *Parser) extractModuleCalls(index *moduleIndex, pm *ParsedModule) {
	for _, block := range index.moduleBlocks() {
		if len(block.Labels) < 1 {
			continue
		}
		call := &ModuleCall{Name: block.Labels[0]}
		parseModuleBlock(call, block.Body, pm)
		pm.ModuleCalls = append(pm.ModuleCalls, call)
	}
}

func parseModuleBlock(call *ModuleCall, body hcl.Body, pm *ParsedModule) {
	content, _, diags := body.PartialContent(moduleCallSchema())
	pm.addDiags(diags)
	if content == nil {
		return
	}

	if src, ok := evalContentStringAttr(content, "source"); ok {
		call.Source = src
		if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
			call.IsLocal = true
			call.ResolvedPath = filepath.Clean(filepath.Join(pm.Path, src))
		}
	}

	if ver, ok := evalContentStringAttr(content, "version"); ok {
		call.Version = ver
	}
}
