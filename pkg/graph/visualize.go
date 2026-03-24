package graph

import (
	"fmt"
	"sort"
	"strings"
)

// ToDOT exports the graph in DOT format with HTML labels for proper line breaks.
func (g *DependencyGraph) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph dependencies {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded, fontname=\"Helvetica\"];\n")
	sb.WriteString("  edge [color=\"#666666\"];\n\n")

	// Group nodes by context for subgraph clustering
	groups := g.groupNodesByContext()
	groupKeys := sortedMapKeys(groups)

	for i, groupKey := range groupKeys {
		ids := groups[groupKey]
		sort.Strings(ids)

		fmt.Fprintf(&sb, "  subgraph cluster_%d {\n", i)
		fmt.Fprintf(&sb, "    label=%q;\n", groupKey)
		sb.WriteString("    style=dashed;\n")
		sb.WriteString("    color=\"#999999\";\n")

		for _, id := range ids {
			parts := strings.Split(id, "/")
			// Show short label: last 2 segments (region/module or module/submodule)
			label := id
			if len(parts) >= 2 {
				label = strings.Join(parts[len(parts)-2:], "/")
			}
			fmt.Fprintf(&sb, "    %q [label=%q];\n", id, label)
		}
		sb.WriteString("  }\n\n")
	}

	for from, tos := range g.edges {
		for _, to := range tos {
			fmt.Fprintf(&sb, "  %q -> %q;\n", from, to)
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// minPartsForRegion is the minimum path segments needed to extract a region sub-group.
const minPartsForRegion = 3

// ToPlantUML exports the graph in PlantUML format with nested grouping.
func (g *DependencyGraph) ToPlantUML() string {
	var sb strings.Builder

	sb.WriteString("@startuml\n")
	sb.WriteString("left to right direction\n")
	sb.WriteString("skinparam componentStyle rectangle\n")
	sb.WriteString("skinparam packageStyle frame\n")
	sb.WriteString("skinparam defaultFontSize 11\n\n")

	// Group by context (first 2 segments), then sub-group by region (3rd segment)
	groups := g.groupNodesByContext()
	groupKeys := sortedMapKeys(groups)

	for _, groupKey := range groupKeys {
		ids := groups[groupKey]
		sort.Strings(ids)

		fmt.Fprintf(&sb, "package %q {\n", groupKey)

		// Sub-group by region (3rd segment) for better readability
		subGroups := make(map[string][]string)
		for _, id := range ids {
			parts := strings.Split(id, "/")
			subKey := "default"
			if len(parts) >= minPartsForRegion {
				subKey = parts[2]
			}
			subGroups[subKey] = append(subGroups[subKey], id)
		}

		subKeys := sortedMapKeys(subGroups)
		needsSubGrouping := len(subKeys) > 1

		for _, subKey := range subKeys {
			subIDs := subGroups[subKey]

			if needsSubGrouping {
				fmt.Fprintf(&sb, "  package %q {\n", subKey)
			}

			for _, id := range subIDs {
				alias := plantUMLAlias(id)
				// Short label: last segment(s) after the group context
				parts := strings.Split(id, "/")
				label := parts[len(parts)-1]
				if len(parts) > minPartsForRegion {
					label = strings.Join(parts[3:], "/")
				}
				indent := "  "
				if needsSubGrouping {
					indent = "    "
				}
				fmt.Fprintf(&sb, "%s[%s] as %s\n", indent, label, alias)
			}

			if needsSubGrouping {
				sb.WriteString("  }\n")
			}
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

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
