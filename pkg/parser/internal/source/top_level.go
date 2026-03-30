package source

import "github.com/hashicorp/hcl/v2"

func (b *indexBuilder) collectTopLevelBlocks(file *hcl.File) {
	content, _, diags := file.Body.PartialContent(topLevelBlockSchema)
	b.AddDiagnostics(diags)
	if content == nil {
		return
	}

	for _, block := range content.Blocks {
		b.snapshot.topLevelBlocks[block.Type] = append(b.snapshot.topLevelBlocks[block.Type], block)
	}
}
