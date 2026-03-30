package extract

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
)

type testSink struct {
	path              string
	locals            map[string]cty.Value
	variables         map[string]cty.Value
	backend           *BackendConfig
	requiredProviders []RequiredProvider
	lockedProviders   []LockedProvider
	remoteStates      []RemoteStateRef
	moduleCalls       []ModuleCall
	diagnostics       hcl.Diagnostics
}

func newTestSink(path string) *testSink {
	return &testSink{
		path:              path,
		locals:            make(map[string]cty.Value),
		variables:         make(map[string]cty.Value),
		requiredProviders: make([]RequiredProvider, 0),
		lockedProviders:   make([]LockedProvider, 0),
		remoteStates:      make([]RemoteStateRef, 0),
		moduleCalls:       make([]ModuleCall, 0),
	}
}

func (s *testSink) Path() string                             { return s.path }
func (s *testSink) Locals() map[string]cty.Value             { return s.locals }
func (s *testSink) Variables() map[string]cty.Value          { return s.variables }
func (s *testSink) AddDiags(diags hcl.Diagnostics)           { s.diagnostics = append(s.diagnostics, diags...) }
func (s *testSink) SetLocal(name string, value cty.Value)    { s.locals[name] = value }
func (s *testSink) SetVariable(name string, value cty.Value) { s.variables[name] = value }
func (s *testSink) SetBackend(backend BackendConfig)         { s.backend = &backend }
func (s *testSink) AppendRequiredProvider(provider RequiredProvider) {
	s.requiredProviders = append(s.requiredProviders, provider)
}
func (s *testSink) AppendLockedProvider(provider LockedProvider) {
	s.lockedProviders = append(s.lockedProviders, provider)
}
func (s *testSink) AppendRemoteState(ref RemoteStateRef) {
	s.remoteStates = append(s.remoteStates, ref)
}
func (s *testSink) AppendModuleCall(call ModuleCall) { s.moduleCalls = append(s.moduleCalls, call) }

func TestRunDefault_ExtractsModuleFacts(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, dir, "locals.tf", `
locals {
  state_bucket = "team-a-state"
}
`)
	testutil.WriteFile(t, dir, "variables.tf", `variable "region" { default = "us-east-1" }`)
	testutil.WriteFile(t, dir, "terraform.tfvars", `region = "eu-west-1"`)
	testutil.WriteFile(t, dir, "backend.tf", `
terraform {
  backend "s3" {
    bucket = local.state_bucket
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
`)
	testutil.WriteFile(t, dir, "providers.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)
	testutil.WriteFile(t, dir, ".terraform.lock.hcl", `
provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.67.0"
  constraints = "~> 5.0"
}
`)
	testutil.WriteFile(t, dir, "data.tf", `
data "terraform_remote_state" "vpc" {
  backend = "s3"
  config = {
    bucket = "shared-state"
    key    = "platform/stage/eu-central-1/vpc/terraform.tfstate"
  }
}
`)
	testutil.WriteFile(t, dir, "modules.tf", `
module "vpc" {
  source  = "../_modules/vpc"
  version = "~> 5.0"
}
`)

	index, err := source.NewLoader().Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	sink := newTestSink(dir)
	RunDefault(&Context{
		Source:      index,
		EvalBuilder: evalctx.NewBuilder([]string{"service", "environment", "region", "module"}),
		Sink:        sink,
	})

	if got := sink.locals["state_bucket"].AsString(); got != "team-a-state" {
		t.Fatalf("local state_bucket = %q, want %q", got, "team-a-state")
	}
	if got := sink.variables["region"].AsString(); got != "eu-west-1" {
		t.Fatalf("variable region = %q, want %q", got, "eu-west-1")
	}
	if sink.backend == nil {
		t.Fatal("expected backend to be extracted")
	}
	if got := sink.backend.Config["bucket"]; got != "team-a-state" {
		t.Fatalf("backend bucket = %q, want %q", got, "team-a-state")
	}
	if len(sink.requiredProviders) != 1 {
		t.Fatalf("required providers = %d, want 1", len(sink.requiredProviders))
	}
	if got := sink.requiredProviders[0].Source; got != "hashicorp/aws" {
		t.Fatalf("required provider source = %q, want %q", got, "hashicorp/aws")
	}
	if len(sink.lockedProviders) != 1 {
		t.Fatalf("locked providers = %d, want 1", len(sink.lockedProviders))
	}
	if got := sink.lockedProviders[0].Version; got != "5.67.0" {
		t.Fatalf("locked provider version = %q, want %q", got, "5.67.0")
	}
	if len(sink.remoteStates) != 1 {
		t.Fatalf("remote states = %d, want 1", len(sink.remoteStates))
	}
	if got := sink.remoteStates[0].Backend; got != "s3" {
		t.Fatalf("remote state backend = %q, want %q", got, "s3")
	}
	if len(sink.moduleCalls) != 1 {
		t.Fatalf("module calls = %d, want 1", len(sink.moduleCalls))
	}
	if !sink.moduleCalls[0].IsLocal {
		t.Fatal("expected local module call")
	}
}
