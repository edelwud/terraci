package update

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	resultsFile = "update-results.json"
	reportFile  = "update-report.json"
)

// Commands returns the CLI commands provided by the update plugin.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		target     string
		bump       string
		write      bool
		modulePath string
		outputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check and update Terraform dependency versions",
		Long: `Check for outdated Terraform provider and module versions.

Queries the Terraform Registry for latest versions and reports available updates.
Use --write to apply version bumps to .tf files.

Examples:
  terraci update
  terraci update --target providers --bump patch
  terraci update --write
  terraci update --module platform/prod/eu-central-1/vpc
  terraci update --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsEnabled() {
				return errors.New("update plugin is not enabled (set plugins.update.enabled: true)")
			}

			log.Info("checking dependency versions")
			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			return p.runCheck(c, ctx, cmd)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "what to check: modules, providers, all")
	cmd.Flags().StringVarP(&bump, "bump", "b", "", "version bump level: patch, minor, major")
	cmd.Flags().BoolVarP(&write, "write", "w", false, "write updated versions back to .tf files")
	cmd.Flags().StringVarP(&modulePath, "module", "m", "", "check a specific module only")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "output format: text, json")

	return []*cobra.Command{cmd}
}
