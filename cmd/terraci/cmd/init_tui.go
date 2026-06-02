package cmd

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/cmd/terraci/internal/initflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
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
	form        *huh.Form
	width       int
	height      int
	result      *config.Config
	err         error
	state       *initwiz.StateMap
	flow        *initflow.Flow
	buildConfig func(*initwiz.StateMap) (*initflow.BuildResult, error)
}

func newInitModel(flow *initflow.Flow) *initModel {
	state := flow.DefaultState()

	m := &initModel{
		state:       state,
		flow:        flow,
		buildConfig: flow.BuildConfig,
	}

	const initialGroupCap = 8
	groups := make([]*huh.Group, 0, initialGroupCap)
	groups = append(groups, m.basicsGroup(flow.ProviderOptions()))
	for _, group := range flow.DisplayGroups() {
		groups = append(groups, buildDisplayGroup(group, state))
	}

	m.form = huh.NewForm(groups...).WithWidth(formWidth).WithShowHelp(true)

	return m
}

func (m *initModel) basicsGroup(providers []initflow.ProviderOption) *huh.Group {
	providerOpts := make([]huh.Option[string], 0, len(providers))
	for _, provider := range providers {
		providerOpts = append(providerOpts, huh.NewOption(provider.Description, provider.Name))
	}

	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("CI Provider").
			Description("Which CI/CD platform do you use?").
			Options(providerOpts...).
			Value(initwiz.ProviderKey.Bind(m.state)),

		huh.NewSelect[string]().
			Title("IaC Tool").
			Description("Which infrastructure-as-code tool do you use?").
			Options(
				huh.NewOption("Terraform", "terraform"),
				huh.NewOption("OpenTofu", "tofu"),
			).Value(initwiz.BinaryKey.Bind(m.state)),

		huh.NewInput().
			Title("Directory Pattern").
			Description("How are your Terraform modules organized?").
			Placeholder("{service}/{environment}/{region}/{module}").
			Value(initwiz.PatternKey.Bind(m.state)),
	).Title("Basics")
}

// buildDisplayGroup converts a typed initflow display group into a huh group.
func buildDisplayGroup(group initflow.DisplayGroup, state *initwiz.StateMap) *huh.Group {
	groupFields := group.Fields()
	fields := make([]huh.Field, 0, len(groupFields))
	for i := range groupFields {
		fields = append(fields, buildPluginField(&groupFields[i], state))
	}

	g := huh.NewGroup(fields...).Title(group.Title())
	g = g.WithHideFunc(func() bool {
		return !group.Visible(state)
	})
	return g
}

// buildPluginField converts an InitField into a huh.Field.
func buildPluginField(f *initwiz.InitField, state *initwiz.StateMap) huh.Field {
	f.ApplyDefault(state)

	switch f.Type() {
	case initwiz.FieldBool:
		return huh.NewConfirm().
			Title(f.Title()).
			Description(f.Description()).
			Value(f.BoolKey().Bind(state))
	case initwiz.FieldSelect:
		options := f.Options()
		opts := make([]huh.Option[string], len(options))
		for i, o := range options {
			opts[i] = huh.NewOption(o.Label, o.Value)
		}
		return huh.NewSelect[string]().
			Title(f.Title()).
			Description(f.Description()).
			Options(opts...).
			Value(f.StringKey().Bind(state))
	case initwiz.FieldString:
		input := huh.NewInput().
			Title(f.Title()).
			Description(f.Description()).
			Value(f.StringKey().Bind(state))
		if f.Placeholder() != "" {
			input = input.Placeholder(f.Placeholder())
		}
		return input
	default:
		return huh.NewInput().
			Title(f.Title()).
			Description(f.Description()).
			Value(f.StringKey().Bind(state))
	}
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
		var result *initflow.BuildResult
		result, m.err = m.build(m.state)
		if result != nil {
			m.result = result.Config
		}
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
	result, err := m.build(m.state)
	if err != nil {
		return previewBorder.Render(previewTitle.Render("  .terraci.yaml") + "\n\n" + previewYAML.Render("# error generating preview: "+err.Error()))
	}
	if result == nil || result.Config == nil {
		return previewBorder.Render(previewTitle.Render("  .terraci.yaml") + "\n\n" + previewYAML.Render("# error generating preview: empty init config"))
	}
	data, err := yaml.Marshal(result.Config)
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

func (m *initModel) build(state *initwiz.StateMap) (*initflow.BuildResult, error) {
	if m.buildConfig != nil {
		return m.buildConfig(state)
	}
	if m.flow == nil {
		return nil, nil
	}
	return m.flow.BuildConfig(state)
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
