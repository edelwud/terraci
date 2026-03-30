package moduleparse

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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

func writeModuleFile(t *testing.T, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
