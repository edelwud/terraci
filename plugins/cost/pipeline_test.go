package cost

import (
	"path/filepath"
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
	if job.Phase != pipeline.PhasePostPlan {
		t.Errorf("job.Phase = %v, want PhasePostPlan", job.Phase)
	}
	if !job.DependsOnPlan {
		t.Error("job.DependsOnPlan should be true")
	}
	if !job.AllowFailure {
		t.Error("job.AllowFailure should be true")
	}
	if len(job.Commands) != 1 || job.Commands[0] != "terraci cost" {
		t.Errorf("job.Commands = %v, want [terraci cost]", job.Commands)
	}

	if job.Artifact.Name != pipeline.ResultArtifactName(jobName) {
		t.Errorf("job.Artifact.Name = %q, want %q", job.Artifact.Name, pipeline.ResultArtifactName(jobName))
	}
	wantPaths := []string{filepath.Join(".terraci", resultsFile), filepath.Join(".terraci", reportFile)}
	if !sameStrings(job.Artifact.Paths, wantPaths) {
		t.Errorf("job.Artifact.Paths = %v, want %v", job.Artifact.Paths, wantPaths)
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
	if !sameStrings(job.Artifact.Paths, wantPaths) {
		t.Errorf("job.Artifact.Paths = %v, want %v", job.Artifact.Paths, wantPaths)
	}
}

func TestPlugin_PipelineContribution_NoSteps(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	contrib := p.PipelineContribution(appCtx)

	if len(contrib.Steps) != 0 {
		t.Errorf("steps count = %d, want 0 (cost plugin contributes jobs, not steps)", len(contrib.Steps))
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
