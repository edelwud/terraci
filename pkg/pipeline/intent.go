package pipeline

import "fmt"

// BuildIntent describes the caller's high-level pipeline intent. BuildProjectIR
// derives concrete plan/apply jobs from this request and from resource needs.
type BuildIntent struct {
	applyEnabled bool
	resources    []ResourceRequest
}

// BuildIntentOptions configures NewBuildIntent.
type BuildIntentOptions struct {
	ApplyEnabled     bool
	ResourceRequests []ResourceRequest
}

// NewBuildIntent creates an immutable pipeline build intent.
func NewBuildIntent(opts BuildIntentOptions) (BuildIntent, error) {
	for i, request := range opts.ResourceRequests {
		if err := validateResourceRequest(request); err != nil {
			return BuildIntent{}, fmt.Errorf("resource_requests[%d]: %w", i, err)
		}
	}
	return BuildIntent{
		applyEnabled: opts.ApplyEnabled,
		resources:    append([]ResourceRequest(nil), opts.ResourceRequests...),
	}, nil
}

// ApplyBuildIntent returns an intent that creates plan and apply jobs.
func ApplyBuildIntent(resources ...ResourceRequest) (BuildIntent, error) {
	return NewBuildIntent(BuildIntentOptions{
		ApplyEnabled:     true,
		ResourceRequests: resources,
	})
}

// PlanBuildIntent returns an intent that creates only resource-driven plan jobs.
func PlanBuildIntent(resources ...ResourceRequest) (BuildIntent, error) {
	return NewBuildIntent(BuildIntentOptions{
		ApplyEnabled:     false,
		ResourceRequests: resources,
	})
}

// ApplyEnabled reports whether apply jobs should be emitted.
func (i BuildIntent) ApplyEnabled() bool { return i.applyEnabled }

// ResourceRequests returns resource requests that influence IR construction.
func (i BuildIntent) ResourceRequests() []ResourceRequest {
	return append([]ResourceRequest(nil), i.resources...)
}

// MergeBuildIntents combines multiple build intents.
func MergeBuildIntents(intents ...BuildIntent) (BuildIntent, error) {
	var merged BuildIntentOptions
	for _, intent := range intents {
		merged.ApplyEnabled = merged.ApplyEnabled || intent.applyEnabled
		merged.ResourceRequests = append(merged.ResourceRequests, intent.resources...)
	}
	return NewBuildIntent(merged)
}
