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
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
)

const (
	stateKeyProvider       = "provider"
	stateKeyBinary         = "binary"
	stateKeyPattern        = "pattern"
	stateKeyPlanEnabled    = "plan_enabled"
	stateKeySummaryEnabled = "summary.enabled"

	providerGitLab = "gitlab"
	providerGitHub = "github"
)

// PluginSource is the minimum plugin source required by init flow
// construction. Production passes a registry; tests can pass a small fake.
type PluginSource interface {
	All() []plugin.Plugin
}

// Overrides describes non-interactive init values supplied by CLI flags.
type Overrides struct {
	Provider string
	Binary   string
	Pattern  string
}

// BuildResult is the output of config construction.
type BuildResult struct {
	Config *config.Config
}

// DisplayGroup describes one init form group after plugin groups have been
// classified, sorted, and merged.
type DisplayGroup struct {
	Title    string
	Fields   []initwiz.InitField
	ShowWhen func(*initwiz.StateMap) bool
}

// ProviderOption describes one CI provider choice for the TUI basics group.
type ProviderOption struct {
	Name        string
	Description string
}

// ContributionError wraps a plugin init contribution failure with the plugin
// name while preserving the original error for errors.As/errors.Is.
type ContributionError struct {
	Plugin string
	Err    error
}

func (e *ContributionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Plugin == "" {
		return fmt.Sprintf("build init config: %v", e.Err)
	}
	return fmt.Sprintf("build init config for %s: %v", e.Plugin, e.Err)
}

func (e *ContributionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Flow is an immutable init orchestration value.
type Flow struct {
	contributors    []contributorBinding
	providerOpts    []ProviderOption
	displayGroups   []DisplayGroup
	defaultProvider string
}

type contributorBinding struct {
	name        string
	contributor initwiz.InitContributor
}

type groupBinding struct {
	plugin string
	spec   *initwiz.InitGroupSpec
}

// New constructs an init flow from the supplied plugin source.
func New(source PluginSource) Flow {
	plugins := pluginsSortedByName(source)

	contributors := make([]contributorBinding, 0, len(plugins))
	providers := make([]ProviderOption, 0, len(plugins))
	var groups []groupBinding

	for _, p := range plugins {
		if provider, ok := p.(plugin.CIInfoProvider); ok {
			providers = append(providers, ProviderOption{
				Name:        provider.ProviderName(),
				Description: provider.Description(),
			})
		}
		if contributor, ok := p.(initwiz.InitContributor); ok {
			contributors = append(contributors, contributorBinding{
				name:        contributor.Name(),
				contributor: contributor,
			})
			for _, spec := range contributor.InitGroups() {
				if spec == nil {
					continue
				}
				groups = append(groups, groupBinding{plugin: contributor.Name(), spec: spec})
			}
		}
	}

	providers = normalizeProviderOptions(providers)

	return Flow{
		contributors:    append([]contributorBinding(nil), contributors...),
		providerOpts:    cloneProviderOptions(providers),
		displayGroups:   buildDisplayGroups(groups),
		defaultProvider: defaultProvider(providers),
	}
}

// DefaultState returns a fresh StateMap initialized with canonical init
// defaults.
func (f Flow) DefaultState() *initwiz.StateMap {
	state := initwiz.NewStateMap()
	if f.defaultProvider != "" {
		state.Set(stateKeyProvider, f.defaultProvider)
	}
	state.Set(stateKeyBinary, config.ExecutionBinaryTerraform)
	state.Set(stateKeyPlanEnabled, true)
	state.Set(stateKeyPattern, config.DefaultConfig().Structure.Pattern)
	state.Set(stateKeySummaryEnabled, true)
	return state
}

// ApplyOverrides applies CLI overrides to state. Empty fields are ignored.
func (f Flow) ApplyOverrides(state *initwiz.StateMap, overrides Overrides) {
	if state == nil {
		return
	}
	if overrides.Provider != "" {
		state.Set(stateKeyProvider, overrides.Provider)
	}
	if overrides.Binary != "" {
		state.Set(stateKeyBinary, overrides.Binary)
	}
	if overrides.Pattern != "" {
		state.Set(stateKeyPattern, overrides.Pattern)
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

	pattern := state.String(stateKeyPattern)
	planEnabled := config.DefaultConfig().Execution.PlanEnabled
	if state.Get(stateKeyPlanEnabled) != nil {
		planEnabled = state.Bool(stateKeyPlanEnabled)
	}

	execution := config.DefaultConfig().Execution
	execution.Binary = state.String(stateKeyBinary)
	execution.InitEnabled = true
	execution.PlanEnabled = planEnabled
	if execution.Binary == "" {
		execution.Binary = config.ExecutionBinaryTerraform
	}
	if planEnabled && state.Bool(stateKeySummaryEnabled) {
		execution.PlanMode = "detailed"
	}

	extensions := make([]config.ExtensionValue, 0, len(f.contributors))
	for _, binding := range f.contributors {
		contribution, err := binding.contributor.BuildInitConfig(state)
		if err != nil {
			return nil, &ContributionError{Plugin: binding.name, Err: err}
		}
		if contribution == nil {
			continue
		}
		extensions = append(extensions, contribution.ExtensionValue())
	}

	extensionSet, err := config.NewExtensionSet(extensions...)
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
	return &BuildResult{Config: cfg}, nil
}

func pluginsSortedByName(source PluginSource) []plugin.Plugin {
	if source == nil {
		return nil
	}
	plugins := append([]plugin.Plugin(nil), source.All()...)
	sort.SliceStable(plugins, func(i, j int) bool {
		return plugins[i].Name() < plugins[j].Name()
	})
	return plugins
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
		byCategory[group.spec.Category] = append(byCategory[group.spec.Category], group)
	}

	out := make([]DisplayGroup, 0, len(groups))
	for _, group := range byCategory[initwiz.CategoryProvider] {
		out = append(out, displayGroupFromSpec(group.spec))
	}
	if pipeline := mergedDisplayGroup("Pipeline", byCategory[initwiz.CategoryPipeline]); len(pipeline.Fields) > 0 {
		out = append(out, pipeline)
	}
	if features := mergedDisplayGroup("Features", byCategory[initwiz.CategoryFeature]); len(features.Fields) > 0 {
		out = append(out, features)
	}
	for _, group := range byCategory[initwiz.CategoryDetail] {
		out = append(out, displayGroupFromSpec(group.spec))
	}
	return out
}

func sortGroupBindings(groups []groupBinding) {
	sort.SliceStable(groups, func(i, j int) bool {
		left := groups[i]
		right := groups[j]
		if left.spec.Order != right.spec.Order {
			return left.spec.Order < right.spec.Order
		}
		if left.spec.Title != right.spec.Title {
			return left.spec.Title < right.spec.Title
		}
		if leftKey, rightKey := firstFieldKey(left.spec), firstFieldKey(right.spec); leftKey != rightKey {
			return leftKey < rightKey
		}
		return left.plugin < right.plugin
	})
}

func displayGroupFromSpec(spec *initwiz.InitGroupSpec) DisplayGroup {
	return DisplayGroup{
		Title:    spec.Title,
		Fields:   cloneFields(spec.Fields),
		ShowWhen: spec.ShowWhen,
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
		for _, field := range group.spec.Fields {
			if field.Key == "" {
				continue
			}
			if _, exists := seen[field.Key]; exists {
				continue
			}
			seen[field.Key] = struct{}{}
			fields = append(fields, cloneField(field))
		}
		if group.spec.ShowWhen != nil {
			showFns = append(showFns, group.spec.ShowWhen)
		}
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

	return DisplayGroup{Title: title, Fields: fields, ShowWhen: showWhen}
}

func firstFieldKey(spec *initwiz.InitGroupSpec) string {
	if spec == nil || len(spec.Fields) == 0 {
		return ""
	}
	return spec.Fields[0].Key
}

func cloneProviderOptions(options []ProviderOption) []ProviderOption {
	if len(options) == 0 {
		return nil
	}
	return append([]ProviderOption(nil), options...)
}

func cloneDisplayGroup(group DisplayGroup) DisplayGroup {
	return DisplayGroup{
		Title:    group.Title,
		Fields:   cloneFields(group.Fields),
		ShowWhen: group.ShowWhen,
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
	field.Options = append([]initwiz.InitOption(nil), field.Options...)
	return field
}
