package costengine

import (
	"path/filepath"
	"testing"
)

func TestBuildSegmentTree(t *testing.T) {
	t.Parallel()

	workDir := "/projects/infra"

	t.Run("builds tree from modules", func(t *testing.T) {
		t.Parallel()

		result := &EstimateResult{
			Modules: []ModuleCost{
				{ModuleID: "svc/prod/us-east-1/vpc", ModulePath: filepath.Join(workDir, "svc", "prod", "us-east-1", "vpc"), AfterCost: 10.0},
				{ModuleID: "svc/prod/us-east-1/rds", ModulePath: filepath.Join(workDir, "svc", "prod", "us-east-1", "rds"), AfterCost: 50.0},
				{ModuleID: "svc/staging/us-east-1/vpc", ModulePath: filepath.Join(workDir, "svc", "staging", "us-east-1", "vpc"), AfterCost: 5.0},
			},
		}

		tree := BuildSegmentTree(result, workDir)

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

		result := &EstimateResult{
			Modules: []ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 0, BeforeCost: 0, Error: ""},
				{ModuleID: "svc/prod/rds", AfterCost: 10.0},
			},
		}

		tree := BuildSegmentTree(result, workDir)

		// Only rds should be in the tree
		if len(tree.Children) != 1 {
			t.Fatalf("root children = %d, want 1", len(tree.Children))
		}
	})

	t.Run("includes zero-cost modules with errors", func(t *testing.T) {
		t.Parallel()

		result := &EstimateResult{
			Modules: []ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 0, Error: "api timeout"},
			},
		}

		tree := BuildSegmentTree(result, workDir)

		if len(tree.Children) != 1 {
			t.Fatalf("root children = %d, want 1 (module with error)", len(tree.Children))
		}
	})

	t.Run("accumulates diff cost", func(t *testing.T) {
		t.Parallel()

		result := &EstimateResult{
			Modules: []ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 10, DiffCost: 5},
				{ModuleID: "svc/prod/rds", AfterCost: 20, DiffCost: -3},
			},
		}

		tree := BuildSegmentTree(result, workDir)
		svc := tree.Children[0]
		if svc.DiffCost != 2.0 {
			t.Errorf("svc diff = %.2f, want 2.00", svc.DiffCost)
		}
	})

	t.Run("leaf nodes have module reference", func(t *testing.T) {
		t.Parallel()

		result := &EstimateResult{
			Modules: []ModuleCost{
				{ModuleID: "svc/prod/vpc", AfterCost: 10},
			},
		}

		tree := BuildSegmentTree(result, workDir)
		leaf := tree.Children[0].Children[0].Children[0] // svc/prod/vpc
		if leaf.Module == nil {
			t.Error("leaf module should not be nil")
		}
		if leaf.Name != "vpc" {
			t.Errorf("leaf name = %q, want vpc", leaf.Name)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()

		tree := BuildSegmentTree(&EstimateResult{}, workDir)
		if len(tree.Children) != 0 {
			t.Errorf("root children = %d, want 0", len(tree.Children))
		}
	})
}

func TestFindChild(t *testing.T) {
	t.Parallel()

	root := &SegmentNode{
		Children: []*SegmentNode{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		c := FindChild(root, "beta")
		if c == nil || c.Name != "beta" {
			t.Errorf("FindChild(beta) = %v, want beta", c)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		c := FindChild(root, "gamma")
		if c != nil {
			t.Errorf("FindChild(gamma) = %v, want nil", c)
		}
	})

	t.Run("empty children", func(t *testing.T) {
		t.Parallel()

		c := FindChild(&SegmentNode{}, "x")
		if c != nil {
			t.Errorf("FindChild on empty = %v, want nil", c)
		}
	})
}

func TestCompactSegmentTree(t *testing.T) {
	t.Parallel()

	t.Run("merges single-child chain", func(t *testing.T) {
		t.Parallel()

		root := &SegmentNode{
			Children: []*SegmentNode{
				{Name: "a", Children: []*SegmentNode{
					{Name: "b", Children: []*SegmentNode{
						{Name: "c", Module: &ModuleCost{AfterCost: 10}},
					}},
				}},
			},
		}

		CompactSegmentTree(root)

		if len(root.Children) != 1 {
			t.Fatalf("children = %d, want 1", len(root.Children))
		}
		if root.Children[0].Name != "a/b/c" {
			t.Errorf("name = %q, want a/b/c", root.Children[0].Name)
		}
	})

	t.Run("stops at branch point", func(t *testing.T) {
		t.Parallel()

		root := &SegmentNode{
			Children: []*SegmentNode{
				{Name: "a", Children: []*SegmentNode{
					{Name: "b1", Module: &ModuleCost{}},
					{Name: "b2", Module: &ModuleCost{}},
				}},
			},
		}

		CompactSegmentTree(root)

		if root.Children[0].Name != "a" {
			t.Errorf("branch should not be compacted, got %q", root.Children[0].Name)
		}
		if len(root.Children[0].Children) != 2 {
			t.Error("branch children should be preserved")
		}
	})

	t.Run("stops at module node", func(t *testing.T) {
		t.Parallel()

		root := &SegmentNode{
			Children: []*SegmentNode{
				{Name: "a", Module: &ModuleCost{}, Children: []*SegmentNode{
					{Name: "b"},
				}},
			},
		}

		CompactSegmentTree(root)

		// a has a module, so should not be merged with b
		if root.Children[0].Name != "a" {
			t.Errorf("module node should not be compacted, got %q", root.Children[0].Name)
		}
	})

	t.Run("empty tree", func(_ *testing.T) {
		root := &SegmentNode{}
		CompactSegmentTree(root) // should not panic
	})
}

func TestStripModulePrefix(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := StripModulePrefix(tt.address, tt.moduleAddr)
			if got != tt.want {
				t.Errorf("StripModulePrefix(%q, %q) = %q, want %q", tt.address, tt.moduleAddr, got, tt.want)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple path", "a/b/c", []string{"a", "b", "c"}},
		{"single segment", "vpc", []string{"vpc"}},
		{"empty", "", nil},
		{"dot", ".", nil},
		{"nested", "platform/prod/eu-central-1/rds", []string{"platform", "prod", "eu-central-1", "rds"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SplitPath(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitPath(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("SplitPath(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDetectRegion(t *testing.T) {
	t.Parallel()

	segments := []string{"service", "environment", "region", "module"}

	tests := []struct {
		name       string
		segments   []string
		modulePath string
		want       string
	}{
		{"extracts region from pattern", segments, "platform/prod/eu-central-1/rds", "eu-central-1"},
		{"extracts us-east-1", segments, "svc/staging/us-east-1/vpc", "us-east-1"},
		{"falls back when no region segment", []string{"service", "module"}, "svc/vpc", "us-east-1"},
		{"falls back on nil segments", nil, "svc/prod/eu-west-1/vpc", "us-east-1"},
		{"falls back when path too short", segments, "svc", "us-east-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectRegion(tt.segments, tt.modulePath)
			if got != tt.want {
				t.Errorf("DetectRegion(%v, %q) = %q, want %q", tt.segments, tt.modulePath, got, tt.want)
			}
		})
	}
}
