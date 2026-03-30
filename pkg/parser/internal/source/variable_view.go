package source

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type VariableBlockView struct {
	block *hcl.Block
}

func (v VariableBlockView) Name() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v VariableBlockView) DefaultValue() (cty.Value, bool, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(variableDefaultSchema)
	if content == nil {
		return cty.NilVal, false, diags
	}

	attr, ok := content.Attributes["default"]
	if !ok {
		return cty.NilVal, false, diags
	}

	val, valDiags := attr.Expr.Value(nil)
	diags = append(diags, valDiags...)
	if valDiags.HasErrors() {
		return cty.NilVal, false, diags
	}

	return val, true, diags
}
