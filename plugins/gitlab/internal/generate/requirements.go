package generate

import (
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func PipelineRequirements(cfg *configpkg.Config) pipeline.BuildRequirements {
	return pipeline.BuildRequirements{PlanOnly: newSettings(cfg, execution.Config{}).planOnly()}
}
