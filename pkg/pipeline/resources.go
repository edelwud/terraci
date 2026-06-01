package pipeline

import "github.com/edelwud/terraci/pkg/ci"

// ResourceKind identifies a data product exchanged between pipeline jobs.
type ResourceKind string

const (
	ResourceKindPlanBinary   ResourceKind = "plan_binary"
	ResourceKindPlanText     ResourceKind = "plan_text"
	ResourceKindPlanJSON     ResourceKind = "plan_json"
	ResourceKindPluginResult ResourceKind = "plugin_result"
	ResourceKindPluginReport ResourceKind = "plugin_report"
)

// ResourceRef is the concrete identity of a pipeline resource.
type ResourceRef struct {
	Kind       ResourceKind
	ModulePath string
	Producer   string
}

// ResourceScope identifies how a ResourceRequest selects concrete resources.
type ResourceScope string

const (
	ResourceScopeAllModules   ResourceScope = "all_modules"
	ResourceScopeModule       ResourceScope = "module"
	ResourceScopeAllProducers ResourceScope = "all_producers"
	ResourceScopeProducer     ResourceScope = "producer"
)

// ResourceSelector is the selector portion of a ResourceRequest.
type ResourceSelector struct {
	scope      ResourceScope
	modulePath string
	producer   string
}

// ResourceRequest selects one or more resources needed by a job.
type ResourceRequest struct {
	kind     ResourceKind
	selector ResourceSelector
	optional bool
}

// ResourceSpec binds a resource identity to its workspace-relative path.
type ResourceSpec struct {
	Ref  ResourceRef
	Path string
}

// PlanResource returns a module-scoped plan resource.
func PlanResource(kind ResourceKind, modulePath, path string) ResourceSpec {
	return ResourceSpec{
		Ref: ResourceRef{
			Kind:       kind,
			ModulePath: WorkspacePath(modulePath),
		},
		Path: WorkspacePath(path),
	}
}

// PluginResource returns a producer-scoped plugin resource.
func PluginResource(kind ResourceKind, producer, path string) ResourceSpec {
	return ResourceSpec{
		Ref: ResourceRef{
			Kind:     kind,
			Producer: producer,
		},
		Path: WorkspacePath(path),
	}
}

// AllPlanResources requests a plan resource from every target module.
func AllPlanResources(kind ResourceKind) ResourceRequest {
	return ResourceRequest{
		kind: kind,
		selector: ResourceSelector{
			scope: ResourceScopeAllModules,
		},
	}
}

// ModulePlanResource requests a plan resource from one module.
func ModulePlanResource(kind ResourceKind, modulePath string) ResourceRequest {
	return ResourceRequest{
		kind: kind,
		selector: ResourceSelector{
			scope:      ResourceScopeModule,
			modulePath: WorkspacePath(modulePath),
		},
	}
}

// AllPluginResources requests a plugin resource from every producer.
func AllPluginResources(kind ResourceKind, optional bool) ResourceRequest {
	return ResourceRequest{
		kind: kind,
		selector: ResourceSelector{
			scope: ResourceScopeAllProducers,
		},
		optional: optional,
	}
}

// PluginProducerResource requests one plugin resource by producer.
func PluginProducerResource(kind ResourceKind, producer string, optional bool) ResourceRequest {
	return ResourceRequest{
		kind: kind,
		selector: ResourceSelector{
			scope:    ResourceScopeProducer,
			producer: producer,
		},
		optional: optional,
	}
}

// Kind returns the requested resource kind.
func (r ResourceRequest) Kind() ResourceKind { return r.kind }

// Selector returns the resource selector.
func (r ResourceRequest) Selector() ResourceSelector { return r.selector }

// Optional reports whether a missing resource should be ignored.
func (r ResourceRequest) Optional() bool { return r.optional }

// Scope returns selector scope.
func (s ResourceSelector) Scope() ResourceScope { return s.scope }

// ModulePath returns the selected module path for module-scoped requests.
func (s ResourceSelector) ModulePath() string { return s.modulePath }

// Producer returns the selected producer for producer-scoped requests.
func (s ResourceSelector) Producer() string { return s.producer }

func PluginResultAndReportResources(serviceDir, producer string) []ResourceSpec {
	return []ResourceSpec{
		PluginResource(ResourceKindPluginResult, producer, WorkspacePath(serviceDir, ci.ResultFilename(producer))),
		PluginResource(ResourceKindPluginReport, producer, WorkspacePath(serviceDir, ci.ReportFilename(producer))),
	}
}

// DependencyNames returns dependency job names in declaration order.
func DependencyNames(deps []JobDependency) []string {
	if len(deps) == 0 {
		return nil
	}

	names := make([]string, 0, len(deps))
	for _, dep := range deps {
		if dep.Job == "" {
			continue
		}
		names = append(names, dep.Job)
	}
	return names
}
