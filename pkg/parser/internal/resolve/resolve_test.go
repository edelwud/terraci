package resolve

import (
	"testing"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
)

func TestResolver_ResolveSimple(t *testing.T) {
	resolver := NewResolver(evalctx.NewBuilder([]string{"service", "environment", "region", "module"}))
	ref := &Ref{
		Name:   "vpc",
		Config: testutil.ParseExpressionMap(t, map[string]string{"key": `"platform/stage/eu-central-1/vpc/terraform.tfstate"`}),
	}

	paths, err := resolver.Resolve(ref, "platform/stage/eu-central-1/eks", nil, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(paths) != 1 || paths[0] != "platform/stage/eu-central-1/vpc/terraform.tfstate" {
		t.Fatalf("paths = %v, want [platform/stage/eu-central-1/vpc/terraform.tfstate]", paths)
	}
}

func TestResolver_ResolveForEach(t *testing.T) {
	resolver := NewResolver(evalctx.NewBuilder([]string{"service", "environment", "region", "module"}))
	ref := &Ref{
		Name: "deps",
		Config: testutil.ParseExpressionMap(t, map[string]string{
			"key": `"${each.value}/terraform.tfstate"`,
		}),
		ForEach: testutil.ParseExpression(t, `tomap({
  vpc = "platform/stage/eu-central-1/vpc"
  rds = "platform/stage/eu-central-1/rds"
})`),
	}

	paths, err := resolver.Resolve(ref, "platform/stage/eu-central-1/app", nil, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v, want 2 paths", paths)
	}
	got := map[string]bool{}
	for _, path := range paths {
		got[path] = true
	}
	for _, want := range []string{
		"platform/stage/eu-central-1/vpc/terraform.tfstate",
		"platform/stage/eu-central-1/rds/terraform.tfstate",
	} {
		if !got[want] {
			t.Fatalf("missing path %q in %v", want, paths)
		}
	}
}
