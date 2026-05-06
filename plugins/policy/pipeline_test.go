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
	if !job.DependsOnPlan {
		t.Error("job.DependsOnPlan should be true")
	}
	if !job.AllowFailure {
		t.Error("job.AllowFailure should be true when on_failure=warn")
	}

	if job.Artifact.Name != pipeline.ResultArtifactName(jobName) {
		t.Errorf("job.Artifact.Name = %q, want %q", job.Artifact.Name, pipeline.ResultArtifactName(jobName))
	}
	wantPaths := []string{
		filepath.Join(appCtx.Config().ServiceDir, resultsFile),
		filepath.Join(appCtx.Config().ServiceDir, reportFile),
	}
	if !slices.Equal(job.Artifact.Paths, wantPaths) {
		t.Errorf("job.Artifact.Paths = %v, want %v", job.Artifact.Paths, wantPaths)
	}
}
