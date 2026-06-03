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

			if _, statErr := os.Stat(configPath); statErr == nil && !forceInit {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}

			hasFlags := initProvider != "" || initBinary != "" || initPattern != ""

			var result *initflow.BuildResult
			if ciMode || hasFlags {
				result, err = buildNonInteractiveInitConfig(prepared, initProvider, initBinary, initPattern)
				if err != nil {
					return err
				}
			} else {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return errors.New(
						"terraci init: stdin is not a TTY — pass --ci to accept defaults, " +
							"or supply --provider / --binary / --pattern to drive non-interactive setup",
					)
				}
				result, err = runInteractiveInit(prepared)
				if err != nil {
					return err
				}
			}

			if result == nil || result.Config == nil {
				return errors.New("init produced empty config")
			}
			if err := result.Config.Save(configPath); err != nil {
				return fmt.Errorf("create config: %w", err)
			}

			log.WithField("file", configPath).Info("configuration created")
			logGenerateHint(result.GenerateCommand)

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

func buildNonInteractiveInitConfig(source initflow.PluginSource, provider, binary, pattern string) (*initflow.BuildResult, error) {
	flow, err := initflow.New(source)
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
	return result, nil
}

func logGenerateHint(command string) {
	if command == "" {
		command = "terraci generate -o .gitlab-ci.yml"
	}
	log.Info("generate your pipeline with:")
	log.IncreasePadding()
	log.Info(command)
	log.DecreasePadding()
}

func runInteractiveInit(source initflow.PluginSource) (*initflow.BuildResult, error) {
	flow, err := initflow.New(source)
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
	if im.result == nil || im.result.Config == nil {
		return nil, errors.New("init canceled")
	}
	return im.result, nil
}
