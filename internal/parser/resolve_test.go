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

// Tests the real-world pattern: locals derived from abspath(path.module) + split + element.
// Creates a realistic directory structure so element(..., length - N) resolves correctly.
func TestResolveWorkspacePath_AbspathSplitElement(t *testing.T) {
	root := t.TempDir()
	moduleDir := createTestModuleDir(t, root, "platform", "stage", "eu-central-1", "eks")

	writeTestFile(t, moduleDir, "locals.tf", `
locals {
  path_arr = split("/", abspath(path.module))

  service     = element(local.path_arr, length(local.path_arr) - 4)
  environment = element(local.path_arr, length(local.path_arr) - 3)
  region      = element(local.path_arr, length(local.path_arr) - 2)
  module      = element(local.path_arr, length(local.path_arr) - 1)
}
`)
	writeTestFile(t, moduleDir, "data.tf", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    key = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
  }
}
`)

	p := NewParser()
	result, err := p.ParseModule(moduleDir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Verify locals were resolved with correct values
	assertLocalStr(t, result, "service", "platform")
	assertLocalStr(t, result, "environment", "stage")
	assertLocalStr(t, result, "region", "eu-central-1")
	assertLocalStr(t, result, "module", "eks")

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], moduleDir, result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	assertPaths(t, paths, "platform/stage/eu-central-1/vpc/terraform.tfstate")
}

// Tests the submodule case: platform/vpn/eu-north-1/proxy/prod with depth-5 abspath pattern.
func TestResolveWorkspacePath_SubmoduleAbspath(t *testing.T) {
	root := t.TempDir()
	moduleDir := createTestModuleDir(t, root, "platform", "vpn", "eu-north-1", "proxy", "prod")

	writeTestFile(t, moduleDir, "locals.tf", `
locals {
  path_arr = split("/", abspath(path.module))

  service     = element(local.path_arr, length(local.path_arr) - 5)
  environment = element(local.path_arr, length(local.path_arr) - 4)
  region      = element(local.path_arr, length(local.path_arr) - 3)
  scope       = element(local.path_arr, length(local.path_arr) - 2)
  module      = element(local.path_arr, length(local.path_arr) - 1)
}
`)
	writeTestFile(t, moduleDir, "data.tf", `
data "terraform_remote_state" "sg" {
  backend = "s3"
  config = {
    key = "${local.service}/${local.environment}/${local.region}/sg/terraform.tfstate"
  }
}
`)

	p := NewParser()
	result, err := p.ParseModule(moduleDir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Verify all locals resolved with correct values
	assertLocalStr(t, result, "service", "platform")
	assertLocalStr(t, result, "environment", "vpn")
	assertLocalStr(t, result, "region", "eu-north-1")
	assertLocalStr(t, result, "scope", "proxy")
	assertLocalStr(t, result, "module", "prod")

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], moduleDir, result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	assertPaths(t, paths, "platform/vpn/eu-north-1/sg/terraform.tfstate")
}

// Tests for_each with abspath-derived locals: dependencies map iterates over modules.
func TestResolveWorkspacePath_AbspathForEach(t *testing.T) {
	root := t.TempDir()
	moduleDir := createTestModuleDir(t, root, "platform", "stage", "eu-central-1", "app")

	writeTestFile(t, moduleDir, "locals.tf", `
locals {
  path_arr = split("/", abspath(path.module))

  service     = element(local.path_arr, length(local.path_arr) - 4)
  environment = element(local.path_arr, length(local.path_arr) - 3)
  region      = element(local.path_arr, length(local.path_arr) - 2)
  module      = element(local.path_arr, length(local.path_arr) - 1)

  dependencies = {
    vpc = "${local.service}/${local.environment}/${local.region}/vpc"
    rds = "${local.service}/${local.environment}/${local.region}/rds"
  }
}
`)
	writeTestFile(t, moduleDir, "data.tf", `
data "terraform_remote_state" "deps" {
  for_each = local.dependencies
  backend  = "s3"
  config = {
    key = "${each.value}/terraform.tfstate"
  }
}
`)

	p := NewParser()
	result, err := p.ParseModule(moduleDir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	assertLocalStr(t, result, "service", "platform")
	assertLocalStr(t, result, "environment", "stage")

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], moduleDir, result.Locals, result.Variables)
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
			t.Errorf("missing path %q, got %v", want, paths)
		}
	}
}

// Tests the complex for_each pattern: ternary + for expression + var from tfvars + lookup(each.value, key).
// This is the pattern from platform/vpn/eu-north-1/fortigate/main.tf.
func TestResolveWorkspacePath_ComplexForEachWithLookup(t *testing.T) {
	root := t.TempDir()
	moduleDir := createTestModuleDir(t, root, "platform", "vpn", "eu-north-1", "fortigate")

	writeTestFile(t, moduleDir, "variables.tf", `
variable "inject_address_groups_from_state" {
  default = true
}

variable "managed_environments" {
  default = []
}
`)
	writeTestFile(t, moduleDir, "terraform.tfvars", `
inject_address_groups_from_state = true

managed_environments = [
  { service = "platform", environment = "infra", region = "eu-central-1" },
  { service = "platform", environment = "stage", region = "eu-central-1" },
  { service = "platform", environment = "vpn",   region = "eu-north-1" },
]
`)
	writeTestFile(t, moduleDir, "data.tf", `
data "terraform_remote_state" "vpc_settings" {
  for_each = var.inject_address_groups_from_state ? { for v in var.managed_environments : "${v.service}-${v.environment}-${v.region}" => v } : {}

  backend = "s3"
  config = {
    key = "${lookup(each.value, "service")}/${lookup(each.value, "environment")}/${lookup(each.value, "region")}/vpc/terraform.tfstate"
  }
}
`)

	p := NewParser()
	result, err := p.ParseModule(moduleDir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("remote states: got %d, want 1", len(result.RemoteStates))
	}

	paths, err := p.ResolveWorkspacePath(result.RemoteStates[0], moduleDir, result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// Should resolve 3 paths from the 3 managed_environments
	if len(paths) != 3 {
		t.Fatalf("paths: got %d, want 3: %v", len(paths), paths)
	}

	pathSet := map[string]bool{}
	for _, p := range paths {
		pathSet[p] = true
	}
	for _, want := range []string{
		"platform/infra/eu-central-1/vpc/terraform.tfstate",
		"platform/stage/eu-central-1/vpc/terraform.tfstate",
		"platform/vpn/eu-north-1/vpc/terraform.tfstate",
	} {
		if !pathSet[want] {
			t.Errorf("missing path %q, got %v", want, paths)
		}
	}
}

func assertLocalStr(t *testing.T, pm *ParsedModule, name, want string) {
	t.Helper()
	val, ok := pm.Locals[name]
	if !ok {
		t.Errorf("local %q not resolved", name)
		return
	}
	got := val.AsString()
	if got != want {
		t.Errorf("local %q = %q, want %q", name, got, want)
	}
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
