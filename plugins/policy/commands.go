package policy

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
)

// CommandSpecs returns the CLI commands provided by the policy plugin.
func (p *Plugin) CommandSpecs() ([]plugin.CommandSpec, error) {
	var (
		pullCacheDir     string
		checkFormat      string
		policyModulePath string
	)

	pullCmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   "pull",
		Short: "Pull policies from configured sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdCtx, current, bindingErr := plugin.CommandPlugin[*Plugin](cmd, p.Name())
			if bindingErr != nil {
				return bindingErr
			}
			if enabledErr := plugin.RequireEnabled(current, "policy checks are not enabled (set extensions.policy.enabled: true)"); enabledErr != nil {
				return enabledErr
			}

			log.Info("pulling policies from configured sources")
			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			return current.runPull(c, cmdCtx.AppContext(), pullCacheDir)
		},
		Configure: func(cmd *cobra.Command) error {
			cmd.Flags().StringVar(&pullCacheDir, "cache-dir", "", "cache directory for materialized policies")
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	checkCmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   "check",
		Short: "Check Terraform plans against policies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdCtx, current, bindingErr := plugin.CommandPlugin[*Plugin](cmd, p.Name())
			if bindingErr != nil {
				return bindingErr
			}
			if enabledErr := plugin.RequireEnabled(current, "policy checks are not enabled (set extensions.policy.enabled: true)"); enabledErr != nil {
				return enabledErr
			}

			log.Info("running policy checks")

			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			return current.runCheck(c, cmdCtx.AppContext(), policyModulePath, checkFormat, cmd.OutOrStdout())
		},
		Configure: func(cmd *cobra.Command) error {
			cmd.Flags().StringVarP(&policyModulePath, "module", "m", "", "check specific module only")
			cmd.Flags().StringVar(&checkFormat, "format", "text", "output format: text, json")
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   "policy",
		Short: "Policy management commands",
		Long:  "Commands for managing and running OPA policy checks against Terraform plans.",
		Subcommands: []plugin.CommandSpec{
			pullCmd,
			checkCmd,
		},
	})
	if err != nil {
		return nil, err
	}

	return []plugin.CommandSpec{cmd}, nil
}
