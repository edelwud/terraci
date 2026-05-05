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
| `PipelineGeneratorFactory` | Create a pipeline generator that transforms IR → YAML |

Optional:

| Interface | Purpose |
|-----------|---------|
| `CommentServiceFactory` | Create MR/PR comment service (for plan summaries) |
| `FlagOverridable` | Support `--plan-only` and `--auto-approve` CLI flags |

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

Core builds the IR once via `pipeline.Build(opts)` and passes it to your factory.
Your generator just renders the IR — it does not need depGraph, modules, or
contributions because the IR already encodes all of them.

```go
func (p *Plugin) NewGenerator(ctx *plugin.AppContext, ir *pipeline.IR) pipeline.Generator {
    return &BitbucketGenerator{
        config: p.Config(),
        ir:     ir,
    }
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

### Working with the Pipeline IR

The IR contains execution levels with module jobs and contributed plugin jobs:

```go
func (g *BitbucketGenerator) Generate() (pipeline.GeneratedPipeline, error) {
    // g.ir.Levels — ordered groups of parallel module jobs
    for _, level := range g.ir.Levels {
        for _, mj := range level.Modules {
            // mj.Module.Path — "platform/prod/eu-central-1/vpc"
            // mj.Plan — *Job (nil if plan disabled)
            // mj.Apply — *Job (nil if plan-only mode)
            // Each Job has: Name, Operation, Dependencies, Steps, Env
        }
    }

    // g.ir.Jobs — contributed jobs from plugins (cost, policy, summary, etc.)
    for _, job := range g.ir.Jobs {
        // job.Name — "cost-estimation", "policy-check", etc.
        // job.Phase — determines stage name
        // job.Dependencies — job names this depends on
        // job.Operation — typed payload; render via cishell.RenderOperation for shell-driven CI
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

## Flag Overrides (Optional)

Implement `FlagOverridable` to support `--plan-only` and `--auto-approve` CLI flags on `terraci generate`:

```go
func (p *Plugin) SetPlanOnly(v bool) {
    if cfg := p.Config(); cfg != nil {
        cfg.PlanOnly = v
    }
}

func (p *Plugin) SetAutoApprove(v bool) {
    if cfg := p.Config(); cfg != nil {
        cfg.AutoApprove = v
    }
}
```

These methods are called directly by the framework when the user passes `--plan-only` or `--auto-approve` to `terraci generate`. The config struct is mutated before pipeline generation begins.

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
    Image       string `yaml:"image"`
    PlanOnly    bool   `yaml:"plan_only"`
    AutoApprove bool   `yaml:"auto_approve"`
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

func (p *Plugin) NewGenerator(ctx *plugin.AppContext, ir *pipeline.IR) pipeline.Generator {
    return &generator{config: p.Config(), ir: ir}
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

- [Pipeline Step Plugin](/plugins/pipeline-plugin) — inject steps without building a full provider
- [Pipeline Generation Guide](/guide/pipeline-generation) — how the IR works
- Built-in [GitLab](/config/gitlab) and [GitHub](/config/github) providers as reference
