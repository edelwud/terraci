package plugintest

import (
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// PipelineContributorContract describes expected generic contribution shape.
type PipelineContributorContract struct {
	Contributor      plugin.PipelineContributor
	AppContext       *plugin.AppContext
	ExpectedJobNames []string
}

// AssertPipelineContributor verifies deterministic contribution shape without
// asserting plugin-specific command/resource details.
func AssertPipelineContributor(tb testing.TB, c PipelineContributorContract) {
	tb.Helper()
	if c.Contributor == nil {
		tb.Fatal("Contributor is nil")
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}

	first := c.Contributor.PipelineContribution(appCtx)
	if first == nil {
		tb.Fatal("PipelineContribution() = nil")
	}
	gotNames := contributedJobNames(first)
	if len(c.ExpectedJobNames) > 0 && !slices.Equal(gotNames, c.ExpectedJobNames) {
		tb.Fatalf("contributed job names = %v, want %v", gotNames, c.ExpectedJobNames)
	}
	for _, name := range gotNames {
		if name == "" {
			tb.Fatalf("contributed job names contain an empty name: %v", gotNames)
		}
	}

	second := c.Contributor.PipelineContribution(appCtx)
	if second == nil {
		tb.Fatal("second PipelineContribution() = nil")
	}
	if gotAgain := contributedJobNames(second); !slices.Equal(gotAgain, gotNames) {
		tb.Fatalf("PipelineContribution() job names are not deterministic: first %v, second %v", gotNames, gotAgain)
	}
}

func contributedJobNames(contribution *pipeline.Contribution) []string {
	if contribution == nil || len(contribution.Jobs) == 0 {
		return nil
	}
	names := make([]string, 0, len(contribution.Jobs))
	for _, job := range contribution.Jobs {
		names = append(names, job.Name)
	}
	return names
}
