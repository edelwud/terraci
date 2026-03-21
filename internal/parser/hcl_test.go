package parser

import (
	"context"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestParseModule_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewParser(nil)

	result, err := p.ParseModule(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Locals) != 0 {
		t.Errorf("locals: got %d, want 0", len(result.Locals))
	}
	if len(result.Variables) != 0 {
		t.Errorf("variables: got %d, want 0", len(result.Variables))
	}
	if len(result.RemoteStates) != 0 {
		t.Errorf("remote states: got %d, want 0", len(result.RemoteStates))
	}
	if len(result.ModuleCalls) != 0 {
		t.Errorf("module calls: got %d, want 0", len(result.ModuleCalls))
	}
}

func TestParseModule_WithLocals(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals {
  service     = "platform"
  environment = "stage"
  region      = "eu-central-1"
}
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertLocalEquals(t, result, "service", "platform")
	assertLocalEquals(t, result, "environment", "stage")
	assertLocalEquals(t, result, "region", "eu-central-1")
}

func TestParseModule_MultipleLocalsBlocks(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"locals.tf": `
locals { service = "platform" }
locals { environment = "stage" }
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Locals) != 2 {
		t.Errorf("locals: got %d, want 2", len(result.Locals))
	}
	assertLocalEquals(t, result, "service", "platform")
	assertLocalEquals(t, result, "environment", "stage")
}

func TestParseModule_WithVariables(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"variables.tf": `
variable "aws_region" { default = "eu-central-1" }
variable "project_name" { default = "my-project" }
variable "no_default" { description = "no default" }
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only variables with defaults
	if len(result.Variables) != 2 {
		t.Errorf("variables: got %d, want 2", len(result.Variables))
	}
	assertVarEquals(t, result, "aws_region", "eu-central-1")
}

func TestParseModule_WithTfvars(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"variables.tf":     `variable "region" { default = "us-east-1" }`,
		"terraform.tfvars": `region = "eu-west-1"`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertVarEquals(t, result, "region", "eu-west-1") // tfvars overrides default
}

func TestParseModule_WithAutoTfvars(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"variables.tf":         `variable "region" { default = "us-east-1" }`,
		"terraform.tfvars":     `region = "eu-west-1"`,
		"override.auto.tfvars": `region = "ap-northeast-1"`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertVarEquals(t, result, "region", "ap-northeast-1") // auto.tfvars highest priority
}

func TestParseModule_WithRemoteState(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = { bucket = "bucket", key = "vpc/terraform.tfstate", region = "eu-central-1" }
}
data "terraform_remote_state" "eks" {
  backend = "s3"
  config = { bucket = "bucket", key = "eks/terraform.tfstate", region = "eu-central-1" }
}
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 2 {
		t.Fatalf("remote states: got %d, want 2", len(result.RemoteStates))
	}

	names := map[string]bool{}
	for _, rs := range result.RemoteStates {
		names[rs.Name] = true
		if rs.Backend != "s3" {
			t.Errorf("rs %s: backend = %q, want s3", rs.Name, rs.Backend)
		}
	}
	for _, want := range []string{"vpc", "eks"} {
		if !names[want] {
			t.Errorf("missing remote state %q", want)
		}
	}
}

func TestParseModule_WithRemoteStateForEach(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"locals.tf": `locals { deps = { vpc = "vpc", rds = "rds" } }`,
		"data.tf": `
data "terraform_remote_state" "deps" {
  for_each = local.deps
  backend  = "s3"
  config   = { bucket = "b", key = "${each.value}/terraform.tfstate" }
}
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("remote states: got %d, want 1", len(result.RemoteStates))
	}
	if result.RemoteStates[0].ForEach == nil {
		t.Error("expected for_each to be set")
	}
}

func TestParseModule_RemoteStateConfigBlock(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"data.tf": `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config {
    bucket = "bucket"
    key    = "vpc/terraform.tfstate"
    region = "eu-central-1"
  }
}
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RemoteStates) != 1 {
		t.Fatalf("remote states: got %d, want 1", len(result.RemoteStates))
	}
	if _, ok := result.RemoteStates[0].Config["key"]; !ok {
		t.Error("config block: missing 'key'")
	}
}

func TestParseModule_WithModuleCalls(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}
module "local_mod" {
  source = "../../../_modules/kafka"
}
module "rel_mod" {
  source = "./submodules/net"
}
`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ModuleCalls) != 3 {
		t.Fatalf("module calls: got %d, want 3", len(result.ModuleCalls))
	}

	byName := map[string]*ModuleCall{}
	for _, mc := range result.ModuleCalls {
		byName[mc.Name] = mc
	}

	// Registry module
	if mc := byName["vpc"]; mc != nil {
		if mc.Source != "terraform-aws-modules/vpc/aws" {
			t.Errorf("vpc source = %q", mc.Source)
		}
		if mc.Version != "5.0.0" {
			t.Errorf("vpc version = %q", mc.Version)
		}
		if mc.IsLocal {
			t.Error("vpc should not be local")
		}
	} else {
		t.Error("missing module 'vpc'")
	}

	// Local modules
	for _, name := range []string{"local_mod", "rel_mod"} {
		if mc := byName[name]; mc != nil {
			if !mc.IsLocal {
				t.Errorf("%s should be local", name)
			}
		} else {
			t.Errorf("missing module %q", name)
		}
	}
}

func TestParseModule_InvalidHCL(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"invalid.tf": `locals { broken = "unclosed string\n}`,
	})

	result, err := NewParser(nil).ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Diagnostics.HasErrors() {
		t.Error("expected diagnostics errors for invalid HCL")
	}
}

// --- assertion helpers ---

func assertLocalEquals(t *testing.T, pm *ParsedModule, name, want string) {
	t.Helper()
	val, ok := pm.Locals[name]
	if !ok {
		t.Errorf("missing local %q", name)
		return
	}
	if val.Type() != cty.String {
		t.Errorf("local %q: type = %s, want string", name, val.Type().FriendlyName())
		return
	}
	if val.AsString() != want {
		t.Errorf("local %q = %q, want %q", name, val.AsString(), want)
	}
}

func assertVarEquals(t *testing.T, pm *ParsedModule, name, want string) {
	t.Helper()
	val, ok := pm.Variables[name]
	if !ok {
		t.Errorf("missing variable %q", name)
		return
	}
	if val.Type() != cty.String {
		t.Errorf("variable %q: type = %s, want string", name, val.Type().FriendlyName())
		return
	}
	if val.AsString() != want {
		t.Errorf("variable %q = %q, want %q", name, val.AsString(), want)
	}
}
