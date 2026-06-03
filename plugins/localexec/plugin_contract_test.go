package localexec

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("command provider", func(t *testing.T) {
		plugintest.AssertCommandProvider(t, plugintest.CommandProviderContract{
			Provider:     &Plugin{},
			ExpectedUses: []string{"local-exec"},
		})
	})
}
