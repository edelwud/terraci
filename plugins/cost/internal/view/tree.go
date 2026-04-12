package view

import (
	"path/filepath"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// SegmentNode is a tree node representing one path segment in text output.
type SegmentNode struct {
	Name      string
	AfterCost float64
	DiffCost  float64
	Children  []*SegmentNode
	Module    *model.ModuleCost
}

// BuildSegmentTree creates a tree from module paths split by "/".
func BuildSegmentTree(result *model.EstimateResult, workDir string) *SegmentNode {
	root := &SegmentNode{Name: ""}

	for i := range result.Modules {
		mc := &result.Modules[i]
		if model.CostIsZero(mc.AfterCost) && model.CostIsZero(mc.BeforeCost) && model.CostIsZero(mc.DiffCost) && mc.Error == "" {
			continue
		}

		moduleID := mc.ModuleID
		if rel, err := filepath.Rel(workDir, mc.ModulePath); err == nil {
			moduleID = filepath.ToSlash(rel)
		}

		parts := strings.Split(moduleID, "/")
		node := root
		for _, part := range parts {
			child := findChild(node, part)
			if child == nil {
				child = &SegmentNode{Name: part}
				node.Children = append(node.Children, child)
			}
			child.AfterCost += mc.AfterCost
			child.DiffCost += mc.DiffCost
			node = child
		}
		node.Module = mc
	}

	return root
}

func findChild(node *SegmentNode, name string) *SegmentNode {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// CompactSegmentTree merges nodes that have exactly one child and no module.
func CompactSegmentTree(node *SegmentNode) {
	for _, child := range node.Children {
		CompactSegmentTree(child)
	}

	for i, child := range node.Children {
		for len(child.Children) == 1 && child.Module == nil {
			merged := child.Children[0]
			merged.Name = child.Name + "/" + merged.Name
			node.Children[i] = merged
			child = merged
		}
	}
}
