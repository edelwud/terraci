package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("terraci %s\n", app.Version)
			fmt.Printf("  commit: %s\n", app.Commit)
			fmt.Printf("  built:  %s\n", app.Date)

			// Version info from plugins (e.g., OPA version from policy plugin)
			for _, vp := range plugin.ByCapability[plugin.VersionProvider]() {
				for k, v := range vp.VersionInfo() {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}

			plugins := plugin.All()
			if len(plugins) > 0 {
				fmt.Printf("  plugins:\n")
				for _, p := range plugins {
					fmt.Printf("    - %s: %s\n", p.Name(), p.Description())
				}
			}
		},
	}
}
