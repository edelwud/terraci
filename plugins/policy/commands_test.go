package policy

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
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

	report := buildPolicyReport(summary)
	if report.Status != ci.ReportStatusFail {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusFail)
	}
	if !strings.Contains(report.Summary, "2 modules") {
		t.Fatalf("Summary = %q, want module count", report.Summary)
	}
	if !strings.Contains(report.Body, "public endpoint forbidden") {
		t.Fatalf("Body = %q, want failure message", report.Body)
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

	report := buildPolicyReport(summary)
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("Status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
	if !strings.Contains(report.Body, "tag missing") {
		t.Fatalf("Body = %q, want warning message", report.Body)
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
	if !strings.Contains(output, "platform/prod/app") {
		t.Fatalf("output = %q, want module path", output)
	}
	if !strings.Contains(output, "tag missing") {
		t.Fatalf("output = %q, want warning message", output)
	}
}
