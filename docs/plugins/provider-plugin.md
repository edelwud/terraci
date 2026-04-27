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
| `CIMetadata` | Return provider name, pipeline ID, commit SHA |
| `GeneratorFactory` | Create a pipeline generator that transforms IR → YAML |

Optional:

| Interface | Purpose |
|-----------|---------|
| `CommentFactory` | Create MR/PR comment service (for plan summaries) |
| `FlagOverridable` | Support `--plan-only` and `--auto-approve` CLI flags |

## Environment Detection

Return `true` when running inside your CI system:

```go
func (p *Plugin) DetectEnv() bool {
    return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}
```

TerraCi checks all registered providers. The first one returning `true` is selected.

## CI Metadata

Provide pipeline context for logging and comment content:

```go
func (p *Plugin) ProviderName() string { return "bitbucket" }
func (p *Plugin) PipelineID() string   { return os.Getenv("BITBUCKET_BUILD_NUMBER") }
func (p *Plugin) CommitSHA() string    { return os.Getenv("BITBUCKET_COMMIT") }
```

## Pipeline Generator

The generator receives the provider-agnostic pipeline IR and transforms it to your CI format:

```go
func (p *Plugin) NewGenerator(
    ctx *plugin.AppContext,
    depGraph *graph.DependencyGraph,
    modules []*discovery.Module,
) pipeline.Generator {
    return &BitbucketGenerator{
        config:   p.Config(),
        depGraph: depGraph,
        modules:  modules,
    }
}
```

The generator must implement `pipeline.Generator`:

```go
type Generator interface {
    Generate(ir *IR) (*GeneratedPipeline, error)
}

type GeneratedPipeline struct {
    Content []byte // the generated YAML
}
```

### Working with the Pipeline IR

The IR contains execution levels with module jobs and contributed plugin jobs:

```go
func (g *BitbucketGenerator) Generate(ir *pipeline.IR) (*pipeline.GeneratedPipeline, error) {
    // ir.Levels — ordered groups of parallel module jobs
    for _, level := range ir.Levels {
        for _, mj := range level.Modules {
            // mj.Module.Path — "platform/prod/eu-central-1/vpc"
            // mj.Plan — *Job (nil if plan disabled)
            // mj.Apply — *Job (nil if plan-only mode)
            // Each Job has: Name, Script, Dependencies, Steps, Env
        }
    }

    // ir.Jobs — contributed jobs from plugins (cost, policy, summary, etc.)
    for _, job := range ir.Jobs {
        // job.Name — "cost-estimation", "policy-check", etc.
        // job.Phase — determines stage name
        // job.Dependencies — job names this depends on
        // job.Script — commands to run
    }

    content := renderBitbucketYAML(ir)
    return &pipeline.GeneratedPipeline{Content: content}, nil
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
        if v {
            cfg.PlanEnabled = true
        }
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
    "github.com/edelwud/terraci/pkg/discovery"
    "github.com/edelwud/terraci/pkg/graph"
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
    Image           string `yaml:"image"`
    TerraformBinary string `yaml:"terraform_binary"`
    PlanEnabled     bool   `yaml:"plan_enabled"`
    AutoApprove     bool   `yaml:"auto_approve"`
}

// --- EnvDetector ---

func (p *Plugin) DetectEnv() bool {
    return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}

// --- CIMetadata ---

func (p *Plugin) ProviderName() string { return "bitbucket" }
func (p *Plugin) PipelineID() string   { return os.Getenv("BITBUCKET_BUILD_NUMBER") }
func (p *Plugin) CommitSHA() string    { return os.Getenv("BITBUCKET_COMMIT") }

// --- GeneratorFactory ---

func (p *Plugin) NewGenerator(
    ctx *plugin.AppContext,
    depGraph *graph.DependencyGraph,
    modules []*discovery.Module,
) pipeline.Generator {
    return &generator{config: p.Config(), depGraph: depGraph, modules: modules}
}

type generator struct {
    config   *Config
    depGraph *graph.DependencyGraph
    modules  []*discovery.Module
}

func (g *generator) Generate(ir *pipeline.IR) (*pipeline.GeneratedPipeline, error) {
    // Transform IR to bitbucket-pipelines.yml format
    // This is where you implement your CI-specific YAML generation
    content := renderBitbucketPipeline(ir, g.config)
    return &pipeline.GeneratedPipeline{Content: content}, nil
}

// --- CommentFactory (optional) ---

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
