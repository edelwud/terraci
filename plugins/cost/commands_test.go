package cost

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	cmds := p.Commands(appCtx)
	if len(cmds) != 1 {
		t.Fatalf("Commands() returned %d commands, want 1", len(cmds))
	}

	cmd := cmds[0]
	if cmd.Use != "cost" {
		t.Errorf("command.Use = %q, want %q", cmd.Use, "cost")
	}

	moduleFlag := cmd.Flags().Lookup("module")
	if moduleFlag == nil {
		t.Error("missing --module flag")
	}
	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("missing --output flag")
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

func TestPlugin_RunEstimation_NoPlanFiles(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
	})

	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)

	err := p.runEstimationWithWriter(context.Background(), appCtx, "", "text", io.Discard)
	if err == nil {
		t.Fatal("expected error when no plan.json files exist")
	}
	if !strings.Contains(err.Error(), "no plan.json") {
		t.Errorf("error = %q, want to contain 'no plan.json'", err.Error())
	}
}

func TestPlugin_RunEstimation_InvalidConfig(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
		BlobCache: &model.BlobCacheConfig{
			TTL: "invalid-duration",
		},
	})

	workDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	writePlanJSON(t, moduleDir, "")

	appCtx := newTestAppContext(t, workDir)

	err := p.runEstimationWithWriter(context.Background(), appCtx, "", "text", io.Discard)
	if err == nil {
		t.Fatal("expected error when config is invalid")
	}
	if !strings.Contains(err.Error(), "invalid cost configuration") {
		t.Errorf("error = %q, want to contain 'invalid cost configuration'", err.Error())
	}
}

func TestPlugin_RunEstimation_Success(t *testing.T) {
	p := newTestPlugin(t)

	enablePlugin(t, p, &model.CostConfig{
		Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
	})

	workDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	writePlanJSON(t, moduleDir, testPlanEC2)

	appCtx := newTestAppContext(t, workDir)
	runtime := newRuntimeWithEstimator(newTestEstimator(t))

	err := runEstimationUseCase(context.Background(), appCtx, runtime, "", "text", io.Discard)
	if err != nil {
		t.Fatalf("runEstimation() error = %v", err)
	}

	result := loadEstimateResult(t, appCtx.ServiceDir())
	if len(result.Modules) != 1 {
		t.Errorf("modules count = %d, want 1", len(result.Modules))
	}
	if result.TotalAfter <= 0 {
		t.Errorf("TotalAfter = %.4f, want > 0", result.TotalAfter)
	}
	if result.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", result.Currency)
	}

	report := loadCostReport(t, appCtx.ServiceDir())
	if report.Plugin != "cost" {
		t.Errorf("report.Plugin = %q, want %q", report.Plugin, "cost")
	}
	if report.Status != ci.ReportStatusPass {
		t.Errorf("report.Status = %q, want %q", report.Status, ci.ReportStatusPass)
	}
}

func TestPlugin_RunEstimation_ModuleFilter(t *testing.T) {
	p := newTestPlugin(t)

	enablePlugin(t, p, &model.CostConfig{
		Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
	})

	workDir := t.TempDir()
	// Create two modules
	vpcDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	eksDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "eks")
	writePlanJSON(t, vpcDir, testPlanEC2)
	writePlanJSON(t, eksDir, testPlanEC2)

	appCtx := newTestAppContext(t, workDir)
	runtime := newRuntimeWithEstimator(newTestEstimator(t))

	// Filter to only VPC module
	err := runEstimationUseCase(context.Background(), appCtx, runtime, "platform/prod/us-east-1/vpc", "text", io.Discard)
	if err != nil {
		t.Fatalf("runEstimation() error = %v", err)
	}

	// Verify only one module was estimated
	result := loadEstimateResult(t, appCtx.ServiceDir())
	if len(result.Modules) != 1 {
		t.Errorf("modules count = %d, want 1 (filter should select only vpc)", len(result.Modules))
	}
}

func TestPlugin_RunEstimation_JSONOutput(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	// Capture JSON output via the io.Writer parameter
	var buf strings.Builder
	err := outputResult(&buf, appCtx.WorkDir(), "json", &model.EstimateResult{
		Modules: []model.ModuleCost{
			{
				ModuleID:   "test/module",
				ModulePath: "/tmp/test",
				Region:     "us-east-1",
				AfterCost:  7.592,
			},
		},
		TotalAfter: 7.592,
		Currency:   "USD",
	})
	if err != nil {
		t.Fatalf("outputResult(json) error = %v", err)
	}

	// Verify output is valid JSON
	var parsed model.EstimateResult
	if jsonErr := json.Unmarshal([]byte(buf.String()), &parsed); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", jsonErr, buf.String())
	}
	if parsed.TotalAfter != 7.592 {
		t.Errorf("TotalAfter = %.3f, want 7.592", parsed.TotalAfter)
	}
	if parsed.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", parsed.Currency)
	}
}

func TestPlugin_RunEstimation_TextOutput(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	output := captureTextOutput(t, func() {
		err := outputResult(io.Discard, appCtx.WorkDir(), "text", &model.EstimateResult{
			Modules: []model.ModuleCost{
				{
					ModuleID:   "platform/prod/us-east-1/vpc",
					ModulePath: "/tmp/platform/prod/us-east-1/vpc",
					Region:     "us-east-1",
					AfterCost:  10.50,
				},
			},
			TotalAfter: 10.50,
			Currency:   "USD",
		})
		if err != nil {
			t.Fatalf("outputResult(text) error = %v", err)
		}
	})
	if !strings.Contains(output, "vpc") {
		t.Fatalf("text output = %q, want to contain module name", output)
	}
	if !strings.Contains(output, "summary") {
		t.Fatalf("text output = %q, want summary block", output)
	}
	if !strings.Contains(output, "monthly") {
		t.Fatalf("text output = %q, want to contain monthly field", output)
	}
}

func TestPlugin_OutputResult_TextOutput_UsesLoggerNotWriter(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	var buf strings.Builder
	output := captureTextOutput(t, func() {
		err := outputResult(&buf, appCtx.WorkDir(), "text", &model.EstimateResult{
			Modules: []model.ModuleCost{
				{
					ModuleID:   "platform/prod/us-east-1/vpc",
					ModulePath: "/tmp/platform/prod/us-east-1/vpc",
					Region:     "us-east-1",
					AfterCost:  10.50,
				},
			},
			TotalAfter: 10.50,
			Currency:   "USD",
		})
		if err != nil {
			t.Fatalf("outputResult(text) error = %v", err)
		}
	})
	if buf.Len() != 0 {
		t.Fatalf("writer output = %q, want empty for logger-backed text mode", buf.String())
	}
	if !strings.Contains(output, "summary") {
		t.Fatalf("captured logger output = %q, want summary block", output)
	}
}

func TestPlugin_OutputResult_TextOutput_SkipsZeroCostModule(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	output := captureTextOutput(t, func() {
		err := outputResult(io.Discard, appCtx.WorkDir(), "text", &model.EstimateResult{
			Modules: []model.ModuleCost{
				{
					ModuleID:   "cdp/infra/eu-central-1/eks",
					ModulePath: "/tmp/cdp/infra/eu-central-1/eks",
					Region:     "eu-central-1",
				},
			},
			Currency: "USD",
		})
		if err != nil {
			t.Fatalf("outputResult(text) error = %v", err)
		}
	})
	if strings.Contains(output, "eks") {
		t.Fatalf("text output = %q, want zero-cost module to be hidden", output)
	}
}

func TestPlugin_OutputResult_TextOutput_SkipsZeroCostSubmoduleHeader(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	output := captureTextOutput(t, func() {
		err := outputResult(io.Discard, appCtx.WorkDir(), "text", &model.EstimateResult{
			Modules: []model.ModuleCost{
				{
					ModuleID:   "cdp/infra/eu-central-1/eks",
					ModulePath: "/tmp/cdp/infra/eu-central-1/eks",
					Region:     "eu-central-1",
					AfterCost:  1.23,
					Resources: []model.ResourceCost{
						{
							Address:    "module.velero_irsa_role.aws_iam_role.this",
							ModuleAddr: "module.velero_irsa_role",
							Status:     model.ResourceEstimateStatusExact,
						},
					},
				},
			},
			TotalAfter: 1.23,
			Currency:   "USD",
		})
		if err != nil {
			t.Fatalf("outputResult(text) error = %v", err)
		}
	})
	if strings.Contains(output, "module.velero_irsa_role") {
		t.Fatalf("text output = %q, want zero-cost submodule header hidden", output)
	}
}

func TestBuildCostReport(t *testing.T) {
	result := &model.EstimateResult{
		Modules: []model.ModuleCost{
			{
				ModuleID:   "platform/prod/vpc",
				ModulePath: "/tmp/vpc",
				Region:     "us-east-1",
				BeforeCost: 5.25,
				AfterCost:  10.50,
				DiffCost:   5.25,
			},
			{
				ModuleID:   "platform/prod/broken",
				ModulePath: "/tmp/broken",
				Error:      "parse error",
			},
		},
		TotalBefore: 5.25,
		TotalAfter:  10.50,
		TotalDiff:   5.25,
		Currency:    "USD",
	}

	report := buildCostReport(result)

	if report.Plugin != "cost" {
		t.Errorf("Plugin = %q, want %q", report.Plugin, "cost")
	}
	if report.Title != "Cost Estimation" {
		t.Errorf("Title = %q, want %q", report.Title, "Cost Estimation")
	}
	if !strings.Contains(report.Summary, "2 modules") {
		t.Errorf("Summary = %q, want to contain '2 modules'", report.Summary)
	}

	// Body should contain markdown table
	if !strings.Contains(report.Body, "| Module |") {
		t.Error("Body missing markdown table header")
	}
	if !strings.Contains(report.Body, "/tmp/vpc") {
		t.Error("Body missing successful module path")
	}
	if !strings.Contains(report.Body, "/tmp/broken") {
		t.Error("Body should contain error module")
	}

	if report.Status != ci.ReportStatusWarn {
		t.Errorf("Status = %q, want %q when report has errors", report.Status, ci.ReportStatusWarn)
	}

	if len(report.Modules) != 2 {
		t.Fatalf("Modules count = %d, want 2 (including errored module)", len(report.Modules))
	}

	m := report.Modules[0]
	if !m.HasCost {
		t.Error("Module.HasCost should be true")
	}
	if m.CostAfter != 10.50 {
		t.Errorf("Module.CostAfter = %.2f, want 10.50", m.CostAfter)
	}
	if m.CostDiff != 5.25 {
		t.Errorf("Module.CostDiff = %.2f, want 5.25", m.CostDiff)
	}
	if report.Modules[1].Error != "parse error" {
		t.Errorf("Error module error = %q, want %q", report.Modules[1].Error, "parse error")
	}
}

func TestBuildCostReport_IncludesPrefetchWarnings(t *testing.T) {
	t.Parallel()

	result := &model.EstimateResult{
		Modules: []model.ModuleCost{
			{
				ModuleID:   "platform/prod/vpc",
				ModulePath: "/tmp/vpc",
				AfterCost:  10.50,
				Resources: []model.ResourceCost{
					{
						Address: "aws_lambda_function.worker",
						Status:  model.ResourceEstimateStatusUsageEstimated,
					},
					{
						Address: "aws_sqs_queue.jobs",
						Status:  model.ResourceEstimateStatusUsageUnknown,
					},
				},
			},
		},
		TotalAfter:     10.50,
		Currency:       "USD",
		UsageEstimated: 1,
		UsageUnknown:   1,
		PrefetchWarnings: []model.PrefetchDiagnostic{
			{
				Kind:         "lookup-failed",
				ResourceType: "aws_db_instance",
				Address:      "aws_db_instance.db",
				Detail:       "missing instance_class",
			},
		},
	}

	report := buildCostReport(result)
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if !strings.Contains(report.Body, "Prefetch warnings") {
		t.Fatalf("report body = %q, want prefetch warnings section", report.Body)
	}
	if !strings.Contains(report.Body, "Resource statuses") {
		t.Fatalf("report body = %q, want resource statuses section", report.Body)
	}
	if !strings.Contains(report.Summary, "usage estimated: 1") {
		t.Fatalf("summary = %q, want usage estimated count", report.Summary)
	}
	if !strings.Contains(report.Summary, "usage unknown: 1") {
		t.Fatalf("summary = %q, want usage unknown count", report.Summary)
	}
	if !strings.Contains(report.Body, "aws_db_instance.db") {
		t.Fatalf("report body = %q, want warning address", report.Body)
	}
}

func TestBuildCostReport_Empty(t *testing.T) {
	result := &model.EstimateResult{
		Modules:  []model.ModuleCost{},
		Currency: "USD",
	}

	report := buildCostReport(result)

	if report.Plugin != "cost" {
		t.Errorf("Plugin = %q, want %q", report.Plugin, "cost")
	}
	if !strings.Contains(report.Summary, "0 modules") {
		t.Errorf("Summary = %q, want to contain '0 modules'", report.Summary)
	}
	if len(report.Modules) != 0 {
		t.Errorf("Modules count = %d, want 0", len(report.Modules))
	}
}

func TestBuildCostReport_AllErrors(t *testing.T) {
	result := &model.EstimateResult{
		Modules: []model.ModuleCost{
			{ModuleID: "a", Error: "fail1"},
			{ModuleID: "b", Error: "fail2"},
		},
		Currency: "USD",
	}

	report := buildCostReport(result)

	if len(report.Modules) != 2 {
		t.Errorf("Modules count = %d, want 2 (all errors should still be visible)", len(report.Modules))
	}
	if report.Status != ci.ReportStatusWarn {
		t.Errorf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if !strings.Contains(report.Body, "| Module |") {
		t.Error("Body missing table header")
	}
	if !strings.Contains(report.Body, "fail1") || !strings.Contains(report.Body, "fail2") {
		t.Error("Body should contain module errors")
	}
}

func TestBuildCostReport_EscapesMarkdownTableCells(t *testing.T) {
	t.Parallel()

	result := &model.EstimateResult{
		Modules: []model.ModuleCost{
			{
				ModuleID:   "a",
				ModulePath: "/tmp/with|pipe\\slash",
				Error:      "line1\nline2 | detail",
			},
		},
		Currency: "USD",
	}

	report := buildCostReport(result)
	if !strings.Contains(report.Body, `/tmp/with\|pipe\\slash`) {
		t.Fatalf("report body = %q, want escaped module path", report.Body)
	}
	if !strings.Contains(report.Body, `line1<br>line2 \| detail`) {
		t.Fatalf("report body = %q, want escaped error note", report.Body)
	}
}
