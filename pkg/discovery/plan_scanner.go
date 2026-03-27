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

// Constants for plan summary formatting
const (
	maxResourcesInSummary = 10
	maxAddressLength      = 80
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
		PipelineID:  detectPipelineID(),
		CommitSHA:   detectCommitSHA(),
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

// detectPipelineID reads pipeline ID from CI environment (GitLab or GitHub)
func detectPipelineID() string {
	if v := os.Getenv("CI_PIPELINE_ID"); v != "" {
		return v
	}
	return os.Getenv("GITHUB_RUN_ID")
}

// detectCommitSHA reads commit SHA from CI environment (GitLab or GitHub)
func detectCommitSHA() string {
	if v := os.Getenv("CI_COMMIT_SHA"); v != "" {
		return v
	}
	return os.Getenv("GITHUB_SHA")
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
		rawPlanOutput = FilterPlanOutput(string(data))
	}

	return ci.PlanResult{
		ModuleID:          strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath:        modulePath,
		Components:        components,
		Status:            getPlanStatus(parsed),
		Summary:           FormatPlanSummary(parsed),
		StructuredDetails: FormatPlanDetails(parsed),
		RawPlanOutput:     rawPlanOutput,
		ExitCode:          getExitCode(parsed),
	}, nil
}

func getPlanStatus(p *plan.ParsedPlan) ci.PlanStatus {
	if !p.HasChanges() {
		return ci.PlanStatusNoChanges
	}
	return ci.PlanStatusChanges
}

func getExitCode(p *plan.ParsedPlan) int {
	if !p.HasChanges() {
		return 0
	}
	return 2
}

// FormatPlanSummary formats a parsed plan as a compact summary line
func FormatPlanSummary(p *plan.ParsedPlan) string {
	if !p.HasChanges() {
		return "No changes"
	}

	var counts []string
	if p.ToAdd > 0 {
		counts = append(counts, fmt.Sprintf("+%d", p.ToAdd))
	}
	if p.ToChange > 0 {
		counts = append(counts, fmt.Sprintf("~%d", p.ToChange))
	}
	if p.ToDestroy > 0 {
		counts = append(counts, fmt.Sprintf("-%d", p.ToDestroy))
	}
	if p.ToImport > 0 {
		counts = append(counts, fmt.Sprintf("↓%d", p.ToImport))
	}
	return strings.Join(counts, " ")
}

// FormatPlanDetails formats a parsed plan as structured resource list grouped by action
func FormatPlanDetails(p *plan.ParsedPlan) string {
	if !p.HasChanges() {
		return ""
	}

	groups := make(map[string][]string)
	for _, r := range p.Resources {
		addr := formatResourceAddress(r)
		groups[r.Action] = append(groups[r.Action], addr)
	}

	var sb strings.Builder

	actionOrder := []struct {
		action string
		label  string
	}{
		{"create", "Create"},
		{"update", "Update"},
		{"replace", "Replace"},
		{"delete", "Delete"},
		{"read", "Read"},
	}

	for _, a := range actionOrder {
		resources := groups[a.action]
		if len(resources) == 0 {
			continue
		}

		fmt.Fprintf(&sb, "**%s:**\n", a.label)
		shown := 0
		for _, addr := range resources {
			if shown >= maxResourcesInSummary {
				fmt.Fprintf(&sb, "- ... +%d more\n", len(resources)-shown)
				break
			}
			fmt.Fprintf(&sb, "- `%s`\n", addr)
			shown++
		}
		sb.WriteString("\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

func formatResourceAddress(r plan.ResourceChange) string {
	addr := r.Address
	if r.ModuleAddr == "" && r.Type != "" && r.Name != "" {
		addr = r.Type + "." + r.Name
	}
	if len(addr) > maxAddressLength {
		addr = addr[:maxAddressLength-3] + "..."
	}
	return addr
}

// FilterPlanOutput extracts only the diff portion from terraform plan output
func FilterPlanOutput(rawOutput string) string {
	lines := strings.Split(rawOutput, "\n")
	var result []string
	inDiff := false
	skipUntilEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Refreshing state...") ||
			strings.HasPrefix(trimmed, "Reading...") ||
			strings.HasPrefix(trimmed, "data.") && strings.Contains(trimmed, "Reading...") {
			skipUntilEmpty = true
			continue
		}

		if skipUntilEmpty {
			if trimmed == "" {
				skipUntilEmpty = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "# ") ||
			strings.HasPrefix(trimmed, "Plan:") ||
			strings.HasPrefix(trimmed, "Changes to Outputs:") ||
			strings.HasPrefix(trimmed, "Terraform will perform") ||
			strings.HasPrefix(trimmed, "No changes.") {
			inDiff = true
		}

		if strings.HasPrefix(line, "  + ") ||
			strings.HasPrefix(line, "  - ") ||
			strings.HasPrefix(line, "  ~ ") ||
			strings.HasPrefix(line, "  # ") {
			inDiff = true
		}

		if inDiff {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return rawOutput
	}

	return strings.Join(result, "\n")
}
