package cmd

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
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
	initStateDefaults(state)

	m := &initModel{state: state}

	// Collect all plugin group specs
	contributors := plugin.ByCapability[plugin.InitContributor]()
	var allSpecs []*plugin.InitGroupSpec
	for _, c := range contributors {
		allSpecs = append(allSpecs, c.InitGroups()...)
	}

	// Sort specs by Order
	sort.Slice(allSpecs, func(i, j int) bool {
		return allSpecs[i].Order < allSpecs[j].Order
	})

	// Categorize specs
	var providerSpecs, pipelineSpecs, featureSpecs, detailSpecs []*plugin.InitGroupSpec
	for _, spec := range allSpecs {
		switch spec.Category {
		case plugin.CategoryProvider:
			providerSpecs = append(providerSpecs, spec)
		case plugin.CategoryPipeline:
			pipelineSpecs = append(pipelineSpecs, spec)
		case plugin.CategoryFeature:
			featureSpecs = append(featureSpecs, spec)
		case plugin.CategoryDetail:
			detailSpecs = append(detailSpecs, spec)
		}
	}

	// Assemble groups in logical order:
	// Basics (hardcoded) → Provider → Pipeline (merged) → Features (merged) → Detail
	const initialGroupCap = 8
	groups := make([]*huh.Group, 0, initialGroupCap)

	// 1. Basics — the only hardcoded group
	groups = append(groups, m.basicsGroup())

	// 2. Provider groups — separate, with ShowWhen
	for _, spec := range providerSpecs {
		groups = append(groups, buildPluginGroup(spec, state))
	}

	// 3. Pipeline — merge all CategoryPipeline fields into one group
	if len(pipelineSpecs) > 0 {
		groups = append(groups, buildMergedGroup("Pipeline", pipelineSpecs, state))
	}

	// 4. Features — merge all CategoryFeature fields into one group
	if len(featureSpecs) > 0 {
		groups = append(groups, buildMergedGroup("Features", featureSpecs, state))
	}

	// 5. Detail groups — separate, with ShowWhen (gated by feature toggles)
	for _, spec := range detailSpecs {
		groups = append(groups, buildPluginGroup(spec, state))
	}

	m.form = huh.NewForm(groups...).WithWidth(formWidth).WithShowHelp(true)

	return m
}

func (m *initModel) basicsGroup() *huh.Group {
	// Build provider options dynamically from registered plugins
	providerPlugins := plugin.ByCapability[plugin.CIMetadata]()
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
			Title("IaC Tool").
			Description("Which infrastructure-as-code tool do you use?").
			Options(
				huh.NewOption("Terraform", "terraform"),
				huh.NewOption("OpenTofu", "tofu"),
			).Value(m.state.StringPtr("binary")),

		huh.NewInput().
			Title("Directory Pattern").
			Description("How are your Terraform modules organized?").
			Placeholder("{service}/{environment}/{region}/{module}").
			Value(m.state.StringPtr("pattern")),
	).Title("Basics")
}

// buildMergedGroup combines fields from multiple specs into a single group.
// Used for CategoryPipeline and CategoryFeature — merges toggle fields
// from different plugins into one cohesive step.
// Fields with duplicate keys are deduplicated (first occurrence wins).
func buildMergedGroup(title string, specs []*plugin.InitGroupSpec, state *plugin.StateMap) *huh.Group {
	var fields []huh.Field
	var showFns []func(*plugin.StateMap) bool
	seen := make(map[string]bool)

	for _, spec := range specs {
		for _, f := range spec.Fields {
			if seen[f.Key] {
				continue
			}
			seen[f.Key] = true
			fields = append(fields, buildPluginField(f, state))
		}
		if spec.ShowWhen != nil {
			showFns = append(showFns, spec.ShowWhen)
		}
	}

	g := huh.NewGroup(fields...).Title(title)

	// If any spec has ShowWhen, hide the merged group when ALL ShowWhen return false
	if len(showFns) > 0 {
		g = g.WithHideFunc(func() bool {
			for _, fn := range showFns {
				if fn(state) {
					return false // at least one wants to show
				}
			}
			return true // all hidden
		})
	}

	return g
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
	pattern := state.String("pattern")
	pluginConfigs := make(map[string]map[string]any)

	for _, c := range plugin.ByCapability[plugin.InitContributor]() {
		contrib := c.BuildInitConfig(state)
		if contrib != nil {
			pluginConfigs[contrib.PluginKey] = contrib.Config
		}
	}

	cfg, err := config.BuildConfigFromPlugins(pattern, pluginConfigs)
	if err != nil {
		log.WithError(err).Warn("failed to build config from init state")
		return config.DefaultConfig()
	}
	return cfg
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
