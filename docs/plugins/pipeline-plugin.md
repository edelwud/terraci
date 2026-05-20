---
title: "Pipeline Job Plugin"
description: "Add standalone resource-aware jobs to generated CI pipelines: policy, cost, summaries, update checks"
outline: deep
---

# Pipeline Job Plugin

Pipeline plugins add standalone DAG jobs to the provider-agnostic pipeline IR.
Jobs declare the resources they read and write; TerraCi derives dependencies,
artifact restore, and detailed plan output from those typed resource requests.

## Use Cases

- **Policy checks** — consume `plan.json`, publish a policy report
- **Cost estimation** — consume `plan.json`, publish a cost report
- **Dependency updates** — publish result and report artifacts
- **Summaries** — consume plan JSON and optional plugin reports, then post a PR/MR comment

## How It Works

There are no pipeline phases and no injected plan/apply steps. The pipeline is
a DAG:

1. Core creates Terraform plan/apply jobs for target modules.
2. Plugins contribute standalone jobs.
3. Each job declares `Consumes` and `Produces` resources.
4. `pipeline.Build` resolves concrete resources, artifact transfer, and job dependencies.
5. Providers render the finished IR without knowing plugin semantics.

## Contributing Jobs

```go
import (
    "github.com/edelwud/terraci/pkg/ci"
    "github.com/edelwud/terraci/pkg/pipeline"
    "github.com/edelwud/terraci/pkg/plugin"
)

func (p *Plugin) PipelineContribution(ctx *plugin.AppContext) (*pipeline.Contribution, error) {
    serviceDir := ctx.Config().ServiceDir()

    job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
        Name:     "security-scan",
        Commands: []string{"terraci security check"},
        Consumes: []pipeline.ResourceRequest{
            pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
        },
        Produces: []pipeline.ResourceSpec{
            pipeline.PluginResource(
                pipeline.ResourceKindPluginReport,
                "security",
                pipeline.WorkspacePath(serviceDir, ci.ReportFilename("security")),
            ),
        },
    })
    if err != nil {
        return nil, err
    }
    contribution, err := pipeline.NewContribution(job)
    if err != nil {
        return nil, err
    }
    return contribution, nil
}
```

### ContributedJob Options And Getters

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Job name in generated pipeline |
| `Commands` | `[]string` | Shell commands to run |
| `Dependencies` | `[]JobDependency` | Optional explicit control dependencies |
| `Consumes` | `[]ResourceRequest` | Typed resources to restore before the job runs |
| `Produces` | `[]ResourceSpec` | Typed resources published by the job |
| `AllowFailure` | `bool` | If `true`, job failure does not fail the pipeline |

Use `pipeline.NewPluginCommandJob` or `pipeline.NewContributedJob` to build the
job, then `pipeline.NewContribution(job)` to return it. Consumers inspect jobs
through `Contribution.Jobs()` and `ContributedJob` getters.
Return builder errors directly; returning `nil, nil` is invalid. Optional jobs
should opt out through `plugin.PipelineContributionGate`.

## Resource Requests

Plan resources are module-scoped:

```go
pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON)
pipeline.ModulePlanResource(pipeline.ResourceKindPlanText, "platform/prod/eu-central-1/vpc")
```

Plugin resources are producer-scoped:

```go
pipeline.AllPluginResources(pipeline.ResourceKindPluginReport, true) // optional
pipeline.PluginProducerResource(pipeline.ResourceKindPluginResult, "policy", false)
```

Requesting `PlanJSON` or `PlanText` automatically makes only the matching
module plan jobs detailed. A job that consumes `PlanJSON` depends on the plan
jobs that produce it and receives their artifacts restored to workspace-relative
paths.

## Summary-Style Jobs

A summary job does not need a special phase. It lands at the end of the DAG
because it consumes resources produced by earlier jobs:

```go
func (p *Plugin) PipelineContribution(_ *plugin.AppContext) (*pipeline.Contribution, error) {
    job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
        Name:     "terraci-summary",
        Commands: []string{"terraci summary"},
        Consumes: []pipeline.ResourceRequest{
            pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
            pipeline.AllPluginResources(pipeline.ResourceKindPluginReport, true),
        },
    })
    if err != nil {
        return nil, err
    }
    contribution, err := pipeline.NewContribution(job)
    if err != nil {
        return nil, err
    }
    return contribution, nil
}
```

## Activation

The registry calls `PipelineContribution` only for enabled plugin configs. If
your plugin has an extra pipeline toggle, implement `plugin.PipelineContributionGate`:

```go
func (p *Plugin) PipelineContributionEnabled(_ *plugin.AppContext) (bool, error) {
    cfg := p.Config()
    return cfg != nil && cfg.Pipeline, nil
}
```

## Generated Output

Providers render DAG layers using their native mechanics. For example, GitLab
uses stages derived from `pipeline.Schedule`, while GitHub renders job `needs`.

```yaml
security-scan:
  stage: deploy-2
  needs:
    - job: plan-platform-prod-eu-central-1-vpc
      artifacts: true
  script:
    - terraci security check
  artifacts:
    paths:
      - .terraci/security-report.json
```

## See Also

- [CI Provider Plugin](/plugins/provider-plugin) — render the IR for a new CI system
- [Pipeline Generation Guide](/guide/pipeline-generation) — how generated pipelines are built
