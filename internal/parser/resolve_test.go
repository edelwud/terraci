package parser

import "testing"

func TestResolveWorkspacePath_Simple(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "b", key = "platform/stage/eu-central-1/vpc/terraform.tfstate" }
}
`,
	})

	p := NewParser()
	result, err := p.ParseModule(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	assertPaths(t, paths, "platform/stage/eu-central-1/vpc/terraform.tfstate")
}

func TestResolveWorkspacePath_WithLocals(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"locals.tf": `locals { service = "platform"; environment = "stage"; region = "eu-central-1" }`,
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "b", key = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate" }
}
`,
	})

	p := NewParser()
	result, err := p.ParseModule(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	assertPaths(t, paths, "platform/stage/eu-central-1/vpc/terraform.tfstate")
}

func TestResolveWorkspacePath_WithForEach(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"locals.tf": `locals { deps = { vpc = "platform/stage/eu-central-1/vpc", rds = "platform/stage/eu-central-1/rds" } }`,
		"data.tf": `
data "terraform_remote_state" "deps" {
  for_each = local.deps
  backend  = "s3"
  config   = { bucket = "b", key = "${each.value}/terraform.tfstate" }
}
`,
	})

	p := NewParser()
	result, err := p.ParseModule(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("paths: got %d, want 2: %v", len(paths), paths)
	}

	pathSet := map[string]bool{}
	for _, p := range paths {
		pathSet[p] = true
	}
	for _, want := range []string{
		"platform/stage/eu-central-1/vpc/terraform.tfstate",
		"platform/stage/eu-central-1/rds/terraform.tfstate",
	} {
		if !pathSet[want] {
			t.Errorf("missing path %q", want)
		}
	}
}

func TestResolveWorkspacePath_InfersFromModulePath(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "b", key = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate" }
}
`,
	})

	p := NewParser()
	result, err := p.ParseModule(dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// No explicit locals — should infer from module path segments
	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	assertPaths(t, paths, "platform/stage/eu-central-1/vpc/terraform.tfstate")
}

func assertPaths(t *testing.T, paths []string, want ...string) {
	t.Helper()
	if len(paths) != len(want) {
		t.Fatalf("paths: got %d, want %d: %v", len(paths), len(want), paths)
	}
	for i, w := range want {
		if paths[i] != w {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], w)
		}
	}
}
