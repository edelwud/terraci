package source

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

var variableDefaultSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{{Name: "default"}},
}

var backendSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{{Type: "backend", LabelNames: []string{"type"}}},
}

var requiredProvidersSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{{Type: "required_providers"}},
}

var moduleCallSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version"},
	},
}

var remoteStateSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "backend", Required: true},
		{Name: "for_each"},
		{Name: "workspace"},
		{Name: "config"},
	},
	Blocks: []hcl.BlockHeaderSchema{{Type: "config"}},
}
