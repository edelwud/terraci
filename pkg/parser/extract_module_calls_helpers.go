package parser

import "github.com/hashicorp/hcl/v2"

func moduleCallSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "source", Required: true},
			{Name: "version"},
		},
	}
}
