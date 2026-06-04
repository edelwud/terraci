package render

import (
	"encoding/json"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func mustReportJSON(tb testing.TB, raw string) *ci.Report {
	tb.Helper()
	var report ci.Report
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		tb.Fatalf("Unmarshal(report) error = %v", err)
	}
	return &report
}
