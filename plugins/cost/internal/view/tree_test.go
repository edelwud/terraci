package view_test

import (
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/view"
)

func TestBuildSegmentTree(t *testing.T) {
	t.Parallel()

	workDir := "/projects/infra"

	t.Run("builds tree from modules", func(t *testing.T) {
		t.Parallel()

		result := &model.EstimateResult{
			Modules: []model.ModuleCost{
				{ModuleID: "svc/prod/us-east-1/vpc", ModulePath: filepath.Join(workDir, "svc", "prod", "us-east-1", "vpc"), AfterCost: 10.0},
				{ModuleID: "svc/prod/us-east-1/rds", ModulePath: filepath.Join(workDir, "svc", "prod", "us-east-1", "rds"), AfterCost: 50.0},
				{ModuleID: "svc/staging/us-east-1/vpc", ModulePath: filepath.Join(workDir, "svc", "staging", "us-east-1", "vpc"), AfterCost: 5.0},
			},
		}

		tree := view.BuildSegmentTree(result, workDir)

		if len(tree.Children) != 1 {
			t.Fatalf("root children = %d, want 1 (svc)", len(tree.Children))
		}
		svc := tree.Children[0]
		if svc.Name != "svc" {
			t.Errorf("first child = %q, want svc", svc.Name)
		}
		if svc.AfterCost != 65.0 {
			t.Errorf("svc cost = %.2f, want 65.00", svc.AfterCost)
		}
		if len(svc.Children) != 2 {
			t.Fatalf("svc children = %d, want 2 (prod, staging)", len(svc.Children))
		}
	})

	t.Run("skips zero-cost modules without errors", func(t *testing.T) {
		t.Parallel()

		result := &model.EstimateResult{
			Modules: []model.ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 0, BeforeCost: 0, Error: ""},
				{ModuleID: "svc/prod/rds", AfterCost: 10.0},
			},
		}

		tree := view.BuildSegmentTree(result, workDir)
		if len(tree.Children) != 1 {
			t.Fatalf("root children = %d, want 1", len(tree.Children))
		}
	})

	t.Run("includes zero-cost modules with errors", func(t *testing.T) {
		t.Parallel()

		result := &model.EstimateResult{
			Modules: []model.ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 0, Error: "api timeout"},
			},
		}

		tree := view.BuildSegmentTree(result, workDir)
		if len(tree.Children) != 1 {
			t.Fatalf("root children = %d, want 1 (module with error)", len(tree.Children))
		}
	})

	t.Run("accumulates diff cost", func(t *testing.T) {
		t.Parallel()

		result := &model.EstimateResult{
			Modules: []model.ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 10, DiffCost: 5},
				{ModuleID: "svc/prod/rds", AfterCost: 20, DiffCost: -3},
			},
		}

		tree := view.BuildSegmentTree(result, workDir)
		svc := tree.Children[0]
		if svc.DiffCost != 2.0 {
			t.Errorf("svc diff = %.2f, want 2.00", svc.DiffCost)
		}
	})

	t.Run("leaf nodes have module reference", func(t *testing.T) {
		t.Parallel()

		result := &model.EstimateResult{
			Modules: []model.ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 10},
			},
		}

		tree := view.BuildSegmentTree(result, workDir)
		leaf := tree.Children[0].Children[0].Children[0]
		if leaf.Module == nil {
			t.Error("leaf module should not be nil")
		}
		if leaf.Name != "vpc" {
			t.Errorf("leaf name = %q, want vpc", leaf.Name)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()

		tree := view.BuildSegmentTree(&model.EstimateResult{}, workDir)
		if len(tree.Children) != 0 {
			t.Errorf("root children = %d, want 0", len(tree.Children))
		}
	})
}

func TestCompactSegmentTree(t *testing.T) {
	t.Parallel()

	t.Run("merges single-child chain", func(t *testing.T) {
		t.Parallel()

		root := &view.SegmentNode{
			Children: []*view.SegmentNode{
				{Name: "a", Children: []*view.SegmentNode{
					{Name: "b", Children: []*view.SegmentNode{
						{Name: "c", Module: &model.ModuleCost{AfterCost: 10}},
					}},
				}},
			},
		}

		view.CompactSegmentTree(root)

		if len(root.Children) != 1 {
			t.Fatalf("children = %d, want 1", len(root.Children))
		}
		if root.Children[0].Name != "a/b/c" {
			t.Errorf("name = %q, want a/b/c", root.Children[0].Name)
		}
	})

	t.Run("stops at branch point", func(t *testing.T) {
		t.Parallel()

		root := &view.SegmentNode{
			Children: []*view.SegmentNode{
				{Name: "a", Children: []*view.SegmentNode{
					{Name: "b1", Module: &model.ModuleCost{}},
					{Name: "b2", Module: &model.ModuleCost{}},
				}},
			},
		}

		view.CompactSegmentTree(root)

		if root.Children[0].Name != "a" {
			t.Errorf("branch should not be compacted, got %q", root.Children[0].Name)
		}
		if len(root.Children[0].Children) != 2 {
			t.Error("branch children should be preserved")
		}
	})

	t.Run("stops at module node", func(t *testing.T) {
		t.Parallel()

		root := &view.SegmentNode{
			Children: []*view.SegmentNode{
				{Name: "a", Module: &model.ModuleCost{}, Children: []*view.SegmentNode{
					{Name: "b"},
				}},
			},
		}

		view.CompactSegmentTree(root)

		if root.Children[0].Name != "a" {
			t.Errorf("module node should not be compacted, got %q", root.Children[0].Name)
		}
	})

	t.Run("empty tree", func(t *testing.T) {
		t.Parallel()

		root := &view.SegmentNode{}
		view.CompactSegmentTree(root)
	})
}
