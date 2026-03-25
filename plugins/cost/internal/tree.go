package costengine

import (
	"path/filepath"
	"strings"
)

// SegmentNode is a tree node representing one path segment (service, environment, region, or module).
type SegmentNode struct {
	Name      string         // segment value, e.g., "prod", "eu-central-1", "rds"
	AfterCost float64        // total cost including all children
	DiffCost  float64        // total diff including all children
	Children  []*SegmentNode // child segments
	Module    *ModuleCost    // non-nil only for leaf nodes (actual modules)
}

// BuildSegmentTree creates a tree from module paths split by "/".
func BuildSegmentTree(result *EstimateResult, workDir string) *SegmentNode {
	root := &SegmentNode{Name: ""}

	for i := range result.Modules {
		mc := &result.Modules[i]
		if mc.AfterCost == 0 && mc.BeforeCost == 0 && mc.Error == "" {
			continue
		}

		moduleID := mc.ModuleID
		if rel, err := filepath.Rel(workDir, mc.ModulePath); err == nil {
			moduleID = filepath.ToSlash(rel)
		}

		parts := strings.Split(moduleID, "/")
		node := root
		for _, part := range parts {
			child := FindChild(node, part)
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

// FindChild returns the child node with the given name, or nil.
func FindChild(node *SegmentNode, name string) *SegmentNode {
	for _, c := range node.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// CompactSegmentTree merges nodes that have exactly one child and no module into "parent/child".
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

// StripModulePrefix removes the "module.x.module.y." prefix from a resource address
// when displayed inside its module group, since it's redundant.
// e.g., "module.runner.aws_instance.web" with prefix "module.runner" → "aws_instance.web"
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

// SplitPath splits a filepath into its component parts.
func SplitPath(p string) []string {
	var parts []string
	for p != "" && p != "." && p != "/" {
		dir, file := filepath.Split(p)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		p = filepath.Clean(dir)
	}
	return parts
}

// DetectRegion extracts region from a module path using configured pattern segments.
// Falls back to "us-east-1" if no region segment is found.
func DetectRegion(segments []string, modulePath string) string {
	parts := SplitPath(modulePath)
	for i, seg := range segments {
		if seg == "region" && i < len(parts) {
			return parts[i]
		}
	}
	return DefaultRegion
}
