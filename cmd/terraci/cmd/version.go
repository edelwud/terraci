package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("terraci %s\n", versionInfo.Version)
		fmt.Printf("  commit: %s\n", versionInfo.Commit)
		fmt.Printf("  built:  %s\n", versionInfo.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
