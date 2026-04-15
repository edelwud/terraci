package planner

import (
	"context"
	"os"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

type mockRegistry struct {
	moduleVersions     map[string][]string
	moduleProviderDeps map[string][]registrymeta.ModuleProviderDep
	providerVersions   map[string][]string
}

func (m *mockRegistry) ModuleVersions(_ context.Context, address sourceaddr.ModuleAddress) ([]string, error) {
	return m.moduleVersions[address.Namespace+"/"+address.Name+"/"+address.Provider], nil
}

func (m *mockRegistry) ModuleProviderDeps(_ context.Context, address sourceaddr.ModuleAddress, version string) ([]registrymeta.ModuleProviderDep, error) {
	return m.moduleProviderDeps[address.Namespace+"/"+address.Name+"/"+address.Provider+"@"+version], nil
}

func (m *mockRegistry) ProviderVersions(_ context.Context, address sourceaddr.ProviderAddress) ([]string, error) {
	return m.providerVersions[address.Namespace+"/"+address.Type], nil
}

func (m *mockRegistry) ProviderPlatforms(context.Context, sourceaddr.ProviderAddress, string) ([]string, error) {
	return nil, nil
}

func (m *mockRegistry) ProviderPackage(context.Context, sourceaddr.ProviderAddress, string, string) (*registrymeta.ProviderPackage, error) {
	return nil, nil
}

func TestSolveModule_IncludesLockOnlyProvider(t *testing.T) {
	moduleParser := parser.NewParser(nil)
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)
	write(".terraform.lock.hcl", `
provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.1.0"
  constraints = "~> 5.0"
  hashes      = ["zh:test"]
}
`)

	parsed, err := moduleParser.ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("ParseModule() error = %v", err)
	}

	solver := New(context.Background(), &tfupdateengine.UpdateConfig{
		Target: tfupdateengine.TargetAll,
		Bump:   tfupdateengine.BumpMinor,
	}, &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0"},
		},
		moduleProviderDeps: map[string][]registrymeta.ModuleProviderDep{
			"terraform-aws-modules/vpc/aws@5.1.0": {{
				Source:  "hashicorp/aws",
				Version: ">= 5.0.0, < 6.0.0",
			}},
		},
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.1.0", "5.2.0"},
		},
	})

	plan, err := solver.SolveModule(&discovery.Module{Path: dir, RelativePath: "svc/prod/mod"}, parsed)
	if err != nil {
		t.Fatalf("SolveModule() error = %v", err)
	}
	if len(plan.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(plan.Providers))
	}
	if !plan.Providers[0].Locked {
		t.Fatalf("provider should be marked as lock-derived")
	}
	if plan.Providers[0].LockedSource != "registry.terraform.io/hashicorp/aws" {
		t.Fatalf("locked source = %q", plan.Providers[0].LockedSource)
	}
	if len(plan.LockSync.Providers) != 1 {
		t.Fatalf("lock sync providers = %d, want 1", len(plan.LockSync.Providers))
	}
}

func TestSolveModule_BuildsLockSyncForTransitiveProvider(t *testing.T) {
	moduleParser := parser.NewParser(nil)
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)

	parsed, err := moduleParser.ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("ParseModule() error = %v", err)
	}

	solver := New(context.Background(), &tfupdateengine.UpdateConfig{
		Target: tfupdateengine.TargetAll,
		Bump:   tfupdateengine.BumpMinor,
	}, &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0"},
		},
		moduleProviderDeps: map[string][]registrymeta.ModuleProviderDep{
			"terraform-aws-modules/vpc/aws@5.1.0": {{
				Source:  "hashicorp/aws",
				Version: ">= 5.0.0, < 6.0.0",
			}},
		},
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.1.0", "5.2.0"},
		},
	})

	plan, err := solver.SolveModule(&discovery.Module{Path: dir, RelativePath: "svc/prod/mod"}, parsed)
	if err != nil {
		t.Fatalf("SolveModule() error = %v", err)
	}
	if len(plan.Providers) != 0 {
		t.Fatalf("providers = %d, want 0 direct providers", len(plan.Providers))
	}
	if len(plan.LockSync.Providers) != 1 {
		t.Fatalf("lock sync providers = %d, want 1", len(plan.LockSync.Providers))
	}
	lockProvider := plan.LockSync.Providers[0]
	if lockProvider.ProviderSource != "hashicorp/aws" {
		t.Fatalf("lock source = %q", lockProvider.ProviderSource)
	}
	if lockProvider.Version != "5.2.0" {
		t.Fatalf("lock version = %q", lockProvider.Version)
	}
	if lockProvider.Constraint != ">= 5.0.0, < 6.0.0" {
		t.Fatalf("lock constraint = %q", lockProvider.Constraint)
	}
	if lockProvider.TerraformFile != dir+"/main.tf" {
		t.Fatalf("lock terraform file = %q", lockProvider.TerraformFile)
	}
}

func TestSolveModule_TargetModulesSuppressesProviderResults(t *testing.T) {
	moduleParser := parser.NewParser(nil)
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/main.tf", []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	parsed, err := moduleParser.ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("ParseModule() error = %v", err)
	}

	solver := New(context.Background(), &tfupdateengine.UpdateConfig{
		Target: tfupdateengine.TargetModules,
		Bump:   tfupdateengine.BumpMinor,
	}, &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0"},
		},
		moduleProviderDeps: map[string][]registrymeta.ModuleProviderDep{
			"terraform-aws-modules/vpc/aws@5.1.0": {{
				Source:  "hashicorp/aws",
				Version: ">= 5.0.0, < 6.0.0",
			}},
		},
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0", "5.2.0"},
		},
	})

	plan, err := solver.SolveModule(&discovery.Module{Path: dir, RelativePath: "svc/prod/mod"}, parsed)
	if err != nil {
		t.Fatalf("SolveModule() error = %v", err)
	}
	if len(plan.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(plan.Modules))
	}
	if len(plan.Providers) != 0 {
		t.Fatalf("providers = %d, want 0 for target=modules", len(plan.Providers))
	}
	if len(plan.LockSync.Providers) != 1 {
		t.Fatalf("lock sync providers = %d, want 1", len(plan.LockSync.Providers))
	}
}

func TestSolveModule_TargetProvidersUsesCurrentModuleConstraintsOnly(t *testing.T) {
	moduleParser := parser.NewParser(nil)
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/main.tf", []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	parsed, err := moduleParser.ParseModule(context.Background(), dir)
	if err != nil {
		t.Fatalf("ParseModule() error = %v", err)
	}

	solver := New(context.Background(), &tfupdateengine.UpdateConfig{
		Target: tfupdateengine.TargetProviders,
		Bump:   tfupdateengine.BumpMinor,
	}, &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0"},
		},
		moduleProviderDeps: map[string][]registrymeta.ModuleProviderDep{
			"terraform-aws-modules/vpc/aws@5.0.0": {{
				Source:  "hashicorp/aws",
				Version: ">= 5.0.0, < 6.0.0",
			}},
			"terraform-aws-modules/vpc/aws@5.1.0": {{
				Source:  "hashicorp/aws",
				Version: ">= 6.0.0",
			}},
		},
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0", "5.2.0", "6.0.0"},
		},
	})

	plan, err := solver.SolveModule(&discovery.Module{Path: dir, RelativePath: "svc/prod/mod"}, parsed)
	if err != nil {
		t.Fatalf("SolveModule() error = %v", err)
	}
	if len(plan.Modules) != 0 {
		t.Fatalf("modules = %d, want 0 for target=providers", len(plan.Modules))
	}
	if len(plan.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(plan.Providers))
	}
	if plan.Providers[0].Selected != "5.2.0" {
		t.Fatalf("provider selected = %q, want 5.2.0", plan.Providers[0].Selected)
	}
	if len(plan.LockSync.Providers) != 1 {
		t.Fatalf("lock sync providers = %d, want 1", len(plan.LockSync.Providers))
	}
	if plan.LockSync.Providers[0].Constraint != ">= 5.0.0, ~> 5.0, < 6.0.0" {
		t.Fatalf("lock constraint = %q", plan.LockSync.Providers[0].Constraint)
	}
}
