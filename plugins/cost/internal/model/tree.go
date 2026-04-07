package model

import (
	"path/filepath"
	"strings"
)

// DefaultRegion is used when no region is specified.
const DefaultRegion = "us-east-1"

// SegmentNode is a tree node representing one path segment.
type SegmentNode struct {
	Name      string
	AfterCost float64
	DiffCost  float64
	Children  []*SegmentNode
	Module    *ModuleCost
}

// BuildSegmentTree creates a tree from module paths split by "/".
func BuildSegmentTree(result *EstimateResult, workDir string) *SegmentNode {
	root := &SegmentNode{Name: ""}

	for i := range result.Modules {
		mc := &result.Modules[i]
		if CostIsZero(mc.AfterCost) && CostIsZero(mc.BeforeCost) && CostIsZero(mc.DiffCost) && mc.Error == "" {
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

// findChild returns the child node with the given name, or nil.
func findChild(node *SegmentNode, name string) *SegmentNode {
	for _, c := range node.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// CompactSegmentTree merges nodes that have exactly one child and no module.
func CompactSegmentTree(node *SegmentNode) {
	for _, c := range node.Children {
		CompactSegmentTree(c)
	}

	for i, c := range node.Children {
		for len(c.Children) == 1 && c.Module == nil {
			merged := c.Children[0]
			merged.Name = c.Name + "/" + merged.Name
			node.Children[i] = merged
			c = merged
		}
	}
}

// StripModulePrefix removes the module prefix from a resource address.
func StripModulePrefix(address, moduleAddr string) string {
	if moduleAddr == "" {
		return address
	}
	prefix := moduleAddr + "."
	if len(address) > len(prefix) && address[:len(prefix)] == prefix {
		return address[len(prefix):]
	}
	return address
}

// splitPath splits a filepath into its OS-independent path segments.
func splitPath(p string) []string {
	return strings.Split(filepath.ToSlash(filepath.Clean(p)), "/")
}

// DetectRegion extracts region from a module path using configured pattern segments.
func DetectRegion(segments []string, modulePath string) string {
	parts := splitPath(modulePath)
	for i, seg := range segments {
		if seg == "region" && i < len(parts) {
			return parts[i]
		}
	}
	return DefaultRegion
}
