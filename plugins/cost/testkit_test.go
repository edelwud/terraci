package cost

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func captureTextOutput(t *testing.T, fn func()) string {
	return plugintest.CaptureLogOutput(t, fn)
}

func loadEstimateResult(t *testing.T, serviceDir string) model.EstimateResult {
	return plugintest.LoadJSONFile[model.EstimateResult](t, serviceDir, resultsFile)
}

func loadCostReport(t *testing.T, serviceDir string) ci.Report {
	return plugintest.LoadPluginReport(t, serviceDir, "cost")
}
