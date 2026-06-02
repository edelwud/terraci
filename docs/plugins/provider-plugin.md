---
title: "CI Provider Plugin"
description: "Add support for new CI systems: Bitbucket Pipelines, Jenkins, CircleCI, Azure DevOps"
outline: deep
---

# CI Provider Plugin

Add support for a new CI system beyond GitLab CI and GitHub Actions. This is the most complex plugin type — it requires implementing multiple interfaces to generate pipeline YAML, detect the CI environment, and optionally post MR/PR comments.

## Use Cases

- **Bitbucket Pipelines** — generate `bitbucket-pipelines.yml`
- **Jenkins** — generate `Jenkinsfile` with parallel stages
- **CircleCI** — generate `.circleci/config.yml`
- **Azure DevOps** — generate `azure-pipelines.yml`

## Required Interfaces

A CI provider must implement at minimum:

| Interface | Purpose |
|-----------|---------|
| `EnvDetector` | Detect if running in your CI environment |
| `CIInfoProvider` | Return provider name, pipeline ID, commit SHA |
| `PipelineGeneratorFactory` | Create an IR-bound generator from the immutable pipeline IR |

Optional:

| Interface | Purpose |
|-----------|---------|
| `CommentServiceFactory` | Create MR/PR comment service (for plan summaries) |

## Environment Detection

Return `true` when running inside your CI system:

```go
func (p *Plugin) DetectEnv() bool {
    return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}
```

TerraCi checks all registered providers. The first one returning `true` is selected.

## CI Info

Provide pipeline context for logging and comment content:

```go
func (p *Plugin) ProviderName() string { return "bitbucket" }
func (p *Plugin) PipelineID() string   { return os.Getenv("BITBUCKET_BUILD_NUMBER") }
func (p *Plugin) CommitSHA() string    { return os.Getenv("BITBUCKET_COMMIT") }
```

## Pipeline Generator

Core builds the IR once via `pipeline.BuildProjectIR(req)`, then asks the
provider for `NewGenerator(ir)`.
Your generator only renders the immutable IR through getters — it does not need
AppContext, depGraph, modules, Terraform runtime config, or contributions
because the IR already encodes all of them.

```go
func (p *Plugin) NewGenerator(ir *pipeline.IR) (pipeline.Generator, error) {
    return &BitbucketGenerator{
        config: p.Config(),
        ir:     ir,
    }, nil
}
```

The generator must implement `pipeline.Generator`:

```go
type Generator interface {
    Generate() (GeneratedPipeline, error)
    DryRun() (*DryRunResult, error)
}

type GeneratedPipeline interface {
    ToYAML() ([]byte, error)
}
```

Provider output should follow the same value-object rule as `pipeline.IR`.
Build a provider-local document through validated constructors/builders, expose
semantic read helpers for tests, and keep `ToYAML()` as the only raw YAML/map
boundary. Avoid one-shot provider document constructors and job-map read APIs;
add jobs through the provider document builder.

### Working with the Pipeline IR

The IR is a flat DAG value object. Every executable item is a `pipeline.Job`;
providers render jobs in declaration order and use `pipeline.Schedule` only
when their CI needs barrier groups, such as GitLab stages. `Schedule` returns
read-only value groups, so providers should use `group.Name()` and
`group.Jobs()` instead of storing mutable job pointers:

```go
func (g *BitbucketGenerator) Generate() (pipeline.GeneratedPipeline, error) {
    for _, job := range g.ir.Jobs() {
        // job.Kind() — plan, apply, or command
        // job.Module() — module metadata for plan/apply jobs
        // job.Dependencies() — required control edges
        // job.InputArtifacts() — artifacts to restore from producer jobs
        // job.Operation() — typed payload; render via cishell.RenderOperation for shell-driven CI
    }

    return renderBitbucketYAML(g.ir), nil
}
```

## Comment Service (Optional)

Post plan summaries to your CI system's PR/MR:

```go
func (p *Plugin) NewCommentService(ctx *plugin.AppContext) ci.CommentService {
    cfg := p.Config()
    if cfg.PR == nil || cfg.PR.Comment == nil || !cfg.PR.Comment.Enabled {
        return &ci.NoOpCommentService{}
    }
    return &BitbucketCommentService{
        apiToken: os.Getenv("BITBUCKET_TOKEN"),
        repoSlug: os.Getenv("BITBUCKET_REPO_SLUG"),
        prID:     os.Getenv("BITBUCKET_PR_ID"),
    }
}
```

The `CommentService` interface:

```go
type CommentService interface {
    IsEnabled() bool
    UpsertComment(ctx context.Context, body string) error
}
```

## Full Skeleton

```go
package bitbucket

import (
    "os"

    "github.com/edelwud/terraci/pkg/ci"
    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.RegisterFactory(func() plugin.Plugin {
        return &Plugin{
            BasePlugin: plugin.BasePlugin[*Config]{
                PluginName: "bitbucket",
                PluginDesc: "Bitbucket Pipelines generation",
                EnableMode: plugin.EnabledWhenConfigured,
                DefaultCfg: func() *Config { return &Config{} },
            },
        }
    })
}

type Plugin struct{ plugin.BasePlugin[*Config] }

type Config struct {
    Image    string `yaml:"image"`
}

func (c *Config) Clone() *Config {
    if c == nil {
        return nil
    }
    out := *c
    return &out
}

// --- EnvDetector ---

func (p *Plugin) DetectEnv() bool {
    return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}

// --- CIInfoProvider ---

func (p *Plugin) ProviderName() string { return "bitbucket" }
func (p *Plugin) PipelineID() string   { return os.Getenv("BITBUCKET_BUILD_NUMBER") }
func (p *Plugin) CommitSHA() string    { return os.Getenv("BITBUCKET_COMMIT") }

// --- PipelineGeneratorFactory ---

func (p *Plugin) NewGenerator(ir *pipeline.IR) (pipeline.Generator, error) {
    return &generator{config: p.Config(), ir: ir}, nil
}

type generator struct {
    config *Config
    ir     *pipeline.IR
}

func (g *generator) Generate() (pipeline.GeneratedPipeline, error) {
    // Transform IR to bitbucket-pipelines.yml format
    return renderBitbucketPipeline(g.ir, g.config), nil
}

func (g *generator) DryRun() (*pipeline.DryRunResult, error) {
    return g.ir.DryRun(countModules(g.ir)), nil
}

// --- CommentServiceFactory (optional) ---

func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
    // Implement Bitbucket PR comment service
    return nil
}
```

## Provider Resolution

TerraCi resolves the active CI provider in this order:

1. **`TERRACI_PROVIDER` env var** — explicit override
2. **Environment detection** — `DetectEnv()` returns `true`
3. **Single active provider** — only one provider is active

Your provider is automatically discovered. No core code changes needed.

## See Also

- [Pipeline Job Plugin](/plugins/pipeline-plugin) — add DAG jobs without building a full provider
- [Pipeline Generation Guide](/guide/pipeline-generation) — how the IR works
- Built-in [GitLab](/config/gitlab) and [GitHub](/config/github) providers as reference
