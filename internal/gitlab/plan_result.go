package gitlab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edelwud/terraci/internal/terraform/plan"
)

// Constants for plan summary formatting
const (
	maxResourcesInSummary = 10 // Maximum number of resources to show
	maxAddressLength      = 80 // Maximum length for resource addresses
)

// PlanResult represents the result of a terraform plan for a single module
type PlanResult struct {
	ModuleID          string     `json:"module_id"`
	ModulePath        string     `json:"module_path"`
	Service           string     `json:"service"`
	Environment       string     `json:"environment"`
	Region            string     `json:"region"`
	Module            string     `json:"module"`
	Submodule         string     `json:"submodule,omitempty"`
	Status            PlanStatus `json:"status"`
	Summary           string     `json:"summary"`
	StructuredDetails string     `json:"structured_details,omitempty"`
	RawPlanOutput     string     `json:"raw_plan_output,omitempty"`
	Error             string     `json:"error,omitempty"`
	ExitCode          int        `json:"exit_code"`
}

// PlanResultCollection is a collection of plan results from multiple jobs
type PlanResultCollection struct {
	Results     []PlanResult `json:"results"`
	PipelineID  string       `json:"pipeline_id,omitempty"`
	CommitSHA   string       `json:"commit_sha,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// ToModulePlans converts plan results to ModulePlan for comment rendering
func (c *PlanResultCollection) ToModulePlans() []ModulePlan {
	plans := make([]ModulePlan, len(c.Results))
	for i := range c.Results {
		r := &c.Results[i]
		plans[i] = ModulePlan{
			ModuleID:          r.ModuleID,
			ModulePath:        r.ModulePath,
			Service:           r.Service,
			Environment:       r.Environment,
			Region:            r.Region,
			Module:            r.Module,
			Status:            r.Status,
			Summary:           r.Summary,
			StructuredDetails: r.StructuredDetails,
			RawPlanOutput:     r.RawPlanOutput,
			Error:             r.Error,
		}
	}
	return plans
}

// ScanPlanResults scans for plan.json files in module directories
// and builds a collection of plan results from their contents.
// This is used by the summary job to collect results from plan job artifacts.
func ScanPlanResults(rootDir string) (*PlanResultCollection, error) {
	collection := &PlanResultCollection{
		Results:     make([]PlanResult, 0),
		GeneratedAt: time.Now().UTC(),
		PipelineID:  os.Getenv("CI_PIPELINE_ID"),
		CommitSHA:   os.Getenv("CI_COMMIT_SHA"),
	}

	// Track processed directories to avoid duplicates
	processedDirs := make(map[string]bool)

	// Find all plan.json files recursively
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Skip walk errors, continue scanning
		}

		if info.IsDir() || info.Name() != "plan.json" {
			return nil
		}

		// Get module path from directory containing plan file
		modulePath := filepath.Dir(path)
		if rootDir != "." {
			if relPath, relErr := filepath.Rel(rootDir, modulePath); relErr == nil {
				modulePath = relPath
			}
		}

		// Skip if we already processed this directory
		if processedDirs[modulePath] {
			return nil
		}

		result, parseErr := parsePlanJSON(path, modulePath)
		if parseErr != nil {
			result = PlanResult{
				ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
				ModulePath: modulePath,
				Status:     PlanStatusFailed,
				Summary:    "Failed to parse plan",
				Error:      parseErr.Error(),
			}
		}

		processedDirs[modulePath] = true
		collection.Results = append(collection.Results, result)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan for plan results: %w", err)
	}

	return collection, nil
}

// parseModulePath parses a module path and extracts components.
// Supports both regular modules (service/env/region/module) and
// submodules (service/env/region/module/submodule).
func parseModulePath(modulePath string) (service, env, region, module, submodule string) {
	parts := strings.Split(modulePath, string(filepath.Separator))

	switch {
	case len(parts) >= 5:
		// Submodule: service/env/region/module/submodule
		service = parts[0]
		env = parts[1]
		region = parts[2]
		module = parts[3]
		submodule = strings.Join(parts[4:], "/")
	case len(parts) >= 4:
		// Regular module: service/env/region/module
		service = parts[0]
		env = parts[1]
		region = parts[2]
		module = parts[3]
	}

	return
}

// parsePlanJSON parses a plan.json file and creates a PlanResult
func parsePlanJSON(jsonPath, modulePath string) (PlanResult, error) {
	parsed, err := plan.ParseJSON(jsonPath)
	if err != nil {
		return PlanResult{}, err
	}

	service, env, region, module, submodule := parseModulePath(modulePath)

	// Read the text output and filter it
	txtPath := strings.TrimSuffix(jsonPath, ".json") + ".txt"
	var rawPlanOutput string
	if data, readErr := os.ReadFile(txtPath); readErr == nil {
		rawPlanOutput = FilterPlanOutput(string(data))
	}

	return PlanResult{
		ModuleID:          strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath:        modulePath,
		Service:           service,
		Environment:       env,
		Region:            region,
		Module:            module,
		Submodule:         submodule,
		Status:            getPlanStatus(parsed),
		Summary:           FormatPlanSummary(parsed),
		StructuredDetails: FormatPlanDetails(parsed),
		RawPlanOutput:     rawPlanOutput,
		ExitCode:          getExitCode(parsed),
	}, nil
}

// getPlanStatus returns the plan status based on parsed plan
func getPlanStatus(p *plan.ParsedPlan) PlanStatus {
	if !p.HasChanges() {
		return PlanStatusNoChanges
	}
	return PlanStatusChanges
}

// getExitCode returns the exit code based on the parsed plan
func getExitCode(p *plan.ParsedPlan) int {
	if !p.HasChanges() {
		return 0 // No changes
	}
	return 2 // Has changes
}

// FormatPlanSummary formats a parsed plan as a compact summary line
// Format: "+2 ~1 -1" or "No changes"
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
// Format:
//
//	**Create:**
//	- `aws_instance.web`
//	- `aws_s3_bucket.data`
//
//	**Update:**
//	- `aws_instance.api`
//
//	**Delete:**
//	- `aws_instance.old`
func FormatPlanDetails(p *plan.ParsedPlan) string {
	if !p.HasChanges() {
		return ""
	}

	// Group resources by action
	groups := make(map[string][]string)
	for _, r := range p.Resources {
		addr := formatResourceAddress(r)
		groups[r.Action] = append(groups[r.Action], addr)
	}

	var sb strings.Builder

	// Order: create, update, replace, delete, read
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

		sb.WriteString(fmt.Sprintf("**%s:**\n", a.label))
		shown := 0
		for _, addr := range resources {
			if shown >= maxResourcesInSummary {
				sb.WriteString(fmt.Sprintf("- ... +%d more\n", len(resources)-shown))
				break
			}
			sb.WriteString(fmt.Sprintf("- `%s`\n", addr))
			shown++
		}
		sb.WriteString("\n")
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// formatResourceAddress returns a shortened resource address for display
func formatResourceAddress(r plan.ResourceChange) string {
	addr := r.Address
	// Use short form if no module prefix
	if r.ModuleAddr == "" && r.Type != "" && r.Name != "" {
		addr = r.Type + "." + r.Name
	}
	// Truncate very long addresses (e.g., with for_each keys)
	if len(addr) > maxAddressLength {
		addr = addr[:maxAddressLength-3] + "..."
	}
	return addr
}

// FilterPlanOutput extracts only the diff portion from terraform plan output,
// removing "Refreshing state...", "Reading...", and other noise.
// Returns the filtered output suitable for display in PR comments.
func FilterPlanOutput(rawOutput string) string {
	lines := strings.Split(rawOutput, "\n")
	var result []string
	inDiff := false
	skipUntilEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip "Refreshing state..." and "Reading..." blocks
		if strings.HasPrefix(trimmed, "Refreshing state...") ||
			strings.HasPrefix(trimmed, "Reading...") ||
			strings.HasPrefix(trimmed, "data.") && strings.Contains(trimmed, "Reading...") {
			skipUntilEmpty = true
			continue
		}

		// Skip lines until we hit an empty line after noise
		if skipUntilEmpty {
			if trimmed == "" {
				skipUntilEmpty = false
			}
			continue
		}

		// Start capturing from resource changes or plan summary
		if strings.HasPrefix(trimmed, "# ") ||
			strings.HasPrefix(trimmed, "Plan:") ||
			strings.HasPrefix(trimmed, "Changes to Outputs:") ||
			strings.HasPrefix(trimmed, "Terraform will perform") ||
			strings.HasPrefix(trimmed, "No changes.") {
			inDiff = true
		}

		// Also start on actual diff lines
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

	// If nothing captured, return original (might be error output)
	if len(result) == 0 {
		return rawOutput
	}

	return strings.Join(result, "\n")
}
