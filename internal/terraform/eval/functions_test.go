package eval

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestLookupFunc(t *testing.T) {
	tests := []struct {
		name     string
		mapVal   cty.Value
		key      string
		defVal   *cty.Value
		expected cty.Value
		wantErr  bool
	}{
		{
			name: "map with existing key",
			mapVal: cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"baz": cty.StringVal("qux"),
			}),
			key:      "foo",
			expected: cty.StringVal("bar"),
		},
		{
			name: "object with existing key",
			mapVal: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"age":  cty.NumberIntVal(42),
			}),
			key:      "name",
			expected: cty.StringVal("test"),
		},
		{
			name: "missing key with default",
			mapVal: cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			key:      "missing",
			defVal:   ptr(cty.StringVal("default")),
			expected: cty.StringVal("default"),
		},
		{
			name: "missing key without default",
			mapVal: cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			key:     "missing",
			wantErr: true,
		},
		{
			name: "object missing key with default",
			mapVal: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			key:      "missing",
			defVal:   ptr(cty.StringVal("default")),
			expected: cty.StringVal("default"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []cty.Value{tt.mapVal, cty.StringVal(tt.key)}
			if tt.defVal != nil {
				args = append(args, *tt.defVal)
			}

			result, err := lookupFunc.Call(args)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Equals(tt.expected).True() {
				t.Errorf("expected %v, got %v", tt.expected.GoString(), result.GoString())
			}
		})
	}
}

func TestLookupFunc_UnknownKey(t *testing.T) {
	mapVal := cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})

	result, err := lookupFunc.Call([]cty.Value{mapVal, cty.UnknownVal(cty.String)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unknown key should return DynamicVal (also unknown)
	if result.IsKnown() {
		t.Errorf("expected unknown value for unknown key, got known: %v", result.GoString())
	}
}

func ptr(v cty.Value) *cty.Value {
	return &v
}
