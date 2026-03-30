package extract

import (
	"path/filepath"
)

func extractLockFile(ctx *Context) {
	lockPath := filepath.Join(ctx.Sink.Path(), lockFileName)
	file, err := ctx.Index.ParseHCLFile(lockPath)
	if err != nil || file == nil {
		return
	}

	bodyContent, _, diags := file.Body.PartialContent(lockFileSchema())
	ctx.Sink.AddDiags(diags)
	if bodyContent == nil {
		return
	}

	for _, block := range bodyContent.Blocks {
		if len(block.Labels) < 1 {
			continue
		}

		lp := LockedProvider{Source: block.Labels[0]}
		attrContent, _, attrDiags := block.Body.PartialContent(lockProviderAttrSchema())
		ctx.Sink.AddDiags(attrDiags)
		if attrContent == nil {
			continue
		}

		if v, ok := evalContentStringAttr(attrContent, "version"); ok {
			lp.Version = v
		}
		if v, ok := evalContentStringAttr(attrContent, "constraints"); ok {
			lp.Constraints = v
		}

		ctx.Sink.AppendLockedProvider(lp)
	}
}
