package cost

import (
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestPlugin_PipelineContribution(t *testing.T) {
	p := newTestPlugin(t)
	p.serviceDirRel = ".terraci"

	contrib := p.PipelineContribution()

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

	wantArtifact := filepath.Join(".terraci", resultsFile)
	if len(job.ArtifactPaths) != 1 || job.ArtifactPaths[0] != wantArtifact {
		t.Errorf("job.ArtifactPaths = %v, want [%s]", job.ArtifactPaths, wantArtifact)
	}
}

func TestPlugin_PipelineContribution_EmptyServiceDir(t *testing.T) {
	p := newTestPlugin(t)
	p.serviceDirRel = ""

	contrib := p.PipelineContribution()
	job := contrib.Jobs[0]

	if len(job.ArtifactPaths) != 1 || job.ArtifactPaths[0] != resultsFile {
		t.Errorf("job.ArtifactPaths = %v, want [%s]", job.ArtifactPaths, resultsFile)
	}
}

func TestPlugin_PipelineContribution_NoSteps(t *testing.T) {
	p := newTestPlugin(t)
	p.serviceDirRel = ".terraci"

	contrib := p.PipelineContribution()

	if len(contrib.Steps) != 0 {
		t.Errorf("steps count = %d, want 0 (cost plugin contributes jobs, not steps)", len(contrib.Steps))
	}
}
