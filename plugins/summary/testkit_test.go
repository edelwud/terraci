package summary

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func plugSummaryOutput(t *testing.T, fn func()) string {
	return plugintest.CaptureLogOutput(t, fn)
}
