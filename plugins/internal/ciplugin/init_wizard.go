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
			return s.Provider() == providerKey
		},
		Fields: []initwiz.InitField{
			{
				Key:         "plan_enabled",
				Title:       "Enable plan jobs?",
				Description: "Generate plan jobs before apply jobs",
				Type:        initwiz.FieldBool,
				Default:     true,
			},
		},
	}
}
