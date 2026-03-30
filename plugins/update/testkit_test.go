package update

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	rawlog "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func captureUpdateTextOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldLogger := rawlog.Log
	var buf bytes.Buffer
	rawlog.Log = rawlog.New(&buf)
	defer func() { rawlog.Log = oldLogger }()
	fn()
	return buf.String()
}

func loadUpdateResult(t *testing.T, serviceDir string) updateengine.UpdateResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(serviceDir, resultsFile))
	if err != nil {
		t.Fatalf("failed to read %s: %v", resultsFile, err)
	}

	var result updateengine.UpdateResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse %s: %v", resultsFile, err)
	}
	return result
}

func loadUpdateReport(t *testing.T, serviceDir string) ci.Report {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(serviceDir, reportFile))
	if err != nil {
		t.Fatalf("failed to read %s: %v", reportFile, err)
	}

	var report ci.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("failed to parse %s: %v", reportFile, err)
	}
	return report
}
