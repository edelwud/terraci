package plugin

import "github.com/spf13/cobra"

// CommandProvider adds CLI subcommands to TerraCi. The framework calls
// Commands() once during root command registration. Inside RunE, plugins
// retrieve the per-run AppContext via plugin.FromContext(cmd.Context()).
type CommandProvider interface {
	Plugin
	Commands() []*cobra.Command
}

// VersionProvider plugins contribute version info to `terraci version`.
type VersionProvider interface {
	Plugin
	VersionInfo() map[string]string
}
