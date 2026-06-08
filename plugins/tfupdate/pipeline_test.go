package tfupdate

import (
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func TestPlugin_PipelineContribution(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true, Pipeline: true})
	appCtx := newTestAppContext(t, t.TempDir())

	plugintest.AssertPipelineContributor(t, plugintest.PipelineContributorContract{
		Contributor:      p,
		AppContext:       appCtx,
		ExpectedJobNames: []string{"tfupdate-check"},
	})

	contrib, err := p.PipelineContribution(appCtx)
	if err != nil {
		t.Fatalf("PipelineContribution() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("PipelineContribution() returned nil")
	}
	jobs := contrib.Jobs()
	if len(jobs) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(jobs))
	}

	job := jobs[0]

	if job.Name() != "tfupdate-check" {
		t.Errorf("job.Name() = %q, want %q", job.Name(), "tfupdate-check")
	}
	if consumes := job.Consumes(); len(consumes) != 0 {
		t.Fatalf("job.Consumes() = %#v, want none", consumes)
	}
	if !job.AllowFailure() {
		t.Error("job.AllowFailure should be true")
	}
	commands := job.Commands()
	if len(commands) != 1 || commands[0] != "terraci tfupdate" {
		t.Errorf("job.Commands() = %v, want [terraci tfupdate]", commands)
	}

	produces := job.Produces()
	if len(produces) != 2 {
		t.Fatalf("job.Produces() = %#v, want result and report", produces)
	}
	wantPaths := []string{pipeline.WorkspacePath(".terraci", resultsFile), pipeline.WorkspacePath(".terraci", reportFile)}
	if !slices.Equal(producedPaths(produces), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(produces), wantPaths)
	}
	if produces[0].Ref.Kind != pipeline.ResourceKindPluginResult || produces[0].Ref.Producer != pluginName {
		t.Fatalf("result resource = %#v", produces[0])
	}
	if produces[1].Ref.Kind != pipeline.ResourceKindPluginReport || produces[1].Ref.Producer != pluginName {
		t.Fatalf("report resource = %#v", produces[1])
	}
}

func TestPlugin_PipelineContribution_NotConfigured(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	enabled, err := p.PipelineContributionEnabled(appCtx)
	if err != nil {
		t.Fatalf("PipelineContributionEnabled() error = %v", err)
	}
	if enabled {
		t.Error("PipelineContributionEnabled() = true, want false for unconfigured plugin")
	}
}

func TestPlugin_PipelineContribution_PipelineFalse(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true, Pipeline: false})

	appCtx := newTestAppContext(t, t.TempDir())

	enabled, err := p.PipelineContributionEnabled(appCtx)
	if err != nil {
		t.Fatalf("PipelineContributionEnabled() error = %v", err)
	}
	if enabled {
		t.Error("PipelineContributionEnabled() = true, want false when Pipeline=false")
	}
}

func TestPlugin_PipelineContribution_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true, Pipeline: true})
	base := newTestAppContext(t, t.TempDir())
	cfg, err := config.Build(config.BuildOptions{ServiceDirSet: true})
	if err != nil {
		t.Fatalf("config.Build() error = %v", err)
	}
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     cfg,
		WorkDir:    base.WorkDir(),
		ServiceDir: base.ServiceDir(),
		Version:    base.Version(),
		Reports:    base.Reports(),
	})

	contrib, err := p.PipelineContribution(appCtx)
	if err != nil {
		t.Fatalf("PipelineContribution() error = %v", err)
	}
	job := contrib.Jobs()[0]

	wantPaths := []string{resultsFile, reportFile}
	if !slices.Equal(producedPaths(job.Produces()), wantPaths) {
		t.Errorf("produced paths = %v, want %v", producedPaths(job.Produces()), wantPaths)
	}
}

func producedPaths(resources []pipeline.ResourceSpec) []string {
	paths := make([]string, 0, len(resources))
	for _, resource := range resources {
		paths = append(paths, resource.Path)
	}
	return paths
}
