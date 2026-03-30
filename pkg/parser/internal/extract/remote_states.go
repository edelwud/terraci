package extract

import (
	"maps"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

func extractRemoteStates(ctx *Context) {
	for _, remoteState := range ctx.Source.RemoteStateBlockViews() {
		ref := RemoteStateRef{
			Name:    remoteState.Name(),
			Config:  make(map[string]hcl.Expression),
			RawBody: remoteState.RawBody(),
		}
		parseRemoteStateBlock(ctx, remoteState, &ref)
		ctx.Sink.AppendRemoteState(ref)
	}
}

func parseRemoteStateBlock(ctx *Context, view source.RemoteStateBlockView, ref *RemoteStateRef) {
	content, diags := view.Content()
	ctx.Sink.AddDiags(diags)
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
	ctx.Sink.AddDiags(configDiags)
	maps.Copy(ref.Config, configAttrs)
}
