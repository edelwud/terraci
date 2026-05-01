package plugin_test

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
)

type exampleRuntime struct {
	workDir string
}

type exampleRuntimePlugin struct {
	plugin.BasePlugin[*exampleRuntimeConfig]
}

type exampleRuntimeConfig struct {
	Enabled bool
}

func (p *exampleRuntimePlugin) Runtime(_ context.Context, appCtx *plugin.AppContext) (any, error) {
	if p.Config() == nil || !p.Config().Enabled {
		return nil, fmt.Errorf("example runtime is not enabled")
	}
	return &exampleRuntime{workDir: appCtx.WorkDir()}, nil
}

func ExampleRuntimeProvider() {
	p := &exampleRuntimePlugin{
		BasePlugin: plugin.BasePlugin[*exampleRuntimeConfig]{
			PluginName:  "example",
			PluginDesc:  "example runtime plugin",
			EnableMode:  plugin.EnabledExplicitly,
			DefaultCfg:  func() *exampleRuntimeConfig { return &exampleRuntimeConfig{} },
			IsEnabledFn: func(cfg *exampleRuntimeConfig) bool { return cfg != nil && cfg.Enabled },
		},
	}
	p.SetTypedConfig(&exampleRuntimeConfig{Enabled: true})

	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		WorkDir:    "/repo",
		ServiceDir: "/repo/.terraci",
		Version:    "test",
	})
	rawRuntime, _ := p.Runtime(context.Background(), appCtx)
	runtime, _ := plugin.RuntimeAs[*exampleRuntime](rawRuntime)

	fmt.Println(runtime.workDir)
	// Output: /repo
}
