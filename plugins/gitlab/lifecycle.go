package gitlab

import (
	"context"

	"github.com/edelwud/terraci/pkg/plugin"
	mrpkg "github.com/edelwud/terraci/plugins/gitlab/internal/mr"
	"github.com/edelwud/terraci/plugins/internal/ciplugin"
)

// Preflight validates the loaded plugin config and detects MR context when
// running inside GitLab CI. Validation runs before the env-detection branch
// so misshapen configs fail fast even on local generate runs.
func (p *Plugin) Preflight(_ context.Context, _ *plugin.AppContext) error {
	var cfg ciplugin.ConfigValidator
	if c := p.Config(); c != nil {
		cfg = c
	}
	return ciplugin.Preflight(cfg, p.DetectEnv, ciplugin.PreflightLog{
		ProviderName: "gitlab",
		ContextLabel: "MR",
		DetectInContext: func() (any, bool) {
			ctx := mrpkg.DetectContext()
			return ctx.MRIID, ctx.InMR
		},
	})
}
