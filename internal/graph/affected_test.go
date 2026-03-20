package graph

import (
	"reflect"
	"sort"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/parser"
)

func TestGetAffectedModules(t *testing.T) {
	g := buildTestGraph()

	t.Run("vpc changes affects all", func(t *testing.T) {
		affected := g.GetAffectedModules([]string{"platform/stage/eu-central-1/vpc"})
		if len(affected) != 4 {
			t.Errorf("expected 4, got %d: %v", len(affected), affected)
		}
	})

	t.Run("eks changes affects eks+app+vpc", func(t *testing.T) {
		affected := g.GetAffectedModules([]string{"platform/stage/eu-central-1/eks"})
		sort.Strings(affected)
		expected := []string{
			"platform/stage/eu-central-1/app",
			"platform/stage/eu-central-1/eks",
			"platform/stage/eu-central-1/vpc",
		}
		if !reflect.DeepEqual(affected, expected) {
			t.Errorf("got %v, want %v", affected, expected)
		}
	})

	t.Run("app changes affects app+eks+rds+vpc", func(t *testing.T) {
		affected := g.GetAffectedModules([]string{"platform/stage/eu-central-1/app"})
		if len(affected) != 4 {
			t.Errorf("expected 4, got %d: %v", len(affected), affected)
		}
	})
}

func TestGetAffectedByLibraryChanges(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
		discovery.TestModule("platform", "stage", "eu-north-1", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/vpc": {},
		"platform/stage/eu-north-1/msk": {
			DependsOn:           []string{"platform/stage/eu-north-1/vpc"},
			LibraryDependencies: []*parser.LibraryDependency{{LibraryPath: "/project/_modules/kafka"}},
		},
		"platform/stage/eu-north-1/app": {DependsOn: []string{"platform/stage/eu-north-1/msk"}},
	}
	g := BuildFromDependencies(modules, deps)

	affected := g.GetAffectedByLibraryChanges([]string{"/project/_modules/kafka"})
	if len(affected) != 1 || affected[0] != "platform/stage/eu-north-1/msk" {
		t.Errorf("expected [msk], got %v", affected)
	}
}

func TestGetAffectedByLibraryChanges_Transitive(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/msk": {
			LibraryDependencies: []*parser.LibraryDependency{{LibraryPath: "/project/_modules/kafka/acl"}},
		},
	}
	g := BuildFromDependencies(modules, deps)

	affected := g.GetAffectedByLibraryChanges([]string{"/project/_modules/kafka"})
	if len(affected) != 1 || affected[0] != "platform/stage/eu-north-1/msk" {
		t.Errorf("expected [msk] via transitive, got %v", affected)
	}
}

func TestGetAffectedModulesWithLibraries(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
		discovery.TestModule("platform", "stage", "eu-north-1", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/vpc": {},
		"platform/stage/eu-north-1/msk": {
			DependsOn:           []string{"platform/stage/eu-north-1/vpc"},
			LibraryDependencies: []*parser.LibraryDependency{{LibraryPath: "/project/_modules/kafka"}},
		},
		"platform/stage/eu-north-1/app": {DependsOn: []string{"platform/stage/eu-north-1/msk"}},
	}
	g := BuildFromDependencies(modules, deps)

	t.Run("library only", func(t *testing.T) {
		affected := g.GetAffectedModulesWithLibraries(nil, []string{"/project/_modules/kafka"})
		sort.Strings(affected)
		expected := []string{"platform/stage/eu-north-1/msk", "platform/stage/eu-north-1/vpc"}
		if !reflect.DeepEqual(affected, expected) {
			t.Errorf("got %v, want %v", affected, expected)
		}
	})

	t.Run("combined", func(t *testing.T) {
		affected := g.GetAffectedModulesWithLibraries(
			[]string{"platform/stage/eu-north-1/app"},
			[]string{"/project/_modules/kafka"},
		)
		sort.Strings(affected)
		expected := []string{
			"platform/stage/eu-north-1/app",
			"platform/stage/eu-north-1/msk",
			"platform/stage/eu-north-1/vpc",
		}
		if !reflect.DeepEqual(affected, expected) {
			t.Errorf("got %v, want %v", affected, expected)
		}
	})
}
