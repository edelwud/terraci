package cmd

import (
	"sort"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"
)

func newListPluginsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-plugins",
		Short: "List available built-in plugins",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("built-in plugins")
			log.IncreasePadding()

			names := make([]string, 0, len(BuiltinPlugins))
			for k := range BuiltinPlugins {
				names = append(names, k)
			}
			sort.Strings(names)

			for _, name := range names {
				log.WithField("module", BuiltinPlugins[name]).Info(name)
			}

			log.DecreasePadding()
		},
	}
}
