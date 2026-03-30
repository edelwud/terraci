package parser

import (
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const lockFileName = ".terraform.lock.hcl"

func (p *Parser) extractLockFile(index *moduleIndex, pm *ParsedModule) {
	lockPath := filepath.Join(pm.Path, lockFileName)
	file, err := index.parseHCLFile(lockPath)
	if err != nil || file == nil {
		return
	}

	bodyContent, _, diags := file.Body.PartialContent(lockFileSchema())
	pm.addDiags(diags)
	if bodyContent == nil {
		return
	}

	for _, block := range bodyContent.Blocks {
		if len(block.Labels) < 1 {
			continue
		}

		lp := &LockedProvider{Source: block.Labels[0]}
		attrContent, _, attrDiags := block.Body.PartialContent(lockProviderAttrSchema())
		pm.addDiags(attrDiags)
		if attrContent == nil {
			continue
		}

		if v, ok := evalContentStringAttr(attrContent, "version"); ok {
			lp.Version = v
		}
		if v, ok := evalContentStringAttr(attrContent, "constraints"); ok {
			lp.Constraints = v
		}

		pm.LockedProviders = append(pm.LockedProviders, lp)
	}
}

func (p *Parser) extractRequiredProviders(index *moduleIndex, pm *ParsedModule) {
	for _, block := range index.terraformBlocks() {
		content, _, diags := block.Body.PartialContent(requiredProvidersSchema())
		pm.addDiags(diags)
		if content == nil {
			continue
		}

		for _, rpBlock := range content.Blocks {
			attrs, attrDiags := rpBlock.Body.JustAttributes()
			pm.addDiags(attrDiags)

			for name, attr := range attrs {
				rp := &RequiredProvider{Name: name}

				if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() && val.Type() == cty.String {
					rp.VersionConstraint = val.AsString()
					pm.RequiredProviders = append(pm.RequiredProviders, rp)
					continue
				}

				if objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr); isObj {
					fillRequiredProviderFromObject(rp, objExpr)
				}

				pm.RequiredProviders = append(pm.RequiredProviders, rp)
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
