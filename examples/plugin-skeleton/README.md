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
report.go    # producer pattern: ci.EncodeSection + ci.SaveResultsAndReport
consumer.go  # consumer pattern: ci.LoadReports + ci.DecodeSection[T]
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

### For a producer plugin (writes a typed report)

1. `plugin.go` — registration shell, `BasePlugin[*Config]`.
2. `report.go` — define `Kind`, payload struct, then call `ci.EncodeSection` + `ci.SaveResultsAndReport`.
3. `commands.go` — at minimum register a CLI command (`CommandProvider`).

Skip the `--consume` branch if you don't need to read other reports.

### For a consumer plugin (reads other reports)

1. `plugin.go` — same shell.
2. `consumer.go` — call `ci.LoadReports(serviceDir)`, filter by `report.Producer`, decode sections via `ci.DecodeSection[T]`.

## Capability extension points

The skeleton implements only `CommandProvider` and `ConfigLoader` (via `BasePlugin`). Adding more capabilities is a matter of implementing the interface on the same `*Plugin` struct:

| Capability               | Interface                       | Built-in reference                |
|--------------------------|---------------------------------|-----------------------------------|
| Lifecycle preflight      | `plugin.Preflightable`          | `plugins/git/lifecycle.go`        |
| Lazy heavy runtime       | `plugin.RuntimeProvider`        | `plugins/cost/runtime.go`         |
| Pipeline contributions   | `plugin.PipelineContributor`    | `plugins/policy/pipeline.go`      |
| Init wizard fields       | `initwiz.InitContributor`       | `plugins/cost/init_wizard.go`     |
| CI provider              | `plugin.PipelineGeneratorFactory` (+ EnvDetector, CIInfoProvider, …) | `plugins/gitlab/generator.go` |
| Blob/KV cache backend    | `plugin.BlobStoreProvider` / `plugin.KVCacheProvider` | `plugins/diskblob/store.go`, `plugins/inmemcache/cache.go` |
| Change detection         | `plugin.ChangeDetectionProvider` | `plugins/git/detect.go`           |

Framework discovery is purely type-assertion-based: `registry.ByCapabilityFrom[T]` walks the registered plugins and returns those that implement `T`. No manual registration of capabilities is needed.

## Anti-patterns to avoid

- **Don't** import another plugin directly. Cross-plugin communication goes through `pkg/plugin` capability interfaces, `pkg/ci` shared types, or file-based reports.
- **Don't** use `ci.MustEncodeSection` in production code paths — it lives in `pkg/ci/citest` for tests only. Use `ci.EncodeSection` and propagate errors.
- **Don't** skip `Provenance: ci.NewProvenance(...)` on persisted reports. Local consumers compare the fingerprint to detect stale artifacts.
- **Don't** mutate `ctx.Config()` (`*config.Config`) — it's a shared pointer behind an `RWMutex`. Treat it as read-only; mutate plugin-local config via `FlagOverridable` if needed.

## See also

- [`pkg/plugin/doc.go`](../../pkg/plugin/doc.go) — lifecycle diagram and thread-safety contract
- [`docs/plugins/`](../../docs/plugins/) — narrative guides
- [`examples/external-plugin/`](../external-plugin/) — minimal command-only plugin
