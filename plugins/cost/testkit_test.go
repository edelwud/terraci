package cost

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	rawlog "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func captureTextOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldLogger := rawlog.Log
	var buf bytes.Buffer
	rawlog.Log = rawlog.New(&buf)
	defer func() { rawlog.Log = oldLogger }()
	fn()
	return buf.String()
}

func loadEstimateResult(t *testing.T, serviceDir string) model.EstimateResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(serviceDir, resultsFile))
	if err != nil {
		t.Fatalf("failed to read %s: %v", resultsFile, err)
	}

	var result model.EstimateResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse %s: %v", resultsFile, err)
	}

	return result
}

func loadCostReport(t *testing.T, serviceDir string) ci.Report {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(serviceDir, "cost-report.json"))
	if err != nil {
		t.Fatalf("failed to read cost-report.json: %v", err)
	}

	var report ci.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("failed to parse cost-report.json: %v", err)
	}

	return report
}
