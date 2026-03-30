package parser

import (
	"context"
	"testing"
)

func TestModuleIndex_BlockAccess(t *testing.T) {
	dir := setupTempModule(t, map[string]string{
		"locals.tf":  `locals { service = "platform" }`,
		"vars.tf":    `variable "region" { default = "us-east-1" }`,
		"backend.tf": `terraform { backend "s3" { bucket = "state" } }`,
		"data.tf":    `data "terraform_remote_state" "vpc" { backend = "s3" config = { key = "vpc/terraform.tfstate" } }`,
		"modules.tf": `module "vpc" { source = "terraform-aws-modules/vpc/aws" version = "~> 5.0" }`,
	})

	index, err := newModuleLoader().Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := len(index.localsBlocks()); got != 1 {
		t.Fatalf("locals blocks = %d, want 1", got)
	}
	if got := len(index.variableBlocks()); got != 1 {
		t.Fatalf("variable blocks = %d, want 1", got)
	}
	if got := len(index.terraformBlocks()); got != 1 {
		t.Fatalf("terraform blocks = %d, want 1", got)
	}
	if got := len(index.dataBlocks()); got != 1 {
		t.Fatalf("data blocks = %d, want 1", got)
	}
	if got := len(index.moduleBlocks()); got != 1 {
		t.Fatalf("module blocks = %d, want 1", got)
	}
	if got := len(index.variableBlockViews()); got != 1 {
		t.Fatalf("variable block views = %d, want 1", got)
	}
	if got := len(index.terraformBlockViews()); got != 1 {
		t.Fatalf("terraform block views = %d, want 1", got)
	}
	if got := len(index.remoteStateBlockViews()); got != 1 {
		t.Fatalf("remote state block views = %d, want 1", got)
	}
	if got := len(index.moduleBlockViews()); got != 1 {
		t.Fatalf("module block views = %d, want 1", got)
	}

	if got := index.variableBlockViews()[0].Name(); got != "region" {
		t.Fatalf("variable name = %q, want region", got)
	}
	if got := index.moduleBlockViews()[0].Name(); got != "vpc" {
		t.Fatalf("module name = %q, want vpc", got)
	}
	if got := index.remoteStateBlockViews()[0].Name(); got != "vpc" {
		t.Fatalf("remote state name = %q, want vpc", got)
	}
}
