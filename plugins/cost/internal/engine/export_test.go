package engine

import "github.com/edelwud/terraci/plugins/cost/internal/model"

// MapTerraformAction exposes the internal Terraform action mapping for adapter tests.
func MapTerraformAction(action string) (model.EstimateAction, error) {
	return mapTerraformAction(action)
}
