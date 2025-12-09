// Package graph provides dependency graph construction and analysis
package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/parser"
)

// DependencyGraph represents the dependency relationships between modules
type DependencyGraph struct {
	// Nodes are all modules in the graph
	nodes map[string]*Node
	// Edges represent dependencies (from -> to means "from depends on to")
	edges map[string][]string
	// Reverse edges (to -> from means "to is depended on by from")
	reverseEdges map[string][]string
}

// Node represents a module in the dependency graph
type Node struct {
	Module *discovery.Module
	// InDegree is the number of dependencies this module has
	InDegree int
	// OutDegree is the number of modules that depend on this one
	OutDegree int
}

// NewDependencyGraph creates a new empty dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:        make(map[string]*Node),
		edges:        make(map[string][]string),
		reverseEdges: make(map[string][]string),
	}
}

// BuildFromDependencies builds a graph from extracted dependencies
func BuildFromDependencies(
	modules []*discovery.Module,
	deps map[string]*parser.ModuleDependencies,
) *DependencyGraph {
	g := NewDependencyGraph()

	// Add all modules as nodes
	for _, m := range modules {
		g.AddNode(m)
	}

	// Add edges from dependencies
	for moduleID, moduleDeps := range deps {
		for _, depID := range moduleDeps.DependsOn {
			g.AddEdge(moduleID, depID)
		}
	}

	return g
}

// AddNode adds a module to the graph
func (g *DependencyGraph) AddNode(m *discovery.Module) {
	if _, exists := g.nodes[m.ID()]; !exists {
		g.nodes[m.ID()] = &Node{
			Module:   m,
			InDegree: 0,
		}
	}
}

// AddEdge adds a dependency edge (from depends on to)
func (g *DependencyGraph) AddEdge(from, to string) {
	// Ensure both nodes exist
	if _, exists := g.nodes[from]; !exists {
		return
	}
	if _, exists := g.nodes[to]; !exists {
		return
	}

	// Check if edge already exists
	for _, existing := range g.edges[from] {
		if existing == to {
			return
		}
	}

	g.edges[from] = append(g.edges[from], to)
	g.reverseEdges[to] = append(g.reverseEdges[to], from)

	// Update degrees
	g.nodes[from].InDegree++
	g.nodes[to].OutDegree++
}

// GetDependencies returns the direct dependencies of a module
func (g *DependencyGraph) GetDependencies(moduleID string) []string {
	return g.edges[moduleID]
}

// GetDependents returns modules that depend on the given module
func (g *DependencyGraph) GetDependents(moduleID string) []string {
	return g.reverseEdges[moduleID]
}

// GetAllDependencies returns all dependencies (transitive) of a module
func (g *DependencyGraph) GetAllDependencies(moduleID string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(id string)
	visit = func(id string) {
		for _, dep := range g.edges[id] {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				visit(dep)
			}
		}
	}

	visit(moduleID)
	return result
}

// GetAllDependents returns all modules that depend on the given module (transitive)
func (g *DependencyGraph) GetAllDependents(moduleID string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(id string)
	visit = func(id string) {
		for _, dep := range g.reverseEdges[id] {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				visit(dep)
			}
		}
	}

	visit(moduleID)
	return result
}

// TopologicalSort returns modules in dependency order (dependencies first)
// Returns an error if there's a cycle
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	// Kahn's algorithm
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = len(g.edges[id])
	}

	// Find all nodes with no dependencies
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Sort queue for deterministic output
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		// Take first node
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Remove edges from this node
		dependents := g.reverseEdges[node]
		for _, dep := range dependents {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				// Re-sort for deterministic output
				sort.Strings(queue)
			}
		}
	}

	// Check for cycles
	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return result, nil
}

// ExecutionLevels returns modules grouped by execution level
// Modules at the same level can be executed in parallel
func (g *DependencyGraph) ExecutionLevels() ([][]string, error) {
	// Calculate the longest path to each node from any root
	levels := make(map[string]int)

	sorted, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Process in topological order
	for _, nodeID := range sorted {
		maxDepLevel := -1
		for _, dep := range g.edges[nodeID] {
			if levels[dep] > maxDepLevel {
				maxDepLevel = levels[dep]
			}
		}
		levels[nodeID] = maxDepLevel + 1
	}

	// Group by level
	maxLevel := 0
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	result := make([][]string, maxLevel+1)
	for nodeID, level := range levels {
		result[level] = append(result[level], nodeID)
	}

	// Sort each level for deterministic output
	for i := range result {
		sort.Strings(result[i])
	}

	return result, nil
}

// DetectCycles returns all cycles in the graph
func (g *DependencyGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range g.edges[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle, extract it
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	for node := range g.nodes {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

// Subgraph returns a new graph containing only the specified modules and their edges
func (g *DependencyGraph) Subgraph(moduleIDs []string) *DependencyGraph {
	sub := NewDependencyGraph()

	// Create a set for quick lookup
	moduleSet := make(map[string]bool)
	for _, id := range moduleIDs {
		moduleSet[id] = true
	}

	// Add nodes
	for id := range moduleSet {
		if node, exists := g.nodes[id]; exists {
			sub.nodes[id] = &Node{
				Module: node.Module,
			}
		}
	}

	// Add edges only between included nodes
	for from := range moduleSet {
		for _, to := range g.edges[from] {
			if moduleSet[to] {
				sub.AddEdge(from, to)
			}
		}
	}

	return sub
}

// GetAffectedModules returns modules affected by changes to the given modules
// This includes:
// - the changed modules themselves
// - all their dependents (modules that depend on changed modules)
// - all their dependencies (modules that changed modules depend on)
func (g *DependencyGraph) GetAffectedModules(changedModules []string) []string {
	affected := make(map[string]bool)

	for _, m := range changedModules {
		affected[m] = true
		// Add all dependents (modules that depend on this one)
		for _, dep := range g.GetAllDependents(m) {
			affected[dep] = true
		}
		// Add all dependencies (modules this one depends on)
		for _, dep := range g.GetAllDependencies(m) {
			affected[dep] = true
		}
	}

	result := make([]string, 0, len(affected))
	for m := range affected {
		result = append(result, m)
	}

	sort.Strings(result)
	return result
}

// ToDOT exports the graph in DOT format for visualization
func (g *DependencyGraph) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph dependencies {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box];\n\n")

	// Add nodes
	for id := range g.nodes {
		// Escape the label
		label := strings.ReplaceAll(id, "/", "\\n")
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", id, label))
	}

	sb.WriteString("\n")

	// Add edges
	for from, tos := range g.edges {
		for _, to := range tos {
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", from, to))
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}

// Stats returns statistics about the graph
type GraphStats struct {
	TotalModules int
	TotalEdges   int
	RootModules  int     // Modules with no dependencies
	LeafModules  int     // Modules with no dependents
	MaxDepth     int     // Maximum dependency chain length
	AverageDepth float64 // Average dependency chain length
	HasCycles    bool
	CycleCount   int
}

// GetStats returns statistics about the dependency graph
func (g *DependencyGraph) GetStats() GraphStats {
	stats := GraphStats{
		TotalModules: len(g.nodes),
	}

	// Count edges
	for _, edges := range g.edges {
		stats.TotalEdges += len(edges)
	}

	// Count roots and leaves
	for id := range g.nodes {
		if len(g.edges[id]) == 0 {
			stats.RootModules++
		}
		if len(g.reverseEdges[id]) == 0 {
			stats.LeafModules++
		}
	}

	// Calculate depths
	levels, err := g.ExecutionLevels()
	if err == nil {
		stats.MaxDepth = len(levels) - 1
		if len(levels) > 0 {
			totalDepth := 0
			for level, nodes := range levels {
				totalDepth += level * len(nodes)
			}
			stats.AverageDepth = float64(totalDepth) / float64(len(g.nodes))
		}
	}

	// Check for cycles
	cycles := g.DetectCycles()
	stats.HasCycles = len(cycles) > 0
	stats.CycleCount = len(cycles)

	return stats
}

// Nodes returns all nodes in the graph
func (g *DependencyGraph) Nodes() map[string]*Node {
	return g.nodes
}

// GetNode returns a specific node by ID
func (g *DependencyGraph) GetNode(id string) *Node {
	return g.nodes[id]
}
