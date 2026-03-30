package model_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestGroupByModule_SingleModule(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "aws_instance.web", ModuleAddr: "", MonthlyCost: 100},
		{Address: "aws_ebs_volume.data", ModuleAddr: "", MonthlyCost: 10},
	}

	roots := model.GroupByModule(resources)
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

	roots := model.GroupByModule(resources)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root (module.eks), got %d", len(roots))
	}

	parent := roots[0]
	// Parent MonthlyCost should be ONLY its direct resources ($73), NOT $73 + $200
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

	// TotalCost should be recursive: 73 + 200 = 273
	if parent.TotalCost() != 273 {
		t.Errorf("parent TotalCost() = %v, want 273", parent.TotalCost())
	}
}

func TestGroupByModule_MultipleRoots(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc", MonthlyCost: 5},
		{Address: "module.rds.aws_db_instance.db", ModuleAddr: "module.rds", MonthlyCost: 300},
	}

	roots := model.GroupByModule(resources)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
}

func TestGroupByModule_Empty(t *testing.T) {
	roots := model.GroupByModule(nil)
	if len(roots) != 0 {
		t.Errorf("expected 0 roots for nil, got %d", len(roots))
	}
}

func TestSubmoduleCost_TotalCost_DeepNesting(t *testing.T) {
	s := model.SubmoduleCost{
		MonthlyCost: 10,
		Children: []model.SubmoduleCost{
			{
				MonthlyCost: 20,
				Children: []model.SubmoduleCost{
					{MonthlyCost: 30},
				},
			},
		},
	}

	if s.TotalCost() != 60 {
		t.Errorf("TotalCost() = %v, want 60", s.TotalCost())
	}
}

func TestSubmoduleCost_TotalCost_NoChildren(t *testing.T) {
	s := model.SubmoduleCost{MonthlyCost: 42}
	if s.TotalCost() != 42 {
		t.Errorf("TotalCost() = %v, want 42", s.TotalCost())
	}
}

func TestGroupByModule_ThreeLevelNesting(t *testing.T) {
	resources := []model.ResourceCost{
		{Address: "module.infra.aws_vpc.main", ModuleAddr: "module.infra", MonthlyCost: 5},
		{Address: "module.infra.module.eks.aws_eks_cluster.main", ModuleAddr: "module.infra.module.eks", MonthlyCost: 73},
		{Address: "module.infra.module.eks.module.nodes.aws_instance.worker", ModuleAddr: "module.infra.module.eks.module.nodes", MonthlyCost: 200},
	}

	roots := model.GroupByModule(resources)
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

	// module.infra should have 1 child: module.infra.module.eks
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

	// module.infra.module.eks should have 1 child: .module.nodes
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

	roots := model.GroupByModule(resources)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots (root + module.compute), got %d", len(roots))
	}

	// Root module (addr="") should have 2 resources
	var rootModule *model.SubmoduleCost
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

	roots := model.GroupByModule(resources)
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots (siblings), got %d", len(roots))
	}

	// None should have children — all are at the same depth
	for _, r := range roots {
		if len(r.Children) != 0 {
			t.Errorf("module %q has %d children, want 0", r.ModuleAddr, len(r.Children))
		}
	}
}

func TestFindParentAddr(t *testing.T) {
	nodes := map[string]*model.SubmoduleCost{
		"module.eks":              {},
		"module.eks.module.nodes": {},
	}

	if got := model.FindParentAddr("module.eks.module.nodes", nodes); got != "module.eks" {
		t.Errorf("findParentAddr = %q, want module.eks", got)
	}

	if got := model.FindParentAddr("module.eks", nodes); got != "" {
		t.Errorf("findParentAddr(root) = %q, want empty", got)
	}

	if got := model.FindParentAddr("", nodes); got != "" {
		t.Errorf("findParentAddr(empty) = %q, want empty", got)
	}
}

func TestFindParentAddr_DeepNesting(t *testing.T) {
	nodes := map[string]*model.SubmoduleCost{
		"module.a":                            {},
		"module.a.module.b":                   {},
		"module.a.module.b.module.c":          {},
		"module.a.module.b.module.c.module.d": {},
	}

	// module.a.module.b.module.c.module.d → parent is module.a.module.b.module.c
	if got := model.FindParentAddr("module.a.module.b.module.c.module.d", nodes); got != "module.a.module.b.module.c" {
		t.Errorf("findParentAddr(d) = %q, want module.a.module.b.module.c", got)
	}

	// module.a.module.b.module.c → parent is module.a.module.b
	if got := model.FindParentAddr("module.a.module.b.module.c", nodes); got != "module.a.module.b" {
		t.Errorf("findParentAddr(c) = %q, want module.a.module.b", got)
	}
}
