package dependency

import "github.com/edelwud/terraci/pkg/discovery"

type dependencyResultBuilder struct {
	module              *discovery.Module
	dependencies        []*Dependency
	libraryDependencies []*LibraryDependency
	errors              []error
}

func newDependencyResultBuilder(module *discovery.Module) *dependencyResultBuilder {
	return &dependencyResultBuilder{
		module:              module,
		dependencies:        make([]*Dependency, 0),
		libraryDependencies: make([]*LibraryDependency, 0),
		errors:              make([]error, 0),
	}
}

func (b *dependencyResultBuilder) AddDependencies(dependencies ...*Dependency) {
	b.dependencies = append(b.dependencies, dependencies...)
}

func (b *dependencyResultBuilder) AddLibraryDependency(dependency *LibraryDependency) {
	if dependency == nil {
		return
	}

	b.libraryDependencies = append(b.libraryDependencies, dependency)
}

func (b *dependencyResultBuilder) AddErrors(errs ...error) {
	for _, err := range errs {
		if err == nil {
			continue
		}

		b.errors = append(b.errors, err)
	}
}

func (b *dependencyResultBuilder) Build() *ModuleDependencies {
	return &ModuleDependencies{
		Module:              b.module,
		Dependencies:        b.dependencies,
		LibraryDependencies: b.libraryDependencies,
		DependsOn:           collectDependsOnIDs(b.dependencies),
		Errors:              b.errors,
	}
}

type remoteStateResolution struct {
	dependencies []*Dependency
	errors       []error
}

func newRemoteStateResolution() *remoteStateResolution {
	return &remoteStateResolution{
		dependencies: make([]*Dependency, 0),
		errors:       make([]error, 0),
	}
}

func (r *remoteStateResolution) AddDependency(dependency *Dependency) {
	if dependency == nil {
		return
	}

	r.dependencies = append(r.dependencies, dependency)
}

func (r *remoteStateResolution) AddError(err error) {
	if err == nil {
		return
	}

	r.errors = append(r.errors, err)
}

func (r *remoteStateResolution) Dependencies() []*Dependency {
	return r.dependencies
}

func (r *remoteStateResolution) Errors() []error {
	return r.errors
}

type dependencyCollectionBuilder struct {
	results map[string]*ModuleDependencies
	errors  []error
}

func newDependencyCollectionBuilder() *dependencyCollectionBuilder {
	return &dependencyCollectionBuilder{
		results: make(map[string]*ModuleDependencies),
		errors:  make([]error, 0),
	}
}

func (b *dependencyCollectionBuilder) Add(module *discovery.Module, deps *ModuleDependencies, err error) {
	if err != nil {
		b.errors = append(b.errors, err)
		return
	}

	if module == nil || deps == nil {
		return
	}

	b.results[module.ID()] = deps
	b.errors = append(b.errors, deps.Errors...)
}

func (b *dependencyCollectionBuilder) Build() (map[string]*ModuleDependencies, []error) {
	return b.results, b.errors
}

func collectDependsOnIDs(dependencies []*Dependency) []string {
	seen := make(map[string]struct{})
	dependsOn := make([]string, 0, len(dependencies))

	for _, dependency := range dependencies {
		if dependency == nil || dependency.To == nil {
			continue
		}

		moduleID := dependency.To.ID()
		if _, ok := seen[moduleID]; ok {
			continue
		}

		seen[moduleID] = struct{}{}
		dependsOn = append(dependsOn, moduleID)
	}

	return dependsOn
}
