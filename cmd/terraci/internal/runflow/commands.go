package runflow

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// PluginCommands returns commands from a fresh plugin registry. Command
// registration happens before command pre-run, so it uses a throwaway registry;
// each RunE later binds to the real command-scoped plugin instance through
// plugin.CommandPlugin.
func PluginCommands(factory RegistryFactory) ([]*cobra.Command, error) {
	if factory == nil {
		factory = registry.New
	}
	plugins := factory()
	if plugins == nil {
		plugins = registry.New()
	}
	return plugins.Commands()
}
