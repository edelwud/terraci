package graph

import "sort"

// Stats contains statistics about the dependency graph.
type Stats struct {
	TotalModules int
	TotalEdges   int
	RootModules  int // Modules with no dependencies
	LeafModules  int // Modules with no dependents
	MaxDepth     int
	AverageDepth float64
	HasCycles    bool
	CycleCount   int

	// Per-level module counts
	LevelCounts []int

	// Top modules by fan-in (most depended upon)
	TopDependedOn []ModuleStat

	// Top modules by fan-out (most dependencies)
	TopDependencies []ModuleStat
}

// ModuleStat holds a module ID and a count.
type ModuleStat struct {
	ID    string
	Count int
}

// GetStats returns statistics about the dependency graph.
func (g *DependencyGraph) GetStats() Stats {
	stats := Stats{
		TotalModules: len(g.nodes),
	}

	for _, edges := range g.edges {
		stats.TotalEdges += len(edges)
	}

	for id := range g.nodes {
		if len(g.edges[id]) == 0 {
			stats.RootModules++
		}
		if len(g.reverseEdges[id]) == 0 {
			stats.LeafModules++
		}
	}

	if levels, err := g.ExecutionLevels(); err == nil && len(levels) > 0 {
		stats.MaxDepth = len(levels) - 1
		stats.LevelCounts = make([]int, len(levels))
		totalDepth := 0
		for level, nodes := range levels {
			stats.LevelCounts[level] = len(nodes)
			totalDepth += level * len(nodes)
		}
		if len(g.nodes) > 0 {
			stats.AverageDepth = float64(totalDepth) / float64(len(g.nodes))
		}
	}

	cycles := g.DetectCycles()
	stats.HasCycles = len(cycles) > 0
	stats.CycleCount = len(cycles)

	stats.TopDependedOn = g.topByFanIn(5)
	stats.TopDependencies = g.topByFanOut(5)

	return stats
}

// topByFanIn returns the top N modules that are most depended upon.
func (g *DependencyGraph) topByFanIn(n int) []ModuleStat {
	var stats []ModuleStat
	for id := range g.nodes {
		if count := len(g.reverseEdges[id]); count > 0 {
			stats = append(stats, ModuleStat{ID: id, Count: count})
		}
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Count > stats[j].Count })
	if len(stats) > n {
		stats = stats[:n]
	}
	return stats
}

// topByFanOut returns the top N modules with the most dependencies.
func (g *DependencyGraph) topByFanOut(n int) []ModuleStat {
	var stats []ModuleStat
	for id := range g.nodes {
		if count := len(g.edges[id]); count > 0 {
			stats = append(stats, ModuleStat{ID: id, Count: count})
		}
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Count > stats[j].Count })
	if len(stats) > n {
		stats = stats[:n]
	}
	return stats
}
