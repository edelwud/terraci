package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// DefaultEIPHourlyCost is $0.005/hr for public IPv4 (since Feb 2024).
const DefaultEIPHourlyCost = 0.005

// EIPHandler handles aws_eip cost estimation.
type EIPHandler struct {
	awskit.RuntimeDeps
}

type eipAttrs struct {
	Instance string
}

func parseEIPAttrs(attrs map[string]any) eipAttrs {
	return eipAttrs{
		Instance: handler.GetStringAttr(attrs, "instance"),
	}
}

func (h *EIPHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *EIPHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseEIPAttrs(attrs)

	// Since Feb 2024, AWS charges $0.005/hr for all public IPv4 addresses.
	// Pricing is under AmazonVPC service with productFamily "None".
	runtime := h.RuntimeOrDefault()
	prefix := runtime.ResolveUsagePrefix(region)

	usagetype := prefix + "-PublicIPv4:InUseAddress"
	if parsed.Instance == "" {
		usagetype = prefix + "-PublicIPv4:IdleAddress"
	}

	// AWS VPC pricing uses group "VPCPublicIPv4Address" and no product family.
	return &pricing.PriceLookup{
		ServiceID: runtime.MustService(awskit.ServiceKeyVPC),
		Region:    region,
		Attributes: map[string]string{
			"location":  runtime.ResolveRegionName(region),
			"usagetype": usagetype,
			"group":     "VPCPublicIPv4Address",
		},
	}, nil
}

func (h *EIPHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if parseEIPAttrs(attrs).Instance != "" {
		d["attached"] = "true"
	} else {
		d["attached"] = "false"
	}
	return d
}

func (h *EIPHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	if price != nil && price.OnDemandUSD > 0 {
		return handler.HourlyCost(price.OnDemandUSD)
	}
	// Fallback: $0.005/hr ($3.65/month) since Feb 2024
	// Attached to running instance still costs $0.005/hr
	return handler.HourlyCost(DefaultEIPHourlyCost)
}
