package cmd

import "github.com/edelwud/terraci/pkg/config"

// App holds shared state for all commands, replacing package-level globals.
type App struct {
	Config  *config.Config
	WorkDir string

	// Global flag values
	cfgFile  string
	logLevel string

	// Version info
	Version string
	Commit  string
	Date    string
}
