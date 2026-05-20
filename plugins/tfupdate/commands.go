package tfupdate

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/cliout"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

var (
	resultsFile = ci.ResultFilename(pluginName)
	reportFile  = ci.ReportFilename(pluginName)
)

// Commands returns the CLI commands provided by the tfupdate plugin.
func (p *Plugin) Commands() []*cobra.Command {
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
		Use:   pluginName,
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
			appCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, p.Name())
			if err != nil {
				return err
			}
			if enabledErr := plugin.RequireEnabled(current, "tfupdate plugin is not enabled (set extensions.tfupdate.enabled: true)"); enabledErr != nil {
				return enabledErr
			}

			log.Info("checking terraform dependency state")
			req, err := parseCheckRequest(cmd)
			if err != nil {
				return err
			}
			timeout, err := resolveCommandTimeout(current.Config(), req)
			if err != nil {
				return err
			}

			c, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			return current.runCheck(c, appCtx, req, os.Stdout)
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

func parseCheckRequest(cmd *cobra.Command) (CheckRequest, error) {
	req := CheckRequest{OutputFormat: cliout.FormatText}
	if flag := cmd.Flags().Lookup("write"); flag != nil {
		req.Write = flag.Value.String() == "true"
	}
	if flag := cmd.Flags().Lookup("module"); flag != nil {
		req.ModulePath = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("output"); flag != nil {
		format, err := cliout.ParseFormat(flag.Value.String())
		if err != nil {
			return CheckRequest{}, err
		}
		req.OutputFormat = format
	}
	if flag := cmd.Flags().Lookup("target"); flag != nil {
		req.Target = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("bump"); flag != nil {
		req.Bump = flag.Value.String()
	}
	if flag := cmd.Flags().Lookup("pin"); flag != nil {
		req.Pin = flag.Value.String() == "true"
	}
	if flag := cmd.Flags().Lookup("timeout"); flag != nil {
		req.Timeout = flag.Value.String()
	}
	if vals, err := cmd.Flags().GetStringSlice("lock-platforms"); err == nil && len(vals) > 0 {
		req.LockPlatforms = vals
	}
	return req, nil
}

func resolveCommandTimeout(cfg *tfupdateengine.UpdateConfig, req CheckRequest) (time.Duration, error) {
	effective := &tfupdateengine.UpdateConfig{}
	if cfg != nil {
		copyCfg := *cfg
		effective = &copyCfg
	}
	if req.Timeout != "" {
		effective.Timeout = req.Timeout
	}
	if effective.Timeout != "" {
		if _, err := time.ParseDuration(effective.Timeout); err != nil {
			return 0, err
		}
	}
	return effective.CommandTimeout(req.Write), nil
}
