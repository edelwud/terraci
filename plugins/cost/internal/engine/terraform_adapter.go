package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	tfplan "github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// TerraformPlanAdapter converts Terraform plan.json files into the engine input model.
type TerraformPlanAdapter struct{}

// NewTerraformPlanAdapter creates a Terraform-backed module plan adapter.
func NewTerraformPlanAdapter() *TerraformPlanAdapter {
	return &TerraformPlanAdapter{}
}

// LoadModule reads a Terraform plan and maps it into the provider-neutral input model.
func (a *TerraformPlanAdapter) LoadModule(modulePath, region string) (*ModulePlan, error) {
	planJSONPath := filepath.Join(modulePath, "plan.json")

	parsedPlan, err := tfplan.ParseJSON(planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("parse plan.json: %w", err)
	}

	modulePlan := &ModulePlan{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		HasChanges: parsedPlan.HasChanges(),
		Resources:  make([]PlannedResource, 0, len(parsedPlan.Resources)),
	}

	for _, rc := range parsedPlan.Resources {
		if rc.Action == tfplan.ActionRead {
			// Data-source refreshes are not cost-bearing changes and should not fail
			// module estimation just because Terraform records them as read actions.
			continue
		}

		action, err := mapTerraformAction(rc.Action)
		if err != nil {
			return nil, fmt.Errorf("map action for %s: %w", rc.Address, err)
		}

		modulePlan.Resources = append(modulePlan.Resources, PlannedResource{
			ResourceType: resourcedef.ResourceType(rc.Type),
			Address:      rc.Address,
			Name:         rc.Name,
			ModuleAddr:   rc.ModuleAddr,
			Action:       action,
			BeforeAttrs:  rc.BeforeValues,
			AfterAttrs:   rc.AfterValues,
		})
	}

	return modulePlan, nil
}

func mapTerraformAction(action string) (model.EstimateAction, error) {
	switch action {
	case tfplan.ActionCreate:
		return model.ActionCreate, nil
	case tfplan.ActionDelete:
		return model.ActionDelete, nil
	case tfplan.ActionUpdate:
		return model.ActionUpdate, nil
	case tfplan.ActionReplace:
		return model.ActionReplace, nil
	case tfplan.ActionNoOp:
		return model.ActionNoOp, nil
	default:
		return "", fmt.Errorf("unsupported action %q", action)
	}
}
