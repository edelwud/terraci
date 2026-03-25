package cmd

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
)

func newBuildCmd(app *App) *cobra.Command {
	var (
		withPlugins    []string
		withoutPlugins []string
		output         string
		skipCleanup    bool
		verbose        bool
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
  xterraci build --without cost                          # exclude built-in
  xterraci build --output ./bin/terraci                  # custom output path`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				log.SetLevel(log.DebugLevel)
			}

			terraciVersion := app.Version
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

			return b.Build(cmd.Context())
		},
	}

	cmd.Flags().StringArrayVar(&withPlugins, "with", nil, "add plugin (module[@version][=replacement])")
	cmd.Flags().StringArrayVar(&withoutPlugins, "without", nil, "exclude built-in plugin by name")
	cmd.Flags().StringVarP(&output, "output", "o", "./terraci", "output binary path")
	cmd.Flags().BoolVar(&skipCleanup, "skip-cleanup", false, "keep temporary build directory")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output including command output")

	return cmd
}
