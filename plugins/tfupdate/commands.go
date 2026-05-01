package tfupdate

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

const (
	resultsFile = "tfupdate-results.json"
	reportFile  = "tfupdate-report.json"
)

// Commands returns the CLI commands provided by the tfupdate plugin.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		target        string
		bump          string
		pin           bool
		timeout       string
		write         bool
		modulePath    string
		outputFmt     string
		lockPlatforms []string
	)

	cmd := &cobra.Command{
		Use:   "tfupdate",
		Short: "Check or apply Terraform dependency version updates",
		Long: `Check Terraform provider and module versions for available updates.

Default mode is read-only and reports available updates.
Use --write to apply version bumps to matching .tf files.

Exit behavior:
  0 when the scan completes without operational errors
  non-zero when parse, registry, or write errors are encountered
  available updates alone do not make the command fail

Examples:
  terraci tfupdate
  terraci tfupdate --target providers --bump patch
  terraci tfupdate --write
  terraci tfupdate --write --pin
  terraci tfupdate --module platform/prod/eu-central-1/vpc
		terraci tfupdate --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			current, err := plugin.CommandInstance[*Plugin](ctx, p.Name())
			if err != nil {
				return err
			}
			if !current.IsEnabled() {
				return errors.New("tfupdate plugin is not enabled (set extensions.tfupdate.enabled: true)")
			}

			log.Info("checking terraform dependency state")
			opts := parseRuntimeOptions(cmd)
			timeout, err := resolveCommandTimeout(current.Config(), opts)
			if err != nil {
				return err
			}

			c, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			return current.runCheck(c, ctx, cmd)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "", "what to check: modules, providers, all")
	cmd.Flags().StringVarP(&bump, "bump", "b", "", "version bump level: patch, minor, major")
	cmd.Flags().BoolVar(&pin, "pin", false, "pin updated dependency constraints to an exact version when writing")
	cmd.Flags().StringVar(&timeout, "timeout", "", "overall timeout for the update run, e.g. 15m")
	cmd.Flags().BoolVarP(&write, "write", "w", false, "write updated versions back to .tf files")
	cmd.Flags().StringVarP(&modulePath, "module", "m", "", "check a specific module only")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "output format: text, json")
	cmd.Flags().StringSliceVar(&lockPlatforms, "lock-platforms", nil, "platforms to hash for lock files (e.g. linux_amd64,darwin_arm64). Default: all")

	return []*cobra.Command{cmd}
}

func resolveCommandTimeout(cfg *tfupdateengine.UpdateConfig, opts runtimeOptions) (time.Duration, error) {
	effective := &tfupdateengine.UpdateConfig{}
	if cfg != nil {
		copyCfg := *cfg
		effective = &copyCfg
	}
	if opts.timeout != "" {
		effective.Timeout = opts.timeout
	}
	if effective.Timeout != "" {
		if _, err := time.ParseDuration(effective.Timeout); err != nil {
			return 0, err
		}
	}
	return effective.CommandTimeout(opts.write), nil
}
