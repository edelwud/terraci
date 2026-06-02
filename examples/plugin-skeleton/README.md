# Plugin skeleton

Annotated reference plugin showing the two most common third-party plugin shapes in TerraCi:

| Pattern  | What it does                                                                  | Built-in references                                                                                  |
|----------|-------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------|
| Producer | Writes a typed CI report (`{producer}-report.json`) consumed by `summary`.    | [`plugins/cost`](../../plugins/cost), [`plugins/policy`](../../plugins/policy), [`plugins/tfupdate`](../../plugins/tfupdate) |
| Consumer | Reads other plugins' reports (and validates provenance against the workspace). | [`plugins/summary`](../../plugins/summary), [`plugins/localexec`](../../plugins/localexec)             |

Copy whichever file matches your use case; the rest exists so the example builds end-to-end.

## File layout

```
plugin.go    # init() + BasePlugin[*Config] registration; plugin.Validator runs at startup
commands.go  # plugin.CommandProvider — registers `terraci skeleton`
runtime.go   # immutable command dependencies
usecases.go  # Request -> Runtime -> Result orchestration + ci.PublishArtifacts
output.go    # writer-based command output
report.go    # producer pattern: ci.NewRenderedReport
consumer.go  # consumer pattern: ci.ReportStore + ci.DecodeRenderSection
plugin_test.go # plugintest + citest contracts for config, commands, reports, artifacts
```

## Build & run

```bash
# Build a TerraCi binary that includes this skeleton
xterraci build \
  --with github.com/edelwud/terraci/examples/plugin-skeleton=./examples/plugin-skeleton \
  --output ./build/terraci-skeleton

# Configure the plugin
cat > .terraci.yaml <<'YAML'
structure:
  pattern: "{service}/{environment}/{region}/{module}"

extensions:
  skeleton:
    enabled: true
    greeting: "Hi from skeleton!"
YAML

# Producer flow — writes .terraci/skeleton-report.json
./build/terraci-skeleton skeleton

# Consumer flow — reads other plugins' *-report.json files
./build/terraci-skeleton skeleton --consume
```

## What you should copy

### For a producer plugin (writes a render-ready report)

1. `plugin.go` — registration shell, `BasePlugin[*Config]`.
2. `commands.go` — register a CLI command (`CommandProvider`) and use `plugin.CommandPlugin[T]` / `plugin.RequireEnabled` in callbacks.
3. `runtime.go`, `usecases.go`, `output.go` — keep `cobra flags -> Request -> Runtime -> Result -> output`.
4. `report.go` — convert your result into constructor-built `ci.RenderBlock` / `ci.RenderValue` values, build an `ci.ArtifactRun`, then publish raw results plus report through `ci.PublishArtifacts`.
5. `plugin_test.go` — copy `plugintest`/`citest` contracts so config immutability, command binding, report validation, and artifact lifecycle stay covered.

Skip the `--consume` branch if you don't need to read other reports.

### For a consumer plugin (reads other reports)

1. `plugin.go` — same shell.
2. `consumer.go` — call `appCtx.Reports().LoadReports(ctx)`, select current artifacts with `ci.SelectCurrentReports`, filter by `report.Producer`, and decode render-ready sections with `ci.DecodeRenderSection`.

## Capability extension points

The skeleton implements only `CommandProvider` and `ConfigLoader` (via `BasePlugin`). Adding more capabilities is a matter of implementing the interface on the same `*Plugin` struct:

| Capability               | Interface                       | Built-in reference                |
|--------------------------|---------------------------------|-----------------------------------|
| Lifecycle preflight      | `plugin.Preflightable`          | `plugins/git/lifecycle.go`        |
| Lazy heavy runtime       | Plugin-local typed builder      | `plugins/cost/runtime.go`         |
| Pipeline contributions   | `plugin.PipelineContributor`    | `plugins/policy/pipeline.go`      |
| Init wizard fields       | `initwiz.InitContributor`       | `plugins/cost/init_wizard.go`     |
| CI provider              | `plugin.PipelineGeneratorFactory` (+ EnvDetector, CIInfoProvider, …) | `plugins/gitlab/generator.go` |
| Blob/KV cache backend    | `plugin.BlobStoreProvider` / `plugin.KVCacheProvider` | `plugins/diskblob/store.go`, `plugins/inmemcache/cache.go` |
| Change detection         | `plugin.ChangeDetectionProvider` over `workflow.ChangeDetector` | `plugins/git/detect.go`           |

Framework discovery is type-assertion-based inside TerraCi's registry, but plugin authors normally do not enumerate capabilities directly. Implement the interface on `*Plugin`; the framework exposes named registry views and resolver methods where it needs them. No manual registration of capabilities is needed.

## Anti-patterns to avoid

- **Don't** import another plugin directly. Cross-plugin communication goes through `pkg/plugin` capability interfaces, `pkg/ci` shared types, or `ci.ReportStore` artifacts.
- **Don't** capture plugin state at command-registration time. Resolve the command-scoped plugin inside `RunE` with `plugin.CommandPlugin[T]`.
- **Don't** panic while building reports in production code paths. Use `ci.NewRenderedReport` and propagate errors.
- **Don't** assemble report payload JSON or render structs by hand. Use `ci.NewTableBlock`, `ci.NewListBlock`, `ci.RenderStatus`, `ci.RenderMoney`, `ci.RenderModulePath`, and related constructors so presentation stays in the shared renderer.
- **Don't** assemble provenance by hand. Build a `ci.ArtifactRun` and pass `run.Artifact` to `ci.NewRenderedReport`; local consumers compare the fingerprint through `ci.SelectCurrentReports`.
- **Don't** mutate project config through `ctx.Config()`. It returns an immutable `config.Snapshot`; use snapshot accessors in production code.
- **Don't** mutate the value returned by `BasePlugin.Config()` expecting plugin state to change. Config types must implement `Clone() C`; `Config()` returns a defensive copy.

## See also

- [`pkg/plugin/doc.go`](../../pkg/plugin/doc.go) — lifecycle diagram and thread-safety contract
- [`docs/plugins/`](../../docs/plugins/) — narrative guides
- [`examples/external-plugin/`](../external-plugin/) — minimal command-only plugin
