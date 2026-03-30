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
		b.snapshot.collectView(block)
	}
}

func (s *Snapshot) collectView(block *hcl.Block) {
	switch block.Type {
	case "variable":
		if len(block.Labels) > 0 {
			s.variableViews = append(s.variableViews, VariableBlockView{block: block})
		}
	case "terraform":
		s.terraformViews = append(s.terraformViews, TerraformBlockView{block: block})
	case "module":
		if len(block.Labels) > 0 {
			s.moduleViews = append(s.moduleViews, ModuleBlockView{block: block})
		}
	case "data":
		if len(block.Labels) >= 2 && block.Labels[0] == "terraform_remote_state" {
			s.remoteViews = append(s.remoteViews, RemoteStateBlockView{block: block})
		}
	}
}
