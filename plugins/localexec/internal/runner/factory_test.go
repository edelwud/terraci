package runner

import (
	"errors"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

type stubExecutionConfigResolver struct {
	cfg execution.Config
}

func (r stubExecutionConfigResolver) Resolve(*plugin.AppContext, Options) execution.Config {
	return r.cfg
}

type stubBinaryResolver struct {
	path string
	err  error
}

func (r stubBinaryResolver) Resolve(string) (string, error) {
	return r.path, r.err
}

func TestDefaultExecutionConfigResolver_UsesProjectParallelismWhenOverrideIsNonPositive(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Execution.Parallelism = 7
	base := plugintest.NewAppContext(t, t.TempDir())
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{Config: cfg, WorkDir: base.WorkDir(), ServiceDir: base.ServiceDir(), Version: base.Version(), Reports: base.Reports()})

	resolver := defaultExecutionConfigResolver{}

	tests := []struct {
		name        string
		parallelism int
	}{
		{name: "zero", parallelism: 0},
		{name: "negative", parallelism: -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolver.Resolve(appCtx, Options{Parallelism: tt.parallelism})
			if got.Parallelism != 7 {
				t.Fatalf("Resolve() parallelism = %d, want project default 7", got.Parallelism)
			}
		})
	}
}

func TestDefaultExecutionConfigResolver_UsesExplicitParallelismOverride(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Execution.Parallelism = 7
	base := plugintest.NewAppContext(t, t.TempDir())
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{Config: cfg, WorkDir: base.WorkDir(), ServiceDir: base.ServiceDir(), Version: base.Version(), Reports: base.Reports()})

	got := defaultExecutionConfigResolver{}.Resolve(appCtx, Options{Parallelism: 3})
	if got.Parallelism != 3 {
		t.Fatalf("Resolve() parallelism = %d, want 3", got.Parallelism)
	}
}

func TestDefaultFactoryBuildReturnsBinaryResolverError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("missing terraform")
	factory := defaultFactory{
		configResolver: stubExecutionConfigResolver{cfg: execution.Config{Binary: "terraform"}},
		binaryResolver: stubBinaryResolver{err: wantErr},
	}

	_, err := factory.Build(plugintest.NewAppContext(t, t.TempDir()), Options{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Build() error = %v, want %v", err, wantErr)
	}
}

func TestDefaultFactoryBuildWiresRuntime(t *testing.T) {
	t.Parallel()

	cfg := execution.Config{
		Binary:      "terraform",
		Parallelism: 4,
		Env:         map[string]string{"TF_IN_AUTOMATION": "1"},
	}
	factory := defaultFactory{
		configResolver: stubExecutionConfigResolver{cfg: cfg},
		binaryResolver: stubBinaryResolver{path: "/bin/terraform"},
	}

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	runtime, err := factory.Build(appCtx, Options{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("Build() runtime = nil")
	}
	if runtime.ExecConfig.Parallelism != cfg.Parallelism {
		t.Fatalf("parallelism = %d, want %d", runtime.ExecConfig.Parallelism, cfg.Parallelism)
	}
	if runtime.Workspace.WorkDir() != appCtx.WorkDir() {
		t.Fatalf("workspace work dir = %q, want %q", runtime.Workspace.WorkDir(), appCtx.WorkDir())
	}
	if runtime.JobRunner == nil {
		t.Fatal("job runner = nil")
	}
}
