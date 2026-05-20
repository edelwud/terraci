package cost

import (
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestPlugin_PipelineContribution(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	plugintest.AssertPipelineContributor(t, plugintest.PipelineContributorContract{
		Contributor:      p,
		AppContext:       appCtx,
		ExpectedJobNames: []string{"cost-estimation"},
	})

	contrib := p.PipelineContribution(appCtx)

	if contrib == nil {
		t.Fatal("PipelineContribution() returned nil")
	}
	jobs := contrib.Jobs()
	if len(jobs) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(jobs))
	}

	job := jobs[0]

	if job.Name() != "cost-estimation" {
		t.Errorf("job.Name() = %q, want %q", job.Name(), "cost-estimation")
	}
	consumes := job.Consumes()
	if len(consumes) != 1 ||
		consumes[0].Kind != pipeline.ResourceKindPlanJSON ||
		consumes[0].Selector.Scope != pipeline.ResourceScopeAllModules {
		t.Fatalf("job.Consumes() = %#v, want all plan JSON", consumes)
	}
	if !job.AllowFailure() {
		t.Error("job.AllowFailure should be true")
	}
	commands := job.Commands()
	if len(commands) != 1 || commands[0] != "terraci cost" {
		t.Errorf("job.Commands() = %v, want [terraci cost]", commands)
	}

	produces := job.Produces()
	if len(produces) != 2 {
		t.Fatalf("job.Produces() = %#v, want result and report", produces)
	}
	wantPaths := []string{pipeline.WorkspacePath(".terraci", resultsFile), pipeline.WorkspacePath(".terraci", reportFile)}
	if !sameStrings(producedPaths(produces), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(produces), wantPaths)
	}
	if produces[0].Ref.Kind != pipeline.ResourceKindPluginResult || produces[0].Ref.Producer != pluginName {
		t.Fatalf("result resource = %#v", produces[0])
	}
	if produces[1].Ref.Kind != pipeline.ResourceKindPluginReport || produces[1].Ref.Producer != pluginName {
		t.Fatalf("report resource = %#v", produces[1])
	}
}

func TestPlugin_PipelineContribution_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	base := newTestAppContext(t, t.TempDir())
	cfg := base.Config().MutableCopy()
	cfg.ServiceDir = ""
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     cfg,
		WorkDir:    base.WorkDir(),
		ServiceDir: base.ServiceDir(),
		Version:    base.Version(),
		Reports:    base.Reports(),
		Resolver:   base.Resolver(),
	})

	contrib := p.PipelineContribution(appCtx)
	job := contrib.Jobs()[0]

	wantPaths := []string{resultsFile, reportFile}
	if !sameStrings(producedPaths(job.Produces()), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(job.Produces()), wantPaths)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func producedPaths(resources []pipeline.ResourceSpec) []string {
	paths := make([]string, 0, len(resources))
	for _, resource := range resources {
		paths = append(paths, resource.Path)
	}
	return paths
}
