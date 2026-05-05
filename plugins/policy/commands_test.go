package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestBuildPolicyReport_WithFailures(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  2,
		PassedModules: 1,
		FailedModules: 1,
		Results: []policyengine.Result{
			{Module: "platform/prod/vpc"},
			{
				Module: "platform/prod/eks",
				Failures: []policyengine.Violation{
					{Namespace: "terraform", Message: "public endpoint forbidden"},
				},
			},
		},
	}

	report, buildErr := buildPolicyReport(summary)
	if buildErr != nil {
		t.Fatalf("buildPolicyReport() error = %v", buildErr)
	}
	if report.Status != ci.ReportStatusFail {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusFail)
	}
	if !strings.Contains(report.Summary, "2 modules") {
		t.Fatalf("Summary = %q, want module count", report.Summary)
	}
	if len(report.Sections) != 1 {
		t.Fatalf("expected one findings section")
	}
	findings, err := ci.DecodeSection[ci.FindingsSection](report.Sections[0])
	if err != nil {
		t.Fatalf("decode findings: %v", err)
	}
	if findings.Rows[0].Findings[0].Message != "public endpoint forbidden" {
		t.Fatalf("unexpected finding: %+v", findings.Rows[0].Findings[0])
	}
}

func TestBuildPolicyReport_WithWarnings(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  1,
		WarnedModules: 1,
		Results: []policyengine.Result{
			{
				Module: "platform/prod/app",
				Warnings: []policyengine.Violation{
					{Namespace: "compliance", Message: "tag missing"},
				},
			},
		},
	}

	report, buildErr := buildPolicyReport(summary)
	if buildErr != nil {
		t.Fatalf("buildPolicyReport() error = %v", buildErr)
	}
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if len(report.Sections) != 1 {
		t.Fatalf("expected one findings section")
	}
	findings, err := ci.DecodeSection[ci.FindingsSection](report.Sections[0])
	if err != nil {
		t.Fatalf("decode findings: %v", err)
	}
	if findings.Rows[0].Findings[0].Message != "tag missing" {
		t.Fatalf("unexpected finding: %+v", findings.Rows[0].Findings[0])
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
	err := outputResult(&buf, "json", summary, false)
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

func TestOutputText_UsesLogger(t *testing.T) {
	summary := &policyengine.Summary{
		TotalModules:  1,
		WarnedModules: 1,
		Results: []policyengine.Result{
			{
				Module: "platform/prod/app",
				Warnings: []policyengine.Violation{
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
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	cmds := p.Commands(appCtx)
	if len(cmds) != 1 {
		t.Fatalf("Commands() returned %d commands, want 1", len(cmds))
	}

	cmd := cmds[0]
	checkCmd, _, err := cmd.Find([]string{"check"})
	if err != nil {
		t.Fatalf("Find(check) error = %v", err)
	}
	outputFlag := checkCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("missing --output flag on policy check")
	}
	if outputFlag.DefValue != "" {
		t.Fatalf("check --output default = %q, want empty default for shared policy output flag", outputFlag.DefValue)
	}
}

func TestPlugin_Commands_RunE_NotConfigured(t *testing.T) {
	p := newTestPlugin()
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	cmds := p.Commands(appCtx)
	cmd := cmds[0]
	checkCmd, _, err := cmd.Find([]string{"check"})
	if err != nil {
		t.Fatalf("Find(check) error = %v", err)
	}
	checkCmd.SetContext(context.Background())

	err = checkCmd.RunE(checkCmd, nil)
	if err == nil {
		t.Fatal("expected error for unconfigured plugin")
	}
	if !strings.Contains(err.Error(), "extensions.policy.enabled: true") {
		t.Fatalf("error = %q, want actionable enablement hint", err.Error())
	}
}
