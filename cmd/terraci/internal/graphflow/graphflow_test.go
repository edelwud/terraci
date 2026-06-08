package graphflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

func TestParseFormat(t *testing.T) {
	for _, raw := range []string{"", "dot", "plantuml", "list", "levels"} {
		if _, err := ParseFormat(raw); err != nil {
			t.Fatalf("ParseFormat(%q) error = %v", raw, err)
		}
	}
	if _, err := ParseFormat("csv"); err == nil {
		t.Fatal("ParseFormat(csv) error = nil")
	}
}

func TestRenderListIncludesLibraries(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	library := discovery.TestLibraryModule("_modules/network", "/abs/_modules/network")
	depGraph := graph.BuildFromDependencies([]*discovery.Module{module}, nil)

	output, err := Render(depGraph, []*discovery.Module{library}, FormatList)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(output, "[library_modules]") {
		t.Fatalf("output missing library section:\n%s", output)
	}
	if !strings.Contains(output, "_modules/network") {
		t.Fatalf("output missing library module:\n%s", output)
	}
}

func TestRunStats(t *testing.T) {
	workDir := graphTestProject(t)
	prepared := prepareGraph(t, workDir)

	result, err := Run(context.Background(), NewRuntime(prepared), Request{ShowStats: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Output != "" {
		t.Fatalf("Output = %q, want empty stats output", result.Output)
	}
	if result.Stats == nil {
		t.Fatal("Stats = nil")
	}
	if result.Stats.Stats.TotalModules != 1 {
		t.Fatalf("TotalModules = %d, want 1", result.Stats.Stats.TotalModules)
	}
}

func graphTestProject(tb testing.TB) string {
	tb.Helper()
	workDir := tb.TempDir()
	cfg := config.Default()
	if err := cfg.Save(filepath.Join(workDir, ".terraci.yaml")); err != nil {
		tb.Fatalf("Save config: %v", err)
	}
	moduleDir := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		tb.Fatalf("mkdir module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte("# test\n"), 0o644); err != nil {
		tb.Fatalf("write module: %v", err)
	}
	return workDir
}

func prepareGraph(tb testing.TB, workDir string) *runflow.Prepared {
	tb.Helper()
	prepared, err := runflow.New(runflow.Options{}).Prepare(context.Background(), runflow.Request{
		CommandName: "graphflow-test",
		WorkDir:     workDir,
		Policy:      runflow.CommandPolicy{SkipPreflight: true},
	})
	if err != nil {
		tb.Fatalf("Prepare() error = %v", err)
	}
	return prepared
}
