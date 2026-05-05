// Package eval provides HCL evaluation context and Terraform function implementations.
package eval

import (
	"path/filepath"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// NewContext creates an HCL evaluation context with Terraform functions.
// modulePath should be the absolute path to the module directory.
func NewContext(locals, variables map[string]cty.Value, modulePath string) *hcl.EvalContext {
	absPath, err := filepath.Abs(modulePath)
	if err != nil {
		absPath = modulePath
	}

	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local": SafeObjectVal(locals),
			"var":   SafeObjectVal(variables),
			"path": cty.ObjectVal(map[string]cty.Value{
				"module": cty.StringVal(absPath),
				"root":   cty.StringVal(absPath),
			}),
		},
		Functions: Functions(),
	}
}

// builtinFunctionsOnce caches the immutable Terraform-function table. The
// dependency-extraction hot path used to allocate this map per remote_state
// expression — modules with N remote_state blocks did N×alloc; with this
// memoization it's exactly one allocation for the lifetime of the process.
var builtinFunctionsOnce = sync.OnceValue(builtinFunctions)

// Functions returns Terraform functions for HCL evaluation. The returned map
// is shared and must not be mutated by callers.
func Functions() map[string]function.Function {
	return builtinFunctionsOnce()
}

// SafeObjectVal creates a cty.ObjectVal, returning an empty object for nil/empty maps.
func SafeObjectVal(m map[string]cty.Value) cty.Value {
	if len(m) == 0 {
		return cty.EmptyObjectVal
	}
	return cty.ObjectVal(m)
}
