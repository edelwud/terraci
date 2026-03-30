package resolve

import "github.com/hashicorp/hcl/v2"

func findPathExpression(ref Ref) hcl.Expression {
	if expr, ok := ref.Config["key"]; ok {
		return expr
	}
	if expr, ok := ref.Config["prefix"]; ok {
		return expr
	}
	return nil
}
