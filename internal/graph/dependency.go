// Package graph provides dependency graph construction and analysis.
package graph

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/parser"
)

// DependencyGraph represents the dependency relationships between modules.
type DependencyGraph struct {
	nodes        map[string]*Node
	edges        map[string][]string // from → [to] (depends on)
	reverseEdges map[string][]string // to → [from] (depended by)
	libraryUsage map[string][]string // library path → [module IDs]
}

// Node represents a module in the dependency graph.
type Node struct {
	Module    *discovery.Module
	InDegree  int
	OutDegree int
}

// NewDependencyGraph creates a new empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:        make(map[string]*Node),
		edges:        make(map[string][]string),
		reverseEdges: make(map[string][]string),
		libraryUsage: make(map[string][]string),
	}
}

// BuildFromDependencies builds a graph from extracted dependencies.
func BuildFromDependencies(modules []*discovery.Module, deps map[string]*parser.ModuleDependencies) *DependencyGraph {
	g := NewDependencyGraph()

	for _, m := range modules {
		g.AddNode(m)
	}

	for moduleID, moduleDeps := range deps {
		for _, depID := range moduleDeps.DependsOn {
			g.AddEdge(moduleID, depID)
		}
		for _, libDep := range moduleDeps.LibraryDependencies {
			g.AddLibraryUsage(libDep.LibraryPath, moduleID)
		}
	}

	return g
}

// --- Node and edge operations ---

// AddNode adds a module to the graph.
func (g *DependencyGraph) AddNode(m *discovery.Module) {
	if _, exists := g.nodes[m.ID()]; !exists {
		g.nodes[m.ID()] = &Node{Module: m}
	}
}

// AddEdge adds a dependency edge (from depends on to).
func (g *DependencyGraph) AddEdge(from, to string) {
	if g.nodes[from] == nil || g.nodes[to] == nil {
		return
	}
	if slices.Contains(g.edges[from], to) {
		return
	}

	g.edges[from] = append(g.edges[from], to)
	g.reverseEdges[to] = append(g.reverseEdges[to], from)
	g.nodes[from].InDegree++
	g.nodes[to].OutDegree++
}

// Nodes returns all nodes in the graph.
func (g *DependencyGraph) Nodes() map[string]*Node { return g.nodes }

// GetNode returns a specific node by ID.
func (g *DependencyGraph) GetNode(id string) *Node { return g.nodes[id] }

// GetDependencies returns the direct dependencies of a module.
func (g *DependencyGraph) GetDependencies(moduleID string) []string { return g.edges[moduleID] }

// GetDependents returns modules that directly depend on the given module.
func (g *DependencyGraph) GetDependents(moduleID string) []string { return g.reverseEdges[moduleID] }

// GetAllDependencies returns all transitive dependencies of a module.
func (g *DependencyGraph) GetAllDependencies(moduleID string) []string {
	return g.collectTransitive(moduleID, g.edges)
}

// GetAllDependents returns all transitive dependents of a module.
func (g *DependencyGraph) GetAllDependents(moduleID string) []string {
	return g.collectTransitive(moduleID, g.reverseEdges)
}

// collectTransitive performs a DFS traversal on the given adjacency map.
func (g *DependencyGraph) collectTransitive(startID string, adjacency map[string][]string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(string)
	visit = func(id string) {
		for _, next := range adjacency[id] {
			if !visited[next] {
				visited[next] = true
				result = append(result, next)
				visit(next)
			}
		}
	}

	visit(startID)
	return result
}

// --- Library usage ---

// AddLibraryUsage records that an executable module uses a library module.
func (g *DependencyGraph) AddLibraryUsage(libraryPath, moduleID string) {
	libraryPath = normalizeLibraryPath(libraryPath)
	if !slices.Contains(g.libraryUsage[libraryPath], moduleID) {
		g.libraryUsage[libraryPath] = append(g.libraryUsage[libraryPath], moduleID)
	}
}

// GetModulesUsingLibrary returns all executable modules that use the given library path.
func (g *DependencyGraph) GetModulesUsingLibrary(libraryPath string) []string {
	return g.libraryUsage[normalizeLibraryPath(libraryPath)]
}

// GetAllLibraryPaths returns all tracked library module paths.
func (g *DependencyGraph) GetAllLibraryPaths() []string {
	paths := make([]string, 0, len(g.libraryUsage))
	for path := range g.libraryUsage {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func normalizeLibraryPath(path string) string {
	return strings.TrimSuffix(path, "/")
}

// --- Subgraph ---

// Subgraph returns a new graph containing only the specified modules and their edges.
func (g *DependencyGraph) Subgraph(moduleIDs []string) *DependencyGraph {
	sub := NewDependencyGraph()
	moduleSet := make(map[string]bool, len(moduleIDs))
	for _, id := range moduleIDs {
		moduleSet[id] = true
	}

	for id := range moduleSet {
		if node, ok := g.nodes[id]; ok {
			sub.nodes[id] = &Node{Module: node.Module}
		}
	}

	for from := range moduleSet {
		for _, to := range g.edges[from] {
			if moduleSet[to] {
				sub.AddEdge(from, to)
			}
		}
	}

	return sub
}

// ScopeToModule returns a subgraph scoped to the given module's dependencies or dependents.
// If showDependents is true, includes the module and all its dependents;
// otherwise includes the module and all its dependencies.
func (g *DependencyGraph) ScopeToModule(moduleID string, showDependents bool) (*DependencyGraph, error) {
	if g.GetNode(moduleID) == nil {
		return nil, fmt.Errorf("module not found: %s", moduleID)
	}

	var ids []string
	if showDependents {
		ids = append([]string{moduleID}, g.GetAllDependents(moduleID)...)
	} else {
		ids = append([]string{moduleID}, g.GetAllDependencies(moduleID)...)
	}

	return g.Subgraph(ids), nil
}
