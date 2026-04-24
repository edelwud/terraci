package moduleparse

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

func TestRun_BuildsParsedModule(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, dir, "locals.tf", `locals { service = "platform" }`)
	writeModuleFile(t, dir, "vars.tf", `variable "region" { default = "us-east-1" }`)
	writeModuleFile(t, dir, "data.tf", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { key = "platform/stage/eu-central-1/vpc/terraform.tfstate" }
}
`)
	writeModuleFile(t, dir, "module.tf", `module "vpc" { source = "../_modules/vpc" }`)

	parsed, err := Run(context.Background(), dir, []string{"service", "environment", "region", "module"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := parsed.Locals["service"].AsString(); got != "platform" {
		t.Fatalf("local service = %q, want %q", got, "platform")
	}
	if len(parsed.RemoteStates) != 1 {
		t.Fatalf("remote states = %d, want 1", len(parsed.RemoteStates))
	}
	if len(parsed.ModuleCalls) != 1 {
		t.Fatalf("module calls = %d, want 1", len(parsed.ModuleCalls))
	}
	if len(parsed.Files) != 4 {
		t.Fatalf("files = %d, want 4", len(parsed.Files))
	}
}

type fakeSourceLoader struct {
	loaded loadedSource
}

func (l fakeSourceLoader) Load(context.Context, string) (loadedSource, error) {
	return l.loaded, nil
}

func TestRunner_UsesInjectedSourceLoader(t *testing.T) {
	dir := t.TempDir()
	writeModuleFile(t, dir, "locals.tf", `locals { service = "platform" }`)

	loaded, err := source.NewLoader().Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	runner := newRunnerWithLoader("ignored", []string{"service", "environment", "region", "module"}, fakeSourceLoader{loaded: loaded})
	parsed, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := parsed.Path; got != "ignored" {
		t.Fatalf("parsed path = %q, want %q", got, "ignored")
	}
	if got := parsed.Locals["service"].AsString(); got != "platform" {
		t.Fatalf("local service = %q, want %q", got, "platform")
	}
}

type diagnosticSource struct {
	diags hcl.Diagnostics
}

func (s diagnosticSource) LocalsBlocks() []*hcl.Block {
	return nil
}

func (s diagnosticSource) VariableBlockViews() []source.VariableBlockView {
	return nil
}

func (s diagnosticSource) TerraformBlockViews() []source.TerraformBlockView {
	return nil
}

func (s diagnosticSource) RemoteStateBlockViews() []source.RemoteStateBlockView {
	return nil
}

func (s diagnosticSource) ModuleBlockViews() []source.ModuleBlockView {
	return nil
}

func (s diagnosticSource) LockFile() (*hcl.File, hcl.Diagnostics) {
	return nil, nil
}

func (s diagnosticSource) ParseHCLFile(string) (*hcl.File, hcl.Diagnostics, error) {
	return nil, nil, nil
}

func (s diagnosticSource) SharedFiles() map[string]*hcl.File {
	return nil
}

func (s diagnosticSource) SharedDiagnostics() hcl.Diagnostics {
	return s.diags
}

func (s diagnosticSource) SharedTopLevelBlockIndex() map[string][]*hcl.Block {
	return nil
}

func TestRunnerFinalizePreservesExtractionDiagnostics(t *testing.T) {
	runner := newRunner("module", []string{"service", "environment", "region", "module"})
	runner.source = diagnosticSource{
		diags: hcl.Diagnostics{{
			Summary: "source diagnostic",
		}},
	}
	runner.parsed.AddDiags(hcl.Diagnostics{{
		Summary: "extraction diagnostic",
	}})

	runner.finalize()

	if len(runner.parsed.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %d, want 2", len(runner.parsed.Diagnostics))
	}
	if runner.parsed.Diagnostics[0].Summary != "extraction diagnostic" {
		t.Fatalf("first diagnostic = %q, want extraction diagnostic", runner.parsed.Diagnostics[0].Summary)
	}
	if runner.parsed.Diagnostics[1].Summary != "source diagnostic" {
		t.Fatalf("second diagnostic = %q, want source diagnostic", runner.parsed.Diagnostics[1].Summary)
	}
}

func writeModuleFile(t *testing.T, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
