package resolve

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
)

var benchResolvedPaths []string

func BenchmarkResolverResolve(b *testing.B) {
	resolver := NewResolver(evalctx.NewBuilder([]string{"service", "environment", "region", "module"}))

	simpleRef := &Ref{
		Name:   "vpc",
		Config: parseBenchExpressionMap(b, map[string]string{"key": `"platform/stage/eu-central-1/vpc/terraform.tfstate"`}),
	}

	forEachRef := &Ref{
		Name: "deps",
		Config: parseBenchExpressionMap(b, map[string]string{
			"key": `"${each.value}/terraform.tfstate"`,
		}),
		ForEach: parseBenchExpression(b, `tomap({
  vpc = "platform/stage/eu-central-1/vpc"
  rds = "platform/stage/eu-central-1/rds"
})`),
	}

	b.Run("simple", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			paths, err := resolver.Resolve(simpleRef, "platform/stage/eu-central-1/eks", nil, nil)
			if err != nil {
				b.Fatalf("Resolve() error = %v", err)
			}
			benchResolvedPaths = paths
		}
	})

	b.Run("foreach", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			paths, err := resolver.Resolve(forEachRef, "platform/stage/eu-central-1/app", nil, nil)
			if err != nil {
				b.Fatalf("Resolve() error = %v", err)
			}
			benchResolvedPaths = paths
		}
	})
}

func parseBenchExpressionMap(tb testing.TB, expressions map[string]string) map[string]hcl.Expression {
	tb.Helper()
	result := make(map[string]hcl.Expression, len(expressions))
	for name, expr := range expressions {
		result[name] = parseBenchExpression(tb, expr)
	}
	return result
}

func parseBenchExpression(tb testing.TB, expr string) hcl.Expression {
	tb.Helper()
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte("value = "+expr), "bench.hcl")
	if diags.HasErrors() {
		tb.Fatalf("parse expr diagnostics: %v", diags)
	}
	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		tb.Fatalf("parse attrs diagnostics: %v", diags)
	}
	return attrs["value"].Expr
}
