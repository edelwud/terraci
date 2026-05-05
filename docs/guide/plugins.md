---
title: Plugin System
description: "Built-in and custom plugins: activation, configuration, and extensibility via xterraci"
outline: deep
---

# Plugin System

TerraCi uses a compile-time plugin system inspired by the `database/sql` driver pattern in Go. Plugins register themselves via `init()` and blank imports at build time.

## Built-in Plugins

| Plugin | Purpose | Config Required |
|--------|---------|----------------|
| `git` | Changed-module detection via `git diff` | No |
| `gitlab` | GitLab CI pipeline generation and MR comments | Yes (presence activates) |
| `github` | GitHub Actions workflow generation and PR comments | Yes (presence activates) |
| `summary` | MR/PR plan summary comments | No (enabled by default) |
| `cost` | Cloud cost estimation (AWS) | Yes (`providers.aws.enabled: true`) |
| `policy` | OPA policy checks | Yes (`enabled: true`) |
| `tfupdate` | Terraform dependency resolver and lock synchronizer | Yes (`enabled: true`) |

## Activation Policies

Each plugin has an activation policy that determines when it participates in the current run.

### Always Active

The **git** plugin requires no configuration. It provides changed-module detection for `--changed-only` mode and is always available.

### Activated by Config Presence

**gitlab** and **github** plugins activate when their config section exists under `extensions:`. Removing the section disables them:

```yaml
extensions:
  gitlab:      # presence of this section activates the plugin
    image: { name: hashicorp/terraform:1.6 }
```

### Active by Default

**summary** is active unless explicitly disabled. It posts plan summaries to MR/PR comments:

```yaml
extensions:
  summary:
    enabled: false   # opt out
```

### Explicitly Enabled

**cost**, **policy**, and **tfupdate** must be explicitly opted in:

```yaml
extensions:
  cost:
    providers:
      aws:
        enabled: true

  policy:
    enabled: true
    sources:
      - path: policies

  tfupdate:
    enabled: true
    policy:
      bump: minor
```

## CI Provider Detection

TerraCi auto-detects the active CI provider at runtime:

1. **`TERRACI_PROVIDER` env var** -- explicit override:
   ```bash
   TERRACI_PROVIDER=gitlab terraci generate -o pipeline.yml
   ```
2. **Environment variables** -- `GITLAB_CI=true` selects GitLab, `GITHUB_ACTIONS=true` selects GitHub
3. **Single active provider** -- if only one CI provider is active, it is used automatically

If multiple providers are configured and no environment is detected, TerraCi returns an error with instructions to set `TERRACI_PROVIDER`.

## Plugin Capabilities

Plugins implement one or more capability interfaces. The framework discovers them at runtime via type assertion:

| Capability | Purpose | Plugins |
|------------|---------|---------|
| `CommandProvider` | CLI subcommands (`terraci cost`, `terraci local-exec`, etc.) | cost, policy, summary, tfupdate, localexec |
| `PipelineContributor` | Inject steps/jobs into pipeline IR | cost, policy, summary, tfupdate |
| `InitContributor` | Form fields for `terraci init` wizard | gitlab, github, cost, policy, summary, tfupdate |
| `PipelineGeneratorFactory` | Create provider-specific pipeline generator (`NewGenerator(ctx, *pipeline.IR)`) | gitlab, github |
| `CommentServiceFactory` | Create MR/PR comment service | gitlab, github |
| `EnvDetector` | Detect CI environment from env vars | gitlab, github |
| `CIInfoProvider` | Provider name, pipeline ID, commit SHA | gitlab, github |
| `ChangeDetectionProvider` | Detect changed modules via VCS diff | git |
| `RuntimeProvider` | Lazy construction of heavy runtime state | cost, policy, tfupdate |
| `Preflightable` | Cheap startup validation before commands run | gitlab, github, git, cost, policy, tfupdate |
| `VersionProvider` | Contribute version info to `terraci version` | policy |
| `KVCacheProvider` | Named key/value cache backend resolution | inmemcache |
| `BlobStoreProvider` | Named blob/object store backend (`NewBlobStore(ctx, appCtx, opts)`) | diskblob |
| `FlagOverridable` | Direct CLI flag overrides (`--plan-only`, `--auto-approve`) | gitlab, github |

A single plugin can implement multiple capabilities. For example, `cost` implements `CommandProvider` (the `terraci cost` command), `PipelineContributor` (adds cost estimation step to pipeline), `InitContributor` (adds toggle to init wizard), `RuntimeProvider` (lazy estimator setup), and `Preflightable` (config validation).

## Plugin Lifecycle

Every plugin goes through the same lifecycle:

```
1. Register    -- init() registers the plugin via registry.RegisterFactory()
2. Configure   -- framework decodes the matching extensions.<key> YAML section
3. Preflight   -- cheap validation (env detection, config checks)
4. Freeze      -- AppContext is frozen, no further config mutations
5. Execute     -- commands lazily build RuntimeProvider runtimes as needed
```

**Preflight** runs for all enabled plugins before any command. It must be fast and side-effect-light -- no network calls, no heavy state. Heavy work (API clients, caches, estimators) belongs in `RuntimeProvider`, which constructs the runtime lazily when a command actually needs it.

## Custom Plugins

### Building with xterraci

`xterraci` produces a custom TerraCi binary with additional or fewer plugins:

```bash
# Add an external plugin
xterraci build --with github.com/your-org/terraci-plugin-slack

# Pin a specific version
xterraci build --with github.com/your-org/terraci-plugin-slack@v1.2.0

# Use a local plugin during development
xterraci build --with github.com/your-org/plugin=../my-plugin

# Remove built-in plugins you don't need
xterraci build --without cost --without policy

# Combine
xterraci build \
  --with github.com/your-org/terraci-plugin-slack \
  --without cost \
  --output ./build/terraci-custom
```

### How It Works

`xterraci build`:
1. Creates a temporary Go module
2. Generates a `main.go` with blank imports of selected plugins
3. Runs `go get` for each external plugin
4. Runs `go build` to produce the binary

The resulting binary is identical to the standard `terraci` but with a different set of plugins compiled in.

### List Built-in Plugins

```bash
xterraci list-plugins
```

### Writing a Plugin

A minimal external plugin needs:

**1. Registration** -- `init()` function that calls `registry.RegisterFactory()`:

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
                PluginDesc: "My custom plugin",
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
    APIKey  string `yaml:"api_key"`
}
```

**2. Capabilities** -- implement the interfaces you need:

```go
// CommandProvider -- adds `terraci myplugin` command
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
    return []*cobra.Command{{
        Use:   "myplugin",
        Short: "Run my custom plugin",
        RunE: func(cmd *cobra.Command, _ []string) error {
            // your logic here
            return nil
        },
    }}
}
```

**3. Go module** -- publish as a Go module with a `go.mod` that depends on `github.com/edelwud/terraci`.

### Plugin File Convention

For larger plugins, follow the one-file-per-capability convention used by built-in plugins:

| File | Contents |
|------|----------|
| `plugin.go` | `init()`, Plugin struct, BasePlugin embedding |
| `lifecycle.go` | `Preflightable` implementation |
| `commands.go` | `CommandProvider` with cobra commands |
| `runtime.go` | `RuntimeProvider` for lazy heavy state |
| `usecases.go` | Command orchestration over typed runtime |
| `pipeline.go` | `PipelineContributor` steps/jobs |
| `init_wizard.go` | `InitContributor` form fields |
| `output.go` | CLI rendering helpers |
| `report.go` | CI report assembly |

### Working Example

See [examples/external-plugin](https://github.com/edelwud/terraci/tree/main/examples/external-plugin) for a complete working example that adds `terraci hello`.

## Configuration Reference

| Plugin | Config page |
|--------|-------------|
| GitLab CI | [config/gitlab](/config/gitlab) |
| GitLab MR | [config/gitlab-mr](/config/gitlab-mr) |
| GitHub Actions | [config/github](/config/github) |
| Cost Estimation | [config/cost](/config/cost) |
| Policy Checks | [config/policy](/config/policy) |
| Dependency Updates | [config/tfupdate](/config/tfupdate) |
