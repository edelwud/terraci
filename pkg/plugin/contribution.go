package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// PipelineContributor plugins add steps or jobs to the generated CI pipeline.
type PipelineContributor interface {
	Plugin
	PipelineContribution(ctx *AppContext) *pipeline.Contribution
}
