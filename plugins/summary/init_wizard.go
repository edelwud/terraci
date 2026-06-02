// Package summary provides the summary plugin for TerraCi.
package summary

import (
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

// InitContributor — contributes summary field to the init wizard.

const initGroupOrder = 199 // before cost/policy so it appears first in Features

// InitGroups returns the summary plugin's form group for the init wizard.
func (p *Plugin) InitGroups() ([]initwiz.InitGroup, error) {
	field, err := initwiz.NewBoolField(initwiz.BoolFieldOptions{
		Key:         initwiz.SummaryEnabledKey,
		Title:       "Enable plan summaries?",
		Description: "Post Terraform plan summaries as merge and pull request comments",
		Default:     true,
	})
	if err != nil {
		return nil, err
	}
	group, err := initwiz.NewInitGroup(initwiz.InitGroupOptions{
		Title:    "Summary",
		Category: initwiz.CategoryFeature,
		Order:    initGroupOrder,
		Fields:   []initwiz.InitField{field},
	})
	if err != nil {
		return nil, err
	}
	return []initwiz.InitGroup{group}, nil
}

// BuildInitConfig builds the summary plugin config from wizard state.
func (p *Plugin) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	enabled := initwiz.SummaryEnabledKey.Get(state)
	if enabled {
		return nil, nil
	}
	return initwiz.NewInitContribution(pluginName, &summaryengine.Config{Enabled: &enabled})
}
