package policy

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func capturePolicyTextOutput(t *testing.T, fn func()) string {
	return plugintest.CaptureLogOutput(t, fn)
}
