package eval

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestLookupFunc(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

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

func TestAbspathFunc(t *testing.T) {
	t.Parallel()

	funcs := builtinFunctions()
	fn := funcs["abspath"]

	t.Run("relative path returns absolute", func(t *testing.T) {
		t.Parallel()

		result, err := fn.Call([]cty.Value{cty.StringVal("some/relative/path")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := result.AsString()
		if got == "some/relative/path" {
			t.Errorf("expected absolute path, got relative: %s", got)
		}
		if got == "" || got[0] != '/' {
			t.Errorf("expected path starting with /, got %s", got)
		}
	})

	t.Run("absolute path stays absolute", func(t *testing.T) {
		t.Parallel()

		result, err := fn.Call([]cty.Value{cty.StringVal("/already/absolute")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "/already/absolute" {
			t.Errorf("expected /already/absolute, got %s", result.AsString())
		}
	})
}

func TestBuiltinFunctions_AllRegistered(t *testing.T) {
	t.Parallel()

	funcs := builtinFunctions()

	expected := []string{
		"format", "join", "split", "lower", "upper", "trimprefix", "trimsuffix", "replace",
		"element", "concat", "contains", "keys", "values", "merge", "flatten", "distinct",
		"tostring", "tonumber", "tobool", "max", "min", "ceil", "floor", "length", "lookup", "abspath",
		"substr", "trim", "trimspace", "regex", "tolist", "toset", "tomap",
	}

	for _, name := range expected {
		if _, ok := funcs[name]; !ok {
			t.Errorf("expected function %q to be registered, but it was not found", name)
		}
	}
}

func TestStringFunctions(t *testing.T) {
	t.Parallel()

	funcs := builtinFunctions()

	t.Run("split", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["split"].Call([]cty.Value{cty.StringVal(","), cty.StringVal("a,b,c")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.LengthInt() != 3 {
			t.Errorf("expected 3 elements, got %d", result.LengthInt())
		}
	})

	t.Run("join", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["join"].Call([]cty.Value{
			cty.StringVal("-"),
			cty.ListVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b"), cty.StringVal("c")}),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "a-b-c" {
			t.Errorf("expected 'a-b-c', got %q", result.AsString())
		}
	})

	t.Run("lower", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["lower"].Call([]cty.Value{cty.StringVal("HELLO")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "hello" {
			t.Errorf("expected 'hello', got %q", result.AsString())
		}
	})

	t.Run("upper", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["upper"].Call([]cty.Value{cty.StringVal("hello")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "HELLO" {
			t.Errorf("expected 'HELLO', got %q", result.AsString())
		}
	})

	t.Run("trimprefix", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["trimprefix"].Call([]cty.Value{cty.StringVal("helloworld"), cty.StringVal("hello")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "world" {
			t.Errorf("expected 'world', got %q", result.AsString())
		}
	})

	t.Run("trimsuffix", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["trimsuffix"].Call([]cty.Value{cty.StringVal("helloworld"), cty.StringVal("world")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "hello" {
			t.Errorf("expected 'hello', got %q", result.AsString())
		}
	})

	t.Run("replace", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["replace"].Call([]cty.Value{cty.StringVal("hello world"), cty.StringVal("world"), cty.StringVal("there")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "hello there" {
			t.Errorf("expected 'hello there', got %q", result.AsString())
		}
	})
}

func TestNumericFunctions(t *testing.T) {
	t.Parallel()

	funcs := builtinFunctions()

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["max"].Call([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5), cty.NumberIntVal(3)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.NumberIntVal(5)).False() {
			t.Errorf("expected 5, got %v", result.GoString())
		}
	})

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["min"].Call([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(5), cty.NumberIntVal(3)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.NumberIntVal(1)).False() {
			t.Errorf("expected 1, got %v", result.GoString())
		}
	})

	t.Run("ceil", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["ceil"].Call([]cty.Value{cty.NumberFloatVal(1.2)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.NumberIntVal(2)).False() {
			t.Errorf("expected 2, got %v", result.GoString())
		}
	})

	t.Run("floor", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["floor"].Call([]cty.Value{cty.NumberFloatVal(1.8)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.NumberIntVal(1)).False() {
			t.Errorf("expected 1, got %v", result.GoString())
		}
	})
}

func TestTypeFunctions(t *testing.T) {
	t.Parallel()

	funcs := builtinFunctions()

	t.Run("tostring from number", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["tostring"].Call([]cty.Value{cty.NumberIntVal(42)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AsString() != "42" {
			t.Errorf("expected '42', got %q", result.AsString())
		}
	})

	t.Run("tonumber from string", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["tonumber"].Call([]cty.Value{cty.StringVal("42")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.NumberIntVal(42)).False() {
			t.Errorf("expected 42, got %v", result.GoString())
		}
	})

	t.Run("tobool from string true", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["tobool"].Call([]cty.Value{cty.StringVal("true")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.True).False() {
			t.Errorf("expected true, got %v", result.GoString())
		}
	})

	t.Run("tobool from string false", func(t *testing.T) {
		t.Parallel()

		result, err := funcs["tobool"].Call([]cty.Value{cty.StringVal("false")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Equals(cty.False).False() {
			t.Errorf("expected false, got %v", result.GoString())
		}
	})
}

func ptr(v cty.Value) *cty.Value {
	return &v
}
