package policy

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands returns the CLI commands provided by the policy plugin.
func (p *Plugin) Commands() []*cobra.Command {
	var (
		pullOutputDir    string
		checkOutputFmt   string
		policyModulePath string
	)

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull policies from configured sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx := plugin.FromContext(cmd.Context())
			current, err := plugin.CommandInstance[*Plugin](appCtx, p.Name())
			if err != nil {
				return err
			}
			if !current.IsEnabled() {
				return errors.New("policy checks are not enabled (set extensions.policy.enabled: true)")
			}

			log.Info("pulling policies from configured sources")
			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			return current.runPull(c, appCtx, pullOutputDir)
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check Terraform plans against policies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx := plugin.FromContext(cmd.Context())
			current, err := plugin.CommandInstance[*Plugin](appCtx, p.Name())
			if err != nil {
				return err
			}
			if !current.IsEnabled() {
				return errors.New("policy checks are not enabled (set extensions.policy.enabled: true)")
			}

			log.Info("running policy checks")

			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			return current.runCheck(c, appCtx, policyModulePath, checkOutputFmt)
		},
	}

	pullCmd.Flags().StringVarP(&pullOutputDir, "output", "o", "", "output directory for materialized policies")
	checkCmd.Flags().StringVarP(&policyModulePath, "module", "m", "", "check specific module only")
	checkCmd.Flags().StringVarP(&checkOutputFmt, "output", "o", "text", "output format: text, json")

	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy management commands",
		Long:  "Commands for managing and running OPA policy checks against Terraform plans.",
	}
	cmd.AddCommand(pullCmd, checkCmd)

	return []*cobra.Command{cmd}
}
