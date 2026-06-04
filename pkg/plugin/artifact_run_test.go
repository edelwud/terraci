package plugin

import (
	"errors"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestNewArtifactRun_NilAppContextUsesExplicitMetadata(t *testing.T) {
	generatedAt := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	run, err := NewArtifactRun(nil, ArtifactRunOptions{
		Producer:    "cost",
		Fingerprint: "explicit",
		GeneratedAt: generatedAt,
	})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}

	artifact := run.Artifact()
	if artifact.PlanResultsFingerprint() != "explicit" {
		t.Fatalf("fingerprint = %q, want explicit", artifact.PlanResultsFingerprint())
	}
	if !artifact.GeneratedAt().Equal(generatedAt) {
		t.Fatalf("generatedAt = %v, want %v", artifact.GeneratedAt(), generatedAt)
	}
	if artifact.ServiceDir() != "" || artifact.WorkDir() != "" {
		t.Fatalf("nil AppContext dirs = (%q, %q), want empty", artifact.ServiceDir(), artifact.WorkDir())
	}
}

func TestNewArtifactRun_UsesAppContextAndCollectionMetadata(t *testing.T) {
	collection := testPluginPlanResultCollection(t, ci.PlanResultCollectionOptions{
		CommitSHA:  "collection-sha",
		PipelineID: "collection-pipeline",
		Results: []ci.PlanResult{testPluginPlanResult(t, ci.PlanResultOptions{
			ModuleID:   "svc",
			ModulePath: "svc",
			Status:     ci.PlanStatusChanges,
		})},
	})
	appCtx := NewAppContext(AppContextOptions{
		WorkDir:    "/repo",
		ServiceDir: "/repo/.terraci",
	})

	run, err := NewArtifactRun(appCtx, ArtifactRunOptions{
		Producer:   "policy",
		Collection: collection,
	})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}

	artifact := run.Artifact()
	if artifact.WorkDir() != "/repo" || artifact.ServiceDir() != "/repo/.terraci" {
		t.Fatalf("dirs = (%q, %q), want AppContext dirs", artifact.WorkDir(), artifact.ServiceDir())
	}
	if artifact.CommitSHA() != "collection-sha" || artifact.PipelineID() != "collection-pipeline" {
		t.Fatalf("metadata = (%q, %q), want collection metadata", artifact.CommitSHA(), artifact.PipelineID())
	}
	if artifact.PlanResultsFingerprint() != collection.Fingerprint() {
		t.Fatalf("fingerprint = %q, want collection fingerprint", artifact.PlanResultsFingerprint())
	}
}

func TestNewArtifactRun_CIProviderMetadataOverridesCollection(t *testing.T) {
	collection := testPluginPlanResultCollection(t, ci.PlanResultCollectionOptions{
		CommitSHA:  "collection-sha",
		PipelineID: "collection-pipeline",
	})
	appCtx := NewAppContext(AppContextOptions{
		Resolvers: NewResolverSet(ResolverSetOptions{
			CI: artifactRunCIResolver{provider: artifactRunProvider{commitSHA: "ci-sha", pipelineID: "ci-pipeline"}},
		}),
	})

	run, err := NewArtifactRun(appCtx, ArtifactRunOptions{
		Producer:   "summary",
		Collection: collection,
	})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}

	artifact := run.Artifact()
	if artifact.CommitSHA() != "ci-sha" || artifact.PipelineID() != "ci-pipeline" {
		t.Fatalf("metadata = (%q, %q), want CI provider metadata", artifact.CommitSHA(), artifact.PipelineID())
	}
}

func TestNewArtifactRun_IgnoresMissingCIProvider(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{
		Resolvers: NewResolverSet(ResolverSetOptions{CI: artifactRunCIResolver{err: errors.New("not configured")}}),
	})

	if _, err := NewArtifactRun(appCtx, ArtifactRunOptions{Producer: "tfupdate"}); err != nil {
		t.Fatalf("NewArtifactRun() error = %v, want best-effort CI metadata", err)
	}
}

func TestNewArtifactRun_ExplicitFingerprintWins(t *testing.T) {
	collection := testPluginPlanResultCollection(t, ci.PlanResultCollectionOptions{
		Results: []ci.PlanResult{testPluginPlanResult(t, ci.PlanResultOptions{
			ModuleID:   "svc",
			ModulePath: "svc",
			Status:     ci.PlanStatusChanges,
		})},
	})
	run, err := NewArtifactRun(NewAppContext(AppContextOptions{}), ArtifactRunOptions{
		Producer:    "cost",
		Collection:  collection,
		Fingerprint: "explicit",
	})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}
	if got := run.Artifact().PlanResultsFingerprint(); got != "explicit" {
		t.Fatalf("fingerprint = %q, want explicit", got)
	}
}

type artifactRunCIResolver struct {
	provider artifactRunProvider
	err      error
}

func (r artifactRunCIResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	if r.err != nil {
		return nil, r.err
	}
	return NewResolvedCIProvider(r.provider, r.provider, r.provider, nil), nil
}

type artifactRunProvider struct {
	commitSHA  string
	pipelineID string
}

func (p artifactRunProvider) Name() string         { return "ci" }
func (p artifactRunProvider) Description() string  { return "CI" }
func (p artifactRunProvider) DetectEnv() bool      { return true }
func (p artifactRunProvider) ProviderName() string { return "ci" }
func (p artifactRunProvider) PipelineID() string   { return p.pipelineID }
func (p artifactRunProvider) CommitSHA() string    { return p.commitSHA }

func (p artifactRunProvider) NewGenerator(*pipeline.IR) (pipeline.Generator, error) {
	return artifactRunGenerator{}, nil
}

type artifactRunGenerator struct{}

func (artifactRunGenerator) Generate() (pipeline.GeneratedPipeline, error) { return nil, nil }
func (artifactRunGenerator) DryRun() (*pipeline.DryRunResult, error)       { return nil, nil }

func testPluginPlanResult(tb testing.TB, opts ci.PlanResultOptions) ci.PlanResult {
	tb.Helper()
	result, err := ci.NewPlanResult(opts)
	if err != nil {
		tb.Fatalf("NewPlanResult() error = %v", err)
	}
	return result
}

func testPluginPlanResultCollection(tb testing.TB, opts ci.PlanResultCollectionOptions) *ci.PlanResultCollection {
	tb.Helper()
	collection, err := ci.NewPlanResultCollection(opts)
	if err != nil {
		tb.Fatalf("NewPlanResultCollection() error = %v", err)
	}
	return collection
}
