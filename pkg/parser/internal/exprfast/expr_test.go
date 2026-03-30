package exprfast

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

func TestEvalString_Literal(t *testing.T) {
	got, ok := EvalString(parseExpression(t, `"hello"`), nil)
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

	got, ok := EvalString(parseExpression(t, `"${local.service}/vpc"`), ctx)
	if !ok || got != "platform/vpc" {
		t.Fatalf("EvalString() = %q, %v, want %q, true", got, ok, "platform/vpc")
	}
}

func TestContentStringAttr(t *testing.T) {
	content := parseBodyContent(t, `source = "../modules/vpc"`)

	got, ok := ContentStringAttr(content, "source", nil)
	if !ok || got != "../modules/vpc" {
		t.Fatalf("ContentStringAttr() = %q, %v, want %q, true", got, ok, "../modules/vpc")
	}
}

func parseExpression(t *testing.T, expr string) hcl.Expression {
	t.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte("value = "+expr), "test.hcl")
	if diags.HasErrors() {
		t.Fatalf("parse expr diagnostics: %v", diags)
	}
	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		t.Fatalf("parse attrs diagnostics: %v", diags)
	}
	return attrs["value"].Expr
}

func parseBodyContent(t *testing.T, src string) *hcl.BodyContent {
	t.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(src), "test.hcl")
	if diags.HasErrors() {
		t.Fatalf("parse body diagnostics: %v", diags)
	}
	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		t.Fatalf("parse attrs diagnostics: %v", diags)
	}
	return &hcl.BodyContent{Attributes: attrs}
}
