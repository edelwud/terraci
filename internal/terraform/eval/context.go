// Package eval provides HCL evaluation context and Terraform function implementations.
package eval

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// NewContext creates an HCL evaluation context with Terraform functions.
func NewContext(locals, variables map[string]cty.Value, modulePath string) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local": safeObjectVal(locals),
			"var":   safeObjectVal(variables),
			"path": cty.ObjectVal(map[string]cty.Value{
				"module": cty.StringVal(modulePath),
			}),
		},
		Functions: Functions(),
	}
}

// Functions returns Terraform functions for HCL evaluation.
func Functions() map[string]function.Function {
	return map[string]function.Function{
		"lookup": lookupFunc,
	}
}

// safeObjectVal creates a cty.ObjectVal, returning an empty object for nil/empty maps.
func safeObjectVal(m map[string]cty.Value) cty.Value {
	if len(m) == 0 {
		return cty.EmptyObjectVal
	}
	return cty.ObjectVal(m)
}
