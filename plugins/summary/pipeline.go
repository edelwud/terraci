package summary

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// PipelineContribution returns the summary job contribution for the pipeline DAG.
// Framework guarantees this is only called when IsEnabled() == true.
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) (*pipeline.Contribution, error) {
	job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:     "terraci-summary",
		Commands: []string{"terraci summary"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			pipeline.AllPluginResources(pipeline.ResourceKindPluginReport, true),
		},
		AllowFailure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("build summary pipeline job: %w", err)
	}
	contribution, err := pipeline.NewContribution(job)
	if err != nil {
		return nil, fmt.Errorf("build summary pipeline contribution: %w", err)
	}
	return contribution, nil
}
