package source

import "github.com/hashicorp/hcl/v2"

type BackendBlockView struct {
	block *hcl.Block
}

func (v BackendBlockView) Type() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v BackendBlockView) Attributes() (map[string]*hcl.Attribute, hcl.Diagnostics) {
	return v.block.Body.JustAttributes()
}

type TerraformBlockView struct {
	block *hcl.Block
}

func (v TerraformBlockView) BackendBlocks() ([]BackendBlockView, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(backendSchema)
	if content == nil {
		return nil, diags
	}

	views := make([]BackendBlockView, 0, len(content.Blocks))
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, BackendBlockView{block: block})
	}
	return views, diags
}

func (v TerraformBlockView) RequiredProviderBlocks() ([]*hcl.Block, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(requiredProvidersSchema)
	if content == nil {
		return nil, diags
	}
	return content.Blocks, diags
}
