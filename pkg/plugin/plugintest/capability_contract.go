package plugintest

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/workflow"
)

// PreflightableContract describes the fixtures for a plugin.Preflightable
// capability check.
type PreflightableContract struct {
	Plugin      plugin.Preflightable
	AppContext  *plugin.AppContext
	Context     context.Context
	WantErr     bool
	AssertError func(testing.TB, error)
}

// AssertPreflightable verifies a preflight implementation through the public
// capability interface without asserting plugin-specific diagnostics.
func AssertPreflightable(tb testing.TB, c PreflightableContract) {
	tb.Helper()
	if c.Plugin == nil {
		tb.Fatal("Plugin is nil")
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}

	err := c.Plugin.Preflight(ctx, appCtx)
	if c.WantErr {
		if err == nil {
			tb.Fatal("Preflight() error = nil")
		}
		if c.AssertError != nil {
			c.AssertError(tb, err)
		}
		return
	}
	if err != nil {
		tb.Fatalf("Preflight() error = %v", err)
	}
}

// InitContributorContract describes generic init wizard contribution checks.
type InitContributorContract struct {
	Contributor        initwiz.InitContributor
	State              *initwiz.StateMap
	ExpectedPluginKey  string
	ExpectContribution bool
	AssertGroups       func(testing.TB, []*initwiz.InitGroupSpec)
	AssertContribution func(testing.TB, *initwiz.InitContribution)
	DecodeTarget       any
}

// AssertInitContributor verifies that init groups have stable, usable shape
// and that BuildInitConfig returns the expected plugin section contract.
func AssertInitContributor(tb testing.TB, c InitContributorContract) {
	tb.Helper()
	if c.Contributor == nil {
		tb.Fatal("Contributor is nil")
	}

	firstGroups := c.Contributor.InitGroups()
	secondGroups := c.Contributor.InitGroups()
	if !sameInitGroupShape(firstGroups, secondGroups) {
		tb.Fatalf("InitGroups() is not deterministic: first %#v, second %#v", initGroupShapes(firstGroups), initGroupShapes(secondGroups))
	}
	assertInitGroupsUsable(tb, firstGroups)
	if c.AssertGroups != nil {
		c.AssertGroups(tb, firstGroups)
	}

	contribution, err := c.Contributor.BuildInitConfig(c.State)
	if err != nil {
		tb.Fatalf("BuildInitConfig() error = %v", err)
	}
	if !c.ExpectContribution {
		if contribution != nil && c.AssertContribution == nil {
			tb.Fatalf("BuildInitConfig() = %#v, want nil", contribution)
		}
		if c.AssertContribution != nil {
			c.AssertContribution(tb, contribution)
		}
		return
	}
	assertInitContribution(tb, contribution, c)
}

func assertInitGroupsUsable(tb testing.TB, groups []*initwiz.InitGroupSpec) {
	tb.Helper()
	for _, group := range groups {
		if group == nil {
			tb.Fatal("InitGroups() contains nil group")
			continue
		}
		if group.Title == "" {
			tb.Fatalf("InitGroups() contains group with empty title: %#v", group)
		}
		for i := range group.Fields {
			field := &group.Fields[i]
			if field.Key() == "" {
				tb.Fatalf("init group %q contains field with empty key", group.Title)
			}
			if field.Title() == "" {
				tb.Fatalf("init group %q field %q has empty title", group.Title, field.Key())
			}
		}
	}
}

func assertInitContribution(tb testing.TB, contribution *initwiz.InitContribution, c InitContributorContract) {
	tb.Helper()
	if contribution == nil {
		tb.Fatal("BuildInitConfig() = nil")
	}
	if c.ExpectedPluginKey != "" && contribution.PluginKey() != c.ExpectedPluginKey {
		tb.Fatalf("BuildInitConfig().PluginKey() = %q, want %q", contribution.PluginKey(), c.ExpectedPluginKey)
	}
	if contribution.ExtensionValue().Key() == "" {
		tb.Fatal("BuildInitConfig().ExtensionValue().Key() is empty")
	}
	if c.DecodeTarget != nil {
		if err := contribution.DecodeConfig(c.DecodeTarget); err != nil {
			tb.Fatalf("BuildInitConfig().DecodeConfig() error = %v", err)
		}
	}
	if c.AssertContribution != nil {
		c.AssertContribution(tb, contribution)
	}
}

type initGroupShape struct {
	Title    string
	Category initwiz.InitCategory
	Order    int
	Fields   []initFieldShape
}

type initFieldShape struct {
	Key         string
	Title       string
	Description string
	Type        initwiz.FieldType
	Options     []initwiz.InitOption
	Placeholder string
}

func sameInitGroupShape(a, b []*initwiz.InitGroupSpec) bool {
	return slices.EqualFunc(initGroupShapes(a), initGroupShapes(b), func(x, y initGroupShape) bool {
		return x.Title == y.Title &&
			x.Category == y.Category &&
			x.Order == y.Order &&
			slices.EqualFunc(x.Fields, y.Fields, sameInitFieldShape)
	})
}

func sameInitFieldShape(a, b initFieldShape) bool {
	return a.Key == b.Key &&
		a.Title == b.Title &&
		a.Description == b.Description &&
		a.Type == b.Type &&
		slices.Equal(a.Options, b.Options) &&
		a.Placeholder == b.Placeholder
}

func initGroupShapes(groups []*initwiz.InitGroupSpec) []initGroupShape {
	shapes := make([]initGroupShape, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			shapes = append(shapes, initGroupShape{})
			continue
		}
		shape := initGroupShape{
			Title:    group.Title,
			Category: group.Category,
			Order:    group.Order,
			Fields:   make([]initFieldShape, 0, len(group.Fields)),
		}
		for i := range group.Fields {
			field := &group.Fields[i]
			shape.Fields = append(shape.Fields, initFieldShape{
				Key:         field.Key(),
				Title:       field.Title(),
				Description: field.Description(),
				Type:        field.Type(),
				Options:     field.Options(),
				Placeholder: field.Placeholder(),
			})
		}
		shapes = append(shapes, shape)
	}
	return shapes
}

// VersionProviderContract describes expected generic version information.
type VersionProviderContract struct {
	Provider     plugin.VersionProvider
	ExpectedKeys []string
}

// AssertVersionProvider verifies that version information is stable and does
// not leak a mutable map across calls.
func AssertVersionProvider(tb testing.TB, c VersionProviderContract) {
	tb.Helper()
	if c.Provider == nil {
		tb.Fatal("Provider is nil")
	}
	first := c.Provider.VersionInfo()
	if len(first) == 0 {
		tb.Fatal("VersionInfo() is empty")
	}
	for key, value := range first {
		if key == "" {
			tb.Fatalf("VersionInfo() contains empty key: %#v", first)
		}
		if value == "" {
			tb.Fatalf("VersionInfo()[%q] is empty", key)
		}
	}
	for _, key := range c.ExpectedKeys {
		if _, ok := first[key]; !ok {
			tb.Fatalf("VersionInfo() missing key %q in %#v", key, first)
		}
	}

	first["__mutated__"] = "mutation"
	second := c.Provider.VersionInfo()
	if _, ok := second["__mutated__"]; ok {
		tb.Fatal("VersionInfo() leaked mutable map state")
	}
}

// KVCacheProviderContract describes a minimal key/value cache round trip.
type KVCacheProviderContract struct {
	Provider   plugin.KVCacheProvider
	AppContext *plugin.AppContext
	Context    context.Context
	Namespace  string
	Key        string
	Value      []byte
	TTL        time.Duration
}

// AssertKVCacheProvider verifies the public KV cache provider lifecycle.
func AssertKVCacheProvider(tb testing.TB, c KVCacheProviderContract) {
	tb.Helper()
	if c.Provider == nil {
		tb.Fatal("Provider is nil")
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}
	namespace := defaultString(c.Namespace, "plugintest")
	key := defaultString(c.Key, "key")
	value := c.Value
	if value == nil {
		value = []byte("value")
	}

	cache, err := c.Provider.NewKVCache(ctx, appCtx)
	if err != nil {
		tb.Fatalf("NewKVCache() error = %v", err)
	}
	if cache == nil {
		tb.Fatal("NewKVCache() = nil")
	}
	if setErr := cache.Set(ctx, namespace, key, value, c.TTL); setErr != nil {
		tb.Fatalf("KVCache.Set() error = %v", setErr)
	}
	got, ok, err := cache.Get(ctx, namespace, key)
	if err != nil {
		tb.Fatalf("KVCache.Get() error = %v", err)
	}
	if !ok {
		tb.Fatal("KVCache.Get() ok = false")
	}
	if !bytes.Equal(got, value) {
		tb.Fatalf("KVCache.Get() = %q, want %q", got, value)
	}
	if err := cache.Delete(ctx, namespace, key); err != nil {
		tb.Fatalf("KVCache.Delete() error = %v", err)
	}
	if _, ok, err := cache.Get(ctx, namespace, key); err != nil {
		tb.Fatalf("KVCache.Get(after delete) error = %v", err)
	} else if ok {
		tb.Fatal("KVCache.Get(after delete) ok = true")
	}
	if err := cache.DeleteNamespace(ctx, namespace); err != nil {
		tb.Fatalf("KVCache.DeleteNamespace() error = %v", err)
	}
}

// BlobStoreProviderContract describes a minimal blob store provider round trip.
type BlobStoreProviderContract struct {
	Provider   plugin.BlobStoreProvider
	AppContext *plugin.AppContext
	Context    context.Context
	Options    plugin.BlobStoreOptions
	Namespace  string
	Key        string
	Value      []byte
}

// AssertBlobStoreProvider verifies the public blob store provider lifecycle.
func AssertBlobStoreProvider(tb testing.TB, c BlobStoreProviderContract) {
	tb.Helper()
	if c.Provider == nil {
		tb.Fatal("Provider is nil")
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}
	namespace := defaultString(c.Namespace, "plugintest")
	key := defaultString(c.Key, "blob")
	value := c.Value
	if value == nil {
		value = []byte("blob-value")
	}

	store, err := c.Provider.NewBlobStore(ctx, appCtx, c.Options)
	if err != nil {
		tb.Fatalf("NewBlobStore() error = %v", err)
	}
	if store == nil {
		tb.Fatal("NewBlobStore() = nil")
	}
	if _, putErr := store.Put(ctx, namespace, key, value, blobcache.PutOptions{}); putErr != nil {
		tb.Fatalf("BlobStore.Put() error = %v", putErr)
	}
	got, ok, _, err := store.Get(ctx, namespace, key)
	if err != nil {
		tb.Fatalf("BlobStore.Get() error = %v", err)
	}
	if !ok {
		tb.Fatal("BlobStore.Get() ok = false")
	}
	if !bytes.Equal(got, value) {
		tb.Fatalf("BlobStore.Get() = %q, want %q", got, value)
	}
	if err := store.Delete(ctx, namespace, key); err != nil {
		tb.Fatalf("BlobStore.Delete() error = %v", err)
	}
	if _, ok, _, err := store.Get(ctx, namespace, key); err != nil {
		tb.Fatalf("BlobStore.Get(after delete) error = %v", err)
	} else if ok {
		tb.Fatal("BlobStore.Get(after delete) ok = true")
	}
	if err := store.DeleteNamespace(ctx, namespace); err != nil {
		tb.Fatalf("BlobStore.DeleteNamespace() error = %v", err)
	}
}

// ChangeDetectorContract describes a generic workflow change detector call.
type ChangeDetectorContract struct {
	Detector     workflow.ChangeDetector
	Request      workflow.ChangeDetectionRequest
	WantErr      bool
	AssertResult func(testing.TB, *workflow.ChangeDetectionResult)
}

// AssertChangeDetector verifies the workflow-level change detector boundary.
func AssertChangeDetector(tb testing.TB, c ChangeDetectorContract) {
	tb.Helper()
	if c.Detector == nil {
		tb.Fatal("Detector is nil")
	}
	result, err := c.Detector.DetectChanges(context.Background(), c.Request)
	if c.WantErr {
		if err == nil {
			tb.Fatal("DetectChanges() error = nil")
		}
		return
	}
	if err != nil {
		tb.Fatalf("DetectChanges() error = %v", err)
	}
	if result == nil {
		tb.Fatal("DetectChanges() = nil")
	}
	if c.AssertResult != nil {
		c.AssertResult(tb, result)
	}
}

// CIProviderContract describes generic CI provider capability checks. The
// fields are explicit so test code can verify focused interfaces separately
// when a provider composes them from different objects.
type CIProviderContract struct {
	EnvDetector     plugin.EnvDetector
	InfoProvider    plugin.CIInfoProvider
	Generator       plugin.PipelineGeneratorFactory
	CommentFactory  plugin.CommentServiceFactory
	AppContext      *plugin.AppContext
	IR              *pipeline.IR
	ExpectedName    string
	AssertEnv       func(testing.TB, bool)
	AssertInfo      func(testing.TB, plugin.CIInfoProvider)
	AssertGenerator func(testing.TB, pipeline.Generator)
	AssertComment   func(testing.TB, ci.CommentService, bool)
}

// AssertCIProvider verifies the focused CI provider contracts without knowing
// provider-specific YAML or API details.
func AssertCIProvider(tb testing.TB, c CIProviderContract) {
	tb.Helper()
	appCtx := c.AppContext
	if appCtx == nil {
		appCtx = plugin.NewAppContext(plugin.AppContextOptions{})
	}

	if c.EnvDetector != nil && c.AssertEnv != nil {
		c.AssertEnv(tb, c.EnvDetector.DetectEnv())
	}

	if c.InfoProvider == nil {
		tb.Fatal("InfoProvider is nil")
	}
	if c.ExpectedName != "" && c.InfoProvider.ProviderName() != c.ExpectedName {
		tb.Fatalf("ProviderName() = %q, want %q", c.InfoProvider.ProviderName(), c.ExpectedName)
	}
	if c.InfoProvider.ProviderName() == "" {
		tb.Fatal("ProviderName() is empty")
	}
	if c.AssertInfo != nil {
		c.AssertInfo(tb, c.InfoProvider)
	}

	if c.Generator != nil && c.IR != nil {
		assertCIGenerator(tb, c, appCtx)
	}

	if c.CommentFactory != nil || c.AssertComment != nil {
		var service ci.CommentService
		ok := false
		if c.CommentFactory != nil {
			service = c.CommentFactory.NewCommentService(appCtx)
			ok = service != nil
		}
		if c.AssertComment != nil {
			c.AssertComment(tb, service, ok)
		}
	}
}

func assertCIGenerator(tb testing.TB, c CIProviderContract, appCtx *plugin.AppContext) {
	tb.Helper()
	generator, err := c.Generator.NewGenerator(appCtx, c.IR)
	if err != nil {
		tb.Fatalf("NewGenerator() error = %v", err)
	}
	if generator == nil {
		tb.Fatal("NewGenerator() = nil")
	}
	if c.AssertGenerator != nil {
		c.AssertGenerator(tb, generator)
	}
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
