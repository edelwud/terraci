package policy

import (
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestPlugin_PipelineContribution_UsesAppContextServiceDir(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyengine.Config{Enabled: true, Decisions: policyengine.Decisions{Deny: policyengine.ActionWarn}})
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	plugintest.AssertPipelineContributor(t, plugintest.PipelineContributorContract{
		Contributor:      p,
		AppContext:       appCtx,
		ExpectedJobNames: []string{"policy-check"},
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
	if job.Name() != "policy-check" {
		t.Errorf("job.Name() = %q, want %q", job.Name(), "policy-check")
	}
	consumes := job.Consumes()
	if len(consumes) != 1 ||
		consumes[0].Kind != pipeline.ResourceKindPlanJSON ||
		consumes[0].Selector.Scope != pipeline.ResourceScopeAllModules {
		t.Fatalf("job.Consumes() = %#v, want all plan JSON", consumes)
	}
	if !job.AllowFailure() {
		t.Error("job.AllowFailure should be true when no global policy action can block")
	}
	commands := job.Commands()
	if len(commands) != 1 || commands[0] != "terraci policy check --format text" {
		t.Fatalf("job.Commands() = %#v, want only policy check", commands)
	}

	produces := job.Produces()
	if len(produces) != 2 {
		t.Fatalf("job.Produces() = %#v, want result and report", produces)
	}
	wantPaths := []string{
		pipeline.WorkspacePath(appCtx.Config().ServiceDir(), resultsFile),
		pipeline.WorkspacePath(appCtx.Config().ServiceDir(), reportFile),
	}
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

func producedPaths(resources []pipeline.ResourceSpec) []string {
	paths := make([]string, 0, len(resources))
	for _, resource := range resources {
		paths = append(paths, resource.Path)
	}
	return paths
}
