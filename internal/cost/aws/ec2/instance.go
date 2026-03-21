package ec2

import (
	"fmt"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// InstanceHandler handles aws_instance cost estimation.
type InstanceHandler struct{}

func (h *InstanceHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *InstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *InstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceType := aws.GetStringAttr(attrs, "instance_type")
	if instanceType == "" {
		return nil, fmt.Errorf("instance_type not found")
	}

	tenancy := aws.GetStringAttr(attrs, "tenancy")
	switch tenancy {
	case "", "default":
		tenancy = "Shared"
	case "dedicated":
		tenancy = "Dedicated"
	case "host":
		tenancy = "Host"
	}

	// Determine OS
	operatingSystem := "Linux"
	if ami := aws.GetStringAttr(attrs, "ami"); ami != "" {
		// Could check AMI patterns, but default to Linux for now
		operatingSystem = "Linux"
	}

	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: "Compute Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":    instanceType,
		"tenancy":         tenancy,
		"operatingSystem": operatingSystem,
		"preInstalledSw":  "NA",
		"capacitystatus":  "Used",
	}), nil
}

func (h *InstanceHandler) CalculateCost(price *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	return aws.HourlyCost(price.OnDemandUSD)
}
