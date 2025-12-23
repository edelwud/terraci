package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/version"
)

// OPAVersion returns the version of the embedded OPA library
func OPAVersion() string {
	return version.Version
}

// Engine evaluates Rego policies against Terraform plan JSON
type Engine struct {
	// policyDirs contains paths to directories with .rego files
	policyDirs []string

	// namespaces to evaluate (e.g., ["terraform.aws", "terraform.security"])
	namespaces []string
}

// NewEngine creates a new policy engine
func NewEngine(policyDirs, namespaces []string) *Engine {
	return &Engine{
		policyDirs: policyDirs,
		namespaces: namespaces,
	}
}

// Evaluate runs policy checks against a Terraform plan JSON file
func (e *Engine) Evaluate(ctx context.Context, planJSONPath string) (*Result, error) {
	// Read the plan JSON
	planData, err := os.ReadFile(planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan JSON: %w", err)
	}

	var input map[string]any
	if unmarshalErr := json.Unmarshal(planData, &input); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", unmarshalErr)
	}

	// Collect all .rego files from policy directories
	regoFiles, err := e.collectRegoFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to collect rego files: %w", err)
	}

	if len(regoFiles) == 0 {
		return &Result{Successes: 0, Skipped: 0}, nil
	}

	result := &Result{}

	// Evaluate each namespace
	for _, ns := range e.namespaces {
		failures, warnings, err := e.evaluateNamespace(ctx, input, regoFiles, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate namespace %s: %w", ns, err)
		}
		result.Failures = append(result.Failures, failures...)
		result.Warnings = append(result.Warnings, warnings...)
	}

	return result, nil
}

// collectRegoFiles finds all .rego files in policy directories
func (e *Engine) collectRegoFiles() ([]string, error) {
	var files []string

	for _, dir := range e.policyDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".rego") && !strings.HasSuffix(path, "_test.rego") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip missing directories
			}
			return nil, err
		}
	}

	return files, nil
}

// evaluateNamespace evaluates policies in a specific namespace
func (e *Engine) evaluateNamespace(ctx context.Context, input map[string]any, regoFiles []string, namespace string) (failures, warnings []Violation, err error) {
	// Build the query for deny rules
	denyQuery := fmt.Sprintf("data.%s.deny", namespace)
	denyViolations, err := e.runQuery(ctx, input, regoFiles, denyQuery, namespace)
	if err != nil {
		// Namespace might not exist in policies, skip it
		if strings.Contains(err.Error(), "undefined") {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	failures = append(failures, denyViolations...)

	// Build the query for warn rules
	warnQuery := fmt.Sprintf("data.%s.warn", namespace)
	warnViolations, err := e.runQuery(ctx, input, regoFiles, warnQuery, namespace)
	if err != nil {
		// warn rules are optional
		if !strings.Contains(err.Error(), "undefined") {
			return nil, nil, err
		}
	}
	warnings = append(warnings, warnViolations...)

	return failures, warnings, nil
}

// runQuery executes a Rego query and returns violations
func (e *Engine) runQuery(ctx context.Context, input map[string]any, regoFiles []string, query, namespace string) ([]Violation, error) {
	// Load all rego files
	opts := []func(*rego.Rego){
		rego.Query(query),
		rego.Input(input),
	}

	for _, f := range regoFiles {
		opts = append(opts, rego.Load([]string{f}, nil))
	}

	r := rego.New(opts...)

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, err
	}

	var violations []Violation

	for _, result := range rs {
		for _, expr := range result.Expressions {
			switch v := expr.Value.(type) {
			case []any:
				for _, item := range v {
					violation := e.parseViolation(item, namespace)
					if violation != nil {
						violations = append(violations, *violation)
					}
				}
			case map[string]any:
				violation := e.parseViolation(v, namespace)
				if violation != nil {
					violations = append(violations, *violation)
				}
			case string:
				violations = append(violations, Violation{
					Message:   v,
					Namespace: namespace,
				})
			}
		}
	}

	return violations, nil
}

// parseViolation parses a violation from OPA result
func (e *Engine) parseViolation(v any, namespace string) *Violation {
	switch val := v.(type) {
	case string:
		return &Violation{
			Message:   val,
			Namespace: namespace,
		}
	case map[string]any:
		violation := &Violation{
			Namespace: namespace,
			Metadata:  make(map[string]any),
		}
		if msg, ok := val["msg"].(string); ok {
			violation.Message = msg
		} else if msg, ok := val["message"].(string); ok {
			violation.Message = msg
		}
		for k, v := range val {
			if k != "msg" && k != "message" {
				violation.Metadata[k] = v
			}
		}
		if violation.Message == "" {
			// If no message, use JSON representation
			if data, marshalErr := json.Marshal(val); marshalErr == nil {
				violation.Message = string(data)
			}
		}
		return violation
	}
	return nil
}
