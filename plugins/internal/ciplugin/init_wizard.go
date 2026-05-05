package ciplugin

import "github.com/edelwud/terraci/pkg/plugin/initwiz"

// PipelineGroup returns the standard "Pipeline" init wizard group used by
// every CI provider plugin (gitlab, github). The group is gated on the
// active provider via providerKey so providers that opt out of plan/apply
// orchestration don't accidentally consume the fields.
//
// The corresponding state keys are "plan_enabled" (bool, default true) and
// "auto_approve" (bool, default false). Plugins consume these via
// state.Bool("plan_enabled") / state.Bool("auto_approve") in BuildInitConfig.
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
				Title:       "Enable plan stage?",
				Description: "Generate separate plan + apply jobs",
				Type:        initwiz.FieldBool,
				Default:     true,
			},
			{
				Key:         "auto_approve",
				Title:       "Auto-approve applies?",
				Description: "Skip manual approval for terraform apply",
				Type:        initwiz.FieldBool,
				Default:     false,
			},
		},
	}
}
