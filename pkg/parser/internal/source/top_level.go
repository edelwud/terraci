package source

import "github.com/hashicorp/hcl/v2"

func (i *Index) collectTopLevelBlocks(file *hcl.File) {
	content, _, diags := file.Body.PartialContent(topLevelBlockSchema)
	i.AddDiagnostics(diags)
	if content == nil {
		return
	}

	for _, block := range content.Blocks {
		i.topLevelBlocks[block.Type] = append(i.topLevelBlocks[block.Type], block)
	}
}
