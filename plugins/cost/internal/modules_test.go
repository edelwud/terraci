package costengine

import "testing"

func TestGroupByModule_SingleModule(t *testing.T) {
	resources := []ResourceCost{
		{Address: "aws_instance.web", ModuleAddr: "", MonthlyCost: 100},
		{Address: "aws_ebs_volume.data", ModuleAddr: "", MonthlyCost: 10},
	}

	roots := groupByModule(resources)
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
	resources := []ResourceCost{
		{Address: "module.eks.aws_eks_cluster.main", ModuleAddr: "module.eks", MonthlyCost: 73},
		{Address: "module.eks.module.nodes.aws_instance.worker", ModuleAddr: "module.eks.module.nodes", MonthlyCost: 200},
	}

	roots := groupByModule(resources)
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
	resources := []ResourceCost{
		{Address: "module.vpc.aws_vpc.main", ModuleAddr: "module.vpc", MonthlyCost: 5},
		{Address: "module.rds.aws_db_instance.db", ModuleAddr: "module.rds", MonthlyCost: 300},
	}

	roots := groupByModule(resources)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
}

func TestGroupByModule_Empty(t *testing.T) {
	roots := groupByModule(nil)
	if len(roots) != 0 {
		t.Errorf("expected 0 roots for nil, got %d", len(roots))
	}
}

func TestSubmoduleCost_TotalCost_DeepNesting(t *testing.T) {
	s := SubmoduleCost{
		MonthlyCost: 10,
		Children: []SubmoduleCost{
			{
				MonthlyCost: 20,
				Children: []SubmoduleCost{
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
	s := SubmoduleCost{MonthlyCost: 42}
	if s.TotalCost() != 42 {
		t.Errorf("TotalCost() = %v, want 42", s.TotalCost())
	}
}

func TestFindParentAddr(t *testing.T) {
	nodes := map[string]*SubmoduleCost{
		"module.eks":              {},
		"module.eks.module.nodes": {},
	}

	if got := findParentAddr("module.eks.module.nodes", nodes); got != "module.eks" {
		t.Errorf("findParentAddr = %q, want module.eks", got)
	}

	if got := findParentAddr("module.eks", nodes); got != "" {
		t.Errorf("findParentAddr(root) = %q, want empty", got)
	}

	if got := findParentAddr("", nodes); got != "" {
		t.Errorf("findParentAddr(empty) = %q, want empty", got)
	}
}
