package registry

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
)

// Commands returns plugin-provided cobra commands in registration order.
func (r *Registry) Commands() ([]*cobra.Command, error) {
	if r == nil {
		return nil, nil
	}
	providers := byCapabilityFrom[plugin.CommandProvider](r)
	var commands []*cobra.Command
	for _, provider := range providers {
		specs, err := provider.CommandSpecs()
		if err != nil {
			return nil, plugin.CommandRegistrationError{Plugin: provider.Name(), Err: err}
		}
		for i := range specs {
			spec := specs[i]
			cmd, err := plugin.BuildCommand(spec)
			if err != nil {
				return nil, plugin.CommandRegistrationError{
					Plugin:  provider.Name(),
					Command: spec.Use(),
					Err:     err,
				}
			}
			commands = append(commands, cmd)
		}
	}
	return commands, nil
}

// DecodeConfig applies extension config documents to config-capable plugins.
func (r *Registry) DecodeConfig(cfg *config.Config) error {
	if r == nil || cfg == nil {
		return nil
	}
	for _, loader := range byCapabilityFrom[plugin.ConfigLoader](r) {
		key := loader.ConfigKey()
		doc, exists := cfg.Extension(key)
		if !exists {
			continue
		}
		if err := loader.DecodeAndSet(doc); err != nil {
			return plugin.ConfigError{Plugin: loader.Name(), Key: key.String(), Err: err}
		}
	}
	return nil
}

// ExtensionSchemas returns extension JSON schema samples by config key.
func (r *Registry) ExtensionSchemas() map[string]any {
	if r == nil {
		return nil
	}
	loaders := byCapabilityFrom[plugin.ConfigLoader](r)
	schemas := make(map[string]any, len(loaders))
	for _, loader := range loaders {
		schemas[loader.ConfigKey().String()] = loader.SchemaConfig()
	}
	return schemas
}

// RunPreflight runs enabled plugin preflights in registration order.
func (r *Registry) RunPreflight(ctx context.Context, appCtx *plugin.AppContext) error {
	if r == nil {
		return nil
	}
	for _, preflight := range r.preflightsForStartup() {
		if err := preflight.Preflight(ctx, appCtx); err != nil {
			return fmt.Errorf("preflight plugin %s: %w", preflight.Name(), err)
		}
	}
	return nil
}

func (r *Registry) preflightsForStartup() []plugin.Preflightable {
	plugins := r.all()
	result := make([]plugin.Preflightable, 0, len(plugins))
	for _, p := range plugins {
		if !isPluginEnabled(p) {
			continue
		}
		if preflightable, ok := p.(plugin.Preflightable); ok {
			result = append(result, preflightable)
		}
	}
	return result
}

// PluginSummary describes one registered plugin for presentation.
type PluginSummary struct {
	name        string
	description string
}

// Name returns the plugin name.
func (s PluginSummary) Name() string { return s.name }

// Description returns the plugin description.
func (s PluginSummary) Description() string { return s.description }

// VersionSnapshot is the registry-owned version command read model.
type VersionSnapshot struct {
	info    map[string]string
	plugins []PluginSummary
}

// Info returns defensive version info key/value copies.
func (s VersionSnapshot) Info() map[string]string {
	if len(s.info) == 0 {
		return nil
	}
	return maps.Clone(s.info)
}

// Plugins returns registered plugin summaries in registration order.
func (s VersionSnapshot) Plugins() []PluginSummary {
	if len(s.plugins) == 0 {
		return nil
	}
	return append([]PluginSummary(nil), s.plugins...)
}

// PluginInventoryItem describes one registered plugin and its public
// capabilities without exposing the plugin instance.
type PluginInventoryItem struct {
	name                string
	description         string
	configLoader        bool
	commandProvider     bool
	preflightable       bool
	pipelineContributor bool
	ciProvider          bool
	initContributor     bool
	versionProvider     bool
	changeDetector      bool
	kvCacheProvider     bool
	blobStoreProvider   bool
}

// Name returns the plugin name.
func (i PluginInventoryItem) Name() string { return i.name }

// Description returns the plugin description.
func (i PluginInventoryItem) Description() string { return i.description }

// HasConfigLoader reports whether the plugin loads extension config.
func (i PluginInventoryItem) HasConfigLoader() bool { return i.configLoader }

// HasCommandProvider reports whether the plugin contributes CLI commands.
func (i PluginInventoryItem) HasCommandProvider() bool { return i.commandProvider }

// HasPreflight reports whether the plugin contributes startup preflight.
func (i PluginInventoryItem) HasPreflight() bool { return i.preflightable }

// HasPipelineContributor reports whether the plugin contributes pipeline jobs.
func (i PluginInventoryItem) HasPipelineContributor() bool { return i.pipelineContributor }

// HasCIProvider reports whether the plugin contributes CI provider metadata.
func (i PluginInventoryItem) HasCIProvider() bool { return i.ciProvider }

// HasInitContributor reports whether the plugin contributes init wizard config.
func (i PluginInventoryItem) HasInitContributor() bool { return i.initContributor }

// HasVersionProvider reports whether the plugin contributes version metadata.
func (i PluginInventoryItem) HasVersionProvider() bool { return i.versionProvider }

// HasChangeDetector reports whether the plugin contributes VCS change detection.
func (i PluginInventoryItem) HasChangeDetector() bool { return i.changeDetector }

// HasKVCacheProvider reports whether the plugin contributes a KV cache backend.
func (i PluginInventoryItem) HasKVCacheProvider() bool { return i.kvCacheProvider }

// HasBlobStoreProvider reports whether the plugin contributes a blob store backend.
func (i PluginInventoryItem) HasBlobStoreProvider() bool { return i.blobStoreProvider }

// PluginInventory is a defensive plugin listing snapshot for tests and
// presentation code.
type PluginInventory struct {
	plugins []PluginInventoryItem
}

// Plugins returns defensive inventory item copies.
func (s PluginInventory) Plugins() []PluginInventoryItem {
	if len(s.plugins) == 0 {
		return nil
	}
	return append([]PluginInventoryItem(nil), s.plugins...)
}

// Inventory returns plugin names, descriptions, and capability flags.
func (r *Registry) Inventory() PluginInventory {
	if r == nil {
		return PluginInventory{}
	}
	plugins := r.all()
	snapshot := PluginInventory{plugins: make([]PluginInventoryItem, 0, len(plugins))}
	for _, p := range plugins {
		_, hasConfig := p.(plugin.ConfigLoader)
		_, hasCommand := p.(plugin.CommandProvider)
		_, hasPreflight := p.(plugin.Preflightable)
		_, hasPipeline := p.(plugin.PipelineContributor)
		_, hasCI := p.(plugin.CIInfoProvider)
		_, hasInit := p.(initwiz.InitContributor)
		_, hasVersion := p.(plugin.VersionProvider)
		_, hasChange := p.(plugin.ChangeDetectionProvider)
		_, hasKV := p.(plugin.KVCacheProvider)
		_, hasBlob := p.(plugin.BlobStoreProvider)
		snapshot.plugins = append(snapshot.plugins, PluginInventoryItem{
			name:                p.Name(),
			description:         p.Description(),
			configLoader:        hasConfig,
			commandProvider:     hasCommand,
			preflightable:       hasPreflight,
			pipelineContributor: hasPipeline,
			ciProvider:          hasCI,
			initContributor:     hasInit,
			versionProvider:     hasVersion,
			changeDetector:      hasChange,
			kvCacheProvider:     hasKV,
			blobStoreProvider:   hasBlob,
		})
	}
	return snapshot
}

// VersionSnapshot returns version info and plugin summaries.
func (r *Registry) VersionSnapshot() VersionSnapshot {
	if r == nil {
		return VersionSnapshot{}
	}
	snapshot := VersionSnapshot{info: make(map[string]string)}
	for _, provider := range byCapabilityFrom[plugin.VersionProvider](r) {
		maps.Copy(snapshot.info, provider.VersionInfo())
	}
	for _, p := range r.all() {
		snapshot.plugins = append(snapshot.plugins, PluginSummary{
			name:        p.Name(),
			description: p.Description(),
		})
	}
	return snapshot
}

// InitProviderOption describes one CI provider option for init presentation.
type InitProviderOption struct {
	name        string
	description string
}

// Name returns the provider name.
func (o InitProviderOption) Name() string { return o.name }

// Description returns the provider description.
func (o InitProviderOption) Description() string { return o.description }

// InitGroupBinding ties an init group to the plugin that contributed it.
type InitGroupBinding struct {
	plugin string
	group  initwiz.InitGroup
}

// Plugin returns the contributing plugin name.
func (b InitGroupBinding) Plugin() string { return b.plugin }

// Group returns a defensive group copy.
func (b InitGroupBinding) Group() initwiz.InitGroup { return b.group.Clone() }

// InitContributorBinding ties a config contribution builder to a plugin name.
type InitContributorBinding struct {
	name        string
	contributor initwiz.InitContributor
}

// Plugin returns the contributing plugin name.
func (b InitContributorBinding) Plugin() string { return b.name }

// BuildInitConfig builds and validates one plugin init contribution.
func (b InitContributorBinding) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	if b.contributor == nil {
		return nil, &InitContributionError{Plugin: b.name, Err: errors.New("init contributor is nil")}
	}
	contribution, err := b.contributor.BuildInitConfig(state)
	if err != nil {
		return nil, &InitContributionError{Plugin: b.name, Err: err}
	}
	return contribution, nil
}

// InitWizardSnapshot is the registry-owned init wizard read model.
type InitWizardSnapshot struct {
	contributors []InitContributorBinding
	providers    []InitProviderOption
	groups       []InitGroupBinding
}

// Contributors returns defensive contributor bindings.
func (s *InitWizardSnapshot) Contributors() []InitContributorBinding {
	if s == nil || len(s.contributors) == 0 {
		return nil
	}
	return append([]InitContributorBinding(nil), s.contributors...)
}

// ProviderOptions returns deterministic CI provider choices.
func (s *InitWizardSnapshot) ProviderOptions() []InitProviderOption {
	if s == nil || len(s.providers) == 0 {
		return nil
	}
	return append([]InitProviderOption(nil), s.providers...)
}

// Groups returns defensive contributed init group bindings.
func (s *InitWizardSnapshot) Groups() []InitGroupBinding {
	if s == nil || len(s.groups) == 0 {
		return nil
	}
	return append([]InitGroupBinding(nil), s.groups...)
}

// InitGroupError wraps a plugin init group construction failure.
type InitGroupError struct {
	Plugin string
	Err    error
}

func (e *InitGroupError) Error() string {
	if e == nil {
		return ""
	}
	if e.Plugin == "" {
		return fmt.Sprintf("build init groups: %v", e.Err)
	}
	return fmt.Sprintf("build init groups for %s: %v", e.Plugin, e.Err)
}

func (e *InitGroupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// InitContributionError wraps a plugin init contribution failure.
type InitContributionError struct {
	Plugin string
	Err    error
}

func (e *InitContributionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Plugin == "" {
		return fmt.Sprintf("build init config: %v", e.Err)
	}
	return fmt.Sprintf("build init config for %s: %v", e.Plugin, e.Err)
}

func (e *InitContributionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// InitWizardSnapshot returns init wizard contributors, provider options, and groups.
func (r *Registry) InitWizardSnapshot() (*InitWizardSnapshot, error) {
	if r == nil {
		return &InitWizardSnapshot{}, nil
	}
	plugins := pluginsSortedByName(r.all())
	snapshot := &InitWizardSnapshot{
		contributors: make([]InitContributorBinding, 0, len(plugins)),
		providers:    make([]InitProviderOption, 0, len(plugins)),
	}

	for _, p := range plugins {
		if provider, ok := p.(plugin.CIInfoProvider); ok {
			snapshot.providers = append(snapshot.providers, InitProviderOption{
				name:        provider.ProviderName(),
				description: provider.Description(),
			})
		}
		if contributor, ok := p.(initwiz.InitContributor); ok {
			snapshot.contributors = append(snapshot.contributors, InitContributorBinding{
				name:        contributor.Name(),
				contributor: contributor,
			})
			groups, err := contributor.InitGroups()
			if err != nil {
				return nil, &InitGroupError{Plugin: contributor.Name(), Err: err}
			}
			for i, group := range groups {
				if err := validateInitGroup(group); err != nil {
					return nil, &InitGroupError{Plugin: contributor.Name(), Err: fmt.Errorf("group %d: %w", i, err)}
				}
				snapshot.groups = append(snapshot.groups, InitGroupBinding{
					plugin: contributor.Name(),
					group:  group.Clone(),
				})
			}
		}
	}
	return snapshot, nil
}

func pluginsSortedByName(plugins []plugin.Plugin) []plugin.Plugin {
	out := append([]plugin.Plugin(nil), plugins...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

func validateInitGroup(group initwiz.InitGroup) error {
	if strings.TrimSpace(group.Title()) == "" {
		return errors.New("init group title is required")
	}
	switch group.Category() {
	case initwiz.CategoryProvider, initwiz.CategoryPipeline, initwiz.CategoryFeature, initwiz.CategoryDetail:
	default:
		return fmt.Errorf("unsupported init group category %q", group.Category())
	}
	if len(group.Fields()) == 0 {
		return fmt.Errorf("init group %q must contain at least one field", group.Title())
	}
	return nil
}
