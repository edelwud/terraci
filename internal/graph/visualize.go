package graph

import (
	"fmt"
	"sort"
	"strings"
)

// ToDOT exports the graph in DOT format for visualization.
func (g *DependencyGraph) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph dependencies {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box];\n\n")

	for id := range g.nodes {
		label := strings.ReplaceAll(id, "/", "\\n")
		fmt.Fprintf(&sb, "  %q [label=%q];\n", id, label)
	}

	sb.WriteString("\n")

	for from, tos := range g.edges {
		for _, to := range tos {
			fmt.Fprintf(&sb, "  %q -> %q;\n", from, to)
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// ToPlantUML exports the graph in PlantUML format for visualization.
func (g *DependencyGraph) ToPlantUML() string {
	var sb strings.Builder

	sb.WriteString("@startuml\n")
	sb.WriteString("left to right direction\n")
	sb.WriteString("skinparam componentStyle rectangle\n\n")

	groups := g.groupNodesByContext()

	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	const minPartsForLabel = 3

	for _, groupKey := range groupKeys {
		ids := groups[groupKey]
		sort.Strings(ids)

		fmt.Fprintf(&sb, "package %q {\n", groupKey)
		for _, id := range ids {
			alias := plantUMLAlias(id)
			parts := strings.Split(id, "/")
			label := id
			if len(parts) >= minPartsForLabel {
				label = strings.Join(parts[2:], "/")
			}
			fmt.Fprintf(&sb, "  [%s] as %s\n", label, alias)
		}
		sb.WriteString("}\n\n")
	}

	for from, tos := range g.edges {
		fromAlias := plantUMLAlias(from)
		for _, to := range tos {
			fmt.Fprintf(&sb, "%s --> %s\n", fromAlias, plantUMLAlias(to))
		}
	}

	sb.WriteString("\n@enduml\n")
	return sb.String()
}

func (g *DependencyGraph) groupNodesByContext() map[string][]string {
	groups := make(map[string][]string)
	for id := range g.nodes {
		parts := strings.Split(id, "/")
		key := "other"
		if len(parts) >= 2 {
			key = parts[0] + "/" + parts[1]
		}
		groups[key] = append(groups[key], id)
	}
	return groups
}

// plantUMLAlias creates a valid PlantUML alias from a module ID.
func plantUMLAlias(id string) string {
	r := strings.NewReplacer("/", "_", "-", "_", ".", "_")
	return r.Replace(id)
}
