package policy

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands returns the CLI commands provided by the policy plugin.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		policyOutput     string
		policyModulePath string
	)

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull policies from configured sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsEnabled() {
				return errors.New("policy checks are not enabled in configuration")
			}

			log.Info("pulling policies from configured sources")
			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			return runPullPolicies(c, ctx, p.Config(), policyOutput)
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check Terraform plans against policies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsEnabled() {
				return errors.New("policy checks are not enabled in configuration")
			}

			log.Info("running policy checks")

			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			return p.runCheck(c, ctx, policyModulePath, policyOutput)
		},
	}

	pullCmd.Flags().StringVarP(&policyOutput, "output", "o", "", "output directory for policies")
	checkCmd.Flags().StringVarP(&policyModulePath, "module", "m", "", "check specific module only")
	checkCmd.Flags().StringVarP(&policyOutput, "output", "o", "", "output format: text, json")

	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy management commands",
		Long:  "Commands for managing and running OPA policy checks against Terraform plans.",
	}
	cmd.AddCommand(pullCmd, checkCmd)

	return []*cobra.Command{cmd}
}
