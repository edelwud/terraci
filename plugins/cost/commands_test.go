package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/cliout"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/internal/reportrender"
)

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin(t)

	cmds := p.Commands()
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

func decodeCostSection(t *testing.T, report *ci.Report) ci.RenderSection {
	t.Helper()
	citest.AssertRenderedReportContract(t, report, citest.RenderedReportContract{
		Producer: pluginName,
		Renderers: []citest.ReportRenderer{
			reportrender.MarkdownReport,
			reportrender.CLIReport,
		},
	})
	if len(report.Sections) != 1 {
		t.Fatalf("Sections count = %d, want 1", len(report.Sections))
	}
	section := report.Sections[0]
	if section.Kind() != ci.ReportSectionKindRendered {
		t.Fatalf("section kind = %q, want %q", section.Kind(), ci.ReportSectionKindRendered)
	}
	rendered, err := ci.DecodeRenderSection(section)
	if err != nil {
		t.Fatalf("decode rendered section payload: %v", err)
	}
	return rendered
}

func renderTableRows(t *testing.T, section ci.RenderSection) [][]string {
	t.Helper()
	blocks := section.Blocks()
	for i := range blocks {
		if blocks[i].Kind() == ci.RenderBlockKindTable && blocks[i].Table() != nil {
			return renderRows(blocks[i].Table().Rows())
		}
	}
	t.Fatal("render section has no table block")
	return nil
}

func renderTableColumns(t *testing.T, section ci.RenderSection) []string {
	t.Helper()
	blocks := section.Blocks()
	for i := range blocks {
		if blocks[i].Kind() == ci.RenderBlockKindTable && blocks[i].Table() != nil {
			columns := blocks[i].Table().Columns()
			result := make([]string, 0, len(columns))
			for _, column := range columns {
				result = append(result, renderValueString(column.Title()))
			}
			return result
		}
	}
	t.Fatal("render section has no table block")
	return nil
}

func renderTableRowsOrNil(section ci.RenderSection) [][]string {
	blocks := section.Blocks()
	for i := range blocks {
		if blocks[i].Kind() == ci.RenderBlockKindTable && blocks[i].Table() != nil {
			return renderRows(blocks[i].Table().Rows())
		}
	}
	return nil
}

func renderListItems(section ci.RenderSection, title string) []string {
	blocks := section.Blocks()
	for i := range blocks {
		if blocks[i].Kind() == ci.RenderBlockKindList && blocks[i].Title() == title {
			items := blocks[i].Items()
			result := make([]string, 0, len(items))
			for _, item := range items {
				result = append(result, renderValueString(item))
			}
			return result
		}
	}
	return nil
}

func renderTextBlocks(section ci.RenderSection) []string {
	var texts []string
	blocks := section.Blocks()
	for i := range blocks {
		if blocks[i].Kind() == ci.RenderBlockKindText {
			texts = append(texts, renderValueString(blocks[i].Text()))
		}
	}
	return texts
}

func renderRows(rows []ci.RenderRow) [][]string {
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := row.Cells()
		rendered := make([]string, 0, len(cells))
		for _, cell := range cells {
			rendered = append(rendered, renderValueString(cell))
		}
		result = append(result, rendered)
	}
	return result
}

func renderValueString(value ci.RenderValue) string {
	switch value.Kind() {
	case ci.RenderValueKindText, ci.RenderValueKindCode, ci.RenderValueKindLabel, ci.RenderValueKindModulePath, ci.RenderValueKindResourceAddress:
		return value.Text()
	case ci.RenderValueKindStatus:
		return reportrender.StatusLabel(value.Status())
	case ci.RenderValueKindMoney:
		return renderMoneyString(value.Amount(), value.Unit(), false)
	case ci.RenderValueKindMoneyDelta:
		return renderMoneyString(value.Amount(), value.Unit(), true)
	case ci.RenderValueKindInline:
		var sb strings.Builder
		for _, part := range value.Parts() {
			sb.WriteString(renderValueString(part))
		}
		return sb.String()
	default:
		return ""
	}
}

func renderMoneyString(amount float64, unit ci.RenderMoneyUnit, signed bool) string {
	prefix := ""
	value := amount
	if signed {
		switch {
		case math.Abs(amount) < 0.0000001:
			value = 0
		case amount > 0:
			prefix = "+"
		default:
			prefix = "-"
			value = -amount
		}
	}
	rendered := prefix + renderMoneyAmountString(value)
	if unit != "" {
		rendered += "/" + string(unit)
	}
	return rendered
}

func renderMoneyAmountString(amount float64) string {
	if math.Abs(amount) < 0.0000001 {
		return "$0"
	}
	if amount < 0 {
		return "-" + renderMoneyAmountString(-amount)
	}
	if amount < 0.01 {
		return "<$0.01"
	}
	if amount >= 1000 {
		return fmt.Sprintf("$%.0f", amount)
	}
	if amount >= 1 {
		return fmt.Sprintf("$%.2f", amount)
	}
	return fmt.Sprintf("$%.4f", amount)
}

func TestPlugin_Commands_RunE_NotConfigured(t *testing.T) {
	p := newTestPlugin(t)
	base := newTestAppContext(t, t.TempDir())
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:        base.Config().MutableCopy(),
		WorkDir:       base.WorkDir(),
		ServiceDir:    base.ServiceDir(),
		Version:       base.Version(),
		Reports:       base.Reports(),
		CommandLookup: plugintest.StaticCommandLookup{pluginName: p},
	})

	cmds := p.Commands()
	cmd := cmds[0]
	cmd.SetContext(plugin.WithContext(context.Background(), appCtx))

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

	estimation, err := runEstimationUseCase(context.Background(), appCtx, runtime, estimateRequest{})
	if err != nil {
		t.Fatalf("runEstimation() error = %v", err)
	}
	if err := saveArtifacts(context.Background(), appCtx, estimation.Result, estimation.PlanResults); err != nil {
		t.Fatalf("saveArtifacts() error = %v", err)
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
	if report.Producer != "cost" {
		t.Errorf("report.Producer = %q, want %q", report.Producer, "cost")
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
	estimation, err := runEstimationUseCase(context.Background(), appCtx, runtime, estimateRequest{ModulePath: "platform/prod/us-east-1/vpc"})
	if err != nil {
		t.Fatalf("runEstimation() error = %v", err)
	}

	// Verify only one module was estimated
	if len(estimation.Result.Modules) != 1 {
		t.Errorf("modules count = %d, want 1 (filter should select only vpc)", len(estimation.Result.Modules))
	}
}

func TestPlugin_RunEstimation_JSONOutput(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	// Capture JSON output via the io.Writer parameter
	var buf strings.Builder
	err := outputResult(&buf, appCtx.WorkDir(), cliout.FormatJSON, &model.EstimateResult{
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
		err := outputResult(io.Discard, appCtx.WorkDir(), cliout.FormatText, &model.EstimateResult{
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
		err := outputResult(&buf, appCtx.WorkDir(), cliout.FormatText, &model.EstimateResult{
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
		err := outputResult(io.Discard, appCtx.WorkDir(), cliout.FormatText, &model.EstimateResult{
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
		err := outputResult(io.Discard, appCtx.WorkDir(), cliout.FormatText, &model.EstimateResult{
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

	report, buildErr := buildCostReport(costReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildCostReport() error = %v", buildErr)
	}

	if report.Producer != "cost" {
		t.Errorf("Plugin = %q, want %q", report.Producer, "cost")
	}
	if report.Title != costReportTitle {
		t.Errorf("Title = %q, want %q", report.Title, costReportTitle)
	}
	if !strings.Contains(report.Summary, "2 modules") {
		t.Errorf("Summary = %q, want to contain '2 modules'", report.Summary)
	}

	if report.Status != ci.ReportStatusWarn {
		t.Errorf("Status = %q, want %q when report has errors", report.Status, ci.ReportStatusWarn)
	}

	section := decodeCostSection(t, report)
	columns := renderTableColumns(t, section)
	if strings.Join(columns, ",") != "Module,Before,After,Diff" {
		t.Fatalf("columns = %v, want cost columns without Notes", columns)
	}
	rows := renderTableRows(t, section)
	if len(rows) != 1 {
		t.Fatalf("Rows count = %d, want 1 cost row", len(rows))
	}

	m := rows[0]
	if m[2] != "$10.50/mo" {
		t.Errorf("Module.After = %q, want $10.50/mo", m[2])
	}
	if m[3] != "+$5.25/mo" {
		t.Errorf("Module.Diff = %q, want +$5.25/mo", m[3])
	}
	errors := renderListItems(section, "Estimation errors")
	if len(errors) != 1 || errors[0] != "/tmp/broken: parse error" {
		t.Fatalf("Estimation errors = %v, want broken module error", errors)
	}
	texts := renderTextBlocks(section)
	if len(texts) == 0 || texts[len(texts)-1] != "Total: $5.25/mo -> $10.50/mo (+$5.25/mo)" {
		t.Fatalf("total blocks = %v, want positive diff total", texts)
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

	report, buildErr := buildCostReport(costReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildCostReport() error = %v", buildErr)
	}
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if !strings.Contains(report.Summary, "usage estimated: 1") {
		t.Fatalf("summary = %q, want usage estimated count", report.Summary)
	}
	if !strings.Contains(report.Summary, "usage unknown: 1") {
		t.Fatalf("summary = %q, want usage unknown count", report.Summary)
	}
	section := decodeCostSection(t, report)
	if len(section.Blocks()) == 0 {
		t.Fatalf("expected render blocks")
	}
	limitations := renderListItems(section, "Limitations")
	if len(limitations) < 3 {
		t.Fatalf("Limitations = %v, want usage and prefetch limitations", limitations)
	}
	if !strings.Contains(strings.Join(limitations, "\n"), "usage-based resources estimated") {
		t.Fatalf("Limitations = %v, want usage estimated item", limitations)
	}
	if !strings.Contains(strings.Join(limitations, "\n"), "lookup-failed") {
		t.Fatalf("Limitations = %v, want prefetch warning item", limitations)
	}
}

func TestBuildCostReport_Empty(t *testing.T) {
	result := &model.EstimateResult{
		Modules:  []model.ModuleCost{},
		Currency: "USD",
	}

	report, buildErr := buildCostReport(costReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildCostReport() error = %v", buildErr)
	}

	if report.Producer != "cost" {
		t.Errorf("Plugin = %q, want %q", report.Producer, "cost")
	}
	if !strings.Contains(report.Summary, "0 modules") {
		t.Errorf("Summary = %q, want to contain '0 modules'", report.Summary)
	}
	section := decodeCostSection(t, report)
	if rows := renderTableRowsOrNil(section); len(rows) != 0 {
		t.Errorf("Rows count = %d, want 0", len(rows))
	}
	texts := renderTextBlocks(section)
	if len(texts) == 0 || texts[len(texts)-1] != "Total: $0/mo" {
		t.Fatalf("total blocks = %v, want zero total without arrow", texts)
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

	report, buildErr := buildCostReport(costReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildCostReport() error = %v", buildErr)
	}

	section := decodeCostSection(t, report)
	if rows := renderTableRowsOrNil(section); len(rows) != 0 {
		t.Errorf("Rows count = %d, want 0 cost rows for all-error report", len(rows))
	}
	if report.Status != ci.ReportStatusWarn {
		t.Errorf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	errors := renderListItems(section, "Estimation errors")
	if len(errors) != 2 || errors[0] != "a: fail1" || errors[1] != "b: fail2" {
		t.Fatalf("Estimation errors = %v, want all module errors", errors)
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

	report, buildErr := buildCostReport(costReportRequest{Result: result})
	if buildErr != nil {
		t.Fatalf("buildCostReport() error = %v", buildErr)
	}
	section := decodeCostSection(t, report)
	errors := renderListItems(section, "Estimation errors")
	if len(errors) != 1 {
		t.Fatalf("Estimation errors = %v, want one escaped module error", errors)
	}
	if !strings.Contains(errors[0], "/tmp/with|pipe\\slash") {
		t.Fatalf("error item = %q, want raw module path preserved", errors[0])
	}
	if !strings.Contains(errors[0], "line1\nline2 | detail") {
		t.Fatalf("error item = %q, want raw error preserved", errors[0])
	}
}

func TestBuildCostTotalBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result *model.EstimateResult
		want   string
	}{
		{
			name:   "all zero",
			result: &model.EstimateResult{},
			want:   "Total: $0/mo",
		},
		{
			name:   "zero diff",
			result: &model.EstimateResult{TotalBefore: 10, TotalAfter: 10},
			want:   "Total: $10.00/mo",
		},
		{
			name:   "positive diff",
			result: &model.EstimateResult{TotalBefore: 5, TotalAfter: 10.5, TotalDiff: 5.5},
			want:   "Total: $5.00/mo -> $10.50/mo (+$5.50/mo)",
		},
		{
			name:   "negative diff",
			result: &model.EstimateResult{TotalBefore: 20, TotalAfter: 7.5, TotalDiff: -12.5},
			want:   "Total: $20.00/mo -> $7.50/mo (-$12.50/mo)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := renderValueString(buildCostTotalBlock(tt.result).Text()); got != tt.want {
				t.Fatalf("buildCostTotalBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}
