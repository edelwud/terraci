package pipeline

// BuildRequirements are provider/runtime requests that must influence IR
// construction before a provider renders the finished pipeline.
type BuildRequirements struct {
	Resources []ResourceRequest
	PlanOnly  bool
}

func RequirementsForResources(resources ...ResourceRequest) BuildRequirements {
	return BuildRequirements{Resources: append([]ResourceRequest(nil), resources...)}
}

func RequirementsForDetailedPlans() BuildRequirements {
	return RequirementsForResources(
		AllPlanResources(ResourceKindPlanText),
		AllPlanResources(ResourceKindPlanJSON),
	)
}

func (r BuildRequirements) Merge(other BuildRequirements) BuildRequirements {
	merged := BuildRequirements{
		Resources: append([]ResourceRequest(nil), r.Resources...),
		PlanOnly:  r.PlanOnly || other.PlanOnly,
	}
	merged.Resources = append(merged.Resources, other.Resources...)
	return merged
}
