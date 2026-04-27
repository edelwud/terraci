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
	appCtx                 *plugin.AppContext
	changeDetectorResolver workflow.ChangeDetectorResolver
}

func NewWorkflowResolver(appCtx *plugin.AppContext, changeDetectorResolver workflow.ChangeDetectorResolver) Resolver {
	if changeDetectorResolver == nil {
		changeDetectorResolver = func() (plugin.ChangeDetectionProvider, error) {
			return plugin.ResolveChangeDetector(appCtx)
		}
	}
	return WorkflowResolver{appCtx: appCtx, changeDetectorResolver: changeDetectorResolver}
}

func (r WorkflowResolver) Resolve(ctx context.Context, req spec.ExecuteRequest, result *workflow.Result) ([]*discovery.Module, error) {
	return workflow.ResolveTargets(ctx, r.appCtx, result, workflow.TargetSelectionOptions{
		ModulePath:             req.ModulePath,
		ChangedOnly:            req.ChangedOnly,
		BaseRef:                req.BaseRef,
		Filters:                req.Filters,
		ChangeDetectorResolver: r.changeDetectorResolver,
	})
}
