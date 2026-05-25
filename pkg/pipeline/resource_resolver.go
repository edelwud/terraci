package pipeline

import (
	"errors"
	"fmt"
)

func isDetailedPlanResource(kind ResourceKind) bool {
	return kind == ResourceKindPlanText || kind == ResourceKindPlanJSON
}

func controlDependencies(names []string) []JobDependency {
	if len(names) == 0 {
		return nil
	}

	deps := make([]JobDependency, 0, len(names))
	for _, name := range names {
		deps = append(deps, JobDependency{Job: name})
	}
	return deps
}

func resultArtifactFromResources(jobName string, resources []ResourceSpec) Artifact {
	if len(resources) == 0 {
		return Artifact{}
	}
	return ResultArtifact(jobName, resourcePaths(resources)...)
}

func mergeJobDependency(deps []JobDependency, dep JobDependency) []JobDependency {
	if dep.Job == "" {
		return deps
	}

	for i := range deps {
		if deps[i].Job != dep.Job {
			continue
		}
		return deps
	}

	return append(deps, dep)
}

type producedResource struct {
	spec     ResourceSpec
	jobName  string
	artifact Artifact
}

type resolvedResource struct {
	resource producedResource
	optional bool
}

type resourceIndex struct {
	byRef map[ResourceRef]producedResource
	all   []producedResource
}

func buildResourceIndex(ir *IR) (*resourceIndex, error) {
	index := &resourceIndex{byRef: make(map[ResourceRef]producedResource)}
	if ir == nil {
		return index, nil
	}
	for i := range ir.jobs {
		job := &ir.jobs[i]
		for _, spec := range job.produces {
			if spec.Ref.Kind == "" {
				return nil, fmt.Errorf("pipeline job %q produces resource without kind", job.name)
			}
			if spec.Path == "" {
				return nil, fmt.Errorf("pipeline job %q produces %s without path", job.name, spec.Ref.Kind)
			}
			if !job.outputArtifact.Configured() {
				return nil, fmt.Errorf("pipeline job %q produces %s without output artifact", job.name, spec.Ref.Kind)
			}
			if existing, exists := index.byRef[spec.Ref]; exists {
				return nil, fmt.Errorf("pipeline resource %s produced by both %q and %q", resourceRefLabel(spec.Ref), existing.jobName, job.name)
			}

			resource := producedResource{
				spec:     spec,
				jobName:  job.name,
				artifact: job.outputArtifact,
			}
			index.byRef[spec.Ref] = resource
			index.all = append(index.all, resource)
		}
	}
	return index, nil
}

func resolveResourceRequestsForJob(requests []ResourceRequest, index *resourceIndex, jobName string, allowEmptyModuleResources bool) ([]ResourceSpec, []InputArtifact, []JobDependency, error) {
	consumes, produced, err := resolveResourceRequests(requests, index, jobName, allowEmptyModuleResources)
	if err != nil {
		return nil, nil, nil, err
	}

	artifacts := make([]InputArtifact, 0, len(produced))
	deps := make([]JobDependency, 0, len(produced))
	seenArtifacts := make(map[string]int, len(produced))
	for i := range produced {
		resolved := &produced[i]
		resource := resolved.resource
		if resource.jobName == jobName {
			continue
		}
		if resource.artifact.Configured() {
			if idx, seen := seenArtifacts[resource.artifact.Name]; seen {
				artifacts[idx].Optional = artifacts[idx].Optional && resolved.optional
			} else {
				artifacts = append(artifacts, InputArtifact{
					Artifact:    resource.artifact,
					ProducerJob: resource.jobName,
					Optional:    resolved.optional,
				})
				seenArtifacts[resource.artifact.Name] = len(artifacts) - 1
			}
		}
		deps = mergeJobDependency(deps, JobDependency{
			Job: resource.jobName,
		})
	}

	return consumes, artifacts, deps, nil
}

func resolveResourceRequests(requests []ResourceRequest, index *resourceIndex, consumer string, allowEmptyModuleResources bool) ([]ResourceSpec, []resolvedResource, error) {
	if len(requests) == 0 {
		return nil, nil, nil
	}

	var specs []ResourceSpec
	var resources []resolvedResource
	seen := make(map[ResourceRef]int)
	for _, request := range requests {
		if err := validateResourceRequest(request); err != nil {
			return nil, nil, err
		}
		matches := matchingResources(request, index, consumer)
		if len(matches) == 0 {
			if request.Optional || emptyResourceRequestAllowed(request, allowEmptyModuleResources) {
				continue
			}
			return nil, nil, fmt.Errorf("%s requires unavailable %s", consumer, resourceRequestLabel(request))
		}
		for _, match := range matches {
			if idx, exists := seen[match.spec.Ref]; exists {
				resources[idx].optional = resources[idx].optional && request.Optional
				continue
			}
			seen[match.spec.Ref] = len(resources)
			specs = append(specs, match.spec)
			resources = append(resources, resolvedResource{
				resource: match,
				optional: request.Optional,
			})
		}
	}
	return specs, resources, nil
}

func emptyResourceRequestAllowed(request ResourceRequest, allowEmptyModuleResources bool) bool {
	if !allowEmptyModuleResources || request.Selector.Scope != ResourceScopeAllModules {
		return false
	}
	return request.Kind == ResourceKindPlanBinary || isDetailedPlanResource(request.Kind)
}

func matchingResources(request ResourceRequest, index *resourceIndex, consumer string) []producedResource {
	if index == nil {
		return nil
	}

	var result []producedResource
	for _, resource := range index.all {
		if resource.jobName == consumer {
			continue
		}
		if resourceMatches(request, resource.spec.Ref) {
			result = append(result, resource)
		}
	}
	return result
}

func resourceMatches(request ResourceRequest, ref ResourceRef) bool {
	if request.Kind != ref.Kind {
		return false
	}
	switch request.Selector.Scope {
	case ResourceScopeModule:
		return ref.ModulePath == request.Selector.ModulePath
	case ResourceScopeProducer:
		return ref.Producer == request.Selector.Producer
	case ResourceScopeAllModules:
		return ref.ModulePath != ""
	case ResourceScopeAllProducers:
		return ref.Producer != ""
	default:
		return false
	}
}

func resourceRefLabel(ref ResourceRef) string {
	switch {
	case ref.ModulePath != "":
		return fmt.Sprintf("%s for module %q", ref.Kind, ref.ModulePath)
	case ref.Producer != "":
		return fmt.Sprintf("%s from producer %q", ref.Kind, ref.Producer)
	default:
		return string(ref.Kind)
	}
}

func resourceRequestLabel(request ResourceRequest) string {
	switch request.Selector.Scope {
	case ResourceScopeModule:
		return fmt.Sprintf("%s for module %q", request.Kind, request.Selector.ModulePath)
	case ResourceScopeProducer:
		return fmt.Sprintf("%s from producer %q", request.Kind, request.Selector.Producer)
	case ResourceScopeAllModules:
		return fmt.Sprintf("%s for all modules", request.Kind)
	case ResourceScopeAllProducers:
		return fmt.Sprintf("%s from all producers", request.Kind)
	default:
		return string(request.Kind)
	}
}

func validateResourceRequest(request ResourceRequest) error {
	if request.Kind == "" {
		return errors.New("resource request kind is required")
	}
	return validateResourceKindScope(request.Kind, request.Selector)
}

func validateResourceKindScope(kind ResourceKind, selector ResourceSelector) error {
	switch selector.Scope {
	case ResourceScopeAllModules:
		if selector.ModulePath != "" || selector.Producer != "" {
			return fmt.Errorf("%s selector %q must not set module_path or producer", kind, selector.Scope)
		}
		if !isPlanResourceKind(kind) {
			return fmt.Errorf("%s cannot use module-scoped selector %q", kind, selector.Scope)
		}
	case ResourceScopeModule:
		if selector.ModulePath == "" {
			return fmt.Errorf("%s selector %q requires module_path", kind, selector.Scope)
		}
		if err := ValidateWorkspacePath(selector.ModulePath); err != nil {
			return fmt.Errorf("%s selector %q module_path invalid: %w", kind, selector.Scope, err)
		}
		if selector.Producer != "" {
			return fmt.Errorf("%s selector %q must not set producer", kind, selector.Scope)
		}
		if !isPlanResourceKind(kind) {
			return fmt.Errorf("%s cannot use module-scoped selector %q", kind, selector.Scope)
		}
	case ResourceScopeAllProducers:
		if selector.ModulePath != "" || selector.Producer != "" {
			return fmt.Errorf("%s selector %q must not set module_path or producer", kind, selector.Scope)
		}
		if !isPluginResourceKind(kind) {
			return fmt.Errorf("%s cannot use producer-scoped selector %q", kind, selector.Scope)
		}
	case ResourceScopeProducer:
		if selector.Producer == "" {
			return fmt.Errorf("%s selector %q requires producer", kind, selector.Scope)
		}
		if err := validateProducerName(selector.Producer); err != nil {
			return fmt.Errorf("%s selector %q producer invalid: %w", kind, selector.Scope, err)
		}
		if selector.ModulePath != "" {
			return fmt.Errorf("%s selector %q must not set module_path", kind, selector.Scope)
		}
		if !isPluginResourceKind(kind) {
			return fmt.Errorf("%s cannot use producer-scoped selector %q", kind, selector.Scope)
		}
	default:
		return fmt.Errorf("%s selector scope is required", kind)
	}
	return nil
}

func isPlanResourceKind(kind ResourceKind) bool {
	return kind == ResourceKindPlanBinary || kind == ResourceKindPlanText || kind == ResourceKindPlanJSON
}

func isPluginResourceKind(kind ResourceKind) bool {
	return kind == ResourceKindPluginResult || kind == ResourceKindPluginReport
}
