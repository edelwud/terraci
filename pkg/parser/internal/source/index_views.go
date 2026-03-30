package source

func (s *Snapshot) VariableBlockViews() []VariableBlockView {
	return s.variableViews
}

func (s *Snapshot) TerraformBlockViews() []TerraformBlockView {
	return s.terraformViews
}

func (s *Snapshot) ModuleBlockViews() []ModuleBlockView {
	return s.moduleViews
}

func (s *Snapshot) RemoteStateBlockViews() []RemoteStateBlockView {
	return s.remoteViews
}
