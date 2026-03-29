package update

import (
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func TestPlugin_PipelineContribution(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true, Pipeline: true})
	p.serviceDirRel = ".terraci"

	contrib := p.PipelineContribution()
	if contrib == nil {
		t.Fatal("PipelineContribution() returned nil")
	}
	if len(contrib.Jobs) != 1 {
		t.Fatalf("jobs count = %d, want 1", len(contrib.Jobs))
	}

	job := contrib.Jobs[0]

	if job.Name != "dependency-update-check" {
		t.Errorf("job.Name = %q, want %q", job.Name, "dependency-update-check")
	}
	if job.Phase != pipeline.PhasePrePlan {
		t.Errorf("job.Phase = %v, want PhasePrePlan", job.Phase)
	}
	if job.DependsOnPlan {
		t.Error("job.DependsOnPlan should be false")
	}
	if !job.AllowFailure {
		t.Error("job.AllowFailure should be true")
	}
	if len(job.Commands) != 1 || job.Commands[0] != "terraci update" {
		t.Errorf("job.Commands = %v, want [terraci update]", job.Commands)
	}

	wantArtifact := filepath.Join(".terraci", resultsFile)
	if len(job.ArtifactPaths) != 1 || job.ArtifactPaths[0] != wantArtifact {
		t.Errorf("job.ArtifactPaths = %v, want [%s]", job.ArtifactPaths, wantArtifact)
	}
}

func TestPlugin_PipelineContribution_NotConfigured(t *testing.T) {
	p := newTestPlugin(t)
	// No config set — Config() returns nil.
	contrib := p.PipelineContribution()
	if contrib != nil {
		t.Errorf("PipelineContribution() = %v, want nil for unconfigured plugin", contrib)
	}
}

func TestPlugin_PipelineContribution_PipelineFalse(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true, Pipeline: false})

	contrib := p.PipelineContribution()
	if contrib != nil {
		t.Errorf("PipelineContribution() = %v, want nil when Pipeline=false", contrib)
	}
}

func TestPlugin_PipelineContribution_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true, Pipeline: true})
	p.serviceDirRel = ""

	contrib := p.PipelineContribution()
	job := contrib.Jobs[0]

	if len(job.ArtifactPaths) != 1 || job.ArtifactPaths[0] != resultsFile {
		t.Errorf("job.ArtifactPaths = %v, want [%s]", job.ArtifactPaths, resultsFile)
	}
}

func TestPlugin_PipelineContribution_NoSteps(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &updateengine.UpdateConfig{Enabled: true, Pipeline: true})
	p.serviceDirRel = ".terraci"

	contrib := p.PipelineContribution()
	if len(contrib.Steps) != 0 {
		t.Errorf("steps count = %d, want 0 (update plugin contributes jobs, not steps)", len(contrib.Steps))
	}
}
