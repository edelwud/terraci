package eval

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// lookupFunc implements Terraform's lookup function
// lookup(map, key, default) - returns map[key] or default if key doesn't exist
var lookupFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "map",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
		{
			Name: "key",
			Type: cty.String,
		},
	},
	VarParam: &function.Parameter{
		Name: "default",
		Type: cty.DynamicPseudoType,
	},
	Type: func(_ []cty.Value) (cty.Type, error) {
		return cty.DynamicPseudoType, nil
	},
	Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
		mapVal := args[0]
		keyVal := args[1]

		if !keyVal.IsKnown() {
			return cty.DynamicVal, nil
		}

		key := keyVal.AsString()

		if mapVal.Type().IsMapType() || mapVal.Type().IsObjectType() {
			if mapVal.Type().IsObjectType() {
				if mapVal.Type().HasAttribute(key) {
					return mapVal.GetAttr(key), nil
				}
			} else {
				if mapVal.HasIndex(cty.StringVal(key)).True() {
					return mapVal.Index(cty.StringVal(key)), nil
				}
			}
		}

		// Return default if provided
		if len(args) > 2 {
			return args[2], nil
		}

		return cty.NilVal, fmt.Errorf("key %q not found in map", key)
	},
})
