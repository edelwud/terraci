package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

func newInitCmd(app *App) *cobra.Command {
	var (
		forceInit       bool
		initProvider    string
		initBinary      string
		initImage       string
		initPattern     string
		initInteractive bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize terraci configuration",
		Long: `Create a .terraci.yaml configuration file in the current directory.

By default, runs an interactive wizard. Use --ci flag for non-interactive mode.

Examples:
  terraci init
  terraci init --ci
  terraci init --provider github
  terraci init --binary tofu --image ghcr.io/opentofu/opentofu:1.6`,
		RunE: func(_ *cobra.Command, _ []string) error {
			configPath := filepath.Join(app.WorkDir, ".terraci.yaml")

			if _, err := os.Stat(configPath); err == nil && !forceInit {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}

			hasFlags := initProvider != "" || initBinary != "" || initImage != "" || initPattern != ""

			var newCfg *config.Config
			var err error

			if initInteractive || hasFlags {
				state := plugin.NewStateMap()

				// Set defaults
				if initProvider != "" {
					state.Set("provider", initProvider)
				} else {
					// Default to first registered provider
					providerPlugins := plugin.ByCapability[plugin.GeneratorProvider]()
					if len(providerPlugins) > 0 {
						state.Set("provider", providerPlugins[0].ProviderName())
					}
				}
				if initBinary != "" {
					state.Set("binary", initBinary)
				} else {
					state.Set("binary", "terraform")
				}
				if initPattern != "" {
					state.Set("pattern", initPattern)
				}
				if initImage != "" {
					state.Set("gitlab.image", initImage)
				}

				// Set pipeline defaults for CI mode
				state.Set("plan_enabled", true)

				newCfg = buildConfigFromState(state)
			} else {
				newCfg, err = runInteractiveInit()
				if err != nil {
					return err
				}
			}

			if err := newCfg.Save(configPath); err != nil {
				return fmt.Errorf("create config: %w", err)
			}

			log.WithField("file", configPath).Info("configuration created")
			logGenerateHint()

			return nil
		},
	}

	cmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing config file")
	cmd.Flags().BoolVar(&initInteractive, "ci", false, "non-interactive mode (skip wizard)")
	cmd.Flags().StringVar(&initProvider, "provider", "", "CI provider: gitlab or github")
	cmd.Flags().StringVar(&initBinary, "binary", "", "terraform binary: terraform or tofu")
	cmd.Flags().StringVar(&initImage, "image", "", "docker image for CI jobs")
	cmd.Flags().StringVar(&initPattern, "pattern", "", "directory pattern")

	return cmd
}

func logGenerateHint() {
	log.Info("generate your pipeline with:")
	log.IncreasePadding()
	resolved, _ := plugin.ResolveProvider() //nolint:errcheck // best-effort hint, non-critical
	if resolved != nil && resolved.ProviderName() == "github" {
		log.Info("terraci generate -o .github/workflows/terraform.yml")
	} else {
		log.Info("terraci generate -o .gitlab-ci.yml")
	}
	log.DecreasePadding()
}

func runInteractiveInit() (*config.Config, error) {
	m := newInitModel()
	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, fmt.Errorf("interactive init: %w", err)
	}
	im, ok := finalModel.(*initModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}
	if im.result == nil {
		return nil, fmt.Errorf("init canceled")
	}
	return im.result, nil
}
