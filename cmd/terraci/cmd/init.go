package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func newInitCmd(app *App) *cobra.Command {
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
		RunE: func(_ *cobra.Command, _ []string) error {
			configPath := filepath.Join(app.WorkDir, ".terraci.yaml")

			if _, err := os.Stat(configPath); err == nil && !forceInit {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}

			hasFlags := initProvider != "" || initBinary != "" || initPattern != ""

			var newCfg *config.Config
			var err error

			if ciMode || hasFlags {
				state := initwiz.NewStateMap()
				initStateDefaults(state)

				// Override defaults with CLI flags
				if initProvider != "" {
					state.Set("provider", initProvider)
				}
				if initBinary != "" {
					state.Set("binary", initBinary)
				}
				if initPattern != "" {
					state.Set("pattern", initPattern)
				}

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
			logGenerateHint(newCfg)

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

// initStateDefaults populates a StateMap with default values for the init wizard.
// Shared between interactive (TUI) and non-interactive (--ci) paths.
func initStateDefaults(state *initwiz.StateMap) {
	if provider := defaultInitProvider(); provider != "" {
		state.Set("provider", provider)
	}
	state.Set("binary", "terraform")
	state.Set("plan_enabled", true)
	state.Set("pattern", config.DefaultConfig().Structure.Pattern)
	// summary (MR/PR comments) enabled by default
	state.Set("summary.enabled", true)
}

func defaultInitProvider() string {
	providerPlugins := registry.ByCapability[plugin.CIInfoProvider]()
	if len(providerPlugins) == 0 {
		return ""
	}

	available := make(map[string]struct{}, len(providerPlugins))
	for _, provider := range providerPlugins {
		available[provider.ProviderName()] = struct{}{}
	}

	for _, preferred := range []string{"gitlab", "github"} {
		if _, ok := available[preferred]; ok {
			return preferred
		}
	}

	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	sort.Strings(names)
	return names[0]
}

func logGenerateHint(cfg *config.Config) {
	log.Info("generate your pipeline with:")
	log.IncreasePadding()
	if cfg != nil {
		if _, ok := cfg.Plugins["github"]; ok {
			log.Info("terraci generate -o .github/workflows/terraform.yml")
			log.DecreasePadding()
			return
		}
		if _, ok := cfg.Plugins["gitlab"]; ok {
			log.Info("terraci generate -o .gitlab-ci.yml")
			log.DecreasePadding()
			return
		}
	}

	resolved, _ := registry.ResolveCIProvider() //nolint:errcheck // best-effort fallback, non-critical
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
		return nil, errors.New("unexpected model type")
	}
	if im.result == nil {
		return nil, errors.New("init canceled")
	}
	return im.result, nil
}
