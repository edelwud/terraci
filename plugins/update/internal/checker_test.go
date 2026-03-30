package updateengine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
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

func TestNewChecker(t *testing.T) {
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
	if prov.ProviderSource != "hashicorp/aws" {
		t.Errorf("ProviderSource = %q", prov.ProviderSource)
	}
	if prov.LatestVersion != "6.0.0" {
		t.Errorf("LatestVersion = %q, want 6.0.0", prov.LatestVersion)
	}
	if !prov.UpdateAvailable {
		t.Error("expected UpdateAvailable = true")
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
	if mod.Source != "terraform-aws-modules/vpc/aws" {
		t.Errorf("Source = %q", mod.Source)
	}
	if !mod.UpdateAvailable {
		t.Error("expected UpdateAvailable = true")
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
	if !result.Providers[0].Skipped {
		t.Error("expected Skipped = true")
	}
	if result.Providers[0].SkipReason != "no source specified" {
		t.Errorf("SkipReason = %q", result.Providers[0].SkipReason)
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
	if !result.Providers[0].Skipped {
		t.Error("expected Skipped = true")
	}
	if result.Providers[0].SkipReason != skipReasonIgnored {
		t.Errorf("SkipReason = %q", result.Providers[0].SkipReason)
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
	if !result.Providers[0].Skipped {
		t.Error("expected Skipped = true for invalid source")
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
	if result.Providers[0].Error == "" {
		t.Error("expected Error for registry error")
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
	if !prov.Skipped {
		t.Error("expected Skipped = true when cannot determine version")
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
	if !result.Modules[0].Skipped {
		t.Error("expected Skipped = true")
	}
	if result.Modules[0].SkipReason != skipReasonIgnored {
		t.Errorf("SkipReason = %q", result.Modules[0].SkipReason)
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
	if result.Modules[0].Error == "" {
		t.Error("expected Error for registry error")
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
	if !result.Modules[0].Applied {
		t.Fatal("expected module update to be applied when write=true")
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
	if !prov.UpdateAvailable {
		t.Error("expected UpdateAvailable = true")
	}
}

func TestApplyUpdates_ErrorSetsError(t *testing.T) {
	c := NewChecker(
		&UpdateConfig{Target: TargetAll, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		true,
	)

	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{UpdateAvailable: true, File: "/nonexistent/file.tf", CallName: "vpc", BumpedVersion: "5.2.0", Constraint: "~> 5.0"},
		},
		Providers: []ProviderVersionUpdate{
			{UpdateAvailable: true, File: "/nonexistent/file.tf", ProviderName: "aws", BumpedVersion: "5.2.0", Constraint: "~> 5.0"},
		},
	}

	c.applyUpdates(result)

	if result.Modules[0].Error == "" {
		t.Error("Module.Error should be set after write error")
	}
	if result.Providers[0].Error == "" {
		t.Error("Provider.Error should be set after write error")
	}
}

func TestApplyUpdates_SkipsNotUpdated(_ *testing.T) {
	c := NewChecker(
		&UpdateConfig{Target: TargetAll, Bump: BumpMinor},
		parser.NewParser(nil),
		&mockRegistry{},
		true,
	)

	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{UpdateAvailable: false, File: "some.tf"},
		},
		Providers: []ProviderVersionUpdate{
			{UpdateAvailable: true, File: ""},
		},
	}

	// Should not panic — skips entries without Updated or without File
	c.applyUpdates(result)
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
			{1, 0, 0, ""}, {2, 0, 0, "beta"}, {1, 5, 0, ""},
		}
		got := latestStable(versions)
		if got != (Version{1, 5, 0, ""}) {
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
			{1, 0, 0, ""}, {3, 0, 0, ""}, {2, 0, 0, ""},
		}
		got := latestStable(versions)
		if got != (Version{3, 0, 0, ""}) {
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
		{"pessimistic", "~> 5.0", Version{5, 0, 0, ""}},
		{"greater_equal", ">= 1.2.3", Version{1, 2, 3, ""}},
		{"plain_version", "5.0.0", Version{5, 0, 0, ""}},
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

func TestMustParseVersion(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got := mustParseVersion("1.2.3")
		if got != (Version{1, 2, 3, ""}) {
			t.Errorf("mustParseVersion(1.2.3) = %v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		got := mustParseVersion("bad")
		if !got.IsZero() {
			t.Errorf("mustParseVersion(bad) = %v, want zero", got)
		}
	})
}

func TestComputeSummary(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{UpdateAvailable: true},
			{Skipped: true},
			{},
		},
		Providers: []ProviderVersionUpdate{
			{UpdateAvailable: true},
			{Skipped: true},
		},
	}

	s := computeSummary(result)
	if s.TotalChecked != 5 {
		t.Errorf("TotalChecked = %d, want 5", s.TotalChecked)
	}
	if s.UpdatesAvailable != 2 {
		t.Errorf("UpdatesAvailable = %d, want 2", s.UpdatesAvailable)
	}
	if s.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", s.Skipped)
	}
}

func TestFindTFFileForModule(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		dir := setupModuleDir(t, map[string]string{
			"main.tf": `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`,
		})
		got := FindTFFileForModule(dir, "vpc")
		if got == "" {
			t.Error("expected to find file for module 'vpc'")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		dir := setupModuleDir(t, map[string]string{
			"main.tf": `
module "other" {
  source = "terraform-aws-modules/eks/aws"
}
`,
		})
		got := FindTFFileForModule(dir, "vpc")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no_tf_files", func(t *testing.T) {
		dir := t.TempDir()
		got := FindTFFileForModule(dir, "vpc")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestFindTFFileForProvider(t *testing.T) {
	t.Run("found", func(t *testing.T) {
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
		got := FindTFFileForProvider(dir, "aws")
		if got == "" {
			t.Error("expected to find file for provider 'aws'")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		dir := setupModuleDir(t, map[string]string{
			"versions.tf": `
terraform {
  required_providers {
    gcp = {
      source = "hashicorp/google"
    }
  }
}
`,
		})
		got := FindTFFileForProvider(dir, "aws")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("no_tf_files", func(t *testing.T) {
		dir := t.TempDir()
		got := FindTFFileForProvider(dir, "aws")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}
