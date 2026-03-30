package dependency

import (
	"errors"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser/model"
)

func TestDependencyResultBuilderBuild(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "app")
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	rds := discovery.TestModule("platform", "stage", "eu-central-1", "rds")

	builder := newDependencyResultBuilder(module)
	builder.AddDependencies(
		&Dependency{From: module, To: vpc, Type: "remote_state", RemoteStateName: "vpc"},
		&Dependency{From: module, To: vpc, Type: "remote_state", RemoteStateName: "vpc_duplicate"},
		&Dependency{From: module, To: rds, Type: "remote_state", RemoteStateName: "rds"},
	)
	builder.AddLibraryDependency(&LibraryDependency{
		ModuleCall:  &model.ModuleCall{Name: "shared"},
		LibraryPath: "_modules/shared",
	})
	builder.AddErrors(errors.New("dependency error"))

	result := builder.Build()

	if result.Module != module {
		t.Fatalf("module mismatch")
	}
	if len(result.Dependencies) != 3 {
		t.Fatalf("dependencies = %d, want 3", len(result.Dependencies))
	}
	if len(result.LibraryDependencies) != 1 {
		t.Fatalf("library dependencies = %d, want 1", len(result.LibraryDependencies))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(result.Errors))
	}
	if len(result.DependsOn) != 2 {
		t.Fatalf("depends_on = %d, want 2", len(result.DependsOn))
	}
}

func TestRemoteStateResolution(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "app")
	target := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")

	resolution := newRemoteStateResolution()
	resolution.AddDependency(&Dependency{
		From:            module,
		To:              target,
		Type:            "remote_state",
		RemoteStateName: "vpc",
	})
	resolution.AddError(errors.New("resolution error"))
	resolution.AddDependency(nil)
	resolution.AddError(nil)

	if len(resolution.Dependencies()) != 1 {
		t.Fatalf("dependencies = %d, want 1", len(resolution.Dependencies()))
	}
	if len(resolution.Errors()) != 1 {
		t.Fatalf("errors = %d, want 1", len(resolution.Errors()))
	}
}

func TestDependencyCollectionBuilderBuild(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "app")
	deps := &ModuleDependencies{
		Module: module,
		Errors: []error{errors.New("soft dependency error")},
	}

	builder := newDependencyCollectionBuilder()
	builder.Add(module, deps, nil)
	builder.Add(discovery.TestModule("platform", "stage", "eu-central-1", "broken"), nil, errors.New("hard extract error"))

	results, errs := builder.Build()

	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if len(errs) != 2 {
		t.Fatalf("errors = %d, want 2", len(errs))
	}
}

func TestRemoteStateTargetResolverResolve(t *testing.T) {
	app := discovery.TestModule("platform", "stage", "eu-central-1", "app")
	app.Path = t.TempDir()
	app.RelativePath = "platform/stage/eu-central-1/app"

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	vpc.RelativePath = "platform/stage/eu-central-1/vpc"

	legacy := discovery.TestModule("network", "shared", "global", "legacy")
	legacy.RelativePath = "legacy/custom"

	engine := NewEngine(nil, discovery.NewModuleIndex([]*discovery.Module{app, vpc, legacy}))
	engine.backendIndex.items[BackendIndexKey(&model.BackendConfig{
		Type: "s3",
		Config: map[string]string{
			"bucket": "shared-state",
			"key":    "legacy/custom/terraform.tfstate",
		},
	}, legacy.RelativePath)] = legacy

	resolver := newRemoteStateTargetResolver(
		engine,
		app,
		map[string]cty.Value{},
		map[string]cty.Value{},
	)

	pathMatched := resolver.Resolve(&model.RemoteStateRef{}, "platform/stage/eu-central-1/vpc/terraform.tfstate")
	if pathMatched == nil || pathMatched.ID() != vpc.ID() {
		t.Fatalf("path match = %v, want %s", pathMatched, vpc.ID())
	}

	backendMatched := resolver.Resolve(&model.RemoteStateRef{
		Name:    "legacy",
		Backend: "s3",
		Config: map[string]hcl.Expression{
			"bucket": mustParseExpression(t, `"shared-state"`),
		},
	}, "legacy/custom/terraform.tfstate")
	if backendMatched == nil || backendMatched.ID() != legacy.ID() {
		t.Fatalf("backend match = %v, want %s", backendMatched, legacy.ID())
	}
}

func mustParseExpression(t *testing.T, src string) hcl.Expression {
	t.Helper()

	expr, diags := hclsyntax.ParseExpression([]byte(src), "test.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse expression %q: %s", src, diags.Error())
	}

	return expr
}
