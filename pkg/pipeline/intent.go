package pipeline

import (
	"errors"
	"fmt"
)

// BuildIntent describes the caller's high-level pipeline intent. BuildProjectIR
// derives concrete plan/apply jobs from this request and from resource needs.
type BuildIntent struct {
	constructed  bool
	applyEnabled bool
	resources    []ResourceRequest
}

type buildIntentOptions struct {
	ApplyEnabled     bool
	ResourceRequests []ResourceRequest
}

func newBuildIntent(opts buildIntentOptions) (BuildIntent, error) {
	for i, request := range opts.ResourceRequests {
		if err := validateResourceRequest(request); err != nil {
			return BuildIntent{}, fmt.Errorf("resource_requests[%d]: %w", i, err)
		}
	}
	return BuildIntent{
		constructed:  true,
		applyEnabled: opts.ApplyEnabled,
		resources:    append([]ResourceRequest(nil), opts.ResourceRequests...),
	}, nil
}

// ApplyBuildIntent returns an intent that creates plan and apply jobs.
func ApplyBuildIntent(resources ...ResourceRequest) (BuildIntent, error) {
	return newBuildIntent(buildIntentOptions{
		ApplyEnabled:     true,
		ResourceRequests: resources,
	})
}

// PlanBuildIntent returns an intent that creates only resource-driven plan jobs.
func PlanBuildIntent(resources ...ResourceRequest) (BuildIntent, error) {
	return newBuildIntent(buildIntentOptions{
		ApplyEnabled:     false,
		ResourceRequests: resources,
	})
}

func (i BuildIntent) validate() error {
	if !i.constructed {
		return errors.New("build intent must be created with pipeline.ApplyBuildIntent or pipeline.PlanBuildIntent")
	}
	return nil
}

// ApplyEnabled reports whether apply jobs should be emitted.
func (i BuildIntent) ApplyEnabled() bool { return i.applyEnabled }

// ResourceRequests returns resource requests that influence IR construction.
func (i BuildIntent) ResourceRequests() []ResourceRequest {
	return append([]ResourceRequest(nil), i.resources...)
}
