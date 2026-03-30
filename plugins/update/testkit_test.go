package update

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func captureUpdateTextOutput(t *testing.T, fn func()) string {
	return plugintest.CaptureLogOutput(t, fn)
}

func loadUpdateResult(t *testing.T, serviceDir string) updateengine.UpdateResult {
	return plugintest.LoadJSONFile[updateengine.UpdateResult](t, serviceDir, resultsFile)
}

func loadUpdateReport(t *testing.T, serviceDir string) ci.Report {
	return plugintest.LoadPluginReport(t, serviceDir, "update")
}
