package parser

import "github.com/hashicorp/hcl/v2"

var topLevelBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "locals"},
		{Type: "variable", LabelNames: []string{"name"}},
		{Type: "terraform"},
		{Type: "data", LabelNames: []string{"type", "name"}},
		{Type: "module", LabelNames: []string{"name"}},
	},
}

func (i *moduleIndex) collectTopLevelBlocks(file *hcl.File) {
	content, _, diags := file.Body.PartialContent(topLevelBlockSchema)
	i.addDiagnostics(diags)
	if content == nil {
		return
	}

	for _, block := range content.Blocks {
		i.topLevelBlocks[block.Type] = append(i.topLevelBlocks[block.Type], block)
	}
}
