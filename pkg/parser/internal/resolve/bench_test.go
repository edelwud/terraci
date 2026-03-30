package resolve

import (
	"testing"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
)

var benchResolvedPaths []string

func BenchmarkResolverResolve(b *testing.B) {
	resolver := NewResolver(evalctx.NewBuilder([]string{"service", "environment", "region", "module"}))

	simpleRef := &Ref{
		Name:   "vpc",
		Config: testutil.ParseExpressionMap(b, map[string]string{"key": `"platform/stage/eu-central-1/vpc/terraform.tfstate"`}),
	}

	forEachRef := &Ref{
		Name: "deps",
		Config: testutil.ParseExpressionMap(b, map[string]string{
			"key": `"${each.value}/terraform.tfstate"`,
		}),
		ForEach: testutil.ParseExpression(b, `tomap({
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
