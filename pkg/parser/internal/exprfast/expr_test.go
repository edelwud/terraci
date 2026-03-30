package exprfast

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
)

func TestEvalString_Literal(t *testing.T) {
	got, ok := EvalString(testutil.ParseExpression(t, `"hello"`), nil)
	if !ok || got != "hello" {
		t.Fatalf("EvalString() = %q, %v, want %q, true", got, ok, "hello")
	}
}

func TestEvalString_Template(t *testing.T) {
	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local": cty.ObjectVal(map[string]cty.Value{
				"service": cty.StringVal("platform"),
			}),
		},
	}

	got, ok := EvalString(testutil.ParseExpression(t, `"${local.service}/vpc"`), ctx)
	if !ok || got != "platform/vpc" {
		t.Fatalf("EvalString() = %q, %v, want %q, true", got, ok, "platform/vpc")
	}
}

func TestContentStringAttr(t *testing.T) {
	content := testutil.ParseBodyContent(t, `source = "../modules/vpc"`)

	got, ok := ContentStringAttr(content, "source", nil)
	if !ok || got != "../modules/vpc" {
		t.Fatalf("ContentStringAttr() = %q, %v, want %q, true", got, ok, "../modules/vpc")
	}
}
