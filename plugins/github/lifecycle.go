package github

import (
	"context"

	"github.com/edelwud/terraci/pkg/plugin"
	prpkg "github.com/edelwud/terraci/plugins/github/internal/pr"
	"github.com/edelwud/terraci/plugins/internal/ciplugin"
)

// Preflight validates the loaded plugin config and detects PR context when
// running inside GitHub Actions. Validation runs before the env-detection
// branch so misshapen configs fail fast even on local generate runs.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	var cfg ciplugin.ConfigValidator
	if c := p.Config(); c != nil {
		cfg = c
	}
	return ciplugin.Preflight(cfg, p.DetectEnv, ciplugin.PreflightLog{
		ProviderName: pluginName,
		ContextLabel: "PR",
		DetectInContext: func() (any, bool) {
			ctx := prpkg.DetectContext()
			return ctx.PRNumber, ctx.InPR
		},
	})
}
