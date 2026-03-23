package cmd

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config"
)

// --- TUI styles ---

const (
	formWidth    = 50
	previewWidth = 46
	gapWidth     = 2
)

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

// YAML syntax highlighting styles
var (
	yamlKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8"))
	yamlValStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
	yamlCommentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525b"))
	yamlBoolStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399"))
)

// --- Model ---

type initModel struct {
	form   *huh.Form
	width  int
	height int
	result *config.Config
	opts   initOptions
}

func newInitModel() *initModel {
	m := &initModel{
		opts: initOptions{
			Provider:    "gitlab",
			Binary:      "terraform",
			Pattern:     "{service}/{environment}/{region}/{module}",
			Image:       defaultTerraformImage,
			RunsOn:      defaultGitHubRunner,
			PlanEnabled: true,
		},
	}

	m.form = huh.NewForm(
		m.basicsGroup(),
		m.structureGroup(),
		m.gitlabGroup(),
		m.githubGroup(),
		m.pipelineGroup(),
	).WithWidth(formWidth).WithShowHelp(true)

	return m
}

func (m *initModel) basicsGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("CI Provider").
			Description("Which CI/CD platform do you use?").
			Options(
				huh.NewOption("GitLab CI", "gitlab"),
				huh.NewOption("GitHub Actions", "github"),
			).Value(&m.opts.Provider),

		huh.NewSelect[string]().
			Title("Terraform Binary").
			Description("Which IaC tool do you use?").
			Options(
				huh.NewOption("Terraform", "terraform"),
				huh.NewOption("OpenTofu", "tofu"),
			).Value(&m.opts.Binary),
	).Title("Basics")
}

func (m *initModel) structureGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Directory Pattern").
			Description("How are your modules organized?").
			Placeholder("{service}/{environment}/{region}/{module}").
			Value(&m.opts.Pattern),
	).Title("Project Structure")
}

func (m *initModel) gitlabGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Docker Image").
			Description("Base image for terraform jobs").
			Placeholder(defaultTerraformImage).
			Value(&m.opts.Image),
	).Title("GitLab CI").WithHideFunc(func() bool {
		return m.opts.Provider != "gitlab"
	})
}

func (m *initModel) githubGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Runner Label").
			Description("GitHub Actions runs-on value").
			Placeholder(defaultGitHubRunner).
			Value(&m.opts.RunsOn),
	).Title("GitHub Actions").WithHideFunc(func() bool {
		return m.opts.Provider != "github"
	})
}

func (m *initModel) pipelineGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Enable plan stage?").
			Description("Generate separate plan + apply jobs").
			Value(&m.opts.PlanEnabled),

		huh.NewConfirm().
			Title("Auto-approve applies?").
			Description("Skip manual approval for terraform apply").
			Value(&m.opts.AutoApprove),

		huh.NewConfirm().
			Title("Enable PR/MR comments?").
			Description("Post plan summaries as comments").
			Value(&m.opts.EnableMR),

		huh.NewConfirm().
			Title("Enable cost estimation?").
			Description("Estimate AWS costs from plans").
			Value(&m.opts.EnableCost),
	).Title("Pipeline Options")
}

// --- Tea interface ---

func (m *initModel) Init() tea.Cmd { return m.form.Init() }

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

	_, cmd := m.form.Update(msg)

	if m.form.State == huh.StateCompleted {
		m.result = m.opts.BuildConfig()
		return m, tea.Quit
	}

	return m, cmd
}

func (m *initModel) View() tea.View {
	if m.form.State == huh.StateCompleted {
		return tea.NewView("")
	}

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.form.View(),
		gapStyle.Render(""),
		m.renderYAMLPreview(),
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// --- YAML preview ---

func (m *initModel) renderYAMLPreview() string {
	data, err := yaml.Marshal(m.opts.BuildConfig())
	if err != nil {
		data = []byte("# error generating preview")
	}

	yamlStr := highlightYAML(string(data))

	lines := strings.Split(yamlStr, "\n")
	const borderAndTitleLines = 6
	maxLines := m.height - borderAndTitleLines
	if maxLines < 5 {
		maxLines = 20
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, previewYAML.Render("  ..."))
	}

	title := previewTitle.Render("  .terraci.yaml")
	return previewBorder.Render(title + "\n\n" + strings.Join(lines, "\n"))
}

func highlightYAML(input string) string {
	var out strings.Builder
	for line := range strings.SplitSeq(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			out.WriteString(yamlCommentStyle.Render(line))
			out.WriteString("\n")
			continue
		}

		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]

		if idx := strings.Index(trimmed, ": "); idx >= 0 {
			key := trimmed[:idx+1]
			val := strings.TrimSpace(trimmed[idx+2:])

			style := yamlValStyle
			if val == "true" || val == "false" {
				style = yamlBoolStyle
			}
			out.WriteString(indent + yamlKeyStyle.Render(key) + " " + style.Render(val))
		} else if strings.HasSuffix(trimmed, ":") {
			out.WriteString(indent + yamlKeyStyle.Render(trimmed))
		} else {
			out.WriteString(indent + yamlValStyle.Render(trimmed))
		}
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n")
}
