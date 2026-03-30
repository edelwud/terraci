package dependency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser/internal/model"
)

type benchParser struct {
	parsed *model.ParsedModule
}

func (p benchParser) ParseModule(context.Context, string) (*model.ParsedModule, error) {
	return p.parsed, nil
}

func (p benchParser) ResolveWorkspacePath(ref *model.RemoteStateRef, _ string, _, _ map[string]cty.Value) ([]string, error) {
	if expr, ok := ref.Config["key"]; ok {
		return []string{mustBenchExprString(expr)}, nil
	}
	return nil, nil
}

var benchDependencyResults map[string]*ModuleDependencies

func BenchmarkEngineExtractAllDependencies(b *testing.B) {
	modules := make([]*discovery.Module, 0, 50)
	for i := range 50 {
		module := discovery.TestModule("platform", "stage", "eu-central-1", fmt.Sprintf("app-%02d", i))
		module.Path = b.TempDir()
		module.RelativePath = module.ID()
		modules = append(modules, module)
	}

	parsed := &model.ParsedModule{
		Locals:    map[string]cty.Value{},
		Variables: map[string]cty.Value{},
		RemoteStates: []*model.RemoteStateRef{
			{
				Name:    "shared",
				Backend: "s3",
				Config: map[string]hcl.Expression{
					"key": mustBenchExpression(b, `"platform/stage/eu-central-1/app-00/terraform.tfstate"`),
				},
			},
		},
		ModuleCalls: []*model.ModuleCall{
			{Name: "lib", Source: "../_modules/lib", IsLocal: true, ResolvedPath: "/tmp/lib"},
		},
	}

	engine := NewEngine(benchParser{parsed: parsed}, discovery.NewModuleIndex(modules))

	b.ReportAllocs()
	for b.Loop() {
		results, errs := engine.ExtractAllDependencies(context.Background())
		if len(errs) != 0 {
			b.Fatalf("ExtractAllDependencies() errors = %v", errs)
		}
		benchDependencyResults = results
	}
}

func mustBenchExpression(tb testing.TB, src string) hcl.Expression {
	tb.Helper()

	dir := tb.TempDir()
	path := filepath.Join(dir, "expr.hcl")
	if err := os.WriteFile(path, []byte("value = "+src), 0o600); err != nil {
		tb.Fatalf("write expr fixture: %v", err)
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte("value = "+src), path)
	if diags.HasErrors() {
		tb.Fatalf("parse expr diagnostics: %v", diags)
	}
	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		tb.Fatalf("parse attrs diagnostics: %v", diags)
	}
	return attrs["value"].Expr
}

func mustBenchExprString(expr hcl.Expression) string {
	val, _ := expr.Value(nil)
	return val.AsString()
}
