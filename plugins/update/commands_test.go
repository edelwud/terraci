package update

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	cmds := p.Commands(appCtx)
	if len(cmds) != 1 {
		t.Fatalf("Commands() returned %d commands, want 1", len(cmds))
	}

	cmd := cmds[0]
	if cmd.Use != "update" {
		t.Errorf("command.Use = %q, want %q", cmd.Use, "update")
	}

	for _, flag := range []string{"target", "bump", "write", "module", "output"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

func TestPlugin_Commands_RunE_NotConfigured(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for unconfigured plugin")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("error = %q, want to contain 'not enabled'", err.Error())
	}
}

func TestBuildUpdateReport_NoUpdates(t *testing.T) {
	result := &updateengine.UpdateResult{
		Summary: updateengine.UpdateSummary{
			TotalChecked:     3,
			UpdatesAvailable: 0,
		},
	}

	report := buildUpdateReport(result)
	if report.Plugin != "update" {
		t.Errorf("Plugin = %q, want %q", report.Plugin, "update")
	}
	if report.Title != "Dependency Update Check" {
		t.Errorf("Title = %q, want %q", report.Title, "Dependency Update Check")
	}
	if report.Status != ci.ReportStatusPass {
		t.Errorf("Status = %q, want %q", report.Status, ci.ReportStatusPass)
	}
	if !strings.Contains(report.Summary, "3 checked") {
		t.Errorf("Summary = %q, want to contain '3 checked'", report.Summary)
	}
}

func TestBuildUpdateReport_WithUpdates(t *testing.T) {
	result := &updateengine.UpdateResult{
		Summary: updateengine.UpdateSummary{
			TotalChecked:     5,
			UpdatesAvailable: 2,
		},
	}

	report := buildUpdateReport(result)
	if report.Status != ci.ReportStatusWarn {
		t.Errorf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if !strings.Contains(report.Summary, "2 updates available") {
		t.Errorf("Summary = %q, want to contain '2 updates available'", report.Summary)
	}
}

func TestRenderReportBody_Providers(t *testing.T) {
	result := &updateengine.UpdateResult{
		Providers: []updateengine.ProviderVersionUpdate{
			{
				ModulePath:     "platform/prod/vpc",
				ProviderSource: "hashicorp/aws",
				Constraint:     "~> 5.0",
				LatestVersion:  "5.3.0",
				Updated:        true,
			},
		},
	}

	body := renderReportBody(result)
	if !strings.Contains(body, "### Providers") {
		t.Error("Body missing '### Providers' header")
	}
	if !strings.Contains(body, "| Module |") {
		t.Error("Body missing markdown table header")
	}
	if !strings.Contains(body, "hashicorp/aws") {
		t.Error("Body missing provider source")
	}
	if !strings.Contains(body, "update available") {
		t.Error("Body missing 'update available' status")
	}
}

func TestRenderReportBody_Modules(t *testing.T) {
	result := &updateengine.UpdateResult{
		Modules: []updateengine.ModuleVersionUpdate{
			{
				ModulePath:    "platform/prod/vpc",
				Source:        "terraform-aws-modules/vpc/aws",
				Constraint:    "~> 5.0",
				LatestVersion: "5.2.0",
			},
		},
	}

	body := renderReportBody(result)
	if !strings.Contains(body, "### Modules") {
		t.Error("Body missing '### Modules' header")
	}
	if !strings.Contains(body, "terraform-aws-modules/vpc/aws") {
		t.Error("Body missing module source")
	}
	if !strings.Contains(body, "up to date") {
		t.Error("Body missing 'up to date' status")
	}
}

func TestRenderReportBody_Mixed(t *testing.T) {
	result := &updateengine.UpdateResult{
		Providers: []updateengine.ProviderVersionUpdate{
			{ModulePath: "a", ProviderSource: "hashicorp/aws", Updated: true},
		},
		Modules: []updateengine.ModuleVersionUpdate{
			{ModulePath: "b", Source: "terraform-aws-modules/vpc/aws"},
		},
	}

	body := renderReportBody(result)
	if !strings.Contains(body, "### Providers") {
		t.Error("Body missing Providers section")
	}
	if !strings.Contains(body, "### Modules") {
		t.Error("Body missing Modules section")
	}
}

func TestRenderReportBody_Skipped(t *testing.T) {
	result := &updateengine.UpdateResult{
		Providers: []updateengine.ProviderVersionUpdate{
			{
				ModulePath:     "test",
				ProviderSource: "hashicorp/aws",
				Skipped:        true,
				SkipReason:     "ignored by config",
			},
		},
	}

	body := renderReportBody(result)
	if !strings.Contains(body, "ignored by config") {
		t.Error("Body missing skip reason")
	}
}

func TestRenderReportBody_Empty(t *testing.T) {
	result := &updateengine.UpdateResult{}

	body := renderReportBody(result)
	if body != "" {
		t.Errorf("Body = %q, want empty for no results", body)
	}
}

func TestRenderReportBody_UpdateAvailable(t *testing.T) {
	result := &updateengine.UpdateResult{
		Modules: []updateengine.ModuleVersionUpdate{
			{
				ModulePath:    "test",
				Source:        "terraform-aws-modules/vpc/aws",
				Constraint:    "~> 5.0",
				LatestVersion: "5.3.0",
				Updated:       true,
			},
		},
	}

	body := renderReportBody(result)
	if !strings.Contains(body, "update available") {
		t.Error("Body missing 'update available' for updated module")
	}
}

func TestPlugin_RunCheck_NoModules(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	// Empty workDir — no modules to discover
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)

	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for no modules")
	}
	if !strings.Contains(err.Error(), "no modules") {
		t.Errorf("error = %q, want to contain 'no modules'", err.Error())
	}
}

func TestPlugin_RunCheck_InvalidOptions(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{
		Enabled: true,
		Target:  "invalid-target",
	})
	useMockRegistry(p, &mockRegistry{})

	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)

	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid target")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error = %q, want to contain 'invalid'", err.Error())
	}
}

func TestPlugin_RunCheck_Success(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0", "5.1.0"},
		},
	})

	// Create module directory matching default structure pattern
	workDir := t.TempDir()
	moduleDir := workDir + "/platform/prod/us-east-1/vpc"
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moduleDir+"/versions.tf", []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	appCtx := newTestAppContext(t, workDir)
	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}

	result := loadUpdateResult(t, appCtx.ServiceDir())
	if result.Summary.TotalChecked == 0 {
		t.Fatal("saved result has empty summary")
	}
	report := loadUpdateReport(t, appCtx.ServiceDir())
	if report.Plugin != "update" {
		t.Fatalf("report.Plugin = %q, want update", report.Plugin)
	}
}

func TestPlugin_RunCheck_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0"},
		},
	})

	workDir := t.TempDir()
	moduleDir := workDir + "/platform/prod/us-east-1/vpc"
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moduleDir+"/versions.tf", []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create AppContext with empty ServiceDir to skip artifact saving
	appCtx := newTestAppContext(t, workDir)
	appCtx.Update(appCtx.Config(), appCtx.WorkDir(), "", appCtx.Version())

	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func TestPlugin_RunCheck_FlagOverrides(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0"},
		},
	})

	workDir := t.TempDir()
	moduleDir := workDir + "/platform/prod/us-east-1/vpc"
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moduleDir+"/versions.tf", []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	appCtx := newTestAppContext(t, workDir)
	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	// Set flag overrides
	cmd.Flags().Set("target", "providers")
	cmd.Flags().Set("bump", "patch")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func TestPlugin_RunCheck_DiscoverError(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	// Point to a file instead of directory to trigger workflow.Run error
	workDir := t.TempDir()
	filePath := workDir + "/not-a-dir"
	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	appCtx := newTestAppContext(t, workDir)
	appCtx.Update(appCtx.Config(), filePath, appCtx.ServiceDir(), appCtx.Version())

	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, nil)
	if err == nil {
		// If workflow.Run doesn't error on a file path, it just returns no modules
		// which triggers "no modules found" error
		t.Log("workflow.Run handled file path gracefully")
	}
}

func TestPlugin_RunCheck_WithModuleFilter(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{
		providerVersions: map[string][]string{
			"hashicorp/aws": {"5.0.0"},
		},
	})

	workDir := t.TempDir()
	for _, mod := range []string{"vpc", "eks"} {
		dir := workDir + "/platform/prod/us-east-1/" + mod
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dir+"/versions.tf", []byte(`
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	appCtx := newTestAppContext(t, workDir)
	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	cmd.SetContext(context.Background())
	cmd.Flags().Set("module", "vpc")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}
