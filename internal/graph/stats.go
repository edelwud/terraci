package graph

// Stats contains statistics about the dependency graph.
type Stats struct {
	TotalModules int
	TotalEdges   int
	RootModules  int     // Modules with no dependencies
	LeafModules  int     // Modules with no dependents
	MaxDepth     int     // Maximum dependency chain length
	AverageDepth float64 // Average dependency chain length
	HasCycles    bool
	CycleCount   int
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
		totalDepth := 0
		for level, nodes := range levels {
			totalDepth += level * len(nodes)
		}
		stats.AverageDepth = float64(totalDepth) / float64(len(g.nodes))
	}

	cycles := g.DetectCycles()
	stats.HasCycles = len(cycles) > 0
	stats.CycleCount = len(cycles)

	return stats
}
