package cmd

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
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
	state  *plugin.StateMap
}

func newInitModel() *initModel {
	state := plugin.NewStateMap()
	// Set provider default dynamically from registered plugins
	providerPlugins := plugin.ByCapability[plugin.GeneratorProvider]()
	if len(providerPlugins) > 0 {
		state.Set("provider", providerPlugins[0].ProviderName())
	}
	state.Set("binary", "terraform")
	state.Set("pattern", config.DefaultConfig().Structure.Pattern)
	state.Set("plan_enabled", true)

	m := &initModel{state: state}

	// Collect plugin groups sorted by Order
	contributors := plugin.ByCapability[plugin.InitContributor]()
	type orderedGroup struct {
		order int
		group *huh.Group
	}
	var pluginGroups []orderedGroup
	for _, c := range contributors {
		spec := c.InitGroup()
		if spec == nil {
			continue
		}
		g := buildPluginGroup(spec, state)
		pluginGroups = append(pluginGroups, orderedGroup{order: spec.Order, group: g})
	}
	sort.Slice(pluginGroups, func(i, j int) bool {
		return pluginGroups[i].order < pluginGroups[j].order
	})

	groups := make([]*huh.Group, 0, 2+len(pluginGroups)+1)
	groups = append(groups, m.basicsGroup(), m.structureGroup())
	for _, pg := range pluginGroups {
		groups = append(groups, pg.group)
	}

	groups = append(groups, m.pipelineGroup())

	m.form = huh.NewForm(groups...).WithWidth(formWidth).WithShowHelp(true)

	return m
}

func (m *initModel) basicsGroup() *huh.Group {
	// Build provider options dynamically from registered plugins
	providerPlugins := plugin.ByCapability[plugin.GeneratorProvider]()
	providerOpts := make([]huh.Option[string], 0, len(providerPlugins))
	for _, pp := range providerPlugins {
		providerOpts = append(providerOpts, huh.NewOption(pp.Description(), pp.ProviderName()))
	}

	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("CI Provider").
			Description("Which CI/CD platform do you use?").
			Options(providerOpts...).
			Value(m.state.StringPtr("provider")),

		huh.NewSelect[string]().
			Title("Terraform Binary").
			Description("Which IaC tool do you use?").
			Options(
				huh.NewOption("Terraform", "terraform"),
				huh.NewOption("OpenTofu", "tofu"),
			).Value(m.state.StringPtr("binary")),
	).Title("Basics")
}

func (m *initModel) structureGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Directory Pattern").
			Description("How are your modules organized?").
			Placeholder("{service}/{environment}/{region}/{module}").
			Value(m.state.StringPtr("pattern")),
	).Title("Project Structure")
}

func (m *initModel) pipelineGroup() *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Enable plan stage?").
			Description("Generate separate plan + apply jobs").
			Value(m.state.BoolPtr("plan_enabled")),

		huh.NewConfirm().
			Title("Auto-approve applies?").
			Description("Skip manual approval for terraform apply").
			Value(m.state.BoolPtr("auto_approve")),
	).Title("Pipeline Options")
}

// buildPluginGroup converts an InitGroupSpec into a huh.Group.
func buildPluginGroup(spec *plugin.InitGroupSpec, state *plugin.StateMap) *huh.Group {
	fields := make([]huh.Field, 0, len(spec.Fields))
	for _, f := range spec.Fields {
		fields = append(fields, buildPluginField(f, state))
	}

	g := huh.NewGroup(fields...).Title(spec.Title)
	if spec.ShowWhen != nil {
		showWhen := spec.ShowWhen
		g = g.WithHideFunc(func() bool {
			return !showWhen(state)
		})
	}
	return g
}

// buildPluginField converts an InitField into a huh.Field.
func buildPluginField(f plugin.InitField, state *plugin.StateMap) huh.Field {
	// Initialize default value
	if f.Default != nil {
		if state.Get(f.Key) == nil {
			state.Set(f.Key, f.Default)
		}
	}

	switch f.Type {
	case "bool":
		return huh.NewConfirm().
			Title(f.Title).
			Description(f.Description).
			Value(state.BoolPtr(f.Key))
	case "select":
		opts := make([]huh.Option[string], len(f.Options))
		for i, o := range f.Options {
			opts[i] = huh.NewOption(o.Label, o.Value)
		}
		return huh.NewSelect[string]().
			Title(f.Title).
			Description(f.Description).
			Options(opts...).
			Value(state.StringPtr(f.Key))
	default: // "string"
		input := huh.NewInput().
			Title(f.Title).
			Description(f.Description).
			Value(state.StringPtr(f.Key))
		if f.Placeholder != "" {
			input = input.Placeholder(f.Placeholder)
		}
		return input
	}
}

// buildConfigFromState collects InitContributor results and builds a Config.
func buildConfigFromState(state *plugin.StateMap) *config.Config {
	pattern, _ := state.Get("pattern").(string) //nolint:errcheck // safe type assertion
	pluginConfigs := make(map[string]map[string]any)

	for _, c := range plugin.ByCapability[plugin.InitContributor]() {
		contrib := c.BuildInitConfig(state)
		if contrib != nil {
			pluginConfigs[contrib.PluginKey] = contrib.Config
		}
	}

	return config.BuildConfigFromPlugins(pattern, pluginConfigs)
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
		m.result = buildConfigFromState(m.state)
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
	data, err := yaml.Marshal(buildConfigFromState(m.state))
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
