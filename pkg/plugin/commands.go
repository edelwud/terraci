package plugin

import "github.com/spf13/cobra"

// CommandProvider adds CLI subcommands to TerraCi.
type CommandProvider interface {
	Plugin
	Commands(ctx *AppContext) []*cobra.Command
}

// FlagOverridable plugins support direct CLI flag overrides on their config.
type FlagOverridable interface {
	Plugin
	SetPlanOnly(bool)
	SetAutoApprove(bool)
}

// VersionProvider plugins contribute version info to `terraci version`.
type VersionProvider interface {
	Plugin
	VersionInfo() map[string]string
}
