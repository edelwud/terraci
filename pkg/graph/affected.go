package graph

import (
	"sort"
	"strings"
)

// GetAffectedModules returns modules affected by changes to the given modules.
// Includes the changed modules themselves plus all transitive dependents and dependencies.
func (g *DependencyGraph) GetAffectedModules(changedModules []string) []string {
	affected := make(map[string]bool)
	for _, m := range changedModules {
		affected[m] = true
		for _, dep := range g.GetAllDependents(m) {
			affected[dep] = true
		}
		for _, dep := range g.GetAllDependencies(m) {
			affected[dep] = true
		}
	}
	return sortedKeys(affected)
}

// GetAffectedByLibraryChanges returns executable modules affected by changes to library modules.
func (g *DependencyGraph) GetAffectedByLibraryChanges(changedLibraryPaths []string) []string {
	affected := make(map[string]bool)

	for _, libPath := range changedLibraryPaths {
		for _, moduleID := range g.GetModulesUsingLibrary(libPath) {
			affected[moduleID] = true
		}
	}

	// Handle transitive: if a changed path is a parent of a tracked library path
	for _, libPath := range changedLibraryPaths {
		libPath = normalizeLibraryPath(libPath)
		for trackedPath, moduleIDs := range g.libraryUsage {
			if strings.HasPrefix(trackedPath, libPath+"/") {
				for _, moduleID := range moduleIDs {
					affected[moduleID] = true
				}
			}
		}
	}

	return sortedKeys(affected)
}

// GetAffectedModulesWithLibraries combines executable and library module changes.
func (g *DependencyGraph) GetAffectedModulesWithLibraries(changedModules, changedLibraryPaths []string) []string {
	affected := make(map[string]bool)

	for _, m := range g.GetAffectedModules(changedModules) {
		affected[m] = true
	}

	for _, m := range g.GetAffectedByLibraryChanges(changedLibraryPaths) {
		affected[m] = true
		for _, dep := range g.GetAllDependencies(m) {
			affected[dep] = true
		}
	}

	return sortedKeys(affected)
}

func sortedKeys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
