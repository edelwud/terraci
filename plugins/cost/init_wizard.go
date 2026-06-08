package cost

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// InitContributor — contributes cost estimation field to the init wizard.

const initGroupOrder = 200

var initConfigKey = config.MustExtensionKey(pluginName)

// providerEnabledKey returns the typed StateMap key for a cloud provider's
// "enabled?" toggle. Centralizes the "cost.providers.<id>.enabled" string
// so the InitGroups field definition and BuildInitConfig consumer cannot
// drift apart on a typo.
func providerEnabledKey(providerID string) initwiz.StateKey[bool] {
	return initwiz.MustStateKey[bool](fmt.Sprintf("cost.providers.%s.enabled", providerID))
}

// InitGroups returns the init wizard group spec for cost estimation.
//
// Cloud providers register themselves through cloud.Register; the wizard
// discovers them at runtime so a future GCP/Azure provider plugin appears
// in the form without a wizard change. If no cloud providers are
// registered, the group collapses to a single explanatory toggle.
func (p *Plugin) InitGroups() ([]initwiz.InitGroup, error) {
	clouds := cloud.Providers()

	fields := make([]initwiz.InitField, 0, len(clouds))
	for _, c := range clouds {
		def := c.Definition()
		field, err := initwiz.NewBoolField(initwiz.BoolFieldOptions{
			Key:         providerEnabledKey(def.Manifest.ID),
			Title:       fmt.Sprintf("Estimate %s costs?", def.Manifest.DisplayName),
			Description: fmt.Sprintf("Run cost estimation against %s plan output", def.Manifest.DisplayName),
			Default:     false,
		})
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	if len(fields) == 0 {
		// No registered cloud providers — surface a single inert toggle so
		// the form group still renders with a clear "no clouds available"
		// label. Users compiling a custom binary without any cloud-pricing
		// plugins will see this immediately.
		field, err := initwiz.NewBoolField(initwiz.BoolFieldOptions{
			Key:         initwiz.MustStateKey[bool]("cost.no_providers"),
			Title:       "Cost estimation unavailable",
			Description: "No cloud-pricing providers compiled into this binary",
			Default:     false,
		})
		if err != nil {
			return nil, err
		}
		fields = []initwiz.InitField{field}
	}

	group, err := initwiz.NewInitGroup(initwiz.InitGroupOptions{
		Title:    costReportTitle,
		Category: initwiz.CategoryFeature,
		Order:    initGroupOrder,
		Fields:   fields,
	})
	if err != nil {
		return nil, err
	}
	return []initwiz.InitGroup{group}, nil
}

// BuildInitConfig builds the cost estimation init contribution. Walks the
// same set of registered cloud providers and emits config for every one the
// user enabled in the wizard. Skips contribution entirely when no provider
// is enabled — keeps `.terraci.yaml` clean of an empty `extensions.cost`.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	providers := model.CostProvidersConfig{}
	for _, c := range cloud.Providers() {
		def := c.Definition()
		if providerEnabledKey(def.Manifest.ID).Get(state) {
			providers[def.Manifest.ID] = model.ProviderConfig{Enabled: true}
		}
	}

	if len(providers) == 0 {
		return nil, nil
	}

	return initwiz.NewInitContribution(initConfigKey, &model.CostConfig{Providers: providers})
}
