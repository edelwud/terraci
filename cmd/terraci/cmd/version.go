package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/policy"
)

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("terraci %s\n", app.Version)
			fmt.Printf("  commit: %s\n", app.Commit)
			fmt.Printf("  built:  %s\n", app.Date)
			fmt.Printf("  opa:    %s\n", policy.OPAVersion())
		},
	}
}
