package gitlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// PlanResultDir is the default directory for plan results
	PlanResultDir = ".terraci-results"
	// PlanResultFilePattern is the pattern for plan result files
	PlanResultFilePattern = "plan-*.json"
)

// PlanResult represents the result of a terraform plan for a single module
type PlanResult struct {
	ModuleID    string        `json:"module_id"`
	ModulePath  string        `json:"module_path"`
	Service     string        `json:"service"`
	Environment string        `json:"environment"`
	Region      string        `json:"region"`
	Module      string        `json:"module"`
	Status      PlanStatus    `json:"status"`
	Summary     string        `json:"summary"`
	Details     string        `json:"details,omitempty"`
	Error       string        `json:"error,omitempty"`
	ExitCode    int           `json:"exit_code"`
	Duration    time.Duration `json:"duration_ns"`
	StartedAt   time.Time     `json:"started_at"`
	FinishedAt  time.Time     `json:"finished_at"`
	JobID       string        `json:"job_id,omitempty"`
	JobURL      string        `json:"job_url,omitempty"`
}

// PlanResultCollection is a collection of plan results from multiple jobs
type PlanResultCollection struct {
	Results     []PlanResult `json:"results"`
	PipelineID  string       `json:"pipeline_id,omitempty"`
	CommitSHA   string       `json:"commit_sha,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// SavePlanResult saves a plan result to a JSON file
func SavePlanResult(result *PlanResult, dir string) error {
	if dir == "" {
		dir = PlanResultDir
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	// Create safe filename from module ID
	safeID := strings.ReplaceAll(result.ModuleID, "/", "-")
	filename := fmt.Sprintf("plan-%s.json", safeID)
	resultPath := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan result: %w", err)
	}

	//nolint:gosec // G306: Plan results need to be readable by other pipeline jobs
	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write plan result: %w", err)
	}

	return nil
}

// LoadPlanResults loads all plan results from a directory
func LoadPlanResults(dir string) (*PlanResultCollection, error) {
	if dir == "" {
		dir = PlanResultDir
	}

	collection := &PlanResultCollection{
		Results:     make([]PlanResult, 0),
		GeneratedAt: time.Now().UTC(),
		PipelineID:  os.Getenv("CI_PIPELINE_ID"),
		CommitSHA:   os.Getenv("CI_COMMIT_SHA"),
	}

	pattern := filepath.Join(dir, PlanResultFilePattern)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob plan results: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		var result PlanResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}

		collection.Results = append(collection.Results, result)
	}

	return collection, nil
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
			Duration:    r.Duration,
		}
	}
	return plans
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

// PlanResultWriter helps build plan results during execution
type PlanResultWriter struct {
	result    *PlanResult
	outputDir string
}

// NewPlanResultWriter creates a new plan result writer
func NewPlanResultWriter(moduleID, modulePath, outputDir string) *PlanResultWriter {
	// Parse module ID components
	parts := strings.Split(moduleID, "/")
	var service, env, region, module string
	if len(parts) >= 4 {
		service = parts[0]
		env = parts[1]
		region = parts[2]
		module = parts[3]
	}

	return &PlanResultWriter{
		result: &PlanResult{
			ModuleID:    moduleID,
			ModulePath:  modulePath,
			Service:     service,
			Environment: env,
			Region:      region,
			Module:      module,
			Status:      PlanStatusPending,
			StartedAt:   time.Now().UTC(),
			JobID:       os.Getenv("CI_JOB_ID"),
			JobURL:      os.Getenv("CI_JOB_URL"),
		},
		outputDir: outputDir,
	}
}

// SetOutput sets the plan output and parses the result
func (w *PlanResultWriter) SetOutput(output string, exitCode int) {
	w.result.Details = output
	w.result.ExitCode = exitCode
	w.result.Status, w.result.Summary = ParsePlanOutput(output, exitCode)
}

// SetError sets an error on the plan result
func (w *PlanResultWriter) SetError(err error) {
	w.result.Status = PlanStatusFailed
	w.result.Error = err.Error()
}

// Finish finalizes and saves the plan result
func (w *PlanResultWriter) Finish() error {
	w.result.FinishedAt = time.Now().UTC()
	w.result.Duration = w.result.FinishedAt.Sub(w.result.StartedAt)
	return SavePlanResult(w.result, w.outputDir)
}

// Result returns the current plan result
func (w *PlanResultWriter) Result() *PlanResult {
	return w.result
}
