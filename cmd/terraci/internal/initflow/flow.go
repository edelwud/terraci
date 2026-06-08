// Package initflow owns the typed orchestration for `terraci init`.
//
// The public command package keeps CLI/TUI rendering and file I/O; this package
// owns defaults, plugin contributor discovery, display group ordering, and
// final config assembly.
package initflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

const (
	providerGitLab = "gitlab"
	providerGitHub = "github"
)

// PluginSource is the minimum plugin source required by init flow
// construction. Production passes a registry; tests can pass a small fake.
type PluginSource interface {
	InitWizardSnapshot() (*registry.InitWizardSnapshot, error)
}

// Overrides describes non-interactive init values supplied by CLI flags.
type Overrides struct {
	Provider string
	Binary   string
	Pattern  string
}

// BuildResult is the output of config construction.
type BuildResult struct {
	Config          *config.Config
	GenerateCommand string
}

// DisplayGroup describes one init form group after plugin groups have been
// classified, sorted, and merged.
type DisplayGroup struct {
	title    string
	fields   []initwiz.InitField
	showWhen func(*initwiz.StateMap) bool
}

// Title returns the display group title.
func (g DisplayGroup) Title() string { return g.title }

// Fields returns defensive field copies.
func (g DisplayGroup) Fields() []initwiz.InitField { return cloneFields(g.fields) }

// Visible reports whether the group should be shown for the current state.
func (g DisplayGroup) Visible(state *initwiz.StateMap) bool {
	if g.showWhen == nil {
		return true
	}
	return g.showWhen(state)
}

// ProviderOption describes one CI provider choice for the TUI basics group.
type ProviderOption struct {
	Name        string
	Description string
}

// Flow is an immutable init orchestration value.
type Flow struct {
	contributors    []contributorBinding
	providerOpts    []ProviderOption
	displayGroups   []DisplayGroup
	defaultProvider string
}

type contributorBinding struct {
	binding registry.InitContributorBinding
}

type groupBinding struct {
	plugin string
	group  initwiz.InitGroup
}

// New constructs an init flow from the supplied plugin source.
func New(source PluginSource) (*Flow, error) {
	var snapshot *registry.InitWizardSnapshot
	if source != nil {
		var err error
		snapshot, err = source.InitWizardSnapshot()
		if err != nil {
			return nil, err
		}
	}
	if snapshot == nil {
		snapshot = &registry.InitWizardSnapshot{}
	}

	contributorBindings := snapshot.Contributors()
	providerOptions := snapshot.ProviderOptions()
	groupBindings := snapshot.Groups()

	contributors := make([]contributorBinding, 0, len(contributorBindings))
	providers := make([]ProviderOption, 0, len(providerOptions))
	var groups []groupBinding

	for _, provider := range providerOptions {
		providers = append(providers, ProviderOption{
			Name:        provider.Name(),
			Description: provider.Description(),
		})
	}
	for _, contributor := range contributorBindings {
		contributors = append(contributors, contributorBinding{binding: contributor})
	}
	for _, group := range groupBindings {
		groups = append(groups, groupBinding{plugin: group.Plugin(), group: group.Group()})
	}

	providers = normalizeProviderOptions(providers)

	return &Flow{
		contributors:    append([]contributorBinding(nil), contributors...),
		providerOpts:    cloneProviderOptions(providers),
		displayGroups:   buildDisplayGroups(groups),
		defaultProvider: defaultProvider(providers),
	}, nil
}

// DefaultState returns a fresh StateMap initialized with canonical init
// defaults.
func (f Flow) DefaultState() *initwiz.StateMap {
	state := initwiz.NewStateMap()
	if f.defaultProvider != "" {
		initwiz.ProviderKey.Set(state, f.defaultProvider)
	}
	initwiz.BinaryKey.Set(state, config.ExecutionBinaryTerraform)
	initwiz.PatternKey.Set(state, config.DefaultConfig().Structure.Pattern)
	initwiz.SummaryEnabledKey.Set(state, true)
	return state
}

// ApplyOverrides applies CLI overrides to state. Empty fields are ignored.
func (f Flow) ApplyOverrides(state *initwiz.StateMap, overrides Overrides) {
	if state == nil {
		return
	}
	if overrides.Provider != "" {
		initwiz.ProviderKey.Set(state, overrides.Provider)
	}
	if overrides.Binary != "" {
		initwiz.BinaryKey.Set(state, overrides.Binary)
	}
	if overrides.Pattern != "" {
		initwiz.PatternKey.Set(state, overrides.Pattern)
	}
}

// DisplayGroups returns defensive copies of init display groups.
func (f Flow) DisplayGroups() []DisplayGroup {
	if len(f.displayGroups) == 0 {
		return nil
	}
	out := make([]DisplayGroup, len(f.displayGroups))
	for i := range f.displayGroups {
		out[i] = cloneDisplayGroup(f.displayGroups[i])
	}
	return out
}

// ProviderOptions returns deterministic CI provider choices for presentation.
func (f Flow) ProviderOptions() []ProviderOption {
	return cloneProviderOptions(f.providerOpts)
}

// BuildConfig assembles the final .terraci.yaml config from state.
func (f Flow) BuildConfig(state *initwiz.StateMap) (*BuildResult, error) {
	if state == nil {
		state = f.DefaultState()
	}

	pattern := initwiz.PatternKey.Get(state)

	execution := config.DefaultConfig().Execution
	execution.Binary = initwiz.BinaryKey.Get(state)
	execution.InitEnabled = true
	if execution.Binary == "" {
		execution.Binary = config.ExecutionBinaryTerraform
	}

	extensions := make([]config.ExtensionValue, 0, len(f.contributors))
	for _, binding := range f.contributors {
		contribution, err := binding.binding.BuildInitConfig(state)
		if err != nil {
			return nil, err
		}
		if contribution == nil {
			continue
		}
		extensions = append(extensions, contribution.ExtensionValue())
	}

	extensionSet, err := config.NewExtensionValueSet(extensions...)
	if err != nil {
		return nil, fmt.Errorf("build init extension set: %w", err)
	}
	cfg, err := config.Build(config.BuildOptions{
		Pattern:    pattern,
		Execution:  &execution,
		Extensions: extensionSet,
	})
	if err != nil {
		return nil, fmt.Errorf("build init config: %w", err)
	}
	return &BuildResult{Config: cfg, GenerateCommand: generateCommand(initwiz.ProviderKey.Get(state))}, nil
}

func generateCommand(provider string) string {
	if provider == providerGitHub {
		return "terraci generate -o .github/workflows/terraform.yml"
	}
	return "terraci generate -o .gitlab-ci.yml"
}

func normalizeProviderOptions(options []ProviderOption) []ProviderOption {
	if len(options) == 0 {
		return nil
	}
	byName := make(map[string]ProviderOption, len(options))
	for _, option := range options {
		option.Name = strings.TrimSpace(option.Name)
		if option.Name == "" {
			continue
		}
		if option.Description == "" {
			option.Description = option.Name
		}
		existing, exists := byName[option.Name]
		if !exists || option.Description < existing.Description {
			byName[option.Name] = option
		}
	}
	out := make([]ProviderOption, 0, len(byName))
	for _, option := range byName {
		out = append(out, option)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func defaultProvider(options []ProviderOption) string {
	available := make(map[string]struct{}, len(options))
	for _, option := range options {
		available[option.Name] = struct{}{}
	}
	for _, preferred := range []string{providerGitLab, providerGitHub} {
		if _, ok := available[preferred]; ok {
			return preferred
		}
	}
	if len(options) == 0 {
		return ""
	}
	return options[0].Name
}

func buildDisplayGroups(groups []groupBinding) []DisplayGroup {
	if len(groups) == 0 {
		return nil
	}
	sortGroupBindings(groups)

	byCategory := map[initwiz.InitCategory][]groupBinding{}
	for _, group := range groups {
		byCategory[group.group.Category()] = append(byCategory[group.group.Category()], group)
	}

	out := make([]DisplayGroup, 0, len(groups))
	for _, group := range byCategory[initwiz.CategoryProvider] {
		out = append(out, displayGroupFromSpec(group.group))
	}
	if pipeline := mergedDisplayGroup("Pipeline", byCategory[initwiz.CategoryPipeline]); len(pipeline.fields) > 0 {
		out = append(out, pipeline)
	}
	if features := mergedDisplayGroup("Features", byCategory[initwiz.CategoryFeature]); len(features.fields) > 0 {
		out = append(out, features)
	}
	for _, group := range byCategory[initwiz.CategoryDetail] {
		out = append(out, displayGroupFromSpec(group.group))
	}
	return out
}

func sortGroupBindings(groups []groupBinding) {
	sort.SliceStable(groups, func(i, j int) bool {
		left := groups[i]
		right := groups[j]
		if left.group.Order() != right.group.Order() {
			return left.group.Order() < right.group.Order()
		}
		if left.group.Title() != right.group.Title() {
			return left.group.Title() < right.group.Title()
		}
		if leftKey, rightKey := firstFieldKey(left.group), firstFieldKey(right.group); leftKey != rightKey {
			return leftKey < rightKey
		}
		return left.plugin < right.plugin
	})
}

func displayGroupFromSpec(group initwiz.InitGroup) DisplayGroup {
	return DisplayGroup{
		title:    group.Title(),
		fields:   cloneFields(group.Fields()),
		showWhen: group.Visible,
	}
}

func mergedDisplayGroup(title string, groups []groupBinding) DisplayGroup {
	if len(groups) == 0 {
		return DisplayGroup{}
	}

	fields := make([]initwiz.InitField, 0)
	seen := make(map[string]struct{})
	showFns := make([]func(*initwiz.StateMap) bool, 0)
	for _, group := range groups {
		groupFields := group.group.Fields()
		for i := range groupFields {
			field := &groupFields[i]
			key := field.Key()
			if key == "" {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			fields = append(fields, cloneField(*field))
		}
		showFns = append(showFns, group.group.Visible)
	}

	var showWhen func(*initwiz.StateMap) bool
	if len(showFns) > 0 {
		showWhen = func(state *initwiz.StateMap) bool {
			for _, fn := range showFns {
				if fn(state) {
					return true
				}
			}
			return false
		}
	}

	return DisplayGroup{title: title, fields: fields, showWhen: showWhen}
}

func firstFieldKey(group initwiz.InitGroup) string {
	fields := group.Fields()
	if len(fields) == 0 {
		return ""
	}
	return fields[0].Key()
}

func cloneProviderOptions(options []ProviderOption) []ProviderOption {
	if len(options) == 0 {
		return nil
	}
	return append([]ProviderOption(nil), options...)
}

func cloneDisplayGroup(group DisplayGroup) DisplayGroup {
	return DisplayGroup{
		title:    group.title,
		fields:   cloneFields(group.fields),
		showWhen: group.showWhen,
	}
}

func cloneFields(fields []initwiz.InitField) []initwiz.InitField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]initwiz.InitField, len(fields))
	for i := range fields {
		out[i] = cloneField(fields[i])
	}
	return out
}

func cloneField(field initwiz.InitField) initwiz.InitField {
	return field.Clone()
}
