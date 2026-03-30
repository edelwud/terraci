package source

func (i *Index) VariableBlockViews() []VariableBlockView {
	blocks := i.VariableBlocks()
	views := make([]VariableBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, VariableBlockView{block: block})
	}
	return views
}

func (i *Index) TerraformBlockViews() []TerraformBlockView {
	blocks := i.TerraformBlocks()
	views := make([]TerraformBlockView, 0, len(blocks))
	for _, block := range blocks {
		views = append(views, TerraformBlockView{block: block})
	}
	return views
}

func (i *Index) ModuleBlockViews() []ModuleBlockView {
	blocks := i.ModuleBlocks()
	views := make([]ModuleBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, ModuleBlockView{block: block})
	}
	return views
}

func (i *Index) RemoteStateBlockViews() []RemoteStateBlockView {
	blocks := i.DataBlocks()
	views := make([]RemoteStateBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) < 2 || block.Labels[0] != "terraform_remote_state" {
			continue
		}
		views = append(views, RemoteStateBlockView{block: block})
	}
	return views
}
