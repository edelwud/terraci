package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print xterraci version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("xterraci %s\n", app.Version)
			fmt.Printf("  commit: %s\n", app.Commit)
			fmt.Printf("  built:  %s\n", app.Date)
		},
	}
}
