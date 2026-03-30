package parser

import "github.com/hashicorp/hcl/v2"

func remoteStateSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: true},
			{Name: "for_each"},
			{Name: "workspace"},
			{Name: "config"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "config"}},
	}
}
