package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
)

// ToDOT exports the graph in DOT format with HTML labels for proper line breaks.
func (g *DependencyGraph) ToDOT() string {
	return g.ToDOTWithLibraries(nil)
}

// ToDOTWithLibraries exports the graph in DOT format and renders the given
// library modules in a separate cluster with dashed style. Library-to-consumer
// dependencies are inferred from the graph's tracked libraryUsage. If
// libraries is empty, output matches ToDOT() exactly.
func (g *DependencyGraph) ToDOTWithLibraries(libraries []*discovery.Module) string {
	var sb strings.Builder

	sb.WriteString("digraph dependencies {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded, fontname=\"Helvetica\"];\n")
	sb.WriteString("  edge [color=\"#666666\"];\n\n")

	groups := g.groupNodesByContext()
	groupKeys := sortedMapKeys(groups)

	clusterIdx := 0
	for _, groupKey := range groupKeys {
		ids := groups[groupKey]
		sort.Strings(ids)

		fmt.Fprintf(&sb, "  subgraph cluster_%d {\n", clusterIdx)
		fmt.Fprintf(&sb, "    label=%q;\n", groupKey)
		sb.WriteString("    style=dashed;\n")
		sb.WriteString("    color=\"#999999\";\n")

		for _, id := range ids {
			parts := strings.Split(id, "/")
			label := id
			if len(parts) >= 2 {
				label = strings.Join(parts[len(parts)-2:], "/")
			}
			fmt.Fprintf(&sb, "    %q [label=%q];\n", id, label)
		}
		sb.WriteString("  }\n\n")
		clusterIdx++
	}

	libsByID := writeLibraryCluster(&sb, libraries, clusterIdx)

	for from, tos := range g.edges {
		for _, to := range tos {
			fmt.Fprintf(&sb, "  %q -> %q;\n", from, to)
		}
	}

	writeLibraryEdges(&sb, g, libsByID)

	sb.WriteString("}\n")
	return sb.String()
}

// writeLibraryCluster emits a dashed DOT cluster for library modules.
// Returns a map of node-id → *Module so writeLibraryEdges can look modules up
// by their absolute path when traversing libraryUsage.
func writeLibraryCluster(sb *strings.Builder, libraries []*discovery.Module, clusterIdx int) map[string]*discovery.Module {
	if len(libraries) == 0 {
		return nil
	}
	byPath := make(map[string]*discovery.Module, len(libraries))
	ids := make([]string, 0, len(libraries))
	for _, m := range libraries {
		if m == nil {
			continue
		}
		byPath[m.Path] = m
		ids = append(ids, m.RelativePath)
	}
	sort.Strings(ids)

	fmt.Fprintf(sb, "  subgraph cluster_%d {\n", clusterIdx)
	sb.WriteString("    label=\"library_modules\";\n")
	sb.WriteString("    style=dashed;\n")
	sb.WriteString("    color=\"#aa6633\";\n")
	sb.WriteString("    node [shape=box, style=\"rounded,dashed\", color=\"#aa6633\", fontcolor=\"#aa6633\"];\n")
	for _, id := range ids {
		fmt.Fprintf(sb, "    %q [label=%q];\n", id, id)
	}
	sb.WriteString("  }\n\n")
	return byPath
}

// writeLibraryEdges emits dashed edges from library nodes to executable
// consumers using the graph's tracked libraryUsage. Only libraries present in
// libsByID are rendered to avoid leaking absolute paths from libraryUsage that
// the caller did not surface as nodes.
func writeLibraryEdges(sb *strings.Builder, g *DependencyGraph, libsByID map[string]*discovery.Module) {
	if len(libsByID) == 0 {
		return
	}
	paths := make([]string, 0, len(g.libraryUsage))
	for p := range g.libraryUsage {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, libPath := range paths {
		consumers := g.libraryUsage[libPath]
		if len(consumers) == 0 {
			continue
		}
		owner := libsByID[libPath]
		if owner == nil {
			// libraryUsage may track a path nested under a library module
			// (e.g. _modules/kafka/acl while only _modules/kafka is in
			// libraries). Walk up to the closest enclosing library module.
			owner = enclosingLibrary(libsByID, libPath)
			if owner == nil {
				continue
			}
		}
		consumersSorted := append([]string(nil), consumers...)
		sort.Strings(consumersSorted)
		for _, consumer := range consumersSorted {
			fmt.Fprintf(sb, "  %q -> %q [style=dashed, color=\"#aa6633\"];\n", owner.RelativePath, consumer)
		}
	}
}

func enclosingLibrary(libsByID map[string]*discovery.Module, libPath string) *discovery.Module {
	var best *discovery.Module
	bestLen := 0
	for path, mod := range libsByID {
		prefix := path + "/"
		if strings.HasPrefix(libPath, prefix) && len(path) > bestLen {
			best = mod
			bestLen = len(path)
		}
	}
	return best
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
