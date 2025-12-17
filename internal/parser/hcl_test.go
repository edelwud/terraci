package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestParseModule_EmptyDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "parser-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Locals) != 0 {
		t.Errorf("expected 0 locals, got %d", len(result.Locals))
	}
	if len(result.Variables) != 0 {
		t.Errorf("expected 0 variables, got %d", len(result.Variables))
	}
	if len(result.RemoteStates) != 0 {
		t.Errorf("expected 0 remote states, got %d", len(result.RemoteStates))
	}
	if len(result.ModuleCalls) != 0 {
		t.Errorf("expected 0 module calls, got %d", len(result.ModuleCalls))
	}
}

func TestParseModule_WithLocals(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals {
  service     = "platform"
  environment = "stage"
  region      = "eu-central-1"
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Locals) != 3 {
		t.Errorf("expected 3 locals, got %d", len(result.Locals))
	}

	expectedLocals := map[string]string{
		"service":     "platform",
		"environment": "stage",
		"region":      "eu-central-1",
	}

	for name, expected := range expectedLocals {
		val, ok := result.Locals[name]
		if !ok {
			t.Errorf("missing local %q", name)
			continue
		}
		if val.Type() != cty.String {
			t.Errorf("local %q: expected string type, got %s", name, val.Type().FriendlyName())
			continue
		}
		if val.AsString() != expected {
			t.Errorf("local %q: expected %q, got %q", name, expected, val.AsString())
		}
	}
}

func TestParseModule_WithVariables(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"variables.tf": `
variable "aws_region" {
  default = "eu-central-1"
}

variable "project_name" {
  default = "my-project"
}

variable "no_default" {
  description = "Variable without default"
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 variables (ones with defaults)
	if len(result.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(result.Variables))
	}

	if val, ok := result.Variables["aws_region"]; ok {
		if val.AsString() != "eu-central-1" {
			t.Errorf("aws_region: expected %q, got %q", "eu-central-1", val.AsString())
		}
	} else {
		t.Error("missing variable aws_region")
	}
}

func TestParseModule_WithTfvars(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"variables.tf": `
variable "region" {
  default = "us-east-1"
}
variable "env" {}
`,
		"terraform.tfvars": `
region = "eu-west-1"
env    = "production"
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// terraform.tfvars should override defaults
	if val, ok := result.Variables["region"]; ok {
		if val.AsString() != "eu-west-1" {
			t.Errorf("region: expected %q (from tfvars), got %q", "eu-west-1", val.AsString())
		}
	} else {
		t.Error("missing variable region")
	}

	// env from tfvars
	if val, ok := result.Variables["env"]; ok {
		if val.AsString() != "production" {
			t.Errorf("env: expected %q, got %q", "production", val.AsString())
		}
	} else {
		t.Error("missing variable env")
	}
}

func TestParseModule_WithAutoTfvars(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"variables.tf": `
variable "region" {
  default = "us-east-1"
}
`,
		"terraform.tfvars": `
region = "eu-west-1"
`,
		"override.auto.tfvars": `
region = "ap-northeast-1"
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// auto.tfvars should have highest priority
	if val, ok := result.Variables["region"]; ok {
		if val.AsString() != "ap-northeast-1" {
			t.Errorf("region: expected %q (from auto.tfvars), got %q", "ap-northeast-1", val.AsString())
		}
	} else {
		t.Error("missing variable region")
	}
}

func TestParseModule_WithRemoteState(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"

  config = {
    bucket = "my-terraform-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}

data "terraform_remote_state" "eks" {
  backend = "s3"

  config = {
    bucket = "my-terraform-state"
    key    = "platform/stage/eu-central-1/eks/terraform.tfstate"
    region = "eu-central-1"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 2 {
		t.Fatalf("expected 2 remote states, got %d", len(result.RemoteStates))
	}

	// Check that both remote states were parsed
	names := make(map[string]bool)
	for _, rs := range result.RemoteStates {
		names[rs.Name] = true
		if rs.Backend != "s3" {
			t.Errorf("remote state %s: expected backend s3, got %s", rs.Name, rs.Backend)
		}
	}

	if !names["vpc"] {
		t.Error("missing remote state 'vpc'")
	}
	if !names["eks"] {
		t.Error("missing remote state 'eks'")
	}
}

func TestParseModule_WithRemoteStateForEach(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals {
  dependencies = {
    vpc = "platform/stage/eu-central-1/vpc"
    rds = "platform/stage/eu-central-1/rds"
  }
}
`,
		"data.tf": `
data "terraform_remote_state" "deps" {
  for_each = local.dependencies
  backend  = "s3"

  config = {
    bucket = "my-terraform-state"
    key    = "${each.value}/terraform.tfstate"
    region = "eu-central-1"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("expected 1 remote state, got %d", len(result.RemoteStates))
	}

	rs := result.RemoteStates[0]
	if rs.Name != "deps" {
		t.Errorf("expected name 'deps', got %q", rs.Name)
	}
	if rs.ForEach == nil {
		t.Error("expected for_each expression to be set")
	}
}

func TestParseModule_WithModuleCalls(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}

module "local_module" {
  source = "../../../_modules/kafka"
}

module "relative_module" {
  source = "./submodules/networking"
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ModuleCalls) != 3 {
		t.Fatalf("expected 3 module calls, got %d", len(result.ModuleCalls))
	}

	// Build map for easier testing
	modules := make(map[string]*ModuleCall)
	for _, mc := range result.ModuleCalls {
		modules[mc.Name] = mc
	}

	// Test registry module
	if vpc, ok := modules["vpc"]; ok {
		if vpc.Source != "terraform-aws-modules/vpc/aws" {
			t.Errorf("vpc source: expected %q, got %q", "terraform-aws-modules/vpc/aws", vpc.Source)
		}
		if vpc.Version != "5.0.0" {
			t.Errorf("vpc version: expected %q, got %q", "5.0.0", vpc.Version)
		}
		if vpc.IsLocal {
			t.Error("vpc should not be marked as local")
		}
	} else {
		t.Error("missing module 'vpc'")
	}

	// Test local module with ../
	if local, ok := modules["local_module"]; ok {
		if !local.IsLocal {
			t.Error("local_module should be marked as local")
		}
		if local.ResolvedPath == "" {
			t.Error("local_module should have resolved path")
		}
	} else {
		t.Error("missing module 'local_module'")
	}

	// Test local module with ./
	if rel, ok := modules["relative_module"]; ok {
		if !rel.IsLocal {
			t.Error("relative_module should be marked as local")
		}
	} else {
		t.Error("missing module 'relative_module'")
	}
}

func TestResolveWorkspacePath_Simple(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "state-bucket"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("expected 1 remote state, got %d", len(result.RemoteStates))
	}

	rs := result.RemoteStates[0]
	paths, err := parser.ResolveWorkspacePath(rs, "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	expected := "platform/stage/eu-central-1/vpc/terraform.tfstate"
	if paths[0] != expected {
		t.Errorf("expected path %q, got %q", expected, paths[0])
	}
}

func TestResolveWorkspacePath_WithLocals(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals {
  service     = "platform"
  environment = "stage"
  region      = "eu-central-1"
}
`,
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "state-bucket"
    key    = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("expected 1 remote state, got %d", len(result.RemoteStates))
	}

	rs := result.RemoteStates[0]
	paths, err := parser.ResolveWorkspacePath(rs, "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	expected := "platform/stage/eu-central-1/vpc/terraform.tfstate"
	if paths[0] != expected {
		t.Errorf("expected path %q, got %q", expected, paths[0])
	}
}

func TestResolveWorkspacePath_WithForEach(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals {
  service     = "platform"
  environment = "stage"
  region      = "eu-central-1"
  dependencies = {
    vpc = "platform/stage/eu-central-1/vpc"
    rds = "platform/stage/eu-central-1/rds"
  }
}
`,
		"data.tf": `
data "terraform_remote_state" "deps" {
  for_each = local.dependencies
  backend  = "s3"
  config = {
    bucket = "state-bucket"
    key    = "${each.value}/terraform.tfstate"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("expected 1 remote state, got %d", len(result.RemoteStates))
	}

	rs := result.RemoteStates[0]
	paths, err := parser.ResolveWorkspacePath(rs, "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}

	// Check both paths are present (order may vary)
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	expectedPaths := []string{
		"platform/stage/eu-central-1/vpc/terraform.tfstate",
		"platform/stage/eu-central-1/rds/terraform.tfstate",
	}

	for _, expected := range expectedPaths {
		if !pathSet[expected] {
			t.Errorf("missing expected path %q", expected)
		}
	}
}

func TestResolveWorkspacePath_InfersFromModulePath(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "state-bucket"
    key    = "${local.service}/${local.environment}/${local.region}/vpc/terraform.tfstate"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rs := result.RemoteStates[0]

	// Should infer locals from module path when not explicitly defined
	paths, err := parser.ResolveWorkspacePath(rs, "platform/stage/eu-central-1/eks", result.Locals, result.Variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	expected := "platform/stage/eu-central-1/vpc/terraform.tfstate"
	if paths[0] != expected {
		t.Errorf("expected path %q, got %q", expected, paths[0])
	}
}

func TestParseModule_InvalidHCL(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"invalid.tf": `
locals {
  broken = "unclosed string
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)

	// Should not return error but have diagnostics
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Diagnostics.HasErrors() {
		t.Error("expected diagnostics to have errors for invalid HCL")
	}
}

func TestParseModule_MultipleLocalsBlocks(t *testing.T) {
	tmpDir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals {
  service = "platform"
}

locals {
  environment = "stage"
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should merge locals from multiple blocks
	if len(result.Locals) != 2 {
		t.Errorf("expected 2 locals, got %d", len(result.Locals))
	}

	if val, ok := result.Locals["service"]; !ok || val.AsString() != "platform" {
		t.Error("missing or incorrect 'service' local")
	}
	if val, ok := result.Locals["environment"]; !ok || val.AsString() != "stage" {
		t.Error("missing or incorrect 'environment' local")
	}
}

func TestParseModule_RemoteStateConfigBlock(t *testing.T) {
	// Test config as block (config { ... }) instead of attribute (config = { ... })
	tmpDir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"

  config {
    bucket = "my-terraform-state"
    key    = "vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}
`,
	})
	defer os.RemoveAll(tmpDir)

	parser := NewParser()
	result, err := parser.ParseModule(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("expected 1 remote state, got %d", len(result.RemoteStates))
	}

	rs := result.RemoteStates[0]
	if rs.Backend != "s3" {
		t.Errorf("expected backend 's3', got %q", rs.Backend)
	}

	// Config should be parsed from block
	if _, ok := rs.Config["key"]; !ok {
		t.Error("expected 'key' in config")
	}
}

// Helper function to create a temporary module directory with files
func setupTempModule(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "parser-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	return tmpDir
}
