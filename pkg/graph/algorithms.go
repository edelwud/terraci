package graph

import (
	"container/heap"
	"errors"
	"sort"
)

// TopologicalSort returns modules in dependency order (dependencies first).
// Returns an error if there's a cycle.
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int, len(g.nodes))
	for id := range g.nodes {
		inDegree[id] = len(g.edges[id])
	}

	// Use a min-heap to maintain deterministic order while keeping
	// better asymptotic behavior on insertions.
	h := &stringHeap{}
	heap.Init(h)
	for id, degree := range inDegree {
		if degree == 0 {
			heap.Push(h, id)
		}
	}

	var result []string
	for h.Len() > 0 {
		v := heap.Pop(h)
		node, ok := v.(string)
		if !ok {
			// Skip non-string entries (shouldn't happen, but defensive)
			continue
		}
		result = append(result, node)

		for _, dep := range g.reverseEdges[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				heap.Push(h, dep)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, errors.New("cycle detected in dependency graph")
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

// stringHeap is a min-heap of strings (lexicographic) used by TopologicalSort.
type stringHeap []string

func (h *stringHeap) Len() int           { return len(*h) }
func (h *stringHeap) Less(i, j int) bool { return (*h)[i] < (*h)[j] }
func (h *stringHeap) Swap(i, j int)      { (*h)[i], (*h)[j] = (*h)[j], (*h)[i] }

func (h *stringHeap) Push(x any) {
	if s, ok := x.(string); ok {
		*h = append(*h, s)
	}
}

func (h *stringHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
