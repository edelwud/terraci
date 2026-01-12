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
	maxResourcesInSummary  = 10 // Maximum number of resources to show
	maxAttributesInSummary = 5  // Maximum number of attributes to show per resource
	maxValueLength         = 40 // Maximum length for attribute values
	minTruncateLength      = 3  // Minimum length for truncation (length of "...")
)

// PlanResult represents the result of a terraform plan for a single module
type PlanResult struct {
	ModuleID    string     `json:"module_id"`
	ModulePath  string     `json:"module_path"`
	Service     string     `json:"service"`
	Environment string     `json:"environment"`
	Region      string     `json:"region"`
	Module      string     `json:"module"`
	Submodule   string     `json:"submodule,omitempty"`
	Status      PlanStatus `json:"status"`
	Summary     string     `json:"summary"`
	Details     string     `json:"details,omitempty"`
	Error       string     `json:"error,omitempty"`
	ExitCode    int        `json:"exit_code"`
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
			ModuleID:    r.ModuleID,
			ModulePath:  r.ModulePath,
			Service:     r.Service,
			Environment: r.Environment,
			Region:      r.Region,
			Module:      r.Module,
			Status:      r.Status,
			Summary:     r.Summary,
			Details:     r.Details,
			Error:       r.Error,
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

	// Read the text output for Details field if available
	txtPath := strings.TrimSuffix(jsonPath, ".json") + ".txt"
	var details string
	if data, readErr := os.ReadFile(txtPath); readErr == nil {
		details = string(data)
	}

	return PlanResult{
		ModuleID:    strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath:  modulePath,
		Service:     service,
		Environment: env,
		Region:      region,
		Module:      module,
		Submodule:   submodule,
		Status:      getPlanStatus(parsed),
		Summary:     FormatPlanSummary(parsed),
		Details:     details,
		ExitCode:    getExitCode(parsed),
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

// FormatPlanSummary formats a parsed plan as a detailed multi-line summary
// Format:
//
//	+2 ~1 -1
//	+ aws_instance.web
//	+ aws_s3_bucket.data
//	~ aws_instance.api: instance_type="t2.micro" → "t2.small", tags.Name="old" → "new"
//	- aws_instance.old
func FormatPlanSummary(p *plan.ParsedPlan) string {
	if !p.HasChanges() {
		return "No changes"
	}

	var sb strings.Builder

	// Header line with counts
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
	sb.WriteString(strings.Join(counts, " "))

	// Resource details
	shown := 0
	for _, r := range p.Resources {
		if shown >= maxResourcesInSummary {
			sb.WriteString(fmt.Sprintf("\n  ... +%d more", len(p.Resources)-shown))
			break
		}

		sb.WriteString("\n")
		sb.WriteString(formatResourceChange(r))
		shown++
	}

	return sb.String()
}

// formatResourceChange formats a single resource change with its attributes
func formatResourceChange(r plan.ResourceChange) string {
	// Action symbol
	var symbol string
	switch r.Action {
	case "create":
		symbol = "+"
	case "update":
		symbol = "~"
	case "delete":
		symbol = "-"
	case "replace":
		symbol = "±"
	case "read":
		symbol = "?"
	default:
		symbol = " "
	}

	// Resource address (use short form if no module prefix)
	addr := r.Address
	if r.ModuleAddr == "" && r.Type != "" && r.Name != "" {
		addr = r.Type + "." + r.Name
	}

	// For create/delete, just show the address
	if r.Action == "create" || r.Action == "delete" || r.Action == "read" {
		return fmt.Sprintf("  %s %s", symbol, addr)
	}

	// For update/replace, show changed attributes with values
	if len(r.Attributes) == 0 {
		return fmt.Sprintf("  %s %s", symbol, addr)
	}

	attrParts := make([]string, 0, len(r.Attributes))
	for i, attr := range r.Attributes {
		if i >= maxAttributesInSummary {
			attrParts = append(attrParts, "...")
			break
		}
		attrParts = append(attrParts, formatAttrDiff(attr))
	}

	return fmt.Sprintf("  %s %s: %s", symbol, addr, strings.Join(attrParts, ", "))
}

// formatAttrDiff formats a single attribute change
func formatAttrDiff(attr plan.AttrDiff) string {
	if attr.Sensitive {
		return fmt.Sprintf("%s=(sensitive)", attr.Path)
	}

	if attr.Computed {
		if attr.OldValue != "" {
			return fmt.Sprintf("%s=%s → (known after apply)", attr.Path, truncateValue(attr.OldValue, maxValueLength))
		}
		return fmt.Sprintf("%s=(known after apply)", attr.Path)
	}

	oldVal := truncateValue(attr.OldValue, maxValueLength)
	newVal := truncateValue(attr.NewValue, maxValueLength)

	// New attribute (no old value)
	if attr.OldValue == "" && attr.NewValue != "" {
		return fmt.Sprintf("%s=%s", attr.Path, newVal)
	}

	// Removed attribute (no new value)
	if attr.OldValue != "" && attr.NewValue == "" {
		return fmt.Sprintf("%s=%s → null", attr.Path, oldVal)
	}

	// Changed attribute
	if attr.ForceNew {
		return fmt.Sprintf("%s=%s → %s (forces replacement)", attr.Path, oldVal, newVal)
	}
	return fmt.Sprintf("%s=%s → %s", attr.Path, oldVal, newVal)
}

// truncateValue truncates a string to maxLen, adding "..." if truncated
func truncateValue(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= minTruncateLength {
		return "..."
	}
	return s[:maxLen-minTruncateLength] + "..."
}
