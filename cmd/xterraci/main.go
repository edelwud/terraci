// xterraci builds custom TerraCi binaries with selected plugins,
// similar to xcaddy for Caddy.
//
// Usage:
//
//	xterraci build [version]
//	xterraci build --with github.com/myco/terraci-plugin-slack
//	xterraci build --without awscost --output ./bin/terraci
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "xterraci",
		Short: "Build custom TerraCi binaries with plugins",
	}

	buildCmd := newBuildCmd()
	rootCmd.AddCommand(buildCmd)

	// Show help if no subcommand given
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		cmd.Help()
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newBuildCmd() *cobra.Command {
	var (
		withPlugins    []string
		withoutPlugins []string
		output         string
		skipCleanup    bool
	)

	cmd := &cobra.Command{
		Use:   "build [terraci-version]",
		Short: "Build a custom TerraCi binary",
		Long: `Build a custom TerraCi binary with selected plugins.

Examples:
  xterraci build                                         # all built-in plugins, latest
  xterraci build v1.5.0                                  # specific version
  xterraci build --with github.com/myco/plugin-slack     # add external plugin
  xterraci build --with github.com/myco/plugin@v0.2.0   # pin version
  xterraci build --with github.com/myco/plugin=../local  # local replace
  xterraci build --without awscost                       # exclude built-in
  xterraci build --output ./bin/terraci                  # custom output path`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			terraciVersion := ""
			if len(args) > 0 {
				terraciVersion = args[0]
			}

			b := &Builder{
				TerraciVersion: terraciVersion,
				WithPlugins:    withPlugins,
				WithoutPlugins: withoutPlugins,
				Output:         output,
				SkipCleanup:    skipCleanup,
			}

			return b.Build()
		},
	}

	cmd.Flags().StringArrayVar(&withPlugins, "with", nil, "add plugin (module[@version][=replacement])")
	cmd.Flags().StringArrayVar(&withoutPlugins, "without", nil, "exclude built-in plugin by name")
	cmd.Flags().StringVarP(&output, "output", "o", "./terraci", "output binary path")
	cmd.Flags().BoolVar(&skipCleanup, "skip-cleanup", false, "keep temporary build directory")

	return cmd
}
