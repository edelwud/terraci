package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/pkg/ci"
)

// defaultPlanSegments is the default pattern segments when none are provided.
var defaultPlanSegments = []string{"service", "environment", "region", "module"}

// ScanPlanResults scans for plan.json files in module directories
// and builds a collection of plan results from their contents.
// If segments is nil or empty, default segments (service/environment/region/module) are used.
func ScanPlanResults(rootDir string, segments []string) (*ci.PlanResultCollection, error) {
	if len(segments) == 0 {
		segments = defaultPlanSegments
	}

	collection := &ci.PlanResultCollection{
		Results:     make([]ci.PlanResult, 0),
		GeneratedAt: time.Now().UTC(),
	}

	moduleDirs, err := FindModulesWithPlan(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for plan results: %w", err)
	}

	for _, dir := range moduleDirs {
		jsonPath := filepath.Join(dir, "plan.json")

		modulePath := dir
		if rootDir != "." {
			if relPath, relErr := filepath.Rel(rootDir, dir); relErr == nil {
				modulePath = relPath
			}
		}

		result, parseErr := parsePlanJSON(jsonPath, modulePath, segments)
		if parseErr != nil {
			result = ci.PlanResult{
				ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
				ModulePath: modulePath,
				Status:     ci.PlanStatusFailed,
				Summary:    "Failed to parse plan",
				Error:      parseErr.Error(),
			}
		}

		collection.Results = append(collection.Results, result)
	}

	return collection, nil
}

// ParseModulePathComponents parses a module path using the given segment names
// and returns a map of component name to value. Extra path parts beyond the
// defined segments are joined as "submodule".
func ParseModulePathComponents(modulePath string, segments []string) map[string]string {
	parts := strings.Split(modulePath, string(filepath.Separator))
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
		return ci.PlanResult{}, err
	}

	components := ParseModulePathComponents(modulePath, segments)

	txtPath := strings.TrimSuffix(jsonPath, ".json") + ".txt"
	var rawPlanOutput string
	if data, readErr := os.ReadFile(txtPath); readErr == nil {
		rawPlanOutput = plan.FilterPlanOutput(string(data))
	}

	return ci.PlanResult{
		ModuleID:          strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath:        modulePath,
		Components:        components,
		Status:            ci.PlanStatusFromPlan(parsed.HasChanges()),
		Summary:           parsed.Summary(),
		StructuredDetails: parsed.Details(),
		RawPlanOutput:     rawPlanOutput,
		ExitCode:          parsed.ExitCode(),
	}, nil
}
