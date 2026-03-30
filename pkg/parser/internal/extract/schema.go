package extract

import "github.com/hashicorp/hcl/v2"

func lockFileSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "provider", LabelNames: []string{"source"}}},
	}
}

func lockProviderAttrSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "version"},
			{Name: "constraints"},
		},
	}
}
