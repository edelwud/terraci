package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/cmd/terraci/internal/initflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func newInitCmd() *cobra.Command {
	var (
		forceInit    bool
		initProvider string
		initBinary   string
		initPattern  string
		ciMode       bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize terraci configuration",
		Long: `Create a .terraci.yaml configuration file in the current directory.

By default, runs an interactive wizard. Use --ci flag for non-interactive mode.

Examples:
  terraci init
  terraci init --ci
  terraci init --ci --provider github
	  terraci init --ci --provider gitlab --binary tofu`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			prepared, err := runflow.FromContext(cmd.Context())
			if err != nil {
				return err
			}
			configPath := filepath.Join(prepared.WorkDir(), ".terraci.yaml")

			if _, err := os.Stat(configPath); err == nil && !forceInit {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}

			hasFlags := initProvider != "" || initBinary != "" || initPattern != ""

			var newCfg *config.Config
			if ciMode || hasFlags {
				cfg, err := buildNonInteractiveInitConfig(prepared.Registry(), initProvider, initBinary, initPattern)
				if err != nil {
					return err
				}
				newCfg = cfg
			} else {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return errors.New(
						"terraci init: stdin is not a TTY — pass --ci to accept defaults, " +
							"or supply --provider / --binary / --pattern to drive non-interactive setup",
					)
				}
				cfg, err := runInteractiveInit(prepared.Registry())
				if err != nil {
					return err
				}
				newCfg = cfg
			}

			if err := newCfg.Save(configPath); err != nil {
				return fmt.Errorf("create config: %w", err)
			}

			log.WithField("file", configPath).Info("configuration created")
			logGenerateHint(prepared.Registry(), newCfg)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing config file")
	cmd.Flags().BoolVar(&ciMode, "ci", false, "non-interactive mode (skip wizard)")
	cmd.Flags().StringVar(&initProvider, "provider", "", "CI provider: gitlab or github")
	cmd.Flags().StringVar(&initBinary, "binary", "", "terraform binary: terraform or tofu")
	cmd.Flags().StringVar(&initPattern, "pattern", "", "directory pattern")

	return cmd
}

func buildNonInteractiveInitConfig(plugins *registry.Registry, provider, binary, pattern string) (*config.Config, error) {
	flow, err := initflow.New(plugins)
	if err != nil {
		return nil, err
	}
	state := flow.DefaultState()
	flow.ApplyOverrides(state, initflow.Overrides{
		Provider: provider,
		Binary:   binary,
		Pattern:  pattern,
	})
	result, err := flow.BuildConfig(state)
	if err != nil {
		return nil, err
	}
	return result.Config, nil
}

func logGenerateHint(plugins *registry.Registry, cfg *config.Config) {
	log.Info("generate your pipeline with:")
	log.IncreasePadding()
	if cfg != nil {
		if _, ok := cfg.Extensions["github"]; ok {
			log.Info("terraci generate -o .github/workflows/terraform.yml")
			log.DecreasePadding()
			return
		}
		if _, ok := cfg.Extensions["gitlab"]; ok {
			log.Info("terraci generate -o .gitlab-ci.yml")
			log.DecreasePadding()
			return
		}
	}

	resolved, _ := plugins.ResolveCIProvider() //nolint:errcheck // best-effort fallback, non-critical
	if resolved != nil && resolved.ProviderName() == "github" {
		log.Info("terraci generate -o .github/workflows/terraform.yml")
	} else {
		log.Info("terraci generate -o .gitlab-ci.yml")
	}
	log.DecreasePadding()
}

func runInteractiveInit(plugins *registry.Registry) (*config.Config, error) {
	flow, err := initflow.New(plugins)
	if err != nil {
		return nil, err
	}
	m := newInitModel(flow)
	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, fmt.Errorf("interactive init: %w", err)
	}
	im, ok := finalModel.(*initModel)
	if !ok {
		return nil, errors.New("unexpected model type")
	}
	if im.err != nil {
		return nil, im.err
	}
	if im.result == nil {
		return nil, errors.New("init canceled")
	}
	return im.result, nil
}
