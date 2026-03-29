package graph

import (
	"fmt"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func buildBenchGraph(n int) *DependencyGraph {
	g := NewDependencyGraph()
	for i := range n {
		mod := discovery.TestModule("svc", "prod", "region", fmt.Sprintf("mod%d", i))
		g.AddNode(mod)
		if i > 0 {
			g.AddEdge(mod.ID(), fmt.Sprintf("svc/prod/region/mod%d", i-1))
		}
	}
	return g
}

func BenchmarkTopologicalSort(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		g := buildBenchGraph(size)
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				g.TopologicalSort()
			}
		})
	}
}

func BenchmarkExecutionLevels(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		g := buildBenchGraph(size)
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				g.ExecutionLevels()
			}
		})
	}
}

func BenchmarkDetectCycles(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		g := buildBenchGraph(size)
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				g.DetectCycles()
			}
		})
	}
}
