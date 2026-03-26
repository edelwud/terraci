package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// CollectContributions gathers pipeline contributions from all PipelineContributor plugins.
func CollectContributions() []*pipeline.Contribution {
	contributors := ByCapability[PipelineContributor]()
	contributions := make([]*pipeline.Contribution, 0, len(contributors))
	for _, c := range contributors {
		if contrib := c.PipelineContribution(); contrib != nil {
			contributions = append(contributions, contrib)
		}
	}
	return contributions
}
