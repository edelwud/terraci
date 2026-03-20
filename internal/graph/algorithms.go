package graph

import (
	"fmt"
	"sort"
)

// TopologicalSort returns modules in dependency order (dependencies first).
// Returns an error if there's a cycle.
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int, len(g.nodes))
	for id := range g.nodes {
		inDegree[id] = len(g.edges[id])
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, dep := range g.reverseEdges[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return result, nil
}

// ExecutionLevels returns modules grouped by execution level.
// Modules at the same level can be executed in parallel.
func (g *DependencyGraph) ExecutionLevels() ([][]string, error) {
	sorted, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	levels := make(map[string]int, len(sorted))
	for _, nodeID := range sorted {
		maxDepLevel := -1
		for _, dep := range g.edges[nodeID] {
			if levels[dep] > maxDepLevel {
				maxDepLevel = levels[dep]
			}
		}
		levels[nodeID] = maxDepLevel + 1
	}

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
	for i := range result {
		sort.Strings(result[i])
	}

	return result, nil
}

// DetectCycles returns all cycles in the graph.
func (g *DependencyGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(string) bool
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
				if start := findIndex(path, neighbor); start >= 0 {
					cycle := make([]string, len(path)-start)
					copy(cycle, path[start:])
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

func findIndex(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}
