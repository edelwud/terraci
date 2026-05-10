package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// PipelineContributor plugins add provider-independent jobs to the pipeline DAG.
type PipelineContributor interface {
	Plugin
	PipelineContribution(ctx *AppContext) *pipeline.Contribution
}

// PipelineContributionGate optionally controls whether an enabled plugin
// should contribute to the current pipeline.
type PipelineContributionGate interface {
	Plugin
	PipelineContributionEnabled(ctx *AppContext) bool
}
