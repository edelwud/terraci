package policy

import (
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestPlugin_PipelineContribution_UsesAppContextServiceDir(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyengine.Config{Enabled: true, OnFailure: policyengine.ActionWarn})
	appCtx := newTestAppContext(t.TempDir())

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

	wantArtifact := filepath.Join(appCtx.Config().ServiceDir, resultsFile)
	if len(job.ArtifactPaths) != 1 || job.ArtifactPaths[0] != wantArtifact {
		t.Errorf("job.ArtifactPaths = %v, want [%s]", job.ArtifactPaths, wantArtifact)
	}
}
