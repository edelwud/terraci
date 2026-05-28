package resourcedef_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

func TestRawAttrsAccessorsAndDefensiveCopies(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"name":    "web",
		"size":    "42.5",
		"count":   float64(3),
		"enabled": "true",
		"tags":    []any{"a", 7, "b"},
		"block":   []any{map[string]any{"value": "first"}},
	}
	attrs := resourcedef.NewRawAttrs(input)
	input["name"] = "mutated"

	if got := attrs.String("name"); got != "web" {
		t.Fatalf("String(name) = %q, want web", got)
	}
	if got := attrs.Float("size"); got != 42.5 {
		t.Fatalf("Float(size) = %v, want 42.5", got)
	}
	if got := attrs.Int("count"); got != 3 {
		t.Fatalf("Int(count) = %d, want 3", got)
	}
	if got := attrs.Bool("enabled"); !got {
		t.Fatal("Bool(enabled) = false, want true")
	}
	if got := attrs.StringSlice("tags"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("StringSlice(tags) = %#v, want [a b]", got)
	}
	if got := attrs.FirstObject("block").String("value"); got != "first" {
		t.Fatalf("FirstObject(block).String(value) = %q, want first", got)
	}

	copied := attrs.Map()
	copied["name"] = "copy-mutated"
	if got := attrs.String("name"); got != "web" {
		t.Fatalf("String(name) after Map mutation = %q, want web", got)
	}
}

func TestDefinitionParseAttrsWrapsParserErrors(t *testing.T) {
	t.Parallel()

	def := resourcedef.Definition{
		Type:     "aws_test",
		Category: resourcedef.CostCategoryFixed,
		Parse: func(resourcedef.RawAttrs) (resourcedef.Attributes, error) {
			return resourcedef.Attributes{}, errParseAttrs
		},
		FixedCost: func(string, resourcedef.Attributes) (float64, float64) {
			return 0, 0
		},
	}

	_, err := def.ParseAttrs(resourcedef.EmptyRawAttrs())
	if err == nil {
		t.Fatal("ParseAttrs() error = nil, want error")
	}
}

var errParseAttrs = &parseAttrsError{}

type parseAttrsError struct{}

func (*parseAttrsError) Error() string { return "bad attrs" }
