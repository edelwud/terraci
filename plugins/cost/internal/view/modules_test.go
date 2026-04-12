package view_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/view"
)

func TestGroupByModule_SingleModule(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "aws_instance.web", ModuleAddr: "", MonthlyCost: 100},
		{Address: "aws_ebs_volume.data", ModuleAddr: "", MonthlyCost: 10},
	}

	roots := view.GroupByModule(resources)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].MonthlyCost != 110 {
		t.Errorf("MonthlyCost = %v, want 110", roots[0].MonthlyCost)
	}
	if len(roots[0].Resources) != 2 {
		t.Errorf("Resources count = %d, want 2", len(roots[0].Resources))
	}
}

func TestGroupByModule_NestedModules_NoDoubleCounting(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "module.eks.aws_eks_cluster.main", ModuleAddr: "module.eks", MonthlyCost: 73},
		{Address: "module.eks.module.nodes.aws_instance.worker", ModuleAddr: "module.eks.module.nodes", MonthlyCost: 200},
	}

	roots := view.GroupByModule(resources)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root (module.eks), got %d", len(roots))
	}

	parent := roots[0]
	if parent.MonthlyCost != 73 {
		t.Errorf("parent MonthlyCost = %v, want 73 (direct only)", parent.MonthlyCost)
	}
	if len(parent.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(parent.Children))
	}

	child := parent.Children[0]
	if child.MonthlyCost != 200 {
		t.Errorf("child MonthlyCost = %v, want 200", child.MonthlyCost)
	}
	if parent.TotalCost() != 273 {
		t.Errorf("parent TotalCost() = %v, want 273", parent.TotalCost())
	}
}

func TestGroupByModule_MultipleRoots(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc", MonthlyCost: 5},
		{Address: "module.rds.aws_db_instance.db", ModuleAddr: "module.rds", MonthlyCost: 300},
	}

	roots := view.GroupByModule(resources)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
}

func TestGroupByModule_Empty(t *testing.T) {
	roots := view.GroupByModule(nil)
	if len(roots) != 0 {
		t.Errorf("expected 0 roots for nil, got %d", len(roots))
	}
}

func TestSubmoduleCost_TotalCost(t *testing.T) {
	s := view.SubmoduleCost{
		MonthlyCost: 10,
		Children: []view.SubmoduleCost{
			{
				MonthlyCost: 20,
				Children: []view.SubmoduleCost{
					{MonthlyCost: 30},
				},
			},
		},
	}

	if s.TotalCost() != 60 {
		t.Errorf("TotalCost() = %v, want 60", s.TotalCost())
	}

	leaf := view.SubmoduleCost{MonthlyCost: 42}
	if leaf.TotalCost() != 42 {
		t.Errorf("TotalCost() = %v, want 42", leaf.TotalCost())
	}
}

func TestGroupByModule_ThreeLevelNesting(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "module.infra.aws_vpc.main", ModuleAddr: "module.infra", MonthlyCost: 5},
		{Address: "module.infra.module.eks.aws_eks_cluster.main", ModuleAddr: "module.infra.module.eks", MonthlyCost: 73},
		{Address: "module.infra.module.eks.module.nodes.aws_instance.worker", ModuleAddr: "module.infra.module.eks.module.nodes", MonthlyCost: 200},
	}

	roots := view.GroupByModule(resources)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root (module.infra), got %d", len(roots))
	}

	root := roots[0]
	if root.ModuleAddr != "module.infra" {
		t.Errorf("root addr = %q, want module.infra", root.ModuleAddr)
	}
	if root.MonthlyCost != 5 {
		t.Errorf("root direct cost = %.2f, want 5 (direct only)", root.MonthlyCost)
	}
	if root.TotalCost() != 278 {
		t.Errorf("root TotalCost = %.2f, want 278 (5+73+200)", root.TotalCost())
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child of root, got %d", len(root.Children))
	}
	eks := root.Children[0]
	if eks.ModuleAddr != "module.infra.module.eks" {
		t.Errorf("child addr = %q, want module.infra.module.eks", eks.ModuleAddr)
	}
	if eks.MonthlyCost != 73 {
		t.Errorf("eks direct cost = %.2f, want 73", eks.MonthlyCost)
	}
	if len(eks.Children) != 1 {
		t.Fatalf("expected 1 child of eks, got %d", len(eks.Children))
	}
	nodes := eks.Children[0]
	if nodes.MonthlyCost != 200 {
		t.Errorf("nodes direct cost = %.2f, want 200", nodes.MonthlyCost)
	}
}

func TestGroupByModule_MixedRootAndModule(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "aws_vpc.main", ModuleAddr: "", MonthlyCost: 0},
		{Address: "module.compute.aws_instance.web", ModuleAddr: "module.compute", MonthlyCost: 100},
		{Address: "aws_route53_zone.dns", ModuleAddr: "", MonthlyCost: 0.50},
	}

	roots := view.GroupByModule(resources)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots (root + module.compute), got %d", len(roots))
	}

	var rootModule *view.SubmoduleCost
	for i := range roots {
		if roots[i].ModuleAddr == "" {
			rootModule = &roots[i]
			break
		}
	}
	if rootModule == nil {
		t.Fatal("missing root module (addr='')")
	}
	if len(rootModule.Resources) != 2 {
		t.Errorf("root resources = %d, want 2", len(rootModule.Resources))
	}
}

func TestGroupByModule_SiblingsAtSameDepth(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc", MonthlyCost: 5},
		{Address: "module.rds.aws_db_instance.main", ModuleAddr: "module.rds", MonthlyCost: 300},
		{Address: "module.eks.aws_eks_cluster.main", ModuleAddr: "module.eks", MonthlyCost: 73},
	}

	roots := view.GroupByModule(resources)
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots (siblings), got %d", len(roots))
	}

	for _, root := range roots {
		if len(root.Children) != 0 {
			t.Errorf("module %q has %d children, want 0", root.ModuleAddr, len(root.Children))
		}
	}
}

func TestFindParentAddr(t *testing.T) {
	nodes := map[string]*view.SubmoduleCost{
		"module.eks":              {},
		"module.eks.module.nodes": {},
	}

	if got := view.FindParentAddr("module.eks.module.nodes", nodes); got != "module.eks" {
		t.Errorf("FindParentAddr = %q, want module.eks", got)
	}
	if got := view.FindParentAddr("module.eks", nodes); got != "" {
		t.Errorf("FindParentAddr(root) = %q, want empty", got)
	}
	if got := view.FindParentAddr("", nodes); got != "" {
		t.Errorf("FindParentAddr(empty) = %q, want empty", got)
	}
}

func TestFindParentAddr_DeepNesting(t *testing.T) {
	nodes := map[string]*view.SubmoduleCost{
		"module.a":                            {},
		"module.a.module.b":                   {},
		"module.a.module.b.module.c":          {},
		"module.a.module.b.module.c.module.d": {},
	}

	if got := view.FindParentAddr("module.a.module.b.module.c.module.d", nodes); got != "module.a.module.b.module.c" {
		t.Errorf("FindParentAddr(d) = %q, want module.a.module.b.module.c", got)
	}
	if got := view.FindParentAddr("module.a.module.b.module.c", nodes); got != "module.a.module.b" {
		t.Errorf("FindParentAddr(c) = %q, want module.a.module.b", got)
	}
}

func TestStripModulePrefix(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		moduleAddr string
		want       string
	}{
		{"empty module addr", "aws_instance.web", "", "aws_instance.web"},
		{"matching prefix", "module.runner.aws_instance.web", "module.runner", "aws_instance.web"},
		{"no match", "aws_instance.web", "module.runner", "aws_instance.web"},
		{"prefix equals address", "module.runner", "module.runner", "module.runner"},
		{"nested modules", "module.a.module.b.aws_instance.web", "module.a.module.b", "aws_instance.web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := view.StripModulePrefix(tt.address, tt.moduleAddr)
			if got != tt.want {
				t.Errorf("StripModulePrefix(%q, %q) = %q, want %q", tt.address, tt.moduleAddr, got, tt.want)
			}
		})
	}
}
