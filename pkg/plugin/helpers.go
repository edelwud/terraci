package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// CollectContributions gathers pipeline contributions from all enabled
// PipelineContributor plugins. Pipeline contribution stays a framework-owned
// capability; plugins should derive paths from AppContext rather than cached
// lifecycle state.
func CollectContributions(ctx *AppContext) []*pipeline.Contribution {
	contributors := ByCapability[PipelineContributor]()
	contributions := make([]*pipeline.Contribution, 0, len(contributors))
	for _, c := range contributors {
		if cl, ok := c.(ConfigLoader); ok && !cl.IsEnabled() {
			continue
		}
		if contrib := c.PipelineContribution(ctx); contrib != nil {
			contributions = append(contributions, contrib)
		}
	}
	return contributions
}
