package dependency

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/model"
)

type ModuleParser interface {
	ParseModule(ctx context.Context, modulePath string) (*model.ParsedModule, error)
	ResolveWorkspacePath(ref *model.RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error)
}

type Dependency = model.Dependency

type LibraryDependency = model.LibraryDependency

type ModuleDependencies = model.ModuleDependencies
