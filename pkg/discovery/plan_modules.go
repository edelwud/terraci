package discovery

import (
	"os"
	"path/filepath"
)

// FindModulesWithPlan finds all directories containing plan.json files.
// Returns deduplicated absolute directory paths.
func FindModulesWithPlan(rootDir string) ([]string, error) {
	var paths []string
	seen := make(map[string]bool)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // skip inaccessible paths
		}
		if info.IsDir() || info.Name() != "plan.json" {
			return nil
		}
		dir := filepath.Dir(path)
		if !seen[dir] {
			seen[dir] = true
			paths = append(paths, dir)
		}
		return nil
	})

	return paths, err
}
