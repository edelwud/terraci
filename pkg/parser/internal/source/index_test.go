package source

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
)

func TestIndexBlockAccess(t *testing.T) {
	dir := testutil.SetupTempModule(t, map[string]string{
		"locals.tf":  `locals { service = "platform" }`,
		"vars.tf":    `variable "region" { default = "us-east-1" }`,
		"backend.tf": `terraform { backend "s3" { bucket = "state" } }`,
		"data.tf":    `data "terraform_remote_state" "vpc" { backend = "s3" config = { key = "vpc/terraform.tfstate" } }`,
		"modules.tf": `module "vpc" { source = "terraform-aws-modules/vpc/aws" version = "~> 5.0" }`,
	})

	index, err := NewLoader().Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := len(index.LocalsBlocks()); got != 1 {
		t.Fatalf("locals blocks = %d, want 1", got)
	}
	if got := len(index.VariableBlocks()); got != 1 {
		t.Fatalf("variable blocks = %d, want 1", got)
	}
	if got := len(index.TerraformBlocks()); got != 1 {
		t.Fatalf("terraform blocks = %d, want 1", got)
	}
	if got := len(index.DataBlocks()); got != 1 {
		t.Fatalf("data blocks = %d, want 1", got)
	}
	if got := len(index.ModuleBlocks()); got != 1 {
		t.Fatalf("module blocks = %d, want 1", got)
	}
	if got := len(index.VariableBlockViews()); got != 1 {
		t.Fatalf("variable block views = %d, want 1", got)
	}
	if got := len(index.TerraformBlockViews()); got != 1 {
		t.Fatalf("terraform block views = %d, want 1", got)
	}
	if got := len(index.RemoteStateBlockViews()); got != 1 {
		t.Fatalf("remote state block views = %d, want 1", got)
	}
	if got := len(index.ModuleBlockViews()); got != 1 {
		t.Fatalf("module block views = %d, want 1", got)
	}

	if got := index.VariableBlockViews()[0].Name(); got != "region" {
		t.Fatalf("variable name = %q, want region", got)
	}
	if got := index.ModuleBlockViews()[0].Name(); got != "vpc" {
		t.Fatalf("module name = %q, want vpc", got)
	}
	if got := index.RemoteStateBlockViews()[0].Name(); got != "vpc" {
		t.Fatalf("remote state name = %q, want vpc", got)
	}
}
