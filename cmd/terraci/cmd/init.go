package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	forceInit       bool
	initProvider    string
	initBinary      string
	initImage       string
	initPattern     string
	initInteractive bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize terraci configuration",
	Long: `Create a .terraci.yaml configuration file in the current directory.

By default, runs an interactive wizard that walks you through configuration.
Use --ci flag to skip the wizard and use defaults or CLI flags.

Examples:
  terraci init                              # Interactive wizard
  terraci init --ci                         # Non-interactive with defaults
  terraci init --provider github            # GitHub Actions preset
  terraci init --provider gitlab            # GitLab CI preset (default)
  terraci init --binary tofu --image ghcr.io/opentofu/opentofu:1.6`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing config file")
	initCmd.Flags().BoolVar(&initInteractive, "ci", false, "non-interactive mode (skip wizard)")
	initCmd.Flags().StringVar(&initProvider, "provider", "", "CI provider: gitlab or github")
	initCmd.Flags().StringVar(&initBinary, "binary", "", "terraform binary: terraform or tofu")
	initCmd.Flags().StringVar(&initImage, "image", "", "docker image for CI jobs")
	initCmd.Flags().StringVar(&initPattern, "pattern", "", "directory pattern (e.g., {service}/{environment}/{region}/{module})")
}

func runInit(_ *cobra.Command, _ []string) error {
	configPath := filepath.Join(workDir, ".terraci.yaml")

	if _, err := os.Stat(configPath); err == nil && !forceInit {
		return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
	}

	var newCfg *config.Config

	if !initInteractive && !hasInitFlags() {
		var err error
		newCfg, err = runInteractiveInit()
		if err != nil {
			return err
		}
	} else {
		newCfg = buildConfigFromFlags()
	}

	if err := newCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	log.WithField("file", configPath).Info("configuration created")

	provider := config.ResolveProvider(newCfg)
	if provider == config.ProviderGitHub {
		log.Info("generate your pipeline with:")
		log.IncreasePadding()
		log.Info("terraci generate -o .github/workflows/terraform.yml")
		log.DecreasePadding()
	} else {
		log.Info("generate your pipeline with:")
		log.IncreasePadding()
		log.Info("terraci generate -o .gitlab-ci.yml")
		log.DecreasePadding()
	}

	return nil
}

func hasInitFlags() bool {
	return initProvider != "" || initBinary != "" || initImage != "" || initPattern != ""
}

func buildConfigFromFlags() *config.Config {
	newCfg := config.DefaultConfig()

	if initProvider != "" {
		newCfg.Provider = initProvider
	}
	if initPattern != "" {
		newCfg.Structure.Pattern = initPattern
	}

	switch initProvider {
	case config.ProviderGitHub:
		newCfg.GitHub = defaultGitHubConfig()
		newCfg.GitLab = nil // clear gitlab defaults
		if initBinary != "" {
			newCfg.GitHub.TerraformBinary = initBinary
		}
	default:
		if initBinary != "" {
			newCfg.GitLab.TerraformBinary = initBinary
		}
		if initImage != "" {
			newCfg.GitLab.Image = config.Image{Name: initImage}
		}
	}

	return newCfg
}

// ---------------------------------------------------------------------------
// Interactive init with live YAML preview
// ---------------------------------------------------------------------------

// initModel is the bubbletea model that wraps the huh form + YAML preview
type initModel struct {
	form   *huh.Form
	width  int
	height int
	result *config.Config

	// form field bindings
	provider    string
	binary      string
	pattern     string
	image       string
	runsOn      string
	planEnabled bool
	autoApprove bool
	submodules  bool
	enableMR    bool
	enableCost  bool
}

func newInitModel() *initModel {
	m := &initModel{
		provider:    "gitlab",
		binary:      "terraform",
		pattern:     "{service}/{environment}/{region}/{module}",
		image:       "hashicorp/terraform:1.6",
		runsOn:      "ubuntu-latest",
		planEnabled: true,
		submodules:  true,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("CI Provider").
				Description("Which CI/CD platform do you use?").
				Options(
					huh.NewOption("GitLab CI", "gitlab"),
					huh.NewOption("GitHub Actions", "github"),
				).
				Value(&m.provider),

			huh.NewSelect[string]().
				Title("Terraform Binary").
				Description("Which IaC tool do you use?").
				Options(
					huh.NewOption("Terraform", "terraform"),
					huh.NewOption("OpenTofu", "tofu"),
				).
				Value(&m.binary),
		).Title("Basics"),

		huh.NewGroup(
			huh.NewInput().
				Title("Directory Pattern").
				Description("How are your modules organized?").
				Placeholder("{service}/{environment}/{region}/{module}").
				Value(&m.pattern),

			huh.NewConfirm().
				Title("Enable submodules?").
				Description("Allow nested modules at depth 5").
				Value(&m.submodules),
		).Title("Project Structure"),

		huh.NewGroup(
			huh.NewInput().
				Title("Docker Image").
				Description("Base image for terraform jobs").
				Placeholder("hashicorp/terraform:1.6").
				Value(&m.image),
		).Title("GitLab CI").WithHideFunc(func() bool {
			return m.provider != "gitlab"
		}),

		huh.NewGroup(
			huh.NewInput().
				Title("Runner Label").
				Description("GitHub Actions runs-on value").
				Placeholder("ubuntu-latest").
				Value(&m.runsOn),
		).Title("GitHub Actions").WithHideFunc(func() bool {
			return m.provider != "github"
		}),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable plan stage?").
				Description("Generate separate plan + apply jobs").
				Value(&m.planEnabled),

			huh.NewConfirm().
				Title("Auto-approve applies?").
				Description("Skip manual approval for terraform apply").
				Value(&m.autoApprove),

			huh.NewConfirm().
				Title("Enable PR/MR comments?").
				Description("Post plan summaries as comments").
				Value(&m.enableMR),

			huh.NewConfirm().
				Title("Enable cost estimation?").
				Description("Estimate AWS costs from plans").
				Value(&m.enableCost),
		).Title("Pipeline Options"),
	).WithWidth(formWidth).WithShowHelp(true)

	return m
}

const (
	formWidth    = 50
	previewWidth = 46
	gapWidth     = 2
)

// style definitions
var (
	previewBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6366f1")).
			Padding(1, 2).
			Width(previewWidth)

	previewTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6366f1")).
			Bold(true)

	previewYAML = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a1a1aa"))

	gapStyle = lipgloss.NewStyle().Width(gapWidth)
)

func (m *initModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	_, cmd = m.form.Update(msg)

	if m.form.State == huh.StateCompleted {
		m.result = m.buildConfig()
		return m, tea.Quit
	}

	return m, cmd
}

func (m *initModel) View() tea.View {
	if m.form.State == huh.StateCompleted {
		return tea.NewView("")
	}

	formView := m.form.View()
	yamlPreview := m.renderYAMLPreview()

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		formView,
		gapStyle.Render(""),
		yamlPreview,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *initModel) renderYAMLPreview() string {
	previewCfg := m.buildConfig()

	data, err := yaml.Marshal(previewCfg)
	if err != nil {
		data = []byte("# error generating preview")
	}

	// Syntax highlight: colorize YAML keys
	yamlStr := highlightYAML(string(data))

	// Truncate to fit
	lines := strings.Split(yamlStr, "\n")
	const borderAndTitleLines = 6
	maxLines := m.height - borderAndTitleLines // account for border + title
	if maxLines < 5 {
		maxLines = 20
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, previewYAML.Render("  ..."))
	}

	title := previewTitle.Render("  .terraci.yaml")
	content := strings.Join(lines, "\n")

	return previewBorder.Render(title + "\n\n" + content)
}

// highlightYAML applies simple syntax highlighting to YAML output
func highlightYAML(input string) string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
	commentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#52525b"))
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399"))

	var out strings.Builder
	for line := range strings.SplitSeq(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			out.WriteString(commentStyle.Render(line))
			out.WriteString("\n")
			continue
		}

		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]

		if idx := strings.Index(trimmed, ": "); idx >= 0 {
			key := trimmed[:idx+1]
			val := strings.TrimSpace(trimmed[idx+2:])

			if val == "true" || val == "false" {
				out.WriteString(indent + keyStyle.Render(key) + " " + boolStyle.Render(val))
			} else {
				out.WriteString(indent + keyStyle.Render(key) + " " + valStyle.Render(val))
			}
		} else {
			switch {
			case strings.HasSuffix(trimmed, ":"):
				out.WriteString(indent + keyStyle.Render(trimmed))
			default:
				out.WriteString(indent + valStyle.Render(trimmed))
			}
		}
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func (m *initModel) buildConfig() *config.Config {
	newCfg := config.DefaultConfig()
	newCfg.Provider = m.provider
	newCfg.Structure.Pattern = m.pattern
	newCfg.Structure.AllowSubmodules = m.submodules
	if m.submodules {
		newCfg.Structure.MaxDepth = newCfg.Structure.MinDepth + 1
	} else {
		newCfg.Structure.MaxDepth = newCfg.Structure.MinDepth
	}

	switch m.provider {
	case config.ProviderGitHub:
		ghCfg := defaultGitHubConfig()
		ghCfg.TerraformBinary = m.binary
		ghCfg.RunsOn = m.runsOn
		ghCfg.PlanEnabled = m.planEnabled
		ghCfg.AutoApprove = m.autoApprove

		if m.binary == "tofu" {
			ghCfg.JobDefaults = &config.GitHubJobDefaults{
				StepsBefore: []config.GitHubStep{
					{Uses: "actions/checkout@v4"},
					{Uses: "opentofu/setup-opentofu@v1"},
				},
			}
		} else {
			ghCfg.JobDefaults = &config.GitHubJobDefaults{
				StepsBefore: []config.GitHubStep{
					{Uses: "actions/checkout@v4"},
					{Uses: "hashicorp/setup-terraform@v3"},
				},
			}
		}

		if m.enableMR {
			ghCfg.Permissions = map[string]string{
				"contents":      "read",
				"pull-requests": "write",
			}
			ghCfg.PR = &config.PRConfig{
				Comment: &config.MRCommentConfig{},
			}
		}

		newCfg.GitHub = ghCfg

	default:
		newCfg.GitLab.TerraformBinary = m.binary
		newCfg.GitLab.PlanEnabled = m.planEnabled
		newCfg.GitLab.AutoApprove = m.autoApprove

		if m.binary == "tofu" {
			newCfg.GitLab.Image = config.Image{Name: "ghcr.io/opentofu/opentofu:1.6"}
		} else {
			newCfg.GitLab.Image = config.Image{Name: m.image}
		}

		if m.enableMR {
			enabled := true
			newCfg.GitLab.MR = &config.MRConfig{
				Comment: &config.MRCommentConfig{
					Enabled: &enabled,
				},
				SummaryJob: &config.SummaryJobConfig{
					Image: &config.Image{Name: "ghcr.io/edelwud/terraci:latest"},
				},
			}
		}
	}

	if m.enableCost {
		newCfg.Cost = &config.CostConfig{
			Enabled:       true,
			ShowInComment: true,
		}
	}

	// Clean up zero-value fields for cleaner preview
	if newCfg.Provider == "gitlab" {
		newCfg.GitHub = nil
	}
	if newCfg.Provider == "github" {
		newCfg.GitLab = nil
	}

	return newCfg
}

func runInteractiveInit() (*config.Config, error) {
	m := newInitModel()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("interactive init failed: %w", err)
	}

	im, ok := finalModel.(*initModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}
	result := im.result
	if result == nil {
		return nil, fmt.Errorf("init canceled")
	}

	return result, nil
}

func defaultGitHubConfig() *config.GitHubConfig {
	return &config.GitHubConfig{
		TerraformBinary: "terraform",
		RunsOn:          "ubuntu-latest",
		PlanEnabled:     true,
		InitEnabled:     true,
	}
}
