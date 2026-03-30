package parser

import "github.com/hashicorp/hcl/v2"

func backendSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "backend", LabelNames: []string{"type"}}},
	}
}
