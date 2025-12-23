package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/policy"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("terraci %s\n", versionInfo.Version)
		fmt.Printf("  commit: %s\n", versionInfo.Commit)
		fmt.Printf("  built:  %s\n", versionInfo.Date)
		fmt.Printf("  opa:    %s\n", policy.OPAVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
