package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/cliout"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/plugins/internal/reportrender"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func buildPolicyCommand(t *testing.T, p *Plugin) *cobra.Command {
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

func TestBuildPolicyReport_WithFailures(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  2,
		PassedModules: 1,
		FailedModules: 1,
		Results: []policyengine.Result{
			{Module: "platform/prod/vpc"},
			{
				Module: "platform/prod/eks",
				Failures: []policyengine.Finding{
					{Namespace: "terraform", Message: "public endpoint forbidden"},
				},
			},
		},
	}

	report, buildErr := buildPolicyReport(policyReportRequest{Summary: summary})
	if buildErr != nil {
		t.Fatalf("buildPolicyReport() error = %v", buildErr)
	}
	citest.AssertRenderedReportContract(t, report, citest.RenderedReportContract{
		Producer: pluginName,
		Status:   ci.ReportStatusFail,
		Renderers: []citest.ReportRenderer{
			reportrender.MarkdownReport,
			reportrender.CLIReport,
		},
	})
	if report.Status != ci.ReportStatusFail {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusFail)
	}
	if !strings.Contains(report.Summary, "2 modules") {
		t.Fatalf("Summary = %q, want module count", report.Summary)
	}
	if len(report.Sections) != 1 {
		t.Fatalf("expected one findings section")
	}
	rendered, err := ci.DecodeRenderSection(report.Sections[0])
	if err != nil {
		t.Fatalf("decode rendered section: %v", err)
	}
	blocks := rendered.Blocks()
	row := blocks[0].Table().Rows()[0].Cells()
	if row[3].Text() != "public endpoint forbidden" {
		t.Fatalf("unexpected finding row: %+v", row)
	}
	if row[1].Status() != ci.ReportStatusFail {
		t.Fatalf("finding severity = %q, want fail", row[1].Status())
	}
}

func TestBuildPolicyReport_WithWarnings(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  1,
		WarnedModules: 1,
		Results: []policyengine.Result{
			{
				Module: "platform/prod/app",
				Warnings: []policyengine.Finding{
					{Namespace: "compliance", Message: "tag missing"},
				},
			},
		},
	}

	report, buildErr := buildPolicyReport(policyReportRequest{Summary: summary})
	if buildErr != nil {
		t.Fatalf("buildPolicyReport() error = %v", buildErr)
	}
	citest.AssertRenderedReportContract(t, report, citest.RenderedReportContract{
		Producer: pluginName,
		Status:   ci.ReportStatusWarn,
		Renderers: []citest.ReportRenderer{
			reportrender.MarkdownReport,
			reportrender.CLIReport,
		},
	})
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if len(report.Sections) != 1 {
		t.Fatalf("expected one findings section")
	}
	rendered, err := ci.DecodeRenderSection(report.Sections[0])
	if err != nil {
		t.Fatalf("decode rendered section: %v", err)
	}
	blocks := rendered.Blocks()
	row := blocks[0].Table().Rows()[0].Cells()
	if row[3].Text() != "tag missing" {
		t.Fatalf("unexpected finding row: %+v", row)
	}
	if row[1].Status() != ci.ReportStatusWarn {
		t.Fatalf("finding severity = %q, want warn", row[1].Status())
	}
}

func TestOutputResult_JSON(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules: 1,
		Results: []policyengine.Result{
			{Module: "platform/prod/app"},
		},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, cliout.FormatJSON, summary, false)
	if err != nil {
		t.Fatalf("outputResult(json) error = %v", err)
	}

	var parsed policyengine.Summary
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid json: %v", err)
	}
	if parsed.TotalModules != 1 {
		t.Fatalf("TotalModules = %d, want 1", parsed.TotalModules)
	}
}

func TestOutputResult_JSONBlocks(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  1,
		FailedModules: 1,
		TotalFailures: 1,
		Results: []policyengine.Result{
			{Module: "platform/prod/app", Failures: []policyengine.Finding{{Message: "denied"}}},
		},
	}

	var buf bytes.Buffer
	err := outputResult(&buf, cliout.FormatJSON, summary, true)
	if err == nil {
		t.Fatal("outputResult(json) error = nil, want blocking error")
	}
	if !strings.Contains(err.Error(), "policy check failed with 1 failures") {
		t.Fatalf("error = %q, want blocking failure count", err.Error())
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("output is not valid json: %s", buf.String())
	}
}

func TestOutputResult_NilSummary(t *testing.T) {
	err := outputResult(&bytes.Buffer{}, cliout.FormatText, nil, false)
	if err == nil {
		t.Fatal("outputResult() error = nil, want nil summary error")
	}
	if !strings.Contains(err.Error(), "policy summary is nil") {
		t.Fatalf("error = %q, want nil summary message", err.Error())
	}
}

func TestBuildPolicyReport_NilSummary(t *testing.T) {
	_, err := buildPolicyReport(policyReportRequest{})
	if err == nil {
		t.Fatal("buildPolicyReport() error = nil, want nil summary error")
	}
	if !strings.Contains(err.Error(), "policy summary is nil") {
		t.Fatalf("error = %q, want nil summary message", err.Error())
	}
}

func TestOutputText_UsesLogger(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  1,
		WarnedModules: 1,
		Results: []policyengine.Result{
			{
				Module: "platform/prod/app",
				Warnings: []policyengine.Finding{
					{Namespace: "compliance", Message: "tag missing"},
				},
			},
		},
	}

	output := capturePolicyTextOutput(t, func() {
		if err := outputText(summary, false); err != nil {
			t.Fatalf("outputText() error = %v", err)
		}
	})
	if !strings.Contains(output, "summary") {
		t.Fatalf("output = %q, want summary header", output)
	}
	if !strings.Contains(output, "platform/prod/app") {
		t.Fatalf("output = %q, want module path", output)
	}
	if !strings.Contains(output, "tag missing") {
		t.Fatalf("output = %q, want warning message", output)
	}
}

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin()

	cmd := buildPolicyCommand(t, p)
	checkCmd, _, err := cmd.Find([]string{"check"})
	if err != nil {
		t.Fatalf("Find(check) error = %v", err)
	}
	formatFlag := checkCmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("missing --format flag on policy check")
	}
	if formatFlag.DefValue != "text" {
		t.Fatalf("check --format default = %q, want text", formatFlag.DefValue)
	}
	if checkCmd.Flags().Lookup("output") != nil {
		t.Fatal("policy check should not expose legacy --output flag")
	}
	pullCmd, _, err := cmd.Find([]string{"pull"})
	if err != nil {
		t.Fatalf("Find(pull) error = %v", err)
	}
	if pullCmd.Flags().Lookup("cache-dir") == nil {
		t.Fatal("missing --cache-dir flag on policy pull")
	}
	if pullCmd.Flags().Lookup("output") != nil {
		t.Fatal("policy pull should not expose legacy --output flag")
	}
}

func TestParseOutputFormat_RejectsUnknown(t *testing.T) {
	_, err := cliout.ParseFormat("yaml")
	if err == nil {
		t.Fatal("ParseFormat() error = nil, want invalid format error")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Fatalf("error = %q, want unsupported format message", err.Error())
	}
}

func TestPlugin_Commands_RunE_NotConfigured(t *testing.T) {
	p := newTestPlugin()
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	cmd := buildPolicyCommand(t, p)
	checkCmd, _, err := cmd.Find([]string{"check"})
	if err != nil {
		t.Fatalf("Find(check) error = %v", err)
	}
	checkCmd.SetContext(plugintest.BindCommandPlugin(context.Background(), t, appCtx, pluginName, p))

	err = checkCmd.RunE(checkCmd, nil)
	if err == nil {
		t.Fatal("expected error for unconfigured plugin")
	}
	if !strings.Contains(err.Error(), "extensions.policy.enabled: true") {
		t.Fatalf("error = %q, want actionable enablement hint", err.Error())
	}
}
