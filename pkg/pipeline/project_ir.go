package pipeline

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/workflow"
)

// ProjectIRRequest describes the canonical project-to-IR build request.
type ProjectIRRequest struct {
	Project       *workflow.ProjectResult
	Terraform     TerraformJobConfig
	Intent        BuildIntent
	Contributions ContributionSet
}

// BuildProjectIR builds a provider-agnostic IR from a planned project.
func BuildProjectIR(req ProjectIRRequest) (*IR, error) {
	if req.Project == nil {
		return nil, errors.New("project is required")
	}
	if req.Project.Workflow == nil {
		return nil, errors.New("project workflow is required")
	}
	result := req.Project.Workflow
	if result.Graph == nil {
		return nil, errors.New("project workflow graph is required")
	}
	if result.Filtered.Index == nil {
		return nil, errors.New("project workflow module index is required")
	}

	targets := req.Project.Targets
	if targets == nil {
		targets = result.Filtered.Modules
	}

	ir, err := buildProjectIR(projectIRBuildInput{
		DepGraph:      result.Graph,
		TargetModules: targets,
		AllModules:    result.Filtered.Modules,
		ModuleIndex:   result.Filtered.Index,
		Terraform:     req.Terraform,
		Contributions: req.Contributions,
		Intent:        req.Intent,
	})
	if err != nil {
		return nil, fmt.Errorf("build project pipeline IR: %w", err)
	}
	return ir, nil
}
