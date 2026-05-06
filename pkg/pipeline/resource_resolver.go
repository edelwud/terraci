package pipeline

import "fmt"

func requestsRequireDetailedPlan(required []ResourceRequest, jobs []ContributedJob) bool {
	for _, request := range required {
		if isDetailedPlanResource(request.Kind) {
			return true
		}
	}
	for _, job := range jobs {
		for _, request := range job.Consumes {
			if isDetailedPlanResource(request.Kind) {
				return true
			}
		}
	}
	return false
}

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
		deps[i].Artifacts = deps[i].Artifacts || dep.Artifacts
		deps[i].Optional = deps[i].Optional && dep.Optional
		return deps
	}

	return append(deps, dep)
}

type producedResource struct {
	spec     ResourceSpec
	jobName  string
	artifact Artifact
}

type resourceIndex struct {
	byRef map[ResourceRef]producedResource
	all   []producedResource
}

func buildResourceIndex(ir *IR) (*resourceIndex, error) {
	index := &resourceIndex{byRef: make(map[ResourceRef]producedResource)}
	for _, ref := range ir.JobRefs() {
		if ref.Job == nil {
			continue
		}
		for _, spec := range ref.Job.Produces {
			if spec.Ref.Kind == "" {
				return nil, fmt.Errorf("pipeline job %q produces resource without kind", ref.Job.Name)
			}
			if spec.Path == "" {
				return nil, fmt.Errorf("pipeline job %q produces %s without path", ref.Job.Name, spec.Ref.Kind)
			}
			if !ref.Job.OutputArtifact.Configured() {
				return nil, fmt.Errorf("pipeline job %q produces %s without output artifact", ref.Job.Name, spec.Ref.Kind)
			}
			if existing, exists := index.byRef[spec.Ref]; exists {
				return nil, fmt.Errorf("pipeline resource %s produced by both %q and %q", resourceRefLabel(spec.Ref), existing.jobName, ref.Job.Name)
			}

			resource := producedResource{
				spec:     spec,
				jobName:  ref.Job.Name,
				artifact: ref.Job.OutputArtifact,
			}
			index.byRef[spec.Ref] = resource
			index.all = append(index.all, resource)
		}
	}
	return index, nil
}

func resolveResourceRequestsForJob(requests []ResourceRequest, index *resourceIndex, jobName string, allowEmptyModuleResources bool) ([]ResourceSpec, []Artifact, []JobDependency, error) {
	consumes, produced, err := resolveResourceRequests(requests, index, jobName, allowEmptyModuleResources)
	if err != nil {
		return nil, nil, nil, err
	}

	artifacts := make([]Artifact, 0, len(produced))
	deps := make([]JobDependency, 0, len(produced))
	seenArtifacts := make(map[string]struct{}, len(produced))
	for _, resource := range produced {
		if resource.jobName == jobName {
			continue
		}
		if resource.artifact.Configured() {
			if _, seen := seenArtifacts[resource.artifact.Name]; !seen {
				artifacts = append(artifacts, resource.artifact)
				seenArtifacts[resource.artifact.Name] = struct{}{}
			}
		}
		deps = mergeJobDependency(deps, JobDependency{
			Job:       resource.jobName,
			Artifacts: resource.artifact.Configured(),
			Optional:  requestOptionalForResource(requests, resource.spec.Ref),
		})
	}

	return consumes, artifacts, deps, nil
}

func resolveResourceRequests(requests []ResourceRequest, index *resourceIndex, consumer string, allowEmptyModuleResources bool) ([]ResourceSpec, []producedResource, error) {
	if len(requests) == 0 {
		return nil, nil, nil
	}

	var specs []ResourceSpec
	var resources []producedResource
	seen := make(map[ResourceRef]struct{})
	for _, request := range requests {
		matches := matchingResources(request, index, consumer)
		if len(matches) == 0 {
			if request.Optional || emptyResourceRequestAllowed(request, allowEmptyModuleResources) {
				continue
			}
			return nil, nil, fmt.Errorf("%s requires unavailable %s", consumer, resourceRequestLabel(request))
		}
		for _, match := range matches {
			if _, exists := seen[match.spec.Ref]; exists {
				continue
			}
			seen[match.spec.Ref] = struct{}{}
			specs = append(specs, match.spec)
			resources = append(resources, match)
		}
	}
	return specs, resources, nil
}

func emptyResourceRequestAllowed(request ResourceRequest, allowEmptyModuleResources bool) bool {
	if !allowEmptyModuleResources || !request.AllModules {
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
	if request.ModulePath != "" {
		return ref.ModulePath == request.ModulePath
	}
	if request.Producer != "" {
		return ref.Producer == request.Producer
	}
	if request.AllModules {
		return ref.ModulePath != ""
	}
	if request.AllProducers {
		return ref.Producer != ""
	}
	return true
}

func requestOptionalForResource(requests []ResourceRequest, ref ResourceRef) bool {
	optional := false
	for _, request := range requests {
		if !resourceMatches(request, ref) {
			continue
		}
		if !request.Optional {
			return false
		}
		optional = true
	}
	return optional
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
	switch {
	case request.ModulePath != "":
		return fmt.Sprintf("%s for module %q", request.Kind, request.ModulePath)
	case request.Producer != "":
		return fmt.Sprintf("%s from producer %q", request.Kind, request.Producer)
	case request.AllModules:
		return fmt.Sprintf("%s for all modules", request.Kind)
	case request.AllProducers:
		return fmt.Sprintf("%s from all producers", request.Kind)
	default:
		return string(request.Kind)
	}
}
