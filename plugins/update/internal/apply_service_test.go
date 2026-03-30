package updateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeApplyTF(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestApplyUpdates_ErrorSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Dependency: ModuleDependency{CallName: "vpc", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
		},
		Providers: []ProviderVersionUpdate{
			{Dependency: ProviderDependency{ProviderName: "aws", Constraint: "~> 5.0"}, Status: StatusUpdateAvailable, File: "/nonexistent/file.tf", BumpedVersion: "5.2.0"},
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

func TestApplyUpdates_SkipsNotUpdated(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{Status: StatusUpToDate, File: "some.tf"},
		},
		Providers: []ProviderVersionUpdate{
			{Status: StatusSkipped, File: "some.tf", Issue: "ignored"},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusUpToDate {
		t.Errorf("Module.Status = %q, want %q", result.Modules[0].Status, StatusUpToDate)
	}
	if result.Providers[0].Status != StatusSkipped {
		t.Errorf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusSkipped)
	}
}

func TestApplyUpdates_ModuleSuccess(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "main.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
}
`)

	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{
				Dependency:    ModuleDependency{ModulePath: "test", CallName: "vpc", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusApplied {
		t.Fatalf("Module.Status = %q, want %q", result.Modules[0].Status, StatusApplied)
	}
	if result.Modules[0].Issue != "" {
		t.Fatalf("Module.Issue = %q, want empty", result.Modules[0].Issue)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), "~> 5.2") {
		t.Fatalf("updated file does not contain bumped constraint: %s", data)
	}
}

func TestApplyUpdates_ProviderSuccess(t *testing.T) {
	dir := t.TempDir()
	path := writeApplyTF(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	result := &UpdateResult{
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          path,
				BumpedVersion: "5.2.0",
			},
		},
	}

	NewApplyService().Apply(result)

	if result.Providers[0].Status != StatusApplied {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusApplied)
	}
	if result.Providers[0].Issue != "" {
		t.Fatalf("Provider.Issue = %q, want empty", result.Providers[0].Issue)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(data), "~> 5.2") {
		t.Fatalf("updated file does not contain bumped constraint: %s", data)
	}
}

func TestApplyUpdates_InvalidBumpedVersionSetsError(t *testing.T) {
	result := &UpdateResult{
		Modules: []ModuleVersionUpdate{
			{
				Dependency:    ModuleDependency{ModulePath: "test", CallName: "vpc", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          "main.tf",
				BumpedVersion: "bad",
			},
		},
		Providers: []ProviderVersionUpdate{
			{
				Dependency:    ProviderDependency{ModulePath: "test", ProviderName: "aws", ProviderSource: "hashicorp/aws", Constraint: "~> 5.0"},
				Status:        StatusUpdateAvailable,
				File:          "versions.tf",
				BumpedVersion: "bad",
			},
		},
	}

	NewApplyService().Apply(result)

	if result.Modules[0].Status != StatusError {
		t.Fatalf("Module.Status = %q, want %q", result.Modules[0].Status, StatusError)
	}
	if !strings.Contains(result.Modules[0].Issue, "failed to build") {
		t.Fatalf("Module.Issue = %q, want apply build error", result.Modules[0].Issue)
	}
	if result.Providers[0].Status != StatusError {
		t.Fatalf("Provider.Status = %q, want %q", result.Providers[0].Status, StatusError)
	}
	if !strings.Contains(result.Providers[0].Issue, "failed to build") {
		t.Fatalf("Provider.Issue = %q, want apply build error", result.Providers[0].Issue)
	}
}

func TestParseVersionOrZero(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got := parseVersionOrZero("1.2.3")
		if got != (Version{1, 2, 3, ""}) {
			t.Errorf("parseVersionOrZero(1.2.3) = %v", got)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		got := parseVersionOrZero("bad")
		if !got.IsZero() {
			t.Errorf("parseVersionOrZero(bad) = %v, want zero", got)
		}
	})
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
}
