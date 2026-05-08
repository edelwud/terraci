package cost

import (
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

func TestPlugin_PipelineContribution(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	contrib := p.PipelineContribution(appCtx)

	if contrib == nil {
		t.Fatal("PipelineContribution() returned nil")
	}
	if len(contrib.Jobs) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(contrib.Jobs))
	}

	job := contrib.Jobs[0]

	if job.Name != "cost-estimation" {
		t.Errorf("job.Name = %q, want %q", job.Name, "cost-estimation")
	}
	if len(job.Consumes) != 1 || job.Consumes[0].Kind != pipeline.ResourceKindPlanJSON || !job.Consumes[0].AllModules {
		t.Fatalf("job.Consumes = %#v, want all plan JSON", job.Consumes)
	}
	if !job.AllowFailure {
		t.Error("job.AllowFailure should be true")
	}
	if len(job.Commands) != 1 || job.Commands[0] != "terraci cost" {
		t.Errorf("job.Commands = %v, want [terraci cost]", job.Commands)
	}

	if len(job.Produces) != 2 {
		t.Fatalf("job.Produces = %#v, want result and report", job.Produces)
	}
	wantPaths := []string{pipeline.WorkspacePath(".terraci", resultsFile), pipeline.WorkspacePath(".terraci", reportFile)}
	if !sameStrings(producedPaths(job.Produces), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(job.Produces), wantPaths)
	}
	if job.Produces[0].Ref.Kind != pipeline.ResourceKindPluginResult || job.Produces[0].Ref.Producer != pluginName {
		t.Fatalf("result resource = %#v", job.Produces[0])
	}
	if job.Produces[1].Ref.Kind != pipeline.ResourceKindPluginReport || job.Produces[1].Ref.Producer != pluginName {
		t.Fatalf("report resource = %#v", job.Produces[1])
	}
}

func TestPlugin_PipelineContribution_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	base := newTestAppContext(t, t.TempDir())
	cfg := base.Config()
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
	job := contrib.Jobs[0]

	wantPaths := []string{resultsFile, reportFile}
	if !sameStrings(producedPaths(job.Produces), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(job.Produces), wantPaths)
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
