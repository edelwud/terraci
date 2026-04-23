package targeting

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type Resolver interface {
	Resolve(ctx context.Context, req spec.ExecuteRequest, result *workflow.Result) ([]*discovery.Module, error)
}

type WorkflowResolver struct {
	appCtx *plugin.AppContext
}

func NewWorkflowResolver(appCtx *plugin.AppContext) Resolver {
	return WorkflowResolver{appCtx: appCtx}
}

func (r WorkflowResolver) Resolve(ctx context.Context, req spec.ExecuteRequest, result *workflow.Result) ([]*discovery.Module, error) {
	return workflow.ResolveTargets(ctx, r.appCtx, result, workflow.TargetSelectionOptions{
		ModulePath:  req.ModulePath,
		ChangedOnly: req.ChangedOnly,
		BaseRef:     req.BaseRef,
		Filters:     req.Filters,
	})
}
