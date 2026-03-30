package checker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

type UpdateConfig = updateengine.UpdateConfig
type UpdateResult = updateengine.UpdateResult
type ModuleVersionUpdate = updateengine.ModuleVersionUpdate
type ProviderVersionUpdate = updateengine.ProviderVersionUpdate
type UpdateStatus = updateengine.UpdateStatus
type Version = updateengine.Version

const (
	TargetAll             = updateengine.TargetAll
	TargetModules         = updateengine.TargetModules
	TargetProviders       = updateengine.TargetProviders
	BumpPatch             = updateengine.BumpPatch
	BumpMinor             = updateengine.BumpMinor
	BumpMajor             = updateengine.BumpMajor
	StatusUpToDate        = updateengine.StatusUpToDate
	StatusUpdateAvailable = updateengine.StatusUpdateAvailable
	StatusApplied         = updateengine.StatusApplied
	StatusSkipped         = updateengine.StatusSkipped
	StatusError           = updateengine.StatusError
)

var (
	NewApplyService    = updateengine.NewApplyService
	BuildUpdateSummary = updateengine.BuildUpdateSummary
)

// mockRegistry implements RegistryClient for testing.
type mockRegistry struct {
	moduleVersions   map[string][]string // key: "ns/name/provider"
	providerVersions map[string][]string // key: "ns/type"
	moduleErr        error
	providerErr      error
}

func (m *mockRegistry) ModuleVersions(_ context.Context, ns, name, provider string) ([]string, error) {
	if m.moduleErr != nil {
		return nil, m.moduleErr
	}
	return m.moduleVersions[ns+"/"+name+"/"+provider], nil
}

func (m *mockRegistry) ProviderVersions(_ context.Context, ns, typeName string) ([]string, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	return m.providerVersions[ns+"/"+typeName], nil
}

// setupModuleDir creates a temp directory with .tf files and returns the path.
func setupModuleDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestNew(t *testing.T) {
	cfg := &UpdateConfig{Target: TargetAll, Bump: BumpMinor}
	p := parser.NewParser(nil)
	reg := &mockRegistry{}
	c := NewChecker(cfg, p, reg, false)
	if c.config != cfg {
		t.Error("config not set")
	}
	if c.parser != p {
		t.Error("parser not set")
	}
	if c.registry != reg {
		t.Error("registry not set")
	}
	if c.write {
		t.Error("write should be false")
	}
}

func TestChecker_Check_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := setupModuleDir(t, map[string]string{
		"main.tf": `resource "null_resource" "x" {}`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetAll, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	_, err := c.Check(ctx, []*discovery.Module{{Path: dir, RelativePath: "test"}})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestChecker_Check_EmptyModule(t *testing.T) {
	// Empty dir — parser returns empty result, no providers/modules to check.
	dir := t.TempDir()
	c := NewChecker(
		&UpdateConfig{Target: TargetAll, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "empty"},
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.Modules) != 0 {
		t.Errorf("modules = %d, want 0", len(result.Modules))
	}
	if len(result.Providers) != 0 {
		t.Errorf("providers = %d, want 0", len(result.Providers))
	}
}

func TestChecker_Check_Providers(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`,
	})

	reg := &mockRegistry{
		providerVersions: map[string][]string{
			// Current resolves to 5.0.0 (base of ~> 5.0), so 5.1.0 and 5.2.0 are minor bumps.
			"hashicorp/aws": {"5.0.0", "5.1.0", "5.2.0", "6.0.0"},
		},
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMajor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "platform/prod/vpc"},
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	prov := result.Providers[0]
	if prov.ProviderSource() != "hashicorp/aws" {
		t.Errorf("ProviderSource = %q", prov.ProviderSource())
	}
	if prov.LatestVersion != "6.0.0" {
		t.Errorf("LatestVersion = %q, want 6.0.0", prov.LatestVersion)
	}
	if prov.Status != StatusUpdateAvailable {
		t.Errorf("Status = %q, want %q", prov.Status, StatusUpdateAvailable)
	}
	if prov.File == "" {
		t.Error("expected provider file to be resolved")
	}
}

func TestChecker_Check_Modules(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`,
	})

	reg := &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0", "5.2.0"},
		},
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "platform/prod/vpc"},
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	mod := result.Modules[0]
	if mod.Source() != "terraform-aws-modules/vpc/aws" {
		t.Errorf("Source = %q", mod.Source())
	}
	if mod.Status != StatusUpdateAvailable {
		t.Errorf("Status = %q, want %q", mod.Status, StatusUpdateAvailable)
	}
	if mod.BumpedVersion == "" {
		t.Error("BumpedVersion should not be empty")
	}
	if mod.File == "" {
		t.Error("expected module file to be resolved")
	}
}

func TestCheckProviders_NoSource(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    null = {}
  }
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	if result.Providers[0].Status != StatusSkipped {
		t.Errorf("Status = %q, want %q", result.Providers[0].Status, StatusSkipped)
	}
	if result.Providers[0].Issue != "no source specified" {
		t.Errorf("Issue = %q", result.Providers[0].Issue)
	}
}

func TestCheckProviders_Ignored(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMinor, Ignore: []string{"hashicorp/aws"}},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Providers[0].Status != StatusSkipped {
		t.Errorf("Status = %q, want %q", result.Providers[0].Status, StatusSkipped)
	}
	if result.Providers[0].Issue != skipReasonIgnored {
		t.Errorf("Issue = %q", result.Providers[0].Issue)
	}
}

func TestCheckProviders_InvalidSource(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    bad = {
      source  = "invalid-no-slash"
      version = "~> 1.0"
    }
  }
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	if result.Providers[0].Status != StatusSkipped {
		t.Errorf("Status = %q, want %q", result.Providers[0].Status, StatusSkipped)
	}
}

func TestCheckProviders_RegistryError(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`,
	})

	reg := &mockRegistry{
		providerErr: context.DeadlineExceeded,
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMinor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Providers[0].Status != StatusError {
		t.Errorf("Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if result.Providers[0].Issue == "" {
		t.Error("expected Issue for registry error")
	}
	if result.Summary.Errors != 1 {
		t.Errorf("Summary.Errors = %d, want 1", result.Summary.Errors)
	}
}

func TestCheckProviders_CannotDetermineVersion(t *testing.T) {
	// Provider with source but no constraint and no lock file -> can't determine current version
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}
`,
	})

	reg := &mockRegistry{
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0", "5.1.0"},
		},
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMinor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	prov := result.Providers[0]
	if prov.Status != StatusSkipped {
		t.Errorf("Status = %q, want %q", prov.Status, StatusSkipped)
	}
	if prov.LatestVersion == "" {
		t.Error("LatestVersion should still be reported")
	}
}

func TestCheckModules_LocalModuleSkipped(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "local" {
  source = "../_modules/vpc"
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 0 {
		t.Errorf("modules = %d, want 0 (local modules skipped)", len(result.Modules))
	}
}

func TestCheckModules_NoVersionSkipped(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 0 {
		t.Errorf("modules = %d, want 0 (modules without version skipped)", len(result.Modules))
	}
}

func TestCheckModules_NonRegistrySkipped(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "git::https://example.com/vpc.git"
  version = "1.0.0"
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 0 {
		t.Errorf("modules = %d, want 0 (non-registry skipped)", len(result.Modules))
	}
}

func TestCheckModules_Ignored(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor, Ignore: []string{"terraform-aws-modules/vpc/aws"}},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	if result.Modules[0].Status != StatusSkipped {
		t.Errorf("Status = %q, want %q", result.Modules[0].Status, StatusSkipped)
	}
	if result.Modules[0].Issue != skipReasonIgnored {
		t.Errorf("Issue = %q", result.Modules[0].Issue)
	}
}

func TestCheckModules_InvalidSource(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "bad" {
  source  = "only/two"
  version = "~> 1.0"
}
`,
	})

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// "only/two" has 2 parts, IsRegistrySource returns false (needs 3 parts)
	// So it's filtered out before reaching checkModules
	if len(result.Modules) != 0 {
		t.Errorf("modules = %d, want 0 (non-registry source filtered)", len(result.Modules))
	}
}

func TestCheckModules_SuccessNoUpdate(t *testing.T) {
	// Module where the current version is the latest — no update.
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.2"
}
`,
	})

	reg := &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0", "5.2.0"},
		},
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	// 5.2 is the base version from constraint, LatestByBump won't find anything > 5.2.0 with same major
	mod := result.Modules[0]
	if mod.CurrentVersion == "" {
		t.Error("CurrentVersion should be set")
	}
	if mod.LatestVersion == "" {
		t.Error("LatestVersion should be set")
	}
}

func TestCheckModules_RegistryError(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`,
	})

	reg := &mockRegistry{
		moduleErr: context.DeadlineExceeded,
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMinor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	if result.Modules[0].Status != StatusError {
		t.Errorf("Status = %q, want %q", result.Modules[0].Status, StatusError)
	}
	if result.Modules[0].Issue == "" {
		t.Error("expected Issue for registry error")
	}
	if result.Summary.Errors != 1 {
		t.Errorf("Summary.Errors = %d, want 1", result.Summary.Errors)
	}
}

func TestChecker_Check_WithWrite(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`,
	})

	reg := &mockRegistry{
		moduleVersions: map[string][]string{
			"terraform-aws-modules/vpc/aws": {"5.0.0", "5.1.0", "5.2.0"},
		},
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetModules, Bump: BumpMajor},
		parser.NewParser(nil),
		reg,
		true, // write=true
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("modules = %d, want 1", len(result.Modules))
	}
	if result.Modules[0].Status != StatusApplied {
		t.Fatalf("Status = %q, want %q", result.Modules[0].Status, StatusApplied)
	}
	data, readErr := os.ReadFile(filepath.Join(dir, "main.tf"))
	if readErr != nil {
		t.Fatalf("read updated file: %v", readErr)
	}
	if !strings.Contains(string(data), "~> 5.2") {
		t.Fatalf("updated file does not contain bumped constraint: %s", data)
	}
}

func TestChecker_Check_ProviderWithLockFile(t *testing.T) {
	dir := setupModuleDir(t, map[string]string{
		"versions.tf": `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`,
		".terraform.lock.hcl": `
provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.67.0"
  constraints = "~> 5.0"
  hashes = [
    "h1:test",
  ]
}
`,
	})

	reg := &mockRegistry{
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.67.0", "5.68.0", "5.69.0"},
		},
	}

	c := NewChecker(
		&UpdateConfig{Target: TargetProviders, Bump: BumpMinor},
		parser.NewParser(nil),
		reg,
		false,
	)

	result, err := c.Check(context.Background(), []*discovery.Module{
		{Path: dir, RelativePath: "test"},
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(result.Providers))
	}
	prov := result.Providers[0]
	if prov.CurrentVersion != "5.67.0" {
		t.Errorf("CurrentVersion = %q, want '5.67.0' (from lock file)", prov.CurrentVersion)
	}
	if prov.Status != StatusUpdateAvailable {
		t.Errorf("Status = %q, want %q", prov.Status, StatusUpdateAvailable)
	}
}

func TestApplyUpdates_ErrorSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Dependency: updateengine.ModuleDependency{CallName: "vpc", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
		Providers: []ProviderVersionUpdate{
			{Dependency: updateengine.ProviderDependency{ProviderName: "aws", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusError {
		t.Errorf("Module.Status = %q, want %q", result.Modules[0].Status, StatusError)
	}
	if result.Modules[0].Issue == "" {
		t.Error("Module.Issue should be set after write error")
	}
	if result.Providers[0].Status != StatusError {
		t.Errorf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if result.Providers[0].Issue == "" {
		t.Error("Provider.Issue should be set after write error")
	}
}

func TestApplyUpdates_SkipsNotUpdated(_ *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpToDate, File: "some.tf"},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusUpdateAvailable, File: ""},
		},
	}

	// Should not panic — skips entries without Updated or without File
	NewApplyService().Apply(result)
}

func TestBuildLockIndex(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		idx := buildLockIndex(nil)
		if len(idx) != 0 {
			t.Errorf("len = %d, want 0", len(idx))
		}
	})

	t.Run("strips_terraform_prefix", func(t *testing.T) {
		idx := buildLockIndex([]*parser.LockedProvider{
			{Source: "registry.terraform.io/hashicorp/aws", Version: "5.67.0"},
		})
		if _, ok := idx["hashicorp/aws"]; !ok {
			t.Error("expected key 'hashicorp/aws'")
		}
	})

	t.Run("strips_opentofu_prefix", func(t *testing.T) {
		idx := buildLockIndex([]*parser.LockedProvider{
			{Source: "registry.opentofu.org/hashicorp/aws", Version: "5.67.0"},
		})
		if _, ok := idx["hashicorp/aws"]; !ok {
			t.Error("expected key 'hashicorp/aws'")
		}
	})

	t.Run("no_prefix", func(t *testing.T) {
		idx := buildLockIndex([]*parser.LockedProvider{
			{Source: "hashicorp/aws", Version: "5.67.0"},
		})
		if _, ok := idx["hashicorp/aws"]; !ok {
			t.Error("expected key 'hashicorp/aws'")
		}
	})
}

func TestStripRegistryPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"registry.terraform.io/hashicorp/aws", "hashicorp/aws"},
		{"registry.opentofu.org/hashicorp/aws", "hashicorp/aws"},
		{"hashicorp/aws", "hashicorp/aws"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := stripRegistryPrefix(tt.input); got != tt.want {
				t.Errorf("stripRegistryPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseVersionList(t *testing.T) {
	t.Run("valid_versions", func(t *testing.T) {
		got := parseVersionList([]string{"1.0.0", "2.0.0"})
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
	})

	t.Run("invalid_skipped", func(t *testing.T) {
		got := parseVersionList([]string{"1.0.0", "bad", "2.0.0"})
		if len(got) != 2 {
			t.Errorf("len = %d, want 2 (invalid skipped)", len(got))
		}
	})

	t.Run("empty", func(t *testing.T) {
		got := parseVersionList(nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

func TestLatestStable(t *testing.T) {
	t.Run("skips_prereleases", func(t *testing.T) {
		versions := []Version{
			{Major: 1, Minor: 0, Patch: 0, Prerelease: ""},
			{Major: 2, Minor: 0, Patch: 0, Prerelease: "beta"},
			{Major: 1, Minor: 5, Patch: 0, Prerelease: ""},
		}
		got := latestStable(versions)
		if got != (Version{Major: 1, Minor: 5, Patch: 0, Prerelease: ""}) {
			t.Errorf("latestStable = %v, want 1.5.0", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		got := latestStable(nil)
		if !got.IsZero() {
			t.Errorf("latestStable(nil) = %v, want zero", got)
		}
	})

	t.Run("finds_highest", func(t *testing.T) {
		versions := []Version{
			{Major: 1, Minor: 0, Patch: 0, Prerelease: ""},
			{Major: 3, Minor: 0, Patch: 0, Prerelease: ""},
			{Major: 2, Minor: 0, Patch: 0, Prerelease: ""},
		}
		got := latestStable(versions)
		if got != (Version{Major: 3, Minor: 0, Patch: 0, Prerelease: ""}) {
			t.Errorf("latestStable = %v, want 3.0.0", got)
		}
	})
}

func TestVersionFromConstraint(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Version
	}{
		{"pessimistic", "~> 5.0", Version{Major: 5, Minor: 0, Patch: 0, Prerelease: ""}},
		{"greater_equal", ">= 1.2.3", Version{Major: 1, Minor: 2, Patch: 3, Prerelease: ""}},
		{"plain_version", "5.0.0", Version{Major: 5, Minor: 0, Patch: 0, Prerelease: ""}},
		{"invalid", "garbage", Version{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionFromConstraint(tt.input)
			if got != tt.want {
				t.Errorf("versionFromConstraint(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeSummary(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpdateAvailable},
			{Status: StatusSkipped},
			{Status: StatusUpToDate},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusApplied},
			{Status: StatusError},
		},
	}

	s := BuildUpdateSummary(result)
	if s.TotalChecked != 5 {
		t.Errorf("TotalChecked = %d, want 5", s.TotalChecked)
	}
	if s.UpdatesAvailable != 1 {
		t.Errorf("UpdatesAvailable = %d, want 1", s.UpdatesAvailable)
	}
	if s.UpdatesApplied != 1 {
		t.Errorf("UpdatesApplied = %d, want 1", s.UpdatesApplied)
	}
	if s.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", s.Skipped)
	}
	if s.Errors != 1 {
		t.Errorf("Errors = %d, want 1", s.Errors)
	}

	s2 := BuildUpdateSummary(result)
	if s2 != s {
		t.Errorf("second BuildUpdateSummary() = %+v, want %+v", s2, s)
	}
}

func TestAnalyzeModuleVersions(t *testing.T) {
	analysis := analyzeModuleVersions(
		"~> 5.0",
		parseVersionList([]string{"5.0.0", "5.2.0", "6.0.0", "6.1.0-beta"}),
		BumpMinor,
	)

	if !analysis.hasCurrent || analysis.current.String() != "5.0.0" {
		t.Fatalf("current = %v (has=%v), want 5.0.0", analysis.current, analysis.hasCurrent)
	}
	if analysis.latest.String() != "6.0.0" {
		t.Errorf("latest = %v, want 6.0.0", analysis.latest)
	}
	if analysis.bumped.String() != "5.2.0" {
		t.Errorf("bumped = %v, want 5.2.0", analysis.bumped)
	}
}

func TestAnalyzeProviderVersions(t *testing.T) {
	t.Run("uses locked current version", func(t *testing.T) {
		analysis := analyzeProviderVersions(
			"~> 5.0",
			"5.67.0",
			parseVersionList([]string{"5.67.0", "5.68.0", "6.0.0"}),
			BumpMinor,
		)

		if !analysis.hasCurrent || analysis.current.String() != "5.67.0" {
			t.Fatalf("current = %v (has=%v), want 5.67.0", analysis.current, analysis.hasCurrent)
		}
		if analysis.latest.String() != "6.0.0" {
			t.Errorf("latest = %v, want 6.0.0", analysis.latest)
		}
		if analysis.bumped.String() != "5.68.0" {
			t.Errorf("bumped = %v, want 5.68.0", analysis.bumped)
		}
	})

	t.Run("falls back to constraint resolution", func(t *testing.T) {
		analysis := analyzeProviderVersions(
			"~> 5.0",
			"",
			parseVersionList([]string{"5.0.0", "5.2.0", "6.0.0"}),
			BumpMinor,
		)

		if !analysis.hasCurrent || analysis.current.String() != "5.2.0" {
			t.Fatalf("current = %v (has=%v), want 5.2.0", analysis.current, analysis.hasCurrent)
		}
		if !analysis.bumped.IsZero() {
			t.Fatalf("expected no bumped version, got %v", analysis.bumped)
		}
	})
}

func TestModuleScanResult_Outcome(t *testing.T) {
	update := ModuleVersionUpdate{}
	result := newModuleScanResult(update, versionAnalysis{
		current:    Version{Major: 5, Minor: 0, Patch: 0},
		latest:     Version{Major: 6, Minor: 0, Patch: 0},
		bumped:     Version{Major: 5, Minor: 2, Patch: 0},
		hasCurrent: true,
	})

	outcome := result.outcome("/tmp/main.tf")
	if outcome.CurrentVersion != "5.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", outcome.CurrentVersion, "5.0.0")
	}
	if outcome.LatestVersion != "6.0.0" {
		t.Errorf("LatestVersion = %q, want %q", outcome.LatestVersion, "6.0.0")
	}
	if outcome.BumpedVersion != "5.2.0" {
		t.Errorf("BumpedVersion = %q, want %q", outcome.BumpedVersion, "5.2.0")
	}
	if outcome.File != "/tmp/main.tf" {
		t.Errorf("File = %q, want %q", outcome.File, "/tmp/main.tf")
	}
	if outcome.Status != StatusUpdateAvailable {
		t.Errorf("Status = %q, want %q", outcome.Status, StatusUpdateAvailable)
	}
}

func TestProviderScanResult_Outcome(t *testing.T) {
	t.Run("cannot determine current version", func(t *testing.T) {
		update := ProviderVersionUpdate{}
		outcome := newProviderScanResult(update, versionAnalysis{}).outcome("/tmp/versions.tf")
		if outcome.Status != StatusSkipped {
			t.Errorf("Status = %q, want %q", outcome.Status, StatusSkipped)
		}
		if outcome.Issue != "cannot determine current version" {
			t.Errorf("Issue = %q, want %q", outcome.Issue, "cannot determine current version")
		}
	})

	t.Run("available update", func(t *testing.T) {
		update := ProviderVersionUpdate{}
		outcome := newProviderScanResult(update, versionAnalysis{
			current:    Version{Major: 5, Minor: 67, Patch: 0},
			latest:     Version{Major: 6, Minor: 0, Patch: 0},
			bumped:     Version{Major: 5, Minor: 68, Patch: 0},
			hasCurrent: true,
		}).outcome("/tmp/versions.tf")

		if outcome.CurrentVersion != "5.67.0" {
			t.Errorf("CurrentVersion = %q, want %q", outcome.CurrentVersion, "5.67.0")
		}
		if outcome.LatestVersion != "6.0.0" {
			t.Errorf("LatestVersion = %q, want %q", outcome.LatestVersion, "6.0.0")
		}
		if outcome.BumpedVersion != "5.68.0" {
			t.Errorf("BumpedVersion = %q, want %q", outcome.BumpedVersion, "5.68.0")
		}
		if outcome.File != "/tmp/versions.tf" {
			t.Errorf("File = %q, want %q", outcome.File, "/tmp/versions.tf")
		}
		if outcome.Status != StatusUpdateAvailable {
			t.Errorf("Status = %q, want %q", outcome.Status, StatusUpdateAvailable)
		}
	})
}
