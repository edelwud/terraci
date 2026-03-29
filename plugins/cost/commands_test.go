package cost

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
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
	enablePlugin(t, p, &costengine.CostConfig{Enabled: true, CacheDir: t.TempDir()})
	p.estimator = newTestEstimator(t)

	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)

	err := p.runEstimation(context.Background(), appCtx, "", "text")
	if err == nil {
		t.Fatal("expected error when no plan.json files exist")
	}
	if !strings.Contains(err.Error(), "no plan.json") {
		t.Errorf("error = %q, want to contain 'no plan.json'", err.Error())
	}
}

func TestPlugin_RunEstimation_NilEstimator(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &costengine.CostConfig{Enabled: true})
	// Do NOT call Initialize — estimator stays nil

	workDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	writePlanJSON(t, moduleDir, "")

	appCtx := newTestAppContext(t, workDir)

	err := p.runEstimation(context.Background(), appCtx, "", "text")
	if err == nil {
		t.Fatal("expected error when estimator is nil")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error = %q, want to contain 'not initialized'", err.Error())
	}
}

func TestPlugin_RunEstimation_Success(t *testing.T) {
	p := newTestPlugin(t)
	ts := fakePricingServer(t)
	cacheDir := t.TempDir()

	enablePlugin(t, p, &costengine.CostConfig{
		Enabled:  true,
		CacheDir: cacheDir,
	})

	// Create estimator with fake pricing server
	fetcher := &awskit.Fetcher{Client: ts.Client(), BaseURL: ts.URL}
	p.estimator = costengine.NewEstimator(cacheDir, 0, fetcher)

	workDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	writePlanJSON(t, moduleDir, testPlanEC2)

	appCtx := newTestAppContext(t, workDir)

	err := p.runEstimation(context.Background(), appCtx, "", "text")
	if err != nil {
		t.Fatalf("runEstimation() error = %v", err)
	}

	// Verify cost-results.json was saved
	resultsPath := filepath.Join(appCtx.ServiceDir, resultsFile)
	if _, statErr := os.Stat(resultsPath); os.IsNotExist(statErr) {
		t.Error("cost-results.json was not saved to serviceDir")
	} else {
		// Parse and validate the results
		data, readErr := os.ReadFile(resultsPath)
		if readErr != nil {
			t.Fatalf("failed to read cost-results.json: %v", readErr)
		}
		var result costengine.EstimateResult
		if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
			t.Fatalf("failed to parse cost-results.json: %v", jsonErr)
		}
		if len(result.Modules) != 1 {
			t.Errorf("modules count = %d, want 1", len(result.Modules))
		}
		if result.TotalAfter <= 0 {
			t.Errorf("TotalAfter = %.4f, want > 0", result.TotalAfter)
		}
		if result.Currency != "USD" {
			t.Errorf("Currency = %q, want USD", result.Currency)
		}
	}

	// Verify cost-report.json was saved
	reportPath := filepath.Join(appCtx.ServiceDir, "cost-report.json")
	if _, statErr := os.Stat(reportPath); os.IsNotExist(statErr) {
		t.Error("cost-report.json was not saved to serviceDir")
	} else {
		data, readErr := os.ReadFile(reportPath)
		if readErr != nil {
			t.Fatalf("failed to read cost-report.json: %v", readErr)
		}
		var report ci.Report
		if jsonErr := json.Unmarshal(data, &report); jsonErr != nil {
			t.Fatalf("failed to parse cost-report.json: %v", jsonErr)
		}
		if report.Plugin != "cost" {
			t.Errorf("report.Plugin = %q, want %q", report.Plugin, "cost")
		}
		if report.Status != ci.ReportStatusPass {
			t.Errorf("report.Status = %q, want %q", report.Status, ci.ReportStatusPass)
		}
	}
}

func TestPlugin_RunEstimation_ModuleFilter(t *testing.T) {
	p := newTestPlugin(t)
	ts := fakePricingServer(t)
	cacheDir := t.TempDir()

	enablePlugin(t, p, &costengine.CostConfig{
		Enabled:  true,
		CacheDir: cacheDir,
	})

	fetcher := &awskit.Fetcher{Client: ts.Client(), BaseURL: ts.URL}
	p.estimator = costengine.NewEstimator(cacheDir, 0, fetcher)

	workDir := t.TempDir()
	// Create two modules
	vpcDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	eksDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "eks")
	writePlanJSON(t, vpcDir, testPlanEC2)
	writePlanJSON(t, eksDir, testPlanEC2)

	appCtx := newTestAppContext(t, workDir)

	// Filter to only VPC module
	err := p.runEstimation(context.Background(), appCtx, "platform/prod/us-east-1/vpc", "text")
	if err != nil {
		t.Fatalf("runEstimation() error = %v", err)
	}

	// Verify only one module was estimated
	resultsPath := filepath.Join(appCtx.ServiceDir, resultsFile)
	data, readErr := os.ReadFile(resultsPath)
	if readErr != nil {
		t.Fatalf("failed to read cost-results.json: %v", readErr)
	}
	var result costengine.EstimateResult
	if jsonErr := json.Unmarshal(data, &result); jsonErr != nil {
		t.Fatalf("failed to parse cost-results.json: %v", jsonErr)
	}
	if len(result.Modules) != 1 {
		t.Errorf("modules count = %d, want 1 (filter should select only vpc)", len(result.Modules))
	}
}

func TestPlugin_RunEstimation_JSONOutput(t *testing.T) {
	p := newTestPlugin(t)
	ts := fakePricingServer(t)
	cacheDir := t.TempDir()

	enablePlugin(t, p, &costengine.CostConfig{
		Enabled:  true,
		CacheDir: cacheDir,
	})

	fetcher := &awskit.Fetcher{Client: ts.Client(), BaseURL: ts.URL}
	p.estimator = costengine.NewEstimator(cacheDir, 0, fetcher)

	workDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "prod", "us-east-1", "vpc")
	writePlanJSON(t, moduleDir, testPlanEC2)

	appCtx := newTestAppContext(t, workDir)

	// Capture JSON output via the io.Writer parameter
	var buf bytes.Buffer
	err := p.outputResult(&buf, appCtx, "json", &costengine.EstimateResult{
		Modules: []costengine.ModuleCost{
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
	var parsed costengine.EstimateResult
	if jsonErr := json.Unmarshal(buf.Bytes(), &parsed); jsonErr != nil {
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
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	// Text output uses log package — just verify it doesn't error
	var buf bytes.Buffer
	err := p.outputResult(&buf, appCtx, "text", &costengine.EstimateResult{
		Modules: []costengine.ModuleCost{
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
}

func TestBuildCostReport(t *testing.T) {
	result := &costengine.EstimateResult{
		Modules: []costengine.ModuleCost{
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
	if report.Status != ci.ReportStatusPass {
		t.Errorf("Status = %q, want %q", report.Status, ci.ReportStatusPass)
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
	// Error module should be excluded from body table rows
	if strings.Contains(report.Body, "/tmp/broken") {
		t.Error("Body should not contain error module")
	}

	// Modules should exclude error entries
	if len(report.Modules) != 1 {
		t.Fatalf("Modules count = %d, want 1 (error module excluded)", len(report.Modules))
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
}

func TestBuildCostReport_Empty(t *testing.T) {
	result := &costengine.EstimateResult{
		Modules:  []costengine.ModuleCost{},
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
	result := &costengine.EstimateResult{
		Modules: []costengine.ModuleCost{
			{ModuleID: "a", Error: "fail1"},
			{ModuleID: "b", Error: "fail2"},
		},
		Currency: "USD",
	}

	report := buildCostReport(result)

	if len(report.Modules) != 0 {
		t.Errorf("Modules count = %d, want 0 (all have errors)", len(report.Modules))
	}
	// Body should have table header but no data rows
	if !strings.Contains(report.Body, "| Module |") {
		t.Error("Body missing table header")
	}
}
