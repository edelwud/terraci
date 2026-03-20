package eval

import (
	"fmt"
	"path/filepath"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// builtinFunctions returns all Terraform-compatible functions for HCL evaluation.
func builtinFunctions() map[string]function.Function {
	return map[string]function.Function{
		// String functions
		"format":     stdlib.FormatFunc,
		"join":       stdlib.JoinFunc,
		"split":      stdlib.SplitFunc,
		"lower":      stdlib.LowerFunc,
		"upper":      stdlib.UpperFunc,
		"trimprefix": stdlib.TrimPrefixFunc,
		"trimsuffix": stdlib.TrimSuffixFunc,
		"replace":    stdlib.ReplaceFunc,
		"substr":     stdlib.SubstrFunc,
		"trim":       stdlib.TrimFunc,
		"trimspace":  stdlib.TrimSpaceFunc,
		"regex":      stdlib.RegexFunc,

		// Collection functions
		"element":  stdlib.ElementFunc,
		"length":   stdlib.LengthFunc,
		"lookup":   lookupFunc, // custom: handles both maps and objects (stdlib doesn't support objects)
		"concat":   stdlib.ConcatFunc,
		"contains": stdlib.ContainsFunc,
		"keys":     stdlib.KeysFunc,
		"values":   stdlib.ValuesFunc,
		"merge":    stdlib.MergeFunc,
		"flatten":  stdlib.FlattenFunc,
		"distinct": stdlib.DistinctFunc,
		"tolist":   stdlib.MakeToFunc(cty.List(cty.DynamicPseudoType)),
		"toset":    stdlib.MakeToFunc(cty.Set(cty.DynamicPseudoType)),
		"tomap":    stdlib.MakeToFunc(cty.Map(cty.DynamicPseudoType)),

		// Numeric functions
		"max":   stdlib.MaxFunc,
		"min":   stdlib.MinFunc,
		"ceil":  stdlib.CeilFunc,
		"floor": stdlib.FloorFunc,

		// Type conversion
		"tostring": stdlib.MakeToFunc(cty.String),
		"tonumber": stdlib.MakeToFunc(cty.Number),
		"tobool":   stdlib.MakeToFunc(cty.Bool),

		// Filesystem (static evaluation only)
		"abspath": abspathFunc,
	}
}

// --- abspath ---

var abspathFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "path", Type: cty.String},
	},
	Type: func(_ []cty.Value) (cty.Type, error) { return cty.String, nil },
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		p := args[0].AsString()
		abs, err := filepath.Abs(p)
		if err != nil {
			return cty.StringVal(p), nil
		}
		return cty.StringVal(abs), nil
	},
})

// --- lookup ---

var lookupFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "map", Type: cty.DynamicPseudoType, AllowDynamicType: true},
		{Name: "key", Type: cty.String},
	},
	VarParam: &function.Parameter{Name: "default", Type: cty.DynamicPseudoType},
	Type:     func(_ []cty.Value) (cty.Type, error) { return cty.DynamicPseudoType, nil },
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		mapVal, keyVal := args[0], args[1]

		if !keyVal.IsKnown() {
			return cty.DynamicVal, nil
		}

		key := keyVal.AsString()

		if val, ok := lookupKey(mapVal, key); ok {
			return val, nil
		}

		if len(args) > 2 {
			return args[2], nil
		}

		return cty.NilVal, fmt.Errorf("key %q not found in map", key)
	},
})

func lookupKey(mapVal cty.Value, key string) (cty.Value, bool) {
	switch {
	case mapVal.Type().IsObjectType() && mapVal.Type().HasAttribute(key):
		return mapVal.GetAttr(key), true
	case mapVal.Type().IsMapType() && mapVal.HasIndex(cty.StringVal(key)).True():
		return mapVal.Index(cty.StringVal(key)), true
	default:
		return cty.NilVal, false
	}
}
