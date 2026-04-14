package tfupdate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func captureUpdateTextOutput(t *testing.T, fn func()) string {
	return plugintest.CaptureLogOutput(t, fn)
}

func loadUpdateResult(t *testing.T, serviceDir string) tfupdateengine.UpdateResult {
	return plugintest.LoadJSONFile[tfupdateengine.UpdateResult](t, serviceDir, resultsFile)
}

func loadUpdateReport(t *testing.T, serviceDir string) ci.Report {
	return plugintest.LoadPluginReport(t, serviceDir, "tfupdate")
}
