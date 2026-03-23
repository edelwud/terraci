package graph

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/parser"
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
