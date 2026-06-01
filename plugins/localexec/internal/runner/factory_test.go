package runner

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/terraformrun"
)

type stubExecutionConfigResolver struct {
	profile terraformrun.Profile
	err     error
}

func (r stubExecutionConfigResolver) Resolve(*plugin.AppContext, Options) (terraformrun.Profile, error) {
	return r.profile, r.err
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

			got, err := resolver.Resolve(appCtx, Options{Parallelism: tt.parallelism})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if got.Parallelism() != 7 {
				t.Fatalf("Resolve() parallelism = %d, want project default 7", got.Parallelism())
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

	got, err := defaultExecutionConfigResolver{}.Resolve(appCtx, Options{Parallelism: 3})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Parallelism() != 3 {
		t.Fatalf("Resolve() parallelism = %d, want 3", got.Parallelism())
	}
}

func TestDefaultFactoryBuildWiresRuntime(t *testing.T) {
	t.Parallel()

	profile := mustProfile(t, terraformrun.ProfileOptions{
		Parallelism: 4,
		Env:         map[string]string{"TF_IN_AUTOMATION": "1"},
	})
	factory := defaultFactory{
		configResolver: stubExecutionConfigResolver{profile: profile},
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
	if runtime.Profile.Parallelism() != profile.Parallelism() {
		t.Fatalf("parallelism = %d, want %d", runtime.Profile.Parallelism(), profile.Parallelism())
	}
	if runtime.Workspace.WorkDir() != appCtx.WorkDir() {
		t.Fatalf("workspace work dir = %q, want %q", runtime.Workspace.WorkDir(), appCtx.WorkDir())
	}
	if runtime.JobRunner == nil {
		t.Fatal("job runner = nil")
	}
}

func mustProfile(tb testing.TB, opts terraformrun.ProfileOptions) terraformrun.Profile {
	tb.Helper()
	profile, err := terraformrun.NewProfile(opts)
	if err != nil {
		tb.Fatalf("NewProfile() error = %v", err)
	}
	return profile
}
