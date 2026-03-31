---
title: "Pipeline Step Plugin"
description: "Inject custom jobs and steps into generated CI pipelines: security scans, approval gates, post-deploy hooks"
outline: deep
---

# Pipeline Step Plugin

Inject custom jobs or steps into the generated CI pipeline. Your plugin's contributions are merged with the built-in pipeline IR and appear in the final GitLab CI / GitHub Actions output.

## Use Cases

- **Security scans** — run tfsec, checkov, or Snyk after plan
- **Approval gates** — require external approval before apply
- **Post-deploy hooks** — trigger smoke tests after apply
- **Notifications** — Slack/Teams messages at specific pipeline phases
- **Summary/cleanup** — aggregate results after all jobs complete

## How It Works

The pipeline is built in phases:

```
PrePlan → Plan → PostPlan → PreApply → Apply → PostApply → Finalize
```

Your plugin can contribute:
- **Steps** — injected into existing plan/apply jobs at a specific phase
- **Jobs** — standalone jobs that run independently

## Pipeline Phases

| Phase | Constant | When | Typical Use |
|-------|----------|------|-------------|
| Pre-Plan | `pipeline.PhasePrePlan` | Before `terraform plan` | Setup, auth, cache warm |
| Post-Plan | `pipeline.PhasePostPlan` | After `terraform plan` | Security scans, policy checks, cost |
| Pre-Apply | `pipeline.PhasePreApply` | Before `terraform apply` | Approval checks, lock acquisition |
| Post-Apply | `pipeline.PhasePostApply` | After `terraform apply` | Smoke tests, notifications, cleanup |
| Finalize | `pipeline.PhaseFinalize` | After all other jobs | Summary comments, aggregated reports |

::: tip Finalize Phase
Jobs in `PhaseFinalize` automatically depend on all other contributed jobs. They always run last. The built-in `summary` plugin uses this phase to post MR/PR comments after all plan and policy jobs complete.
:::

## Contributing Steps

Steps are injected into every plan or apply job. Each step has a single command:

```go
import "github.com/edelwud/terraci/pkg/pipeline"

func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    cfg := p.Config()
    if cfg == nil || !cfg.Pipeline {
        return nil
    }

    return &pipeline.Contribution{
        Steps: []pipeline.Step{
            {
                Phase:   pipeline.PhasePostPlan,
                Name:    "tfsec",
                Command: "tfsec --format json --out tfsec-report.json .",
            },
        },
    }
}
```

### How Steps Are Filtered

The framework filters steps by phase when building jobs:
- **Plan jobs** receive `PhasePrePlan` + `PhasePostPlan` steps
- **Apply jobs** receive `PhasePreApply` + `PhasePostApply` steps
- `PhaseFinalize` steps are **not injected** into module jobs — use contributed jobs instead

## Contributing Jobs

Standalone jobs run as separate CI jobs:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:          "security-scan",
                Phase:         pipeline.PhasePostPlan,
                DependsOnPlan: true,
                Commands: []string{
                    "checkov -d . --output json > checkov-report.json",
                },
            },
        },
    }
}
```

### ContributedJob Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Job name in generated pipeline |
| `Phase` | `Phase` | Determines stage name (`Phase.String()`) |
| `Commands` | `[]string` | Shell commands to run |
| `DependsOnPlan` | `bool` | If `true`, depends on all plan jobs |
| `ArtifactPaths` | `[]string` | Paths to collect as CI artifacts |
| `AllowFailure` | `bool` | If `true`, job failure doesn't fail the pipeline |

### Finalize Jobs

For jobs that must run after everything else (summaries, cleanup), use `PhaseFinalize`:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    return &pipeline.Contribution{
        Jobs: []pipeline.ContributedJob{
            {
                Name:          "my-summary",
                Phase:         pipeline.PhaseFinalize,
                DependsOnPlan: true,
                Commands:      []string{"terraci my-summary"},
            },
        },
    }
}
```

`PhaseFinalize` jobs automatically depend on all other contributed jobs (cost, policy, etc.). You don't need to specify these dependencies manually.

## Combining Steps and Jobs

A single plugin can contribute both:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    return &pipeline.Contribution{
        Steps: []pipeline.Step{
            {
                Phase:   pipeline.PhasePostPlan,
                Command: "echo 'Plan complete for $TF_MODULE_PATH'",
            },
        },
        Jobs: []pipeline.ContributedJob{
            {
                Name:          "security-report",
                Phase:         pipeline.PhasePostPlan,
                DependsOnPlan: true,
                Commands:      []string{"aggregate-reports.sh"},
            },
        },
    }
}
```

## Framework Filtering

You don't need to check `IsEnabled()` in `PipelineContribution` — the framework only calls it for enabled plugins. Return `nil` if your plugin has nothing to contribute:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    if cfg := p.Config(); cfg == nil || !cfg.Pipeline {
        return nil
    }
    // ...
}
```

## Generated Output

Your contributed steps appear in the generated YAML:

**GitLab CI:**
```yaml
plan-vpc:
  stage: deploy-plan-0
  script:
    - cd platform/prod/eu-central-1/vpc
    - terraform init
    - terraform plan -out=plan.tfplan
    - tfsec --format json --out tfsec-report.json .  # ← PostPlan step

security-scan:
  stage: post-plan
  needs: [plan-vpc, plan-eks, plan-rds]               # ← DependsOnPlan: true
  script:
    - checkov -d . --output json > checkov-report.json
```

**GitHub Actions:**
```yaml
plan-vpc:
  steps:
    - run: terraform init
    - run: terraform plan -out=plan.tfplan
    - run: tfsec --format json --out tfsec-report.json .  # ← PostPlan step
```

## Configuration

Add a `pipeline` toggle so users can opt out of pipeline contributions:

```go
type Config struct {
    Enabled  bool `yaml:"enabled"`
    Pipeline bool `yaml:"pipeline"`
}
```

```yaml
plugins:
  security:
    enabled: true
    pipeline: true   # inject steps into CI pipeline
```

## Full Example

```go
package security

import (
    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
    "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
    registry.Register(&Plugin{
        BasePlugin: plugin.BasePlugin[*Config]{
            PluginName:  "security",
            PluginDesc:  "Security scanning for Terraform plans",
            EnableMode:  plugin.EnabledExplicitly,
            DefaultCfg:  func() *Config { return &Config{} },
            IsEnabledFn: func(cfg *Config) bool { return cfg != nil && cfg.Enabled },
        },
    })
}

type Plugin struct{ plugin.BasePlugin[*Config] }

type Config struct {
    Enabled  bool   `yaml:"enabled"`
    Pipeline bool   `yaml:"pipeline"`
    Tool     string `yaml:"tool"` // tfsec, checkov, snyk
}

func (p *Plugin) PipelineContribution(_ *plugin.AppContext) *pipeline.Contribution {
    cfg := p.Config()
    if cfg == nil || !cfg.Pipeline {
        return nil
    }

    tool := cfg.Tool
    if tool == "" {
        tool = "tfsec"
    }

    return &pipeline.Contribution{
        Steps: []pipeline.Step{
            {
                Phase:   pipeline.PhasePostPlan,
                Command: tool + " --format json .",
            },
        },
    }
}
```

## See Also

- [CLI Command Plugin](/plugins/command-plugin) — add CLI commands
- [Pipeline Generation Guide](/guide/pipeline-generation) — how the pipeline IR works
