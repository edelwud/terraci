package policy

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestPlugin_PipelineContribution_UsesAppContextServiceDir(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyengine.Config{Enabled: true, OnFailure: policyengine.ActionWarn})
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	contrib := p.PipelineContribution(appCtx)
	if contrib == nil {
		t.Fatal("PipelineContribution() returned nil")
	}
	if len(contrib.Jobs) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(contrib.Jobs))
	}

	job := contrib.Jobs[0]
	if job.Name != "policy-check" {
		t.Errorf("job.Name = %q, want %q", job.Name, "policy-check")
	}
	if job.Phase != pipeline.PhasePostPlan {
		t.Errorf("job.Phase = %v, want PhasePostPlan", job.Phase)
	}
	if len(job.Consumes) != 1 || job.Consumes[0].Kind != pipeline.ResourceKindPlanJSON || !job.Consumes[0].AllModules {
		t.Fatalf("job.Consumes = %#v, want all plan JSON", job.Consumes)
	}
	if !job.AllowFailure {
		t.Error("job.AllowFailure should be true when on_failure=warn")
	}

	if len(job.Produces) != 2 {
		t.Fatalf("job.Produces = %#v, want result and report", job.Produces)
	}
	wantPaths := []string{
		filepath.Join(appCtx.Config().ServiceDir, resultsFile),
		filepath.Join(appCtx.Config().ServiceDir, reportFile),
	}
	if !slices.Equal(producedPaths(job.Produces), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(job.Produces), wantPaths)
	}
	if job.Produces[0].Ref.Kind != pipeline.ResourceKindPluginResult || job.Produces[0].Ref.Producer != pluginName {
		t.Fatalf("result resource = %#v", job.Produces[0])
	}
	if job.Produces[1].Ref.Kind != pipeline.ResourceKindPluginReport || job.Produces[1].Ref.Producer != pluginName {
		t.Fatalf("report resource = %#v", job.Produces[1])
	}
}

func producedPaths(resources []pipeline.ResourceSpec) []string {
	paths := make([]string, 0, len(resources))
	for _, resource := range resources {
		paths = append(paths, resource.Path)
	}
	return paths
}
