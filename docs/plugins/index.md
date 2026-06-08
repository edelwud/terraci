---
title: Plugins
description: "Extend TerraCi with custom plugins: CLI commands, pipeline jobs, cost providers, policy engines, and more"
outline: deep
---

# Plugins

TerraCi is built as a plugin-first system. Every feature — pipeline generation, cost estimation, policy checks, MR comments — is a plugin. You can add your own plugins to integrate with any tool or service your team uses.

## What Can Plugins Do?

| Plugin Type | What It Adds | Example Use Cases |
|-------------|-------------|-------------------|
| [CLI Command](/plugins/command-plugin) | New `terraci <command>` subcommand | Slack notifications, custom reports, infra audits |
| [Pipeline Job](/plugins/pipeline-plugin) | Standalone DAG jobs added to generated CI pipelines | Security scans, compliance checks, deployment gates |
| [CI Provider](/plugins/provider-plugin) | Support for a new CI system (beyond GitLab/GitHub) | Bitbucket Pipelines, Jenkins, CircleCI |
| [Init Wizard Field](/plugins/init-plugin) | Configuration fields in `terraci init` TUI | Custom plugin settings, team-specific defaults |

## Quick Start

Build a custom TerraCi with your plugin in 3 steps:

```bash
# 1. Write your plugin (see guides below)

# 2. Build a custom binary
xterraci build --with github.com/your-org/terraci-plugin-slack

# 3. Use it
./terraci slack --channel #deploys
```

## Architecture

```
                    ┌──────────────────────────┐
                    │      TerraCi Core         │
                    │                           │
                    │  discovery → parser →     │
                    │  graph → pipeline IR      │
                    └──────────┬───────────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
        ┌─────┴─────┐   ┌─────┴─────┐   ┌─────┴─────┐
        │  Built-in  │   │  Built-in  │   │  Your     │
        │  Plugins   │   │  Plugins   │   │  Plugin   │
        │            │   │            │   │           │
        │  gitlab    │   │  cost      │   │  slack    │
        │  github    │   │  policy    │   │  jira     │
        │  git       │   │  summary   │   │  vault    │
        │            │   │  tfupdate  │   │  ...      │
        └────────────┘   └────────────┘   └───────────┘
```

Plugins are compiled into the binary. There is no runtime plugin loading — this means zero overhead and full type safety.

## Guides

<div class="cards">

### [CLI Command Plugin](/plugins/command-plugin)
Add a new `terraci <command>`. The most common plugin type — perfect for notifications, reports, integrations.

### [Pipeline Job Plugin](/plugins/pipeline-plugin)
Add standalone resource-aware jobs to generated CI pipelines. Use for security scans, approval gates, or post-deploy checks.

### [CI Provider Plugin](/plugins/provider-plugin)
Add support for a new CI system. Implement pipeline generation, environment detection, and MR/PR comments.

### [Init Wizard Plugin](/plugins/init-plugin)
Add configuration fields to the `terraci init` interactive wizard. Users configure your plugin through a TUI form.

</div>

## Built-in Plugins

| Plugin | Capabilities | Config |
|--------|-------------|--------|
| **git** | ChangeDetection, Preflight | Always active |
| **gitlab** | Generator, EnvDetector, Comments, Preflight, Init | [config/gitlab](/config/gitlab) |
| **github** | Generator, EnvDetector, Comments, Preflight, Init | [config/github](/config/github) |
| **summary** | Command, Pipeline, Init | Enabled by default |
| **cost** | Command, Pipeline, Runtime, Preflight, Init | [config/cost](/config/cost) |
| **policy** | Command, Pipeline, Runtime, Preflight, Version, Init | [config/policy](/config/policy) |
| **tfupdate** | Command, Pipeline, Runtime, Preflight, Init | [config/tfupdate](/config/tfupdate) |

## Plugin Basics

### Registration

Every plugin registers itself in `init()`:

```go
package myplugin

import (
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.RegisterFactory(func() plugin.Plugin {
        return &Plugin{
            BasePlugin: plugin.BasePlugin[*Config]{
                PluginName: "myplugin",
                PluginDesc: "What my plugin does",
                EnableMode: plugin.EnabledExplicitly,
                DefaultCfg: func() *Config { return &Config{} },
                IsEnabledFn: func(cfg *Config) bool {
                    return cfg != nil && cfg.Enabled
                },
            },
        }
    })
}

type Plugin struct {
    plugin.BasePlugin[*Config]
}

type Config struct {
    Enabled bool   `yaml:"enabled"`
}

func (c *Config) Clone() *Config {
    if c == nil {
        return nil
    }
    out := *c
    return &out
}
```

Users configure your plugin in `.terraci.yaml`:

```yaml
extensions:
  myplugin:
    enabled: true
```

### Activation Policies

| Policy | Behavior | Used by |
|--------|----------|---------|
| `EnabledAlways` | Always active, no config needed | git |
| `EnabledWhenConfigured` | Active when config section exists | gitlab, github |
| `EnabledByDefault` | Active unless `enabled: false` | summary, diskblob, inmemcache |
| `EnabledExplicitly` | Requires explicit opt-in | cost, policy, tfupdate |

### Lifecycle

```
Register → Configure → Preflight → Bind → Execute
```

### SDK contract tests

Use the public test kits for SDK behavior instead of re-testing framework internals by hand:

- `pkg/plugin/plugintest`: `AssertBaseConfigPlugin`, `AssertCommandBinding`, `AssertRequireEnabled`, `AssertRuntimeBuilder`, `AssertPipelineContributor`, `AssertPreflightable`, `AssertInitContributor`, `AssertVersionProvider`, `AssertKVCacheProvider`, `AssertBlobStoreProvider`, `AssertChangeDetector`, `AssertCIProvider`.
- `pkg/ci/citest`: `AssertRenderedReportContract`, `AssertPublishArtifactsContract`, and rendered report builders.

Keep plugin-specific tests focused on your domain logic, APIs, and rendering decisions. The contract helpers verify that your plugin follows the same config immutability, command binding, report, and artifact lifecycle rules as the built-in plugins.

1. **Register** — `init()` runs at import time
2. **Configure** — framework decodes `extensions.<key>` from YAML
3. **Preflight** — cheap validation (no network, no heavy state)
4. **Bind** — runflow builds immutable prepared command state and AppContext
5. **Execute** — commands lazily build runtime as needed

### AppContext

Every capability receives `*plugin.AppContext` with:

```go
ctx.WorkDir()    // project root directory
ctx.ServiceDir() // resolved .terraci directory (absolute path)
ctx.Config()     // immutable config.Config; use accessors such as ServiceDir()
ctx.Version()    // TerraCi version string
ctx.Reports()    // shared ci.ReportStore for plugin artifacts and reports
ctx.CIResolver()             // CI provider resolver — never nil
ctx.ChangeDetectorResolver() // change detector resolver — never nil
ctx.KVCacheResolver()        // KV cache backend resolver — never nil
ctx.BlobStoreResolver()      // blob backend resolver — never nil
```

The framework constructs the context once via `plugin.NewAppContext(plugin.AppContextOptions{...})` and binds it for the duration of a command run. Plugins receive a fully built context — no construction needed.

### Capability Resolvers

Use the narrow resolver accessor for the capability your plugin needs:

```go
ctx.CIResolver().ResolveCIProvider()
ctx.ChangeDetectorResolver().ResolveChangeDetector()
ctx.KVCacheResolver().ResolveKVCacheProvider(name)
ctx.BlobStoreResolver().ResolveBlobStoreProvider(name)
```

Resolver accessors are never nil — framework wiring binds a `plugin.ResolverSet`, and test contexts without real resolvers get no-op capability resolvers that return sentinel errors instead of nil dereferences. Framework lifecycle enumeration such as preflight and pipeline contribution collection is owned by the CLI runflow, not plugin code.

### Building

```bash
# From published module
xterraci build --with github.com/your-org/terraci-plugin-slack

# From local directory during development
xterraci build --with github.com/your-org/plugin=./my-plugin

# Exclude built-in plugins you don't need
xterraci build --without cost --without policy

# Pin specific version
xterraci build --with github.com/your-org/plugin@v1.2.0
```

## See Also

- [examples/external-plugin](https://github.com/edelwud/terraci/tree/main/examples/external-plugin) — working example
- [Plugin System Overview](/guide/plugins) — architecture deep dive
- [xterraci CLI](/cli/) — build custom binaries
