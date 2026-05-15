package render

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestLogOutputCompleted_LogsDAGGroupNames(t *testing.T) {
	output := LogOutput{}
	result := &execution.Result{
		Groups: []execution.GroupResult{
			{Name: "dag-level-0", JobCount: 1},
			{Name: "dag-level-1", JobCount: 1},
		},
		Jobs: []*execution.JobResult{{
			Name:       "plan-platform-stage-eu-central-1-vpc",
			Status:     execution.JobStatusSucceeded,
			StartedAt:  time.Now(),
			FinishedAt: time.Now().Add(10 * time.Millisecond),
		}},
	}

	logs := plugintest.CaptureLogOutput(t, func() {
		output.Completed(result, nil)
	})

	for _, wanted := range []string{"dag-level-0", "dag-level-1"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
}

func TestLogOutputCompleted_EmptyResultUsesConsistentSummary(t *testing.T) {
	output := LogOutput{}
	result := &execution.Result{
		Groups: []execution.GroupResult{
			{Name: "dag-level-0", JobCount: 0},
		},
	}

	logs := plugintest.CaptureLogOutput(t, func() {
		output.Completed(result, nil)
	})

	for _, unwanted := range []string{"local execution completed (no jobs)", "── summary ──────────────────────────"} {
		if strings.Contains(logs, unwanted) {
			t.Fatalf("logs unexpectedly contain %q:\n%s", unwanted, logs)
		}
	}
	for _, wanted := range []string{"── stages ───────────────────────────", "dag-level-0", "jobs=0", "stages=1", "local execution completed"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
	for _, wanted := range []string{"succeeded=0", "failed=0", "duration=0s"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
}

func TestLogOutputCompleted_NilResultUsesConsistentSummary(t *testing.T) {
	output := LogOutput{}

	logs := plugintest.CaptureLogOutput(t, func() {
		output.Completed(nil, nil)
	})

	for _, unwanted := range []string{"local execution completed (no jobs)", "── stages ───────────────────────────", "── summary ──────────────────────────"} {
		if strings.Contains(logs, unwanted) {
			t.Fatalf("logs unexpectedly contain %q:\n%s", unwanted, logs)
		}
	}
	for _, wanted := range []string{"jobs=0", "stages=0", "local execution completed"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
	for _, wanted := range []string{"succeeded=0", "failed=0", "duration=0s"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
}

func TestLogOutputCompleted_NilSummaryReportSkipsCLISection(t *testing.T) {
	output := LogOutput{}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = originalStdout }()

	output.Completed(&execution.Result{}, nil)

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	renderedBytes, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	if rendered := string(renderedBytes); rendered != "" {
		t.Fatalf("expected no rendered summary output, got:\n%s", rendered)
	}
}

func TestLogOutputCompleted_WithSummaryReport(t *testing.T) {
	report := &ci.Report{
		Producer: "summary",
		Title:    "Terraform Plan Summary",
		Summary:  "1 modules: 1 with changes, 0 no changes, 0 failed",
		Sections: []ci.ReportSection{citest.MustEncodeRenderSection(
			"Summary",
			"",
			ci.ReportStatusWarn,
			ci.RenderTextBlock("1 module changed"),
		)},
	}
	output := LogOutput{}
	result := &execution.Result{}

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = originalStdout }()

	output.Completed(result, report)

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	renderedBytes, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}

	rendered := string(renderedBytes)
	if !strings.Contains(rendered, "Terraform Plan Summary") {
		t.Fatalf("rendered summary missing report title:\n%s", rendered)
	}
}

func TestProgressReporter_LogsStageAndModule(t *testing.T) {
	reporter := ProgressReporter{}
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	job := &pipeline.Job{
		Name:      "plan-platform-stage-eu-central-1-vpc",
		Module:    module,
		Operation: pipeline.Operation{Type: pipeline.OperationTypeTerraformPlan},
	}
	result := &execution.JobResult{
		Name:   job.Name,
		Status: execution.JobStatusSucceeded,
	}

	logs := plugintest.CaptureLogOutput(t, func() {
		reporter.JobStarted(job)
		reporter.JobFinished(job, result)
	})

	for _, wanted := range []string{"job=plan-platform-stage-eu-central-1-vpc", "operation=terraform_plan", "module=platform/stage/eu-central-1/vpc", "job started", "job finished"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
}

func TestProgressReporter_LogsFailureStatus(t *testing.T) {
	reporter := ProgressReporter{}
	job := &pipeline.Job{
		Name:      "summary",
		Operation: pipeline.Operation{Type: pipeline.OperationTypeCommands},
	}
	result := &execution.JobResult{
		Name:   job.Name,
		Status: execution.JobStatusFailed,
		Err:    errors.New("boom"),
	}

	logs := plugintest.CaptureLogOutput(t, func() {
		reporter.JobFinished(job, result)
	})

	for _, wanted := range []string{"job=summary", "operation=commands", "status=failed", "error=boom", "job finished"} {
		if !strings.Contains(logs, wanted) {
			t.Fatalf("logs missing %q:\n%s", wanted, logs)
		}
	}
}
