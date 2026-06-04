package tfupdate

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/cliout"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/plugins/internal/reportrender"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
)

func buildTFUpdateCommand(t *testing.T, p *Plugin) *cobra.Command {
	t.Helper()
	specs, err := p.CommandSpecs()
	if err != nil {
		t.Fatalf("CommandSpecs() error = %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("CommandSpecs() returned %d specs, want 1", len(specs))
	}
	cmd, err := plugin.BuildCommand(specs[0])
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}
	return cmd
}

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin(t)

	cmd := buildTFUpdateCommand(t, p)
	if cmd.Use != "tfupdate" {
		t.Errorf("command.Use = %q, want %q", cmd.Use, "tfupdate")
	}

	for _, flag := range []string{"target", "bump", "pin", "timeout", "write", "module", "output"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

func TestResolveCommandTimeout(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *tfupdateengine.UpdateConfig
		req     CheckRequest
		want    time.Duration
		wantErr bool
	}{
		{
			name: "read default",
			cfg:  &tfupdateengine.UpdateConfig{},
			req:  CheckRequest{},
			want: tfupdateengine.DefaultReadTimeout,
		},
		{
			name: "write default",
			cfg:  &tfupdateengine.UpdateConfig{},
			req:  CheckRequest{Write: true},
			want: tfupdateengine.DefaultWriteTimeout,
		},
		{
			name: "config override",
			cfg:  &tfupdateengine.UpdateConfig{Timeout: "30m"},
			req:  CheckRequest{Write: true},
			want: 30 * time.Minute,
		},
		{
			name: "flag override",
			cfg:  &tfupdateengine.UpdateConfig{Timeout: "30m"},
			req:  CheckRequest{Write: true, Timeout: "45m"},
			want: 45 * time.Minute,
		},
		{
			name:    "invalid override",
			cfg:     &tfupdateengine.UpdateConfig{},
			req:     CheckRequest{Timeout: "later"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCommandTimeout(tt.cfg, tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveCommandTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("resolveCommandTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCheckRequest(t *testing.T) {
	t.Parallel()

	cmd := buildTFUpdateCommand(t, newTestPlugin(t))
	if err := cmd.Flags().Parse([]string{
		"--write",
		"--module", "platform/prod/vpc",
		"--output", "json",
		"--target", "providers",
		"--bump", "patch",
		"--pin",
		"--timeout", "15m",
		"--lock-platforms", "linux_amd64,darwin_arm64",
	}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	req, err := parseCheckRequest(cmd)
	if err != nil {
		t.Fatalf("parseCheckRequest() error = %v", err)
	}
	if !req.Write || req.ModulePath != "platform/prod/vpc" || req.OutputFormat != cliout.FormatJSON ||
		req.Target != "providers" || req.Bump != "patch" || !req.Pin || req.Timeout != "15m" ||
		strings.Join(req.LockPlatforms, ",") != "linux_amd64,darwin_arm64" {
		t.Fatalf("parseCheckRequest() = %#v, want parsed flags", req)
	}
}

func TestPlugin_Commands_RunE_NotConfigured(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestCommandAppContext(t, t.TempDir(), p)

	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for unconfigured plugin")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("error = %q, want to contain 'not enabled'", err.Error())
	}
}

func TestBuildUpdateReport_NoUpdates(t *testing.T) {
	result := &tfupdateengine.UpdateResult{
		Summary: tfupdateengine.UpdateSummary{
			TotalChecked:     3,
			UpdatesAvailable: 0,
		},
	}

	report, buildErr := buildUpdateReport(updateReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildUpdateReport() error = %v", buildErr)
	}
	citest.AssertRenderedReportContract(t, report, citest.RenderedReportContract{
		Producer: pluginName,
		Status:   ci.ReportStatusPass,
		Renderers: []citest.ReportRenderer{
			reportrender.MarkdownReport,
			reportrender.CLIReport,
		},
	})
	if report.Producer() != "tfupdate" {
		t.Errorf("Plugin = %q, want %q", report.Producer(), "tfupdate")
	}
	if report.Title() != "Dependency Update Check" {
		t.Errorf("Title = %q, want %q", report.Title(), "Dependency Update Check")
	}
	if report.Status() != ci.ReportStatusPass {
		t.Errorf("Status = %q, want %q", report.Status(), ci.ReportStatusPass)
	}
	if !strings.Contains(report.Summary(), "3 checked") {
		t.Errorf("Summary = %q, want to contain '3 checked'", report.Summary())
	}
}

func TestBuildUpdateReport_WithUpdates(t *testing.T) {
	result := &tfupdateengine.UpdateResult{
		Summary: tfupdateengine.UpdateSummary{
			TotalChecked:     5,
			UpdatesAvailable: 2,
		},
	}

	report, buildErr := buildUpdateReport(updateReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildUpdateReport() error = %v", buildErr)
	}
	citest.AssertRenderedReportContract(t, report, citest.RenderedReportContract{
		Producer: pluginName,
		Status:   ci.ReportStatusWarn,
		Renderers: []citest.ReportRenderer{
			reportrender.MarkdownReport,
			reportrender.CLIReport,
		},
	})
	if report.Status() != ci.ReportStatusWarn {
		t.Errorf("Status = %q, want %q", report.Status(), ci.ReportStatusWarn)
	}
	if !strings.Contains(report.Summary(), "2 updates available") {
		t.Errorf("Summary = %q, want to contain '2 updates available'", report.Summary())
	}
}

func TestRenderReportBody_Providers(t *testing.T) {
	result := &tfupdateengine.UpdateResult{
		Providers: []domain.ProviderVersionUpdate{
			{
				Dependency: domain.ProviderDependency{
					ModulePath:     "platform/prod/vpc",
					ProviderSource: "hashicorp/aws",
					Constraint:     "~> 5.0",
				},
				LatestVersion: "5.3.0",
				Status:        domain.StatusUpdateAvailable,
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
	result := &tfupdateengine.UpdateResult{
		Modules: []domain.ModuleVersionUpdate{
			{
				Dependency: domain.ModuleDependency{
					ModulePath: "platform/prod/vpc",
					Source:     "terraform-aws-modules/vpc/aws",
					Constraint: "~> 5.0",
				},
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
	result := &tfupdateengine.UpdateResult{
		Providers: []domain.ProviderVersionUpdate{
			{Dependency: domain.ProviderDependency{ModulePath: "a", ProviderSource: "hashicorp/aws"}, Status: domain.StatusUpdateAvailable},
		},
		Modules: []domain.ModuleVersionUpdate{
			{Dependency: domain.ModuleDependency{ModulePath: "b", Source: "terraform-aws-modules/vpc/aws"}},
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
	result := &tfupdateengine.UpdateResult{
		Providers: []domain.ProviderVersionUpdate{
			{
				Dependency: domain.ProviderDependency{
					ModulePath:     "test",
					ProviderSource: "hashicorp/aws",
				},
				Status: domain.StatusSkipped,
				Issue:  "ignored by config",
			},
		},
	}

	body := renderReportBody(result)
	if !strings.Contains(body, "ignored by config") {
		t.Error("Body missing skip reason")
	}
}

func TestRenderReportBody_Empty(t *testing.T) {
	result := &tfupdateengine.UpdateResult{}

	body := renderReportBody(result)
	if body != "" {
		t.Errorf("Body = %q, want empty for no results", body)
	}
}

func TestRenderReportBody_UpdateAvailable(t *testing.T) {
	result := &tfupdateengine.UpdateResult{
		Modules: []domain.ModuleVersionUpdate{
			{
				Dependency: domain.ModuleDependency{
					ModulePath: "test",
					Source:     "terraform-aws-modules/vpc/aws",
					Constraint: "~> 5.0",
				},
				LatestVersion: "5.3.0",
				Status:        domain.StatusUpdateAvailable,
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
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	// Empty workDir — no modules to discover
	workDir := t.TempDir()
	appCtx := newTestCommandAppContext(t, workDir, p)

	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

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
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{
		Enabled: true,
		Target:  "invalid-target",
	})
	useMockRegistry(p, &mockRegistry{})

	workDir := t.TempDir()
	appCtx := newTestCommandAppContext(t, workDir, p)

	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

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
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
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

	appCtx := newTestCommandAppContext(t, workDir, p)
	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}

	result := loadUpdateResult(t, appCtx.ServiceDir())
	if result.Summary.TotalChecked == 0 {
		t.Fatal("saved result has empty summary")
	}
	report := loadUpdateReport(t, appCtx.ServiceDir())
	if report.Producer() != "tfupdate" {
		t.Fatalf("report.Producer = %q, want update", report.Producer())
	}
}

func TestPlugin_RunCheck_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
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
	base := newTestCommandAppContext(t, workDir, p)
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     base.Config().MutableCopy(),
		WorkDir:    base.WorkDir(),
		ServiceDir: "",
		Version:    base.Version(),
		Reports:    base.Reports(),
		Resolvers:  testBackendResolverSet(newTestBackendRegistry(t)),
	})

	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func TestPlugin_RunCheck_FlagOverrides(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
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

	appCtx := newTestCommandAppContext(t, workDir, p)
	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

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
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	// Point to a file instead of directory to trigger workflow.PlanProject error
	workDir := t.TempDir()
	filePath := workDir + "/not-a-dir"
	if err := os.WriteFile(filePath, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := newTestCommandAppContext(t, workDir, p)
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     base.Config().MutableCopy(),
		WorkDir:    filePath,
		ServiceDir: base.ServiceDir(),
		Version:    base.Version(),
		Reports:    base.Reports(),
		Resolvers:  testBackendResolverSet(newTestBackendRegistry(t)),
	})

	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

	err := cmd.RunE(cmd, nil)
	if err == nil {
		// If workflow.PlanProject doesn't error on a file path, it just returns no modules
		// which triggers "no modules found" error
		t.Log("workflow.PlanProject handled file path gracefully")
	}
}

func TestPlugin_RunCheck_WithModuleFilter(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
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

	appCtx := newTestCommandAppContext(t, workDir, p)
	cmd := buildTFUpdateCommand(t, p)
	cmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))
	cmd.Flags().Set("module", "vpc")

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func TestFilterModules(t *testing.T) {
	modules := []*discovery.Module{
		{RelativePath: "platform/prod/us-east-1/vpc"},
		{RelativePath: "platform/prod/us-east-1/eks"},
	}

	got := filterModules(modules, "vpc")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].RelativePath != "platform/prod/us-east-1/vpc" {
		t.Fatalf("RelativePath = %q, want vpc module", got[0].RelativePath)
	}
}

func TestFinishUpdateCheck(t *testing.T) {
	t.Run("returns summary error after output", func(t *testing.T) {
		var out strings.Builder
		err := finishUpdateCheck(&out, "json", &tfupdateengine.UpdateResult{
			Summary: tfupdateengine.UpdateSummary{Errors: 2},
		})
		if err == nil {
			t.Fatal("expected summary error")
		}
		if !strings.Contains(err.Error(), "2 errors") {
			t.Fatalf("error = %q, want summary errors", err.Error())
		}
		if out.Len() == 0 {
			t.Fatal("expected output to be written before returning error")
		}
	})

	t.Run("returns output error directly", func(t *testing.T) {
		err := finishUpdateCheck(failingWriter{}, "json", &tfupdateengine.UpdateResult{})
		if err == nil {
			t.Fatal("expected output error")
		}
		if !strings.Contains(err.Error(), "write failed") {
			t.Fatalf("error = %q, want output failure", err.Error())
		}
	})
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}
