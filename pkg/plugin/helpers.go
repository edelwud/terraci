package plugin

import "github.com/edelwud/terraci/pkg/pipeline"

// CollectContributions gathers pipeline contributions from all enabled PipelineContributor plugins.
// Plugins that implement ConfigLoader and are not enabled are skipped (framework-level filtering).
func CollectContributions() []*pipeline.Contribution {
	contributors := ByCapability[PipelineContributor]()
	contributions := make([]*pipeline.Contribution, 0, len(contributors))
	for _, c := range contributors {
		if cl, ok := c.(ConfigLoader); ok && !cl.IsEnabled() {
			continue
		}
		if contrib := c.PipelineContribution(); contrib != nil {
			contributions = append(contributions, contrib)
		}
	}
	return contributions
}
