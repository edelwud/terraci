package eval

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// lookupFunc implements Terraform's lookup(map, key, default).
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

// lookupKey retrieves a value by key from a map or object.
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
