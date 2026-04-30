package workflow

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
)

type stubChangeDetector struct {
	changedModules   []*discovery.Module
	changedLibraries []string
}

func (d stubChangeDetector) Name() string        { return "stub-detector" }
func (d stubChangeDetector) Description() string { return "stub detector" }

func (d stubChangeDetector) DetectChangedModules(context.Context, string, string, *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	return d.changedModules, nil, nil
}

func (d stubChangeDetector) DetectChangedLibraries(context.Context, string, string, []string) ([]string, error) {
	return d.changedLibraries, nil
}

func TestResolveTargets_ModulePathIntersectedWithChangedModules(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()

	result := &Result{
		All:      NewModuleSet([]*discovery.Module{vpc, eks}),
		Filtered: NewModuleSet([]*discovery.Module{vpc, eks}),
		Graph: func() *graph.DependencyGraph {
			depGraph := graph.NewDependencyGraph()
			depGraph.AddNode(vpc)
			depGraph.AddNode(eks)
			depGraph.AddEdge(eks.ID(), vpc.ID())
			return depGraph
		}(),
	}

	targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
		ChangedOnly: true,
		ModulePath:  vpc.RelativePath,
		ChangeDetectorResolver: func() (ChangeDetector, error) {
			return stubChangeDetector{changedModules: []*discovery.Module{eks}}, nil
		},
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if got := moduleIDs(targets); !reflect.DeepEqual(got, []string{vpc.ID()}) {
		t.Fatalf("module ids = %v, want [%s]", got, vpc.ID())
	}
}

func TestResolveTargets_ChangedLibrariesRespectFilters(t *testing.T) {
	t.Parallel()

	stage := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	prod := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}

	depGraph := graph.BuildFromDependencies([]*discovery.Module{stage, prod}, nil)
	depGraph.AddLibraryUsage("_modules/network", stage.ID())
	depGraph.AddLibraryUsage("_modules/network", prod.ID())

	result := &Result{
		All:      NewModuleSet([]*discovery.Module{stage, prod}),
		Filtered: NewModuleSet([]*discovery.Module{stage}),
		Graph:    depGraph,
	}

	targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
		ChangedOnly: true,
		Filters:     &filter.Flags{SegmentArgs: []string{"environment=stage"}},
		ChangeDetectorResolver: func() (ChangeDetector, error) {
			return stubChangeDetector{changedLibraries: []string{"_modules/network"}}, nil
		},
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if got := moduleIDs(targets); !reflect.DeepEqual(got, []string{stage.ID()}) {
		t.Fatalf("module ids = %v, want [%s]", got, stage.ID())
	}
}

func TestResolveTargets_ChangedOnlyNoTargetsReturnsEmpty(t *testing.T) {
	t.Parallel()

	app := discovery.TestModule("svc", "stage", "eu", "app")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()
	result := &Result{
		All:      NewModuleSet([]*discovery.Module{app}),
		Filtered: NewModuleSet([]*discovery.Module{app}),
		Graph:    graph.BuildFromDependencies([]*discovery.Module{app}, nil),
	}

	targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
		ChangedOnly: true,
		ModulePath:  "missing/path",
		ChangeDetectorResolver: func() (ChangeDetector, error) {
			return stubChangeDetector{changedModules: []*discovery.Module{app}}, nil
		},
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("target count = %d, want 0", len(targets))
	}
}

func TestResolveTargets_ModulePathDoesNotMutateFilteredModules(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()
	result := &Result{
		All:      NewModuleSet([]*discovery.Module{vpc, eks}),
		Filtered: NewModuleSet([]*discovery.Module{vpc, eks}),
		Graph:    graph.BuildFromDependencies([]*discovery.Module{vpc, eks}, nil),
	}

	targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
		ModulePath: vpc.RelativePath,
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if got := moduleIDs(targets); !reflect.DeepEqual(got, []string{vpc.ID()}) {
		t.Fatalf("target ids = %v, want [%s]", got, vpc.ID())
	}
	if got := moduleIDs(result.Filtered.Modules); !reflect.DeepEqual(got, []string{vpc.ID(), eks.ID()}) {
		t.Fatalf("filtered modules were mutated: %v", got)
	}
}

func TestResolveTargets_ChangedOnlyAppliesModuleAfterFiltersAndAffectedModules(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	prodVPC := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()
	cfg.Exclude = []string{"platform/prod/**"}
	flags := &filter.Flags{SegmentArgs: []string{"environment=stage"}}

	depGraph := graph.NewDependencyGraph()
	for _, module := range []*discovery.Module{vpc, eks, prodVPC} {
		depGraph.AddNode(module)
	}
	depGraph.AddEdge(eks.ID(), vpc.ID())
	filteredModules := ApplyFilters(cfg, flags, []*discovery.Module{vpc, eks, prodVPC})

	result := &Result{
		All:      NewModuleSet([]*discovery.Module{vpc, eks, prodVPC}),
		Filtered: NewModuleSet(filteredModules),
		Graph:    depGraph,
	}

	tests := []struct {
		name       string
		modulePath string
		wantIDs    []string
	}{
		{
			name:       "module path intersects affected and filtered modules",
			modulePath: vpc.RelativePath,
			wantIDs:    []string{vpc.ID()},
		},
		{
			name:       "module path outside affected set returns empty",
			modulePath: prodVPC.RelativePath,
			wantIDs:    []string{},
		},
		{
			name:    "without module path returns affected modules inside filters",
			wantIDs: []string{eks.ID(), vpc.ID()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
				ChangedOnly: true,
				ModulePath:  tt.modulePath,
				Filters:     flags,
				ChangeDetectorResolver: func() (ChangeDetector, error) {
					return stubChangeDetector{changedModules: []*discovery.Module{eks}}, nil
				},
			})
			if err != nil {
				t.Fatalf("resolveTargets() error = %v", err)
			}

			if got := sortedModuleIDs(targets); !reflect.DeepEqual(got, tt.wantIDs) {
				t.Fatalf("module ids = %v, want %v", got, tt.wantIDs)
			}
		})
	}
}

func TestResolveTargets_ChangedOnlyPreservesFilteredModuleOrder(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	app := discovery.TestModule("platform", "stage", "eu-central-1", "app")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()

	depGraph := graph.NewDependencyGraph()
	for _, module := range []*discovery.Module{vpc, eks, app} {
		depGraph.AddNode(module)
	}

	result := &Result{
		All:      NewModuleSet([]*discovery.Module{vpc, eks, app}),
		Filtered: NewModuleSet([]*discovery.Module{vpc, eks, app}),
		Graph:    depGraph,
	}

	targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
		ChangedOnly: true,
		ChangeDetectorResolver: func() (ChangeDetector, error) {
			return stubChangeDetector{changedModules: []*discovery.Module{app, vpc}}, nil
		},
	})
	if err != nil {
		t.Fatalf("resolveTargets() error = %v", err)
	}

	if got := moduleIDs(targets); !reflect.DeepEqual(got, []string{vpc.ID(), app.ID()}) {
		t.Fatalf("module ids = %v, want [%s %s]", got, vpc.ID(), app.ID())
	}
}

func TestResolveTargets_ChangedLibrariesIntersectModuleAndFilters(t *testing.T) {
	t.Parallel()

	stageVPC := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	stageEKS := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	prodVPC := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	cfg := config.DefaultConfig()
	workDir := t.TempDir()
	cfg.Exclude = []string{"platform/prod/**"}
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}
	flags := &filter.Flags{SegmentArgs: []string{"environment=stage"}}

	tests := []struct {
		name             string
		libraryUsagePath string
		modulePath       string
		wantIDs          []string
	}{
		{
			name:             "module path intersects library affected dependency",
			libraryUsagePath: "_modules/network",
			modulePath:       stageVPC.RelativePath,
			wantIDs:          []string{stageVPC.ID()},
		},
		{
			name:             "without module path returns library users and dependencies after filters",
			libraryUsagePath: "_modules/network",
			wantIDs:          []string{stageEKS.ID(), stageVPC.ID()},
		},
		{
			name:             "changed parent library path matches tracked child usage",
			libraryUsagePath: "_modules/network/vpc",
			wantIDs:          []string{stageEKS.ID(), stageVPC.ID()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			depGraph := graph.NewDependencyGraph()
			for _, module := range []*discovery.Module{stageVPC, stageEKS, prodVPC} {
				depGraph.AddNode(module)
			}
			depGraph.AddEdge(stageEKS.ID(), stageVPC.ID())
			depGraph.AddLibraryUsage(tt.libraryUsagePath, stageEKS.ID())
			depGraph.AddLibraryUsage(tt.libraryUsagePath, prodVPC.ID())
			filteredModules := ApplyFilters(cfg, flags, []*discovery.Module{stageVPC, stageEKS, prodVPC})
			result := &Result{
				All:      NewModuleSet([]*discovery.Module{stageVPC, stageEKS, prodVPC}),
				Filtered: NewModuleSet(filteredModules),
				Graph:    depGraph,
			}

			targets, err := resolveTargets(context.Background(), workDir, cfg, result, TargetSelectionOptions{
				ChangedOnly: true,
				ModulePath:  tt.modulePath,
				Filters:     flags,
				ChangeDetectorResolver: func() (ChangeDetector, error) {
					return stubChangeDetector{changedLibraries: []string{"_modules/network"}}, nil
				},
			})
			if err != nil {
				t.Fatalf("resolveTargets() error = %v", err)
			}

			if got := sortedModuleIDs(targets); !reflect.DeepEqual(got, tt.wantIDs) {
				t.Fatalf("module ids = %v, want %v", got, tt.wantIDs)
			}
		})
	}
}

func sortedModuleIDs(modules []*discovery.Module) []string {
	ids := moduleIDs(modules)
	sort.Strings(ids)
	return ids
}
