package ciplugin

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// PipelineGroup returns the standard "Pipeline" init wizard group used by
// every CI provider plugin (gitlab, github). The group is gated on the active
// provider via providerKey.
//
// The corresponding state key is "plan_enabled" (bool, default true).
func PipelineGroup(providerKey string) *initwiz.InitGroupSpec {
	return &initwiz.InitGroupSpec{
		Title:    "Pipeline",
		Category: initwiz.CategoryPipeline,
		Order:    100,
		ShowWhen: func(s *initwiz.StateMap) bool {
			return initwiz.ProviderKey.Get(s) == providerKey
		},
		Fields: []initwiz.InitField{
			initwiz.NewBoolField(initwiz.BoolFieldOptions{
				Key:         initwiz.PlanEnabledKey,
				Title:       "Enable plan jobs?",
				Description: "Generate plan jobs before apply jobs",
				Default:     true,
			}),
		},
	}
}
