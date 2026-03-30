package model

import "github.com/edelwud/terraci/pkg/discovery"

type Dependency struct {
	From            *discovery.Module
	To              *discovery.Module
	Type            string
	RemoteStateName string
}

type LibraryDependency struct {
	ModuleCall  *ModuleCall
	LibraryPath string
}

type ModuleDependencies struct {
	Module              *discovery.Module
	Dependencies        []*Dependency
	LibraryDependencies []*LibraryDependency
	DependsOn           []string
	Errors              []error
}
