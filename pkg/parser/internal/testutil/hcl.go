package testutil

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

func ParseExpression(tb testing.TB, expr string) hcl.Expression {
	tb.Helper()

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte("value = "+expr), "test.hcl")
	if diags.HasErrors() {
		tb.Fatalf("parse expr diagnostics: %v", diags)
	}

	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		tb.Fatalf("parse attrs diagnostics: %v", diags)
	}

	return attrs["value"].Expr
}

func ParseExpressionMap(tb testing.TB, expressions map[string]string) map[string]hcl.Expression {
	tb.Helper()

	result := make(map[string]hcl.Expression, len(expressions))
	for name, expr := range expressions {
		result[name] = ParseExpression(tb, expr)
	}

	return result
}

func ParseBodyContent(tb testing.TB, src string) *hcl.BodyContent {
	tb.Helper()

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(src), "test.hcl")
	if diags.HasErrors() {
		tb.Fatalf("parse body diagnostics: %v", diags)
	}

	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		tb.Fatalf("parse attrs diagnostics: %v", diags)
	}

	return &hcl.BodyContent{Attributes: attrs}
}
