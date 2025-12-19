package gitlab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PlanResult represents the result of a terraform plan for a single module
type PlanResult struct {
	ModuleID    string     `json:"module_id"`
	ModulePath  string     `json:"module_path"`
	Service     string     `json:"service"`
	Environment string     `json:"environment"`
	Region      string     `json:"region"`
	Module      string     `json:"module"`
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

// ScanPlanResults scans for plan.txt files in module directories
// and builds a collection of plan results from their contents.
// This is used by the summary job to collect results from plan job artifacts.
func ScanPlanResults(rootDir string) (*PlanResultCollection, error) {
	collection := &PlanResultCollection{
		Results:     make([]PlanResult, 0),
		GeneratedAt: time.Now().UTC(),
		PipelineID:  os.Getenv("CI_PIPELINE_ID"),
		CommitSHA:   os.Getenv("CI_COMMIT_SHA"),
	}

	// Find all plan.txt files recursively
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Skip walk errors, continue scanning
		}

		if info.IsDir() {
			return nil
		}

		if info.Name() != "plan.txt" {
			return nil
		}

		// Read plan output
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil //nolint:nilerr // Skip unreadable files, continue scanning
		}

		// Get module path from directory containing plan.txt
		modulePath := filepath.Dir(path)
		if rootDir != "." {
			if relPath, relErr := filepath.Rel(rootDir, modulePath); relErr == nil {
				modulePath = relPath
			}
		}

		// Determine exit code based on content
		output := string(data)
		exitCode := inferExitCode(output)
		status, summary := ParsePlanOutput(output, exitCode)

		// Parse module ID from path (service/env/region/module)
		parts := strings.Split(modulePath, string(filepath.Separator))
		var service, env, region, module string
		if len(parts) >= 4 {
			service = parts[0]
			env = parts[1]
			region = parts[2]
			module = parts[3]
		}

		result := PlanResult{
			ModuleID:    strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
			ModulePath:  modulePath,
			Service:     service,
			Environment: env,
			Region:      region,
			Module:      module,
			Status:      status,
			Summary:     summary,
			Details:     output,
			ExitCode:    exitCode,
		}

		collection.Results = append(collection.Results, result)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan for plan results: %w", err)
	}

	return collection, nil
}

// ParsePlanOutput parses terraform plan output and extracts summary
func ParsePlanOutput(output string, exitCode int) (status PlanStatus, summary string) {
	output = strings.TrimSpace(output)

	switch exitCode {
	case 0:
		// Success - no changes
		if strings.Contains(output, "No changes.") || strings.Contains(output, "Your infrastructure matches the configuration") {
			return PlanStatusNoChanges, "No changes. Infrastructure is up-to-date."
		}
		return PlanStatusSuccess, extractPlanSummary(output)
	case 1:
		// Error
		return PlanStatusFailed, ""
	case 2:
		// Success - has changes
		return PlanStatusChanges, extractPlanSummary(output)
	default:
		return PlanStatusFailed, ""
	}
}

// extractPlanSummary extracts the summary line from plan output
func extractPlanSummary(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "Plan: X to add, Y to change, Z to destroy"
		if strings.HasPrefix(line, "Plan:") {
			return line
		}
		// Also check for "No changes" variations
		if strings.Contains(line, "No changes") {
			return line
		}
	}
	return ""
}

// inferExitCode tries to determine the terraform plan exit code from output
func inferExitCode(output string) int {
	// Check for error indicators
	if strings.Contains(output, "Error:") {
		return 1 // Error
	}

	// Check for no changes
	if strings.Contains(output, "No changes.") ||
		strings.Contains(output, "Your infrastructure matches the configuration") {
		return 0 // No changes
	}

	// Check for changes
	if strings.Contains(output, "Plan:") &&
		(strings.Contains(output, "to add") ||
			strings.Contains(output, "to change") ||
			strings.Contains(output, "to destroy")) {
		return 2 // Has changes
	}

	// Default to success if we can't determine
	return 0
}
