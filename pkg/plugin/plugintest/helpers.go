package plugintest

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	rawlog "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// ansiEscapeRE matches CSI escape sequences emitted by caarlos0/log when
// CLICOLOR_FORCE is on (notably under CI=1, where the logger force-enables
// ANSI). Stripping them keeps strings.Contains assertions deterministic
// across CI and developer machines without leaking environment overrides
// into the rest of the test process.
var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func CaptureLogOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldLogger := rawlog.Log
	var buf bytes.Buffer
	rawlog.Log = rawlog.New(&buf)
	defer func() { rawlog.Log = oldLogger }()
	fn()
	return ansiEscapeRE.ReplaceAllString(buf.String(), "")
}

func LoadJSONFile[T any](t *testing.T, dir, filename string) T {
	t.Helper()

	var value T
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("failed to read %s: %v", filename, err)
	}
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("failed to parse %s: %v", filename, err)
	}
	return value
}

func LoadReport(t *testing.T, serviceDir, filename string) ci.Report {
	t.Helper()
	return LoadJSONFile[ci.Report](t, serviceDir, filename)
}

func LoadPluginReport(t *testing.T, serviceDir, pluginName string) ci.Report {
	t.Helper()
	return LoadReport(t, serviceDir, ci.ReportFilename(pluginName))
}

func NewAppContext(t *testing.T, workDir string) *plugin.AppContext {
	t.Helper()

	return NewAppContextWithResolvers(t, workDir, RegistryResolverSet(registry.New()))
}

func NewRegistry(t *testing.T, factories ...registry.Factory) *registry.Registry {
	t.Helper()

	return registry.NewFromFactories(factories...)
}

func RegistryResolverSet(plugins *registry.Registry) plugin.ResolverSet {
	return plugin.NewResolverSet(plugin.ResolverSetOptions{
		CI:             plugins,
		ChangeDetector: plugins,
		KVCache:        plugins,
		BlobStore:      plugins,
	})
}

func NewAppContextWithResolvers(t *testing.T, workDir string, resolvers plugin.ResolverSet) *plugin.AppContext {
	t.Helper()

	serviceDir := filepath.Join(t.TempDir(), ".terraci")
	cfg, err := config.Build(config.BuildOptions{ServiceDir: ".terraci"})
	if err != nil {
		t.Fatalf("config.Build() error = %v", err)
	}
	return plugin.NewAppContext(plugin.AppContextOptions{
		Config:     cfg,
		WorkDir:    workDir,
		ServiceDir: serviceDir,
		Version:    "test",
		Reports:    ci.NewFileReportStore(serviceDir),
		Resolvers:  resolvers,
	})
}

func MustRuntimeFromBuilder[T any](
	t *testing.T,
	build func(context.Context, *plugin.AppContext) (T, error),
	appCtx *plugin.AppContext,
) T {
	t.Helper()

	runtime, err := build(context.Background(), appCtx)
	if err != nil {
		t.Fatalf("runtime builder error = %v", err)
	}

	return runtime
}
