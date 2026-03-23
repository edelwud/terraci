package cmd

import "github.com/edelwud/terraci/pkg/config"

// initOptions wraps config.InitOptions for use by CLI and TUI init commands.
type initOptions = config.InitOptions

// Constants re-exported for TUI usage.
const (
	defaultTerraformImage = config.DefaultTerraformImage
	defaultTofuImage      = config.DefaultTofuImage
	defaultGitHubRunner   = config.DefaultGitHubRunner
	terraCIImage          = config.TerraCIImage
)
