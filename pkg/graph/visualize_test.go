package graph

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
)

func TestToDOT(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
	}
	g := BuildFromDependencies(modules, deps)

	dot := g.ToDOT()
	for _, want := range []string{"digraph dependencies", "platform/stage/eu-central-1/vpc", "->"} {
		if !strings.Contains(dot, want) {
			t.Errorf("DOT output missing %q", want)
		}
	}
}

func TestToPlantUML(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
	}
	g := BuildFromDependencies(modules, deps)

	uml := g.ToPlantUML()
	for _, want := range []string{"@startuml", "@enduml", "platform/stage", "-->"} {
		if !strings.Contains(uml, want) {
			t.Errorf("PlantUML output missing %q", want)
		}
	}
}

func TestToDOTWithLibraries(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	kafkaLib := discovery.TestLibraryModule("_modules/kafka", "/abs/_modules/kafka")
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
	}
	g := BuildFromDependencies([]*discovery.Module{vpc, eks}, deps)
	g.AddLibraryUsage("/abs/_modules/kafka", eks.ID())

	dot := g.ToDOTWithLibraries([]*discovery.Module{kafkaLib})

	for _, want := range []string{
		"library_modules",
		"_modules/kafka",
		"\"_modules/kafka\" -> \"platform/stage/eu-central-1/eks\" [style=dashed",
	} {
		if !strings.Contains(dot, want) {
			t.Errorf("DOT output missing %q\n--- output ---\n%s", want, dot)
		}
	}
}

func TestToDOTWithLibrariesNestedPath(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	g := BuildFromDependencies([]*discovery.Module{vpc}, nil)

	// libraryUsage tracks a path under a library module (e.g. _modules/kafka/acl)
	// while only the parent _modules/kafka is surfaced as a library node.
	kafka := discovery.TestLibraryModule("_modules/kafka", "/abs/_modules/kafka")
	g.AddLibraryUsage("/abs/_modules/kafka/acl", vpc.ID())

	dot := g.ToDOTWithLibraries([]*discovery.Module{kafka})
	if !strings.Contains(dot, "\"_modules/kafka\" -> \"platform/stage/eu-central-1/vpc\"") {
		t.Errorf("expected dashed edge from enclosing library; got\n%s", dot)
	}
}

func TestHasLibraryConsumers(t *testing.T) {
	t.Parallel()

	g := NewDependencyGraph()
	g.AddLibraryUsage("/abs/_modules/kafka", "consumer")

	if !g.HasLibraryConsumers("/abs/_modules/kafka") {
		t.Error("direct path should have consumers")
	}
	if !g.HasLibraryConsumers("/abs/_modules") {
		t.Error("parent path should report nested consumers")
	}
	if g.HasLibraryConsumers("/abs/_modules/other") {
		t.Error("unrelated path should not have consumers")
	}
}

func TestPlantUMLAlias(t *testing.T) {
	t.Parallel()

	tests := []struct{ input, want string }{
		{"platform/stage/eu-central-1/vpc", "platform_stage_eu_central_1_vpc"},
		{"simple", "simple"},
		{"with.dots/and-dashes", "with_dots_and_dashes"},
	}
	for _, tt := range tests {
		if got := plantUMLAlias(tt.input); got != tt.want {
			t.Errorf("plantUMLAlias(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
