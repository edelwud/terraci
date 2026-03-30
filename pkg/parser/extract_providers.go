package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const lockFileName = ".terraform.lock.hcl"

func extractLockFile(ctx *extractContext) {
	lockPath := filepath.Join(ctx.parsed.Path, lockFileName)
	file, err := ctx.index.parseHCLFile(lockPath)
	if err != nil || file == nil {
		return
	}

	bodyContent, _, diags := file.Body.PartialContent(lockFileSchema())
	ctx.addDiags(diags)
	if bodyContent == nil {
		return
	}

	for _, block := range bodyContent.Blocks {
		if len(block.Labels) < 1 {
			continue
		}

		lp := &LockedProvider{Source: block.Labels[0]}
		attrContent, _, attrDiags := block.Body.PartialContent(lockProviderAttrSchema())
		ctx.addDiags(attrDiags)
		if attrContent == nil {
			continue
		}

		if v, ok := evalContentStringAttr(attrContent, "version"); ok {
			lp.Version = v
		}
		if v, ok := evalContentStringAttr(attrContent, "constraints"); ok {
			lp.Constraints = v
		}

		ctx.parsed.LockedProviders = append(ctx.parsed.LockedProviders, lp)
	}
}

func extractRequiredProviders(ctx *extractContext) {
	for _, terraformBlock := range ctx.index.terraformBlockViews() {
		requiredProviders, diags := terraformBlock.RequiredProviderBlocks()
		ctx.addDiags(diags)
		for _, rpBlock := range requiredProviders {
			attrs, attrDiags := rpBlock.Body.JustAttributes()
			ctx.addDiags(attrDiags)

			for name, attr := range attrs {
				rp := &RequiredProvider{Name: name}

				if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() && val.Type() == cty.String {
					rp.VersionConstraint = val.AsString()
					ctx.parsed.RequiredProviders = append(ctx.parsed.RequiredProviders, rp)
					continue
				}

				if objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr); isObj {
					fillRequiredProviderFromObject(rp, objExpr)
				}

				ctx.parsed.RequiredProviders = append(ctx.parsed.RequiredProviders, rp)
			}
		}
	}
}

func fillRequiredProviderFromObject(rp *RequiredProvider, objExpr *hclsyntax.ObjectConsExpr) {
	for _, item := range objExpr.Items {
		keyVal, keyDiags := item.KeyExpr.Value(nil)
		if keyDiags.HasErrors() || keyVal.Type() != cty.String {
			continue
		}
		switch keyVal.AsString() {
		case "source":
			if v, ok := evalStringExpr(item.ValueExpr, nil); ok {
				rp.Source = v
			}
		case "version":
			if v, ok := evalStringExpr(item.ValueExpr, nil); ok {
				rp.VersionConstraint = v
			}
		}
	}
}
