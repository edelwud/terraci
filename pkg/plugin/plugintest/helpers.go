package plugintest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	rawlog "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

func CaptureLogOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldLogger := rawlog.Log
	var buf bytes.Buffer
	rawlog.Log = rawlog.New(&buf)
	defer func() { rawlog.Log = oldLogger }()
	fn()
	return buf.String()
}

func LoadJSONFile[T any](t *testing.T, dir, filename string) T {
	t.Helper()

	var value T
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("failed to read %s: %v", filename, err)
	}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("failed to parse %s: %v", filename, err)
	}
	return value
}

func LoadReport(t *testing.T, serviceDir, filename string) ci.Report {
	t.Helper()
	return LoadJSONFile[ci.Report](t, serviceDir, filename)
}

func NewAppContext(t *testing.T, workDir string) *plugin.AppContext {
	t.Helper()

	serviceDir := filepath.Join(t.TempDir(), ".terraci")
	cfg := config.DefaultConfig()
	cfg.ServiceDir = ".terraci"
	return plugin.NewAppContext(cfg, workDir, serviceDir, "test", plugin.NewReportRegistry())
}
