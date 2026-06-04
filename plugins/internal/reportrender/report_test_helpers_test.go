package reportrender

import (
	"encoding/json"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func mustRenderedReport(t *testing.T, opts ci.RenderedReportOptions) *ci.Report {
	t.Helper()
	report, err := ci.NewRenderedReport(opts)
	if err != nil {
		t.Fatalf("NewRenderedReport() error = %v", err)
	}
	return report
}

func mustReportJSON(t *testing.T, raw string) *ci.Report {
	t.Helper()
	var report ci.Report
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("Unmarshal(report) error = %v", err)
	}
	return &report
}
