package source

import "github.com/hashicorp/hcl/v2"

type ModuleBlockView struct {
	block *hcl.Block
}

func (v ModuleBlockView) Name() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v ModuleBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(moduleCallSchema)
	return content, diags
}
