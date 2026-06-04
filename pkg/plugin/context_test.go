package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestAppContext_Accessors(t *testing.T) {
	cfg := config.DefaultConfig()
	r := ci.NewMemoryReportStore()
	ctx := NewAppContext(AppContextOptions{
		Config:     cfg,
		WorkDir:    "/tmp",
		ServiceDir: "/tmp/.terraci",
		Version:    "1.0",
		Reports:    r,
	})

	if !ctx.Config().Present() {
		t.Error("Config() should return a present configuration snapshot")
	}
	if ctx.Config().ServiceDir() != cfg.ServiceDir {
		t.Errorf("Config().ServiceDir() = %q, want %q", ctx.Config().ServiceDir(), cfg.ServiceDir)
	}
	if ctx.WorkDir() != "/tmp" {
		t.Errorf("WorkDir() = %q, want /tmp", ctx.WorkDir())
	}
	if ctx.ServiceDir() != "/tmp/.terraci" {
		t.Errorf("ServiceDir() = %q, want /tmp/.terraci", ctx.ServiceDir())
	}
	if ctx.Version() != "1.0" {
		t.Errorf("Version() = %q, want 1.0", ctx.Version())
	}
	if ctx.Reports() != r {
		t.Error("Reports() should be the same registry instance")
	}
	if ctx.CIResolver() == nil || ctx.ChangeDetectorResolver() == nil || ctx.KVCacheResolver() == nil || ctx.BlobStoreResolver() == nil {
		t.Error("resolver accessors should never return nil")
	}
}

func TestAppContext_NoResolverFallsBackToNoop(t *testing.T) {
	ctx := NewAppContext(AppContextOptions{})
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "ci",
			err:  errFromCallSite(ctx.CIResolver().ResolveCIProvider),
		},
		{
			name: "change detector",
			err:  errFromCallSite(ctx.ChangeDetectorResolver().ResolveChangeDetector),
		},
		{
			name: "kv cache",
			err:  errFromCallSite(func() (KVCacheProvider, error) { return ctx.KVCacheResolver().ResolveKVCacheProvider("cache") }),
		},
		{
			name: "blob store",
			err:  errFromCallSite(func() (BlobStoreProvider, error) { return ctx.BlobStoreResolver().ResolveBlobStoreProvider("blob") }),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, ErrNoResolver) {
				t.Fatalf("resolver error = %v, want ErrNoResolver", tt.err)
			}
		})
	}
}

func errFromCallSite[T any](fn func() (T, error)) error {
	_, err := fn()
	return err
}

var errContextTestResolver = errors.New("context test resolver")

type contextCIResolver struct{}

func (contextCIResolver) ResolveCIProvider() (*ResolvedCIProvider, error) {
	return nil, errContextTestResolver
}

func TestResolverSet_UsesProvidedNarrowResolvers(t *testing.T) {
	set := NewResolverSet(ResolverSetOptions{CI: contextCIResolver{}})
	if err := errFromCallSite(set.CIResolver().ResolveCIProvider); !errors.Is(err, errContextTestResolver) {
		t.Fatalf("CIResolver() error = %v, want context resolver error", err)
	}
	if err := errFromCallSite(set.ChangeDetectorResolver().ResolveChangeDetector); !errors.Is(err, ErrNoResolver) {
		t.Fatalf("ChangeDetectorResolver() error = %v, want ErrNoResolver", err)
	}
}

func TestResolverSet_ZeroValueIsNoop(t *testing.T) {
	var set ResolverSet
	if err := errFromCallSite(set.CIResolver().ResolveCIProvider); !errors.Is(err, ErrNoResolver) {
		t.Fatalf("zero ResolverSet CI error = %v, want ErrNoResolver", err)
	}
	if err := errFromCallSite(NoopResolverSet().CIResolver().ResolveCIProvider); !errors.Is(err, ErrNoResolver) {
		t.Fatalf("NoopResolverSet CI error = %v, want ErrNoResolver", err)
	}
}

func TestAppContext_NoReportsCreatesDefaultStore(t *testing.T) {
	ctx := NewAppContext(AppContextOptions{})
	if ctx.Reports() == nil {
		t.Fatal("Reports() should not be nil for empty options")
	}
}

func TestCommandContext_PipelineContributionsAreSnapshot(t *testing.T) {
	job, err := pipeline.NewContributedJob(pipeline.ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"terraci summary"},
	})
	if err != nil {
		t.Fatalf("NewContributedJob() error = %v", err)
	}
	contrib, err := pipeline.NewContribution(job)
	if err != nil {
		t.Fatalf("NewContribution() error = %v", err)
	}
	appCtx := NewAppContext(AppContextOptions{})
	binding := mustCommandBindingWithContributions(t, appCtx, contextTestSource{plugin: &contextTestPlugin{name: "cmd"}}, []*pipeline.Contribution{contrib})

	cmdCtx := binding.CommandContext()
	if cmdCtx.AppContext() != appCtx {
		t.Fatalf("CommandContext.AppContext() = %p, want %p", cmdCtx.AppContext(), appCtx)
	}
	got := cmdCtx.PipelineContributions()
	if len(got) != 1 || got[0] == contrib {
		t.Fatalf("CommandContext.PipelineContributions() = %#v, want defensive contribution copy", got)
	}
	if jobs := got[0].Jobs(); len(jobs) != 1 || jobs[0].Name() != "summary" {
		t.Fatalf("CommandContext.PipelineContributions()[0].Jobs() = %#v, want summary", jobs)
	}
	got[0] = nil
	if again := cmdCtx.PipelineContributions(); len(again) != 1 || again[0] == nil || again[0] == contrib {
		t.Fatalf("CommandContext.PipelineContributions() was mutated through returned slice: %#v", again)
	}
}

func TestCommandBinding_ClonesInputContributions(t *testing.T) {
	job, err := pipeline.NewContributedJob(pipeline.ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"terraci summary"},
	})
	if err != nil {
		t.Fatalf("NewContributedJob() error = %v", err)
	}
	contrib, err := pipeline.NewContribution(job)
	if err != nil {
		t.Fatalf("NewContribution() error = %v", err)
	}
	input := []*pipeline.Contribution{contrib}
	binding := mustCommandBindingWithContributions(t, NewAppContext(AppContextOptions{}), contextTestSource{plugin: &contextTestPlugin{name: "cmd"}}, input)
	input[0] = nil

	got := binding.CommandContext().PipelineContributions()
	if len(got) != 1 || got[0] == nil || got[0] == contrib {
		t.Fatalf("bound PipelineContributions() = %#v, want defensive copy independent of input slice", got)
	}
}

func TestCommandPlugin_ReturnsContributions(t *testing.T) {
	job, err := pipeline.NewContributedJob(pipeline.ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"terraci summary"},
	})
	if err != nil {
		t.Fatalf("NewContributedJob() error = %v", err)
	}
	contrib, err := pipeline.NewContribution(job)
	if err != nil {
		t.Fatalf("NewContribution() error = %v", err)
	}
	target := &contextTestPlugin{name: "cmd", enabled: true}
	appCtx := NewAppContext(AppContextOptions{})
	cmd := &cobra.Command{}
	cmd.SetContext(BindCommandContext(context.Background(), mustCommandBindingWithContributions(t, appCtx, contextTestSource{plugin: target}, []*pipeline.Contribution{contrib})))

	cmdCtx, gotPlugin, err := CommandPlugin[*contextTestPlugin](cmd, "cmd")
	if err != nil {
		t.Fatalf("CommandPlugin() error = %v", err)
	}
	if cmdCtx.AppContext() != appCtx {
		t.Fatalf("CommandPlugin() AppContext = %p, want %p", cmdCtx.AppContext(), appCtx)
	}
	if gotPlugin != target {
		t.Fatalf("CommandPlugin() plugin = %p, want %p", gotPlugin, target)
	}
	if got := cmdCtx.PipelineContributions(); len(got) != 1 || got[0] == contrib {
		t.Fatalf("CommandPlugin().PipelineContributions() = %#v, want defensive copy", got)
	}
}

func TestCommandContext_ZeroValue(t *testing.T) {
	var ctx CommandContext
	if ctx.AppContext() != nil {
		t.Fatal("zero CommandContext AppContext should be nil")
	}
	if got := ctx.PipelineContributions(); got != nil {
		t.Fatalf("zero CommandContext PipelineContributions() = %#v, want nil", got)
	}
}

func TestAppContext_DoesNotCarryPipelineContributions(t *testing.T) {
	ctx := NewAppContext(AppContextOptions{})
	if ctx == nil {
		t.Fatal("NewAppContext() returned nil")
	}
}

func TestAppContext_ConfigIsImmutableSnapshot(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ServiceDir = ".terraci"
	cfg.Exclude = []string{"old"}
	ctx := NewAppContext(AppContextOptions{Config: cfg})

	cfg.ServiceDir = ".changed"
	cfg.Exclude[0] = "changed"
	if got := ctx.Config().ServiceDir(); got != ".terraci" {
		t.Fatalf("Config().ServiceDir() = %q, want original snapshot", got)
	}
	if got := ctx.Config().Exclude(); len(got) != 1 || got[0] != "old" {
		t.Fatalf("Config().Exclude() = %#v, want original snapshot", got)
	}

	mutable := ctx.Config().MutableCopy()
	mutable.ServiceDir = ".copy"
	mutable.Exclude[0] = "copy"
	if got := ctx.Config().ServiceDir(); got != ".terraci" {
		t.Fatalf("mutable copy changed snapshot ServiceDir to %q", got)
	}
	if got := ctx.Config().Exclude(); len(got) != 1 || got[0] != "old" {
		t.Fatalf("mutable copy changed snapshot Exclude to %#v", got)
	}
}

type contextTestPlugin struct {
	name    string
	enabled bool
}

func (p *contextTestPlugin) Name() string        { return p.name }
func (p *contextTestPlugin) Description() string { return p.name }
func (p *contextTestPlugin) IsEnabled() bool     { return p.enabled }

type contextTestSource struct {
	plugin Plugin
}

func (s contextTestSource) LookupCommandPlugin(string) (Plugin, bool) {
	return s.plugin, s.plugin != nil
}

func TestNewCommandBinding_ValidatesInputs(t *testing.T) {
	target := &contextTestPlugin{name: "cmd"}
	appCtx := NewAppContext(AppContextOptions{})
	if binding, err := NewCommandBinding(CommandBindingOptions{AppContext: appCtx, Source: contextTestSource{plugin: target}}); err != nil {
		t.Fatalf("NewCommandBinding() error = %v", err)
	} else if binding.AppContext() != appCtx {
		t.Fatalf("CommandBinding.AppContext() = %p, want %p", binding.AppContext(), appCtx)
	}
	if _, err := NewCommandBinding(CommandBindingOptions{Source: contextTestSource{plugin: target}}); err == nil {
		t.Fatal("NewCommandBinding(missing context) error = nil")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingContext)
	}
	if _, err := NewCommandBinding(CommandBindingOptions{AppContext: appCtx}); err == nil {
		t.Fatal("NewCommandBinding(missing source) error = nil")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingLookup)
	}
}

func TestCommandPluginLookup_LooksUpFromBindingSource(t *testing.T) {
	target := &contextTestPlugin{name: "cmd"}
	binding := mustCommandBinding(t, NewAppContext(AppContextOptions{}), contextTestSource{plugin: target})

	got, err := commandInstance[*contextTestPlugin](binding, "cmd")
	if err != nil {
		t.Fatalf("commandInstance() error = %v", err)
	}
	if got != target {
		t.Fatalf("commandInstance() = %p, want %p", got, target)
	}
}

func TestCommandPluginLookup_DoesNotUseResolverAsLookup(t *testing.T) {
	target := &contextTestPlugin{name: "cmd"}
	appCtx := NewAppContext(AppContextOptions{
		Resolvers: NewResolverSet(ResolverSetOptions{CI: contextCIResolver{}}),
	})

	cmd := &cobra.Command{}
	cmd.SetContext(BindCommandContext(context.Background(), mustCommandBinding(t, appCtx, contextTestSource{})))
	if _, _, err := CommandPlugin[*contextTestPlugin](cmd, target.Name()); err == nil {
		t.Fatal("CommandPlugin() error = nil, want not found without explicit command source")
	} else {
		assertCommandBindingReason(t, err, CommandBindingNotFound)
	}
}

func TestCommandPluginLookup_RejectsNilContext(t *testing.T) {
	if _, err := commandInstance[*contextTestPlugin](nil, "cmd"); err == nil {
		t.Fatal("commandInstance(nil) error = nil, want missing context error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingContext)
	}
}

func TestCommandPluginLookup_RejectsMissingCommandLookup(t *testing.T) {
	binding := &CommandBinding{commandCtx: CommandContext{appCtx: NewAppContext(AppContextOptions{})}}
	if _, err := commandInstance[*contextTestPlugin](binding, "cmd"); err == nil {
		t.Fatal("commandInstance() error = nil, want missing command lookup error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingLookup)
	}
}

func TestCommandPlugin_ReturnsContextAndPlugin(t *testing.T) {
	target := &contextTestPlugin{name: "cmd", enabled: true}
	appCtx := NewAppContext(AppContextOptions{})
	cmd := &cobra.Command{}
	cmd.SetContext(BindCommandContext(context.Background(), mustCommandBinding(t, appCtx, contextTestSource{plugin: target})))

	gotCtx, gotPlugin, err := CommandPlugin[*contextTestPlugin](cmd, "cmd")
	if err != nil {
		t.Fatalf("CommandPlugin() error = %v", err)
	}
	if gotCtx.AppContext() != appCtx {
		t.Fatalf("CommandPlugin() ctx AppContext = %p, want %p", gotCtx.AppContext(), appCtx)
	}
	if gotPlugin != target {
		t.Fatalf("CommandPlugin() plugin = %p, want %p", gotPlugin, target)
	}
}

func TestCommandPlugin_RejectsNilCommand(t *testing.T) {
	if _, _, err := CommandPlugin[*contextTestPlugin](nil, "cmd"); err == nil {
		t.Fatal("CommandPlugin(nil) error = nil, want error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingNilCommand)
	}
}

func TestCommandPlugin_RejectsMissingAppContext(t *testing.T) {
	cmd := &cobra.Command{}
	if _, _, err := CommandPlugin[*contextTestPlugin](cmd, "cmd"); err == nil {
		t.Fatal("CommandPlugin() error = nil, want missing app context error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingContext)
	}
}

func TestCommandPlugin_RejectsMissingCommandLookup(t *testing.T) {
	if _, err := NewCommandBinding(CommandBindingOptions{AppContext: NewAppContext(AppContextOptions{})}); err == nil {
		t.Fatal("NewCommandBinding() error = nil, want missing command lookup error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingMissingLookup)
	}
}

func TestCommandPlugin_RejectsWrongPluginType(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{})
	cmd := &cobra.Command{}
	cmd.SetContext(BindCommandContext(context.Background(), mustCommandBinding(t, appCtx, contextTestSource{plugin: &contextTestPlugin{name: "cmd"}})))

	if _, _, err := CommandPlugin[*otherContextTestPlugin](cmd, "cmd"); err == nil {
		t.Fatal("CommandPlugin() error = nil, want wrong type error")
	} else {
		assertCommandBindingReason(t, err, CommandBindingWrongType)
	}
}

func TestRequireEnabled(t *testing.T) {
	if err := RequireEnabled(&contextTestPlugin{enabled: true}, "disabled"); err != nil {
		t.Fatalf("RequireEnabled(enabled) error = %v", err)
	}
	if err := RequireEnabled(&contextTestPlugin{}, "disabled"); err == nil || err.Error() != "disabled" {
		t.Fatalf("RequireEnabled(disabled) error = %v, want disabled", err)
	} else {
		var disabled *DisabledPluginError
		if !errors.As(err, &disabled) {
			t.Fatalf("RequireEnabled(disabled) error type = %T, want DisabledPluginError", err)
		}
	}
	if err := RequireEnabled(nil, "missing"); err == nil || err.Error() != "missing" {
		t.Fatalf("RequireEnabled(nil) error = %v, want missing", err)
	}
}

type otherContextTestPlugin struct{}

func (p *otherContextTestPlugin) Name() string        { return "other" }
func (p *otherContextTestPlugin) Description() string { return "other" }

func TestCommandContextBinding_RoundTrips(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{Version: "v"})
	binding := mustCommandBinding(t, appCtx, contextTestSource{plugin: &contextTestPlugin{name: "cmd"}})
	carrier := BindCommandContext(context.Background(), binding)
	got := commandBindingFromContext(carrier)
	if got != binding {
		t.Fatalf("commandBindingFromContext() = %p, want %p", got, binding)
	}
	if commandBindingFromContext(context.Background()) != nil {
		t.Fatal("commandBindingFromContext on empty context should be nil")
	}
	if commandBindingFromContext(context.TODO()) != nil {
		// context.TODO is the canonical "no-value" Context; fromContext
		// must still return nil for a context that has no AppContext key
		// attached.
		t.Fatal("commandBindingFromContext(empty context) should be nil")
	}
}

func TestCommandBindingFromCommand(t *testing.T) {
	appCtx := NewAppContext(AppContextOptions{Version: "v"})
	target := &contextTestPlugin{name: "cmd"}
	cmd := &cobra.Command{}
	cmd.SetContext(BindCommandContext(context.Background(), mustCommandBinding(t, appCtx, contextTestSource{plugin: target})))

	got, plugin, err := CommandPlugin[*contextTestPlugin](cmd, "cmd")
	if err != nil {
		t.Fatalf("CommandPlugin() error = %v", err)
	}
	if got.AppContext() != appCtx {
		t.Fatalf("CommandPlugin() ctx AppContext = %p, want %p", got.AppContext(), appCtx)
	}
	if plugin != target {
		t.Fatalf("CommandPlugin() plugin = %p, want %p", plugin, target)
	}
}

func mustCommandBinding(t *testing.T, appCtx *AppContext, source CommandBindingSource) *CommandBinding {
	t.Helper()
	binding, err := NewCommandBinding(CommandBindingOptions{AppContext: appCtx, Source: source})
	if err != nil {
		t.Fatalf("NewCommandBinding() error = %v", err)
	}
	return binding
}

func mustCommandBindingWithContributions(t *testing.T, appCtx *AppContext, source CommandBindingSource, contributions []*pipeline.Contribution) *CommandBinding {
	t.Helper()
	binding, err := NewCommandBinding(CommandBindingOptions{
		AppContext:            appCtx,
		Source:                source,
		PipelineContributions: contributions,
	})
	if err != nil {
		t.Fatalf("NewCommandBinding() error = %v", err)
	}
	return binding
}

func assertCommandBindingReason(t *testing.T, err error, reason CommandBindingReason) {
	t.Helper()

	var bindingErr *CommandBindingError
	if !errors.As(err, &bindingErr) {
		t.Fatalf("error type = %T, want CommandBindingError", err)
	}
	if bindingErr.Reason != reason {
		t.Fatalf("CommandBindingError.Reason = %q, want %q", bindingErr.Reason, reason)
	}
}
