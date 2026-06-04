package planresults

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// defaultPlanSegments is the default pattern segments when none are provided.
var defaultPlanSegments = []string{"service", "environment", "region", "module"}

// Scan scans for plan.json files in module directories
// and builds a collection of plan results from their contents.
// If segments is nil or empty, default segments (service/environment/region/module) are used.
func Scan(rootDir string, segments []string) (*ci.PlanResultCollection, error) {
	if len(segments) == 0 {
		segments = defaultPlanSegments
	}

	moduleDirs, err := FindModulesWithPlan(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for plan results: %w", err)
	}

	results := make([]ci.PlanResult, 0, len(moduleDirs))
	for _, dir := range moduleDirs {
		jsonPath := filepath.Join(dir, pipeline.PlanJSONFilename)

		modulePath := dir
		if rootDir != "." {
			if relPath, relErr := filepath.Rel(rootDir, dir); relErr == nil {
				modulePath = filepath.ToSlash(relPath)
			}
		}

		result, parseErr := parsePlanJSON(jsonPath, modulePath, segments)
		if parseErr != nil {
			result, err = ci.NewPlanResult(ci.PlanResultOptions{
				ModuleID:   filepath.ToSlash(modulePath),
				ModulePath: modulePath,
				Status:     ci.PlanStatusFailed,
				Summary:    "Failed to parse plan",
				Error:      parseErr.Error(),
			})
			if err != nil {
				return nil, err
			}
		}

		results = append(results, result)
	}

	return ci.NewPlanResultCollection(ci.PlanResultCollectionOptions{
		Results:     results,
		GeneratedAt: time.Now().UTC(),
	})
}

// ParseModulePathComponents parses a module path using the given segment names
// and returns a map of component name to value. Extra path parts beyond the
// defined segments are joined as "submodule".
func ParseModulePathComponents(modulePath string, segments []string) map[string]string {
	parts := strings.Split(filepath.ToSlash(modulePath), "/")
	components := make(map[string]string, len(segments)+1)

	if len(parts) >= len(segments) {
		for i, seg := range segments {
			components[seg] = parts[i]
		}
		// Extra parts become submodule
		if len(parts) > len(segments) {
			components["submodule"] = strings.Join(parts[len(segments):], "/")
		}
	}

	return components
}

func parsePlanJSON(jsonPath, modulePath string, segments []string) (ci.PlanResult, error) {
	parsed, err := plan.ParseJSON(jsonPath)
	if err != nil {
		var empty ci.PlanResult
		return empty, err
	}

	components := ParseModulePathComponents(modulePath, segments)

	txtPath := strings.TrimSuffix(jsonPath, ".json") + ".txt"
	var rawPlanOutput string
	if data, readErr := os.ReadFile(txtPath); readErr == nil {
		rawPlanOutput = plan.FilterPlanOutput(string(data))
	}

	return ci.NewPlanResult(ci.PlanResultOptions{
		ModuleID:          filepath.ToSlash(modulePath),
		ModulePath:        modulePath,
		Components:        components,
		Status:            ci.PlanStatusFromPlan(parsed.HasChanges()),
		Summary:           parsed.Summary(),
		StructuredDetails: parsed.Details(),
		RawPlanOutput:     rawPlanOutput,
		ExitCode:          parsed.ExitCode(),
	})
}
