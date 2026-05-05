package cmd

import (
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/workflow"
)

func TestComputeLibraryModulesSummary_NoConfig(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	result := &workflow.Result{Graph: graph.NewDependencyGraph()}

	if got := computeLibraryModulesSummary(cfg, result); got != nil {
		t.Errorf("expected nil summary when library_modules is unset, got %+v", got)
	}
}

func TestComputeLibraryModulesSummary_OrphanDetection(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}

	used := discovery.TestLibraryModule("_modules/kafka", "/abs/_modules/kafka")
	orphan := discovery.TestLibraryModule("_modules/unused", "/abs/_modules/unused")
	nestedConsumed := discovery.TestLibraryModule("_modules/kafka_acl", "/abs/_modules/kafka_acl")

	g := graph.NewDependencyGraph()
	g.AddLibraryUsage("/abs/_modules/kafka", "platform/stage/eu-central-1/msk")
	// nested usage should keep parent module from being reported as orphan
	g.AddLibraryUsage("/abs/_modules/kafka_acl/sub", "platform/stage/eu-central-1/msk")

	result := &workflow.Result{
		Libraries: workflow.NewModuleSet([]*discovery.Module{used, orphan, nestedConsumed}),
		Graph:     g,
	}

	summary := computeLibraryModulesSummary(cfg, result)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.ConfiguredPaths != 1 {
		t.Errorf("ConfiguredPaths = %d, want 1", summary.ConfiguredPaths)
	}
	if summary.Discovered != 3 {
		t.Errorf("Discovered = %d, want 3", summary.Discovered)
	}
	if summary.Consumers != 1 {
		t.Errorf("Consumers = %d, want 1 (one unique executable)", summary.Consumers)
	}
	if !reflect.DeepEqual(summary.Orphans, []string{"_modules/unused"}) {
		t.Errorf("Orphans = %v, want [_modules/unused]", summary.Orphans)
	}
}
