package parser

import (
	"maps"

	"github.com/hashicorp/hcl/v2"
)

func extractRemoteStates(ctx *extractContext) {
	for _, remoteState := range ctx.index.remoteStateBlockViews() {
		ref := &RemoteStateRef{
			Name:    remoteState.Name(),
			Config:  make(map[string]hcl.Expression),
			RawBody: remoteState.RawBody(),
		}
		parseRemoteStateBlock(ctx, remoteState, ref)
		ctx.parsed.RemoteStates = append(ctx.parsed.RemoteStates, ref)
	}
}

func parseRemoteStateBlock(ctx *extractContext, view remoteStateBlockView, ref *RemoteStateRef) {
	content, diags := view.Content()
	ctx.addDiags(diags)
	if content == nil {
		return
	}

	if val, ok := evalContentStringAttr(content, "backend"); ok {
		ref.Backend = val
	}
	if attr, ok := content.Attributes["for_each"]; ok {
		ref.ForEach = attr.Expr
	}

	if _, ok := content.Attributes["config"]; ok {
		maps.Copy(ref.Config, view.InlineConfigExpressions(content))
	}

	configAttrs, configDiags := view.ConfigBlockAttributes(content)
	ctx.addDiags(configDiags)
	maps.Copy(ref.Config, configAttrs)
}
